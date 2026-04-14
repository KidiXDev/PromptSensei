package services

import (
	"context"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/dataset/search"
	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/prompting"
)

type PromptService struct {
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

func (s *PromptService) Enhance(ctx context.Context, req domain.EnhanceRequest) (*domain.EnhanceResult, []string, error) {
	mode := req.Mode
	if mode == "" {
		mode = s.cfg.General.DefaultMode
	}

	repo, err := s.dataset.OpenRepository()
	if err != nil {
		return nil, nil, err
	}
	defer repo.Close()

	retriever := search.NewRetriever(repo)
	retrieval, err := retriever.Retrieve(ctx, req.Prompt, mode)
	if err != nil {
		return nil, nil, err
	}

	persona, err := s.instructions.LoadPersona()
	if err != nil {
		return nil, nil, err
	}
	systemRules, err := s.instructions.LoadSystem()
	if err != nil {
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

	if s.providers.Enabled() {
		resp, err := s.providers.Generate(ctx, domain.GenerateRequest{
			SystemPrompt: assembled.SystemPrompt,
			UserPrompt:   assembled.UserPrompt,
			Model:        s.cfg.Provider.Model,
			Temperature:  s.cfg.Provider.Temperature,
			MaxTokens:    s.cfg.Provider.MaxTokens,
		})
		if err != nil {
			return nil, warnings, err
		}
		if resp != nil && strings.TrimSpace(resp.Text) != "" {
			finalOutput = strings.TrimSpace(resp.Text)
			usedProvider = true
			providerName = resp.Provider
		}
	}

	validationApplied := false
	if (req.StrictBooru || s.cfg.General.StrictBooruValidation) && mode == domain.ModeBooru {
		finalOutput = prompting.FilterBooruOutput(finalOutput, retrieval)
		validationApplied = true
	}

	return &domain.EnhanceResult{
		Output:            finalOutput,
		Retrieval:         retrieval,
		SystemPrompt:      assembled.SystemPrompt,
		UserPrompt:        assembled.UserPrompt,
		ProviderName:      providerName,
		UsedProvider:      usedProvider,
		ValidationApplied: validationApplied,
	}, warnings, nil
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
