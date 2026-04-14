package services

import (
	"testing"

	"github.com/kidixdev/PromptSensei/internal/domain"
)

func TestDeterministicFallbackHybridIncludesPromptAndTags(t *testing.T) {
	out := deterministicFallback(domain.ModeHybrid, "girl in city", domain.RetrievalResult{
		CharacterTags: []domain.TagCandidate{{Name: "1girl"}},
		SuggestedTags: []domain.TagCandidate{{Name: "cityscape"}},
	})
	if out == "" {
		t.Fatalf("expected fallback output")
	}
	if out != "masterpiece, best quality, newest, girl in city, 1girl, cityscape" {
		t.Fatalf("unexpected fallback output: %s", out)
	}
}

func TestJoinTagsDeduplicatesCandidates(t *testing.T) {
	out := joinTags(domain.RetrievalResult{
		CharacterTags: []domain.TagCandidate{{Name: "1girl"}},
		ConfirmedTags: []domain.TagCandidate{{Name: "1girl"}, {Name: "smile"}},
		SuggestedTags: []domain.TagCandidate{{Name: "smile"}, {Name: "sunset"}},
	})
	if out != "1girl, smile, sunset" {
		t.Fatalf("unexpected joined tags: %s", out)
	}
}
