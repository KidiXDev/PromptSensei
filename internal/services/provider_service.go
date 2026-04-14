package services

import (
	"context"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/providers"
)

type ProviderService struct {
	cfg      config.ProviderConfig
	provider domain.Provider
}

func NewProviderService(cfg config.ProviderConfig) (*ProviderService, error) {
	service := &ProviderService{cfg: cfg}
	if !cfg.Enabled {
		return service, nil
	}

	p, err := providers.BuildProvider(cfg)
	if err != nil {
		return nil, err
	}
	service.provider = p
	return service, nil
}

func (s *ProviderService) Enabled() bool {
	return s.cfg.Enabled && s.provider != nil
}

func (s *ProviderService) Name() string {
	if s.provider != nil {
		return s.provider.Name()
	}
	return strings.ToLower(strings.TrimSpace(s.cfg.Name))
}

func (s *ProviderService) Generate(ctx context.Context, req domain.GenerateRequest) (*domain.GenerateResponse, error) {
	if !s.Enabled() {
		return nil, nil
	}
	return s.provider.Generate(ctx, req)
}
