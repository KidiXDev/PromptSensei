package services

import (
	"context"
	"strings"
	"sync"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/logging"
	"github.com/kidixdev/PromptSensei/internal/providers"
)

type ProviderService struct {
	mu       sync.RWMutex
	cfg      config.ProviderConfig
	provider domain.Provider
}

func NewProviderService(cfg config.ProviderConfig) (*ProviderService, error) {
	service := &ProviderService{}
	if err := service.UpdateConfig(cfg); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *ProviderService) Enabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Enabled && s.provider != nil
}

func (s *ProviderService) Name() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.provider != nil {
		return s.provider.Name()
	}
	return strings.ToLower(strings.TrimSpace(s.cfg.Name))
}

func (s *ProviderService) Config() config.ProviderConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *ProviderService) UpdateConfig(cfg config.ProviderConfig) error {
	var p domain.Provider
	if cfg.Enabled {
		var err error
		p, err = providers.BuildProvider(cfg)
		if err != nil {
			logging.Error("provider build failed", "provider", cfg.Name, "error", err)
			return err
		}
	}

	s.mu.Lock()
	s.cfg = cfg
	s.provider = p
	s.mu.Unlock()
	logging.Info("provider config updated",
		"enabled", cfg.Enabled,
		"name", cfg.Name,
		"model", cfg.Model,
		"api_base", cfg.APIBaseURL,
		"max_tokens", cfg.MaxTokens,
		"timeout_seconds", cfg.TimeoutSeconds,
	)
	return nil
}

func (s *ProviderService) Generate(ctx context.Context, req domain.GenerateRequest) (*domain.GenerateResponse, error) {
	s.mu.RLock()
	enabled := s.cfg.Enabled && s.provider != nil
	p := s.provider
	s.mu.RUnlock()
	if !enabled {
		logging.Debug("provider call skipped; provider disabled")
		return nil, nil
	}
	logging.Debug("provider generate start", "provider", p.Name(), "model", req.Model, "temperature", req.Temperature, "max_tokens", req.MaxTokens)
	resp, err := p.Generate(ctx, req)
	if err != nil {
		logging.Error("provider generate failed", "provider", p.Name(), "error", err)
		return nil, err
	}
	if resp == nil {
		logging.Warn("provider returned nil response", "provider", p.Name())
		return nil, nil
	}
	logging.Debug("provider generate complete",
		"provider", resp.Provider,
		"prompt_tokens", resp.Usage.PromptTokens,
		"completion_tokens", resp.Usage.CompletionTokens,
		"total_tokens", resp.Usage.TotalTokens,
	)
	return resp, nil
}
