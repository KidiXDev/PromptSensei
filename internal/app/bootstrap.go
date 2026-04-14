package app

import (
	"context"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/services"
)

func NewRuntime(ctx context.Context) (*Runtime, []string, error) {
	paths, err := config.ResolvePaths()
	if err != nil {
		return nil, nil, err
	}
	if err := config.Bootstrap(paths); err != nil {
		return nil, nil, err
	}

	cfg, err := config.Load(paths)
	if err != nil {
		return nil, nil, err
	}

	dataset := services.NewDatasetService(cfg)

	instructions := services.NewInstructionService(paths)
	knowledge := services.NewKnowledgeService(paths)
	providers, err := services.NewProviderService(cfg.Provider)
	if err != nil {
		return nil, nil, err
	}
	promptService := services.NewPromptService(cfg, dataset, instructions, knowledge, providers)

	return &Runtime{
		Paths:        paths,
		Config:       cfg,
		Dataset:      dataset,
		Instructions: instructions,
		Knowledge:    knowledge,
		Providers:    providers,
		Prompt:       promptService,
	}, nil, nil
}
