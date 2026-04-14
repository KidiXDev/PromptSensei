package providers

import (
	"fmt"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/providers/nanogpt"
	"github.com/kidixdev/PromptSensei/internal/providers/openai"
	"github.com/kidixdev/PromptSensei/internal/providers/openrouter"
)

func BuildProvider(cfg config.ProviderConfig) (domain.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Name)) {
	case "openai":
		return openai.NewClient(cfg), nil
	case "openrouter":
		return openrouter.NewClient(cfg), nil
	case "nanogpt", "nano-gpt":
		return nanogpt.NewClient(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported provider %q", cfg.Name)
	}
}
