package prompting

import (
	"strings"
	"testing"

	"github.com/kidixdev/PromptSensei/internal/domain"
)

func TestEnsureQualityPrefixAddsMissingQuality(t *testing.T) {
	out := EnsureQualityPrefix("1girl, beach, sunset", domain.ModeBooru)
	if !strings.HasPrefix(strings.ToLower(out), "masterpiece, best quality, newest") {
		t.Fatalf("expected quality prefix, got: %s", out)
	}
}

func TestEnsureQualityPrefixKeepsExistingQualityButReorders(t *testing.T) {
	out := EnsureQualityPrefix("1girl, newest, best quality, masterpiece, beach", domain.ModeBooru)
	if !strings.HasPrefix(strings.ToLower(out), "masterpiece, best quality, newest") {
		t.Fatalf("expected ordered quality prefix, got: %s", out)
	}
}
