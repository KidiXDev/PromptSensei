package providers

import (
	"testing"

	"github.com/kidixdev/PromptSensei/internal/config"
)

func TestBuildProviderSupportsNanoGPT(t *testing.T) {
	cfg := config.ProviderConfig{Name: "nanogpt"}
	provider, err := BuildProvider(cfg)
	if err != nil {
		t.Fatalf("build provider failed: %v", err)
	}
	if provider.Name() != "nanogpt" {
		t.Fatalf("expected provider name nanogpt, got %s", provider.Name())
	}
}

func TestBuildProviderSupportsNanoGPTHyphenAlias(t *testing.T) {
	cfg := config.ProviderConfig{Name: "nano-gpt"}
	provider, err := BuildProvider(cfg)
	if err != nil {
		t.Fatalf("build provider failed: %v", err)
	}
	if provider.Name() != "nanogpt" {
		t.Fatalf("expected provider name nanogpt, got %s", provider.Name())
	}
}

func TestBuildProviderSupportsFireworks(t *testing.T) {
	cfg := config.ProviderConfig{Name: "fireworks"}
	provider, err := BuildProvider(cfg)
	if err != nil {
		t.Fatalf("build provider failed: %v", err)
	}
	if provider.Name() != "fireworks" {
		t.Fatalf("expected provider name fireworks, got %s", provider.Name())
	}
}
