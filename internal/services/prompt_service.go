package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/dataset/search"
	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/logging"
	"github.com/kidixdev/PromptSensei/internal/prompting"
)

type PromptService struct {
	mu           sync.RWMutex
	cfg          config.Config
	dataset      *DatasetService
	instructions *InstructionService
	knowledge    *KnowledgeService
	providers    *ProviderService
}

func NewPromptService(
	cfg config.Config,
	dataset *DatasetService,
	instructions *InstructionService,
	knowledge *KnowledgeService,
	providers *ProviderService,
) *PromptService {
	return &PromptService{
		cfg:          cfg,
		dataset:      dataset,
		instructions: instructions,
		knowledge:    knowledge,
		providers:    providers,
	}
}

func (s *PromptService) UpdateConfig(cfg config.Config) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
}

func (s *PromptService) configSnapshot() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *PromptService) Enhance(ctx context.Context, req domain.EnhanceRequest) (*domain.EnhanceResult, []string, error) {
	cfg := s.configSnapshot()

	mode := req.Mode
	if mode == "" {
		mode = cfg.General.DefaultMode
	}
	logging.Debug("enhance start",
		"mode", mode,
		"create_mode", req.CreateMode,
		"strict", req.StrictBooru,
		"provider_enabled", s.providers.Enabled(),
		"prompt", req.Prompt,
		"knowledge_files", strings.Join(req.KnowledgeFiles, ", "),
	)

	repo, err := s.dataset.OpenRepository()
	if err != nil {
		logging.Error("open dataset repository failed", "error", err)
		return nil, nil, err
	}
	defer repo.Close()

	retriever := search.NewRetriever(repo)
	retrieval, err := retriever.Retrieve(ctx, req.Prompt, mode)
	if err != nil {
		logging.Error("retrieval failed", "error", err)
		return nil, nil, err
	}
	logging.Debug("retrieval done",
		"confirmed", len(retrieval.ConfirmedTags),
		"character", len(retrieval.CharacterTags),
		"suggested", len(retrieval.SuggestedTags),
		"rejected", len(retrieval.RejectedTags),
	)

	persona, err := s.instructions.LoadPersona()
	if err != nil {
		logging.Error("load persona failed", "error", err)
		return nil, nil, err
	}
	systemRules, err := s.instructions.LoadSystem()
	if err != nil {
		logging.Error("load system rules failed", "error", err)
		return nil, nil, err
	}

	knowledgeDocs, warnings := s.knowledge.LoadSelected(req.KnowledgeFiles)
	assembled := prompting.Assemble(prompting.AssemblyInput{
		Mode:        mode,
		SystemRules: systemRules,
		Persona:     persona,
		UserPrompt:  req.Prompt,
		Knowledge:   knowledgeDocs,
		Retrieval:   retrieval,
		CreateMode:  req.CreateMode,
	})

	finalOutput := deterministicFallback(mode, req.Prompt, retrieval)
	usedProvider := false
	providerName := s.providers.Name()
	chainApplied := false
	chainStages := 0

	if s.providers.Enabled() {
		chainOutput, chainProvider, chainWarnings, err := s.generateWithPromptChain(ctx, cfg, req, mode, retrieval, assembled)
		warnings = append(warnings, chainWarnings...)
		if err != nil {
			logging.Error("prompt chain failed", "error", err)
			return nil, warnings, err
		}
		if strings.TrimSpace(chainOutput) != "" {
			finalOutput = strings.TrimSpace(chainOutput)
			usedProvider = true
			chainApplied = true
			chainStages = 3
			if strings.TrimSpace(chainProvider) != "" {
				providerName = chainProvider
			}
		}
	}

	validationApplied := false
	if (req.StrictBooru || cfg.General.StrictBooruValidation) && mode == domain.ModeBooru {
		finalOutput = prompting.FilterBooruOutput(finalOutput, retrieval)
		validationApplied = true
		logging.Debug("strict booru validation applied")
	}
	logging.Info("enhance complete",
		"mode", mode,
		"provider", providerName,
		"provider_used", usedProvider,
		"chain_applied", chainApplied,
		"chain_stages", chainStages,
		"validation_applied", validationApplied,
		"warnings", len(warnings),
		"output", finalOutput,
	)

	return &domain.EnhanceResult{
		Output:            finalOutput,
		Retrieval:         retrieval,
		SystemPrompt:      assembled.SystemPrompt,
		UserPrompt:        assembled.UserPrompt,
		ProviderName:      providerName,
		UsedProvider:      usedProvider,
		ValidationApplied: validationApplied,
		ChainApplied:      chainApplied,
		ChainStages:       chainStages,
	}, warnings, nil
}

