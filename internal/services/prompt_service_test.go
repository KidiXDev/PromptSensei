package services

import (
	"testing"

	"github.com/kidixdev/PromptSensei/internal/domain"
)

func TestParsePromptChainPlanFromWrappedJSON(t *testing.T) {
	raw := "plan:\n{\"intent\":\"enhance\",\"subject\":\"miku\",\"style\":\"anime\",\"composition\":[\"close-up\"],\"lighting\":[\"rim light\"],\"must_include_tags\":[\"hatsune_miku\"],\"avoid_tags\":[\"lowres\"],\"quality_signals\":[\"coherent\"]}\nend"
	plan, err := parsePromptChainPlan(raw)
	if err != nil {
		t.Fatalf("parse plan: %v", err)
	}
	if plan.Intent != "enhance" {
		t.Fatalf("expected intent enhance, got %q", plan.Intent)
	}
	if len(plan.MustInclude) != 1 || plan.MustInclude[0] != "hatsune_miku" {
		t.Fatalf("unexpected must include tags: %#v", plan.MustInclude)
	}
}

func TestFallbackPromptChainPlanUsesRetrieval(t *testing.T) {
	plan := fallbackPromptChainPlan("miku in city", domain.RetrievalResult{
		CharacterTags: []domain.TagCandidate{{Name: "hatsune_miku"}},
		RejectedTags:  []domain.RejectedTag{{Name: "day", Reason: "conflict"}},
	}, true)
	if len(plan.MustInclude) == 0 || plan.MustInclude[0] != "hatsune_miku" {
		t.Fatalf("expected retrieval tags in fallback plan")
	}
	if len(plan.Avoid) == 0 || plan.Avoid[0] != "day" {
		t.Fatalf("expected rejected tags in fallback avoid list")
	}
}
