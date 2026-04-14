package services

import (
	"context"
	"fmt"
	"sort"
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

	systemRules, err := s.instructions.LoadSystem()
	if err != nil {
		logging.Error("load system rules failed", "error", err)
		return nil, nil, err
	}

	knowledgeDocs, warnings := s.knowledge.LoadSelected(req.KnowledgeFiles)
	assembled := prompting.Assemble(prompting.AssemblyInput{
		Mode:            mode,
		SystemRules:     systemRules,
		UserPrompt:      req.Prompt,
		OptionalContext: req.Context,
		Knowledge:       knowledgeDocs,
		Retrieval:       retrieval,
		CreateMode:      req.CreateMode,
	})

	finalOutput := deterministicFallback(mode, req.Prompt, retrieval)
	usedProvider := false
	providerName := s.providers.Name()

	if s.providers.Enabled() {
		logPromptSequenceRequest(cfg.Provider.Model, cfg.Provider.Temperature, cfg.Provider.MaxTokens, assembled.SystemPrompt, assembled.UserPrompts)
		resp, err := s.providers.Generate(ctx, domain.GenerateRequest{
			SystemPrompt: assembled.SystemPrompt,
			UserPrompts:  assembled.UserPrompts,
			Model:        cfg.Provider.Model,
			Temperature:  cfg.Provider.Temperature,
			MaxTokens:    cfg.Provider.MaxTokens,
		})
		if err != nil {
			logging.Error("provider generation failed", "error", err)
			return nil, warnings, err
		}
		if resp != nil && strings.TrimSpace(resp.Text) != "" {
			finalOutput = strings.TrimSpace(resp.Text)
			usedProvider = true
			providerName = resp.Provider
			logging.Debug("provider output", "provider", providerName, "output", finalOutput)
		} else {
			warnings = append(warnings, "provider returned empty output; fallback used")
			logging.Warn("provider returned empty output; fallback used", "fallback", finalOutput)
		}
	}

	finalOutput = prompting.EnsureQualityPrefix(finalOutput, mode)

	validationApplied := false
	if (req.StrictBooru || cfg.General.StrictBooruValidation) && mode == domain.ModeBooru {
		finalOutput = prompting.FilterBooruOutput(finalOutput, retrieval)
		validationApplied = true
		logging.Debug("strict booru validation applied")
	}
	
	if cfg.General.TagWhitespace {
		finalOutput = strings.ReplaceAll(finalOutput, "_", " ")
		logging.Debug("tag whitespace replacement applied")
	}

	logging.Info("enhance complete",
		"mode", mode,
		"provider", providerName,
		"provider_used", usedProvider,
		"validation_applied", validationApplied,
		"warnings", len(warnings),
		"output", finalOutput,
	)

	primaryUserPrompt := ""
	if len(assembled.UserPrompts) > 0 {
		primaryUserPrompt = assembled.UserPrompts[0]
	}

	return &domain.EnhanceResult{
		Output:            finalOutput,
		Retrieval:         retrieval,
		SystemPrompt:      assembled.SystemPrompt,
		UserPrompt:        primaryUserPrompt,
		ProviderName:      providerName,
		UsedProvider:      usedProvider,
		ValidationApplied: validationApplied,
		ChainApplied:      false,
		ChainStages:       0,
	}, warnings, nil
}

func logPromptSequenceRequest(model string, temperature float64, maxTokens int, systemPrompt string, userPrompts []string) {
	logging.Debug("llm sequence request",
		"model", model,
		"temperature", temperature,
		"max_tokens", maxTokens,
		"system_prompt", systemPrompt,
		"user_prompt_count", len(userPrompts),
	)
	for i, prompt := range userPrompts {
		logging.Debug("llm user prompt", "index", i+1, "content", prompt)
	}
}

func deterministicFallback(mode domain.Mode, prompt string, retrieval domain.RetrievalResult) string {
	qualityPrefix := "masterpiece, best quality, newest"
	switch mode {
	case domain.ModeBooru:
		tags := joinTags(retrieval)
		if tags == "" {
			return qualityPrefix
		}
		return qualityPrefix + ", " + tags
	case domain.ModeHybrid:
		tags := joinTags(retrieval)
		if tags == "" {
			return qualityPrefix + ", " + strings.TrimSpace(prompt)
		}
		return qualityPrefix + ", " + strings.TrimSpace(prompt) + ", " + tags
	default:
		tags := joinTags(retrieval)
		if tags == "" {
			return qualityPrefix + ", " + strings.TrimSpace(prompt)
		}
		return qualityPrefix + ", " + strings.TrimSpace(prompt) + ". tags: " + tags
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

func formatRetrievalContext(retrieval domain.RetrievalResult) string {
	lines := []string{
		tagsToLine("confirmed_tags", retrieval.ConfirmedTags),
		tagsToLine("character_tags", retrieval.CharacterTags),
		tagsToLine("suggested_tags", retrieval.SuggestedTags),
		rejectedTagsToLine(retrieval.RejectedTags),
	}
	return strings.Join(lines, "\n")
}

func tagsToLine(label string, tags []domain.TagCandidate) string {
	if len(tags) == 0 {
		return label + ": (none)"
	}
	items := make([]string, 0, len(tags))
	for _, tag := range tags {
		items = append(items, tag.Name)
	}
	return label + ": " + strings.Join(items, ", ")
}

func rejectedTagsToLine(tags []domain.RejectedTag) string {
	if len(tags) == 0 {
		return "rejected_tags: (none)"
	}
	items := make([]string, 0, len(tags))
	for _, tag := range tags {
		items = append(items, fmt.Sprintf("%s (%s)", tag.Name, tag.Reason))
	}
	sort.Strings(items)
	return "rejected_tags: " + strings.Join(items, ", ")
}