type promptChainPlan struct {
	Intent         string   `json:"intent"`
	Subject        string   `json:"subject"`
	Style          string   `json:"style"`
	Composition    []string `json:"composition"`
	Lighting       []string `json:"lighting"`
	MustInclude    []string `json:"must_include_tags"`
	Avoid          []string `json:"avoid_tags"`
	QualitySignals []string `json:"quality_signals"`
}

func (s *PromptService) generateWithPromptChain(
	ctx context.Context,
	cfg config.Config,
	req domain.EnhanceRequest,
	mode domain.Mode,
	retrieval domain.RetrievalResult,
	assembled prompting.AssemblyOutput,
) (string, string, []string, error) {
	var warnings []string
	providerName := s.providers.Name()

	plannerTemp := math.Min(cfg.Provider.Temperature, 0.4)
	plannerMaxTokens := clampInt(cfg.Provider.MaxTokens/2, 220, 480)
	plannerSystemPrompt := promptChainPlannerSystemPrompt(mode)
	plannerUserPrompt := promptChainPlannerUserPrompt(req.Prompt, mode, retrieval, req.CreateMode)
	logPromptStageRequest("planner", cfg.Provider.Model, plannerTemp, plannerMaxTokens, plannerSystemPrompt, plannerUserPrompt)
	planResp, err := s.providers.Generate(ctx, domain.GenerateRequest{
		SystemPrompt: plannerSystemPrompt,
		UserPrompt:   plannerUserPrompt,
		Model:        cfg.Provider.Model,
		Temperature:  plannerTemp,
		MaxTokens:    plannerMaxTokens,
	})
	if err != nil {
		logging.Error("planner stage failed", "error", err)
		return "", providerName, warnings, err
	}
	if planResp != nil && strings.TrimSpace(planResp.Provider) != "" {
		providerName = planResp.Provider
	}

	plan, err := parsePromptChainPlan(responseText(planResp))
	if err != nil {
		plan = fallbackPromptChainPlan(req.Prompt, retrieval, req.CreateMode)
		warnings = append(warnings, "planning stage returned non-JSON output; using fallback plan")
		logging.Warn("planner returned non-json output; fallback used", "output", responseText(planResp))
	}
	planJSON, _ := json.MarshalIndent(plan, "", "  ")
	logging.Debug("planner stage output", "provider", providerName, "plan", string(planJSON))

	draftMaxTokens := clampInt(cfg.Provider.MaxTokens, 280, 1400)
	draftSystemPrompt := promptChainDraftSystemPrompt(assembled.SystemPrompt)
	draftUserPrompt := promptChainDraftUserPrompt(assembled.UserPrompt, planJSON)
	logPromptStageRequest("draft", cfg.Provider.Model, cfg.Provider.Temperature, draftMaxTokens, draftSystemPrompt, draftUserPrompt)
	draftResp, err := s.providers.Generate(ctx, domain.GenerateRequest{
		SystemPrompt: draftSystemPrompt,
		UserPrompt:   draftUserPrompt,
		Model:        cfg.Provider.Model,
		Temperature:  cfg.Provider.Temperature,
		MaxTokens:    draftMaxTokens,
	})
	if err != nil {
		logging.Error("draft stage failed", "error", err)
		return "", providerName, warnings, err
	}
	if draftResp != nil && strings.TrimSpace(draftResp.Provider) != "" {
		providerName = draftResp.Provider
	}
	draft := strings.TrimSpace(responseText(draftResp))
	logging.Debug("draft stage output", "provider", providerName, "output", draft)
	if draft == "" {
		draft = deterministicFallback(mode, req.Prompt, retrieval)
		warnings = append(warnings, "draft stage returned empty text; using deterministic fallback")
		logging.Warn("draft stage empty output; deterministic fallback used", "fallback", draft)
	}

	refineMaxTokens := clampInt(cfg.Provider.MaxTokens, 280, 1400)
	refineSystemPrompt := promptChainRefineSystemPrompt(assembled.SystemPrompt)
	refineUserPrompt := promptChainRefineUserPrompt(req.Prompt, mode, retrieval, planJSON, draft)
	logPromptStageRequest("refine", cfg.Provider.Model, 0.25, refineMaxTokens, refineSystemPrompt, refineUserPrompt)
	refineResp, err := s.providers.Generate(ctx, domain.GenerateRequest{
		SystemPrompt: refineSystemPrompt,
		UserPrompt:   refineUserPrompt,
		Model:        cfg.Provider.Model,
		Temperature:  0.25,
		MaxTokens:    refineMaxTokens,
	})
	if err != nil {
		warnings = append(warnings, "refinement stage failed; using draft output")
		logging.Warn("refine stage failed; using draft output", "error", err, "draft", draft)
		return draft, providerName, warnings, nil
	}
	if refineResp != nil && strings.TrimSpace(refineResp.Provider) != "" {
		providerName = refineResp.Provider
	}
	refined := strings.TrimSpace(responseText(refineResp))
	logging.Debug("refine stage output", "provider", providerName, "output", refined)
	if refined == "" {
		warnings = append(warnings, "refinement stage returned empty text; using draft output")
		logging.Warn("refine stage empty output; using draft", "draft", draft)
		return draft, providerName, warnings, nil
	}

	return refined, providerName, warnings, nil
}

func promptChainPlannerSystemPrompt(mode domain.Mode) string {
	return strings.TrimSpace(fmt.Sprintf(`You are a prompt-planning engine for image generation prompts.
Mode: %s

Return exactly one compact JSON object with these keys:
- intent (string)
- subject (string)
- style (string)
- composition (array of strings)
- lighting (array of strings)
- must_include_tags (array of booru-style tags)
- avoid_tags (array of booru-style tags)
- quality_signals (array of strings)

Rules:
- Output JSON only.
- Do not use markdown.
- Keep arrays concise and non-redundant.
- must_include_tags should focus on high-confidence tags only.`, mode))
}

func promptChainPlannerUserPrompt(prompt string, mode domain.Mode, retrieval domain.RetrievalResult, createMode bool) string {
	intent := "enhance"
	if createMode {
		intent = "create"
	}
	return strings.TrimSpace(fmt.Sprintf(`Task: %s
Mode: %s
User input:
%s

Retrieval digest:
%s`, intent, mode, strings.TrimSpace(prompt), retrievalDigest(retrieval)))
}

func promptChainDraftSystemPrompt(baseSystem string) string {
	return strings.TrimSpace(baseSystem + `

Prompt chain stage:
- You are generating a first-pass prompt candidate from a structured plan.
- Follow the requested mode exactly.
- Return only the draft prompt text.`)
}

func promptChainDraftUserPrompt(baseUser string, planJSON []byte) string {
	return strings.TrimSpace(baseUser + "\n\nStructured plan (JSON):\n" + string(planJSON) + "\n\nReturn only the draft prompt.")
}

func promptChainRefineSystemPrompt(baseSystem string) string {
	return strings.TrimSpace(baseSystem + `

Prompt chain stage:
- You are a strict prompt refiner.
- Resolve conflicting tags and remove redundancy.
- Keep the output compact and generation-ready.
- Return only the final prompt text.`)
}

func promptChainRefineUserPrompt(prompt string, mode domain.Mode, retrieval domain.RetrievalResult, planJSON []byte, draft string) string {
	return strings.TrimSpace(fmt.Sprintf(`Mode: %s
Original user input:
%s

Retrieval digest:
%s

Plan (JSON):
%s

Draft candidate:
%s

Return only the final improved prompt.`, mode, strings.TrimSpace(prompt), retrievalDigest(retrieval), string(planJSON), strings.TrimSpace(draft)))
}

func parsePromptChainPlan(text string) (promptChainPlan, error) {
	raw := strings.TrimSpace(text)
	if raw == "" {
		return promptChainPlan{}, fmt.Errorf("empty planner output")
	}

	raw = extractFirstJSONObject(raw)
	if raw == "" {
		return promptChainPlan{}, fmt.Errorf("planner output does not contain a json object")
	}

	var plan promptChainPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return promptChainPlan{}, err
	}
	plan.Intent = strings.TrimSpace(plan.Intent)
	plan.Subject = strings.TrimSpace(plan.Subject)
	plan.Style = strings.TrimSpace(plan.Style)
	plan.Composition = cleanStringSlice(plan.Composition, false)
	plan.Lighting = cleanStringSlice(plan.Lighting, false)
	plan.MustInclude = cleanStringSlice(plan.MustInclude, true)
	plan.Avoid = cleanStringSlice(plan.Avoid, true)
	plan.QualitySignals = cleanStringSlice(plan.QualitySignals, false)

	return plan, nil
}

func fallbackPromptChainPlan(prompt string, retrieval domain.RetrievalResult, createMode bool) promptChainPlan {
	intent := "Enhance the original prompt while preserving core intent."
	if createMode {
		intent = "Create a high-quality prompt from the user idea."
	}
	return promptChainPlan{
		Intent:         intent,
		Subject:        strings.TrimSpace(prompt),
		Style:          "anime illustration",
		Composition:    []string{"clear subject focus", "balanced framing"},
		Lighting:       []string{"cinematic lighting"},
		MustInclude:    topTagNames(retrieval.CharacterTags, 6),
		Avoid:          topRejectedTagNames(retrieval.RejectedTags, 4),
		QualitySignals: []string{"coherent details", "non-conflicting tags"},
	}
}

func retrievalDigest(retrieval domain.RetrievalResult) string {
	return strings.Join([]string{
		"character_tags: " + strings.Join(topTagNames(retrieval.CharacterTags, 12), ", "),
		"confirmed_tags: " + strings.Join(topTagNames(retrieval.ConfirmedTags, 16), ", "),
		"suggested_tags: " + strings.Join(topTagNames(retrieval.SuggestedTags, 18), ", "),
		"rejected_tags: " + strings.Join(topRejectedNames(retrieval.RejectedTags, 8), ", "),
	}, "\n")
}

func topTagNames(tags []domain.TagCandidate, maxCount int) []string {
	if len(tags) == 0 || maxCount <= 0 {
		return []string{"(none)"}
	}
	if len(tags) > maxCount {
		tags = tags[:maxCount]
	}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if name != "" {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return []string{"(none)"}
	}
	return out
}

func topRejectedNames(tags []domain.RejectedTag, maxCount int) []string {
	if len(tags) == 0 || maxCount <= 0 {
		return []string{"(none)"}
	}
	if len(tags) > maxCount {
		tags = tags[:maxCount]
	}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		reason := strings.TrimSpace(tag.Reason)
		if name == "" {
			continue
		}
		if reason == "" {
			out = append(out, name)
			continue
		}
		out = append(out, name+" ("+reason+")")
	}
	if len(out) == 0 {
		return []string{"(none)"}
	}
	return out
}

func topRejectedTagNames(tags []domain.RejectedTag, maxCount int) []string {
	if len(tags) == 0 || maxCount <= 0 {
		return nil
	}
	if len(tags) > maxCount {
		tags = tags[:maxCount]
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func cleanStringSlice(values []string, canonicalTag bool) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if canonicalTag {
			v = strings.ReplaceAll(v, " ", "_")
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	return out
}

func extractFirstJSONObject(raw string) string {
	start := strings.IndexByte(raw, '{')
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(raw); i++ {
		ch := raw[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(raw[start : i+1])
			}
		}
	}
	return ""
}

func responseText(resp *domain.GenerateResponse) string {
	if resp == nil {
		return ""
	}
	return strings.TrimSpace(resp.Text)
}

func logPromptStageRequest(stage string, model string, temperature float64, maxTokens int, systemPrompt string, userPrompt string) {
	logging.Debug("llm stage request",
		"stage", stage,
		"model", model,
		"temperature", temperature,
		"max_tokens", maxTokens,
		"system_prompt", systemPrompt,
		"user_prompt", userPrompt,
	)
}

func clampInt(v int, minValue int, maxValue int) int {
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}

func deterministicFallback(mode domain.Mode, prompt string, retrieval domain.RetrievalResult) string {
	switch mode {
	case domain.ModeBooru:
		return joinTags(retrieval)
	case domain.ModeHybrid:
		tags := joinTags(retrieval)
		if tags == "" {
			return strings.TrimSpace(prompt)
		}
		return strings.TrimSpace(prompt) + ", " + tags
	default:
		tags := joinTags(retrieval)
		if tags == "" {
			return strings.TrimSpace(prompt)
		}
		return strings.TrimSpace(prompt) + ". tags: " + tags
	}
}

func joinTags(retrieval domain.RetrievalResult) string {
	seen := map[string]struct{}{}
	var out []string
	appendTags := func(tags []domain.TagCandidate, max int) {
		for _, t := range tags {
			if len(out) >= max {
				return
			}
			if _, ok := seen[t.Name]; ok {
				continue
			}
			seen[t.Name] = struct{}{}
			out = append(out, t.Name)
		}
	}
	appendTags(retrieval.CharacterTags, 12)
	appendTags(retrieval.ConfirmedTags, 20)
	appendTags(retrieval.SuggestedTags, 28)
	return strings.Join(out, ", ")
}
