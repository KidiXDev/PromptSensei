package prompting

import (
	"strings"
	"testing"

	"github.com/kidixdev/PromptSensei/internal/domain"
)

func TestAssembleIncludesKeySections(t *testing.T) {
	out := Assemble(AssemblyInput{
		Mode:        domain.ModeBooru,
		SystemRules: "base system",
		UserPrompt:  "miku in city",
		Knowledge: []domain.KnowledgeDoc{
			{Name: "Anima.md", Content: "knowledge text"},
		},
		Retrieval: domain.RetrievalResult{
			ConfirmedTags: []domain.TagCandidate{{Name: "hatsune_miku"}},
			SuggestedTags: []domain.TagCandidate{{Name: "cityscape"}},
			Characters: []domain.CharacterRetrievalContext{
				{
					Name:        "hatsune_miku",
					MatchType:   "trigger",
					MatchedTerm: "miku",
					AnchorTags:  []string{"blue_hair", "twintails"},
				},
			},
		},
	})

	if !strings.Contains(out.SystemPrompt, "base system") {
		t.Fatalf("system prompt missing base rules")
	}
	if len(out.UserPrompts) < 2 {
		t.Fatalf("expected multiple user prompts, got %d", len(out.UserPrompts))
	}
	if !strings.Contains(strings.Join(out.UserPrompts, "\n"), "confirmed_tags: hatsune_miku") {
		t.Fatalf("user prompt missing confirmed tags")
	}
	if !strings.Contains(strings.Join(out.UserPrompts, "\n"), "Knowledge context (Anima.md)") {
		t.Fatalf("user prompts missing knowledge context")
	}
	if !strings.Contains(strings.Join(out.UserPrompts, "\n"), "Character context #1") {
		t.Fatalf("user prompts missing per-character context")
	}
	if !strings.Contains(strings.Join(out.UserPrompts, "\n"), "Strict ordering: metadata -> subject -> character") {
		t.Fatalf("user prompts missing strict ordering guidance")
	}
}

func TestFilterBooruOutput(t *testing.T) {
	retrieval := domain.RetrievalResult{
		ConfirmedTags: []domain.TagCandidate{
			{Name: "hatsune_miku"},
			{Name: "1girl"},
		},
		SuggestedTags: []domain.TagCandidate{
			{Name: "cityscape"},
		},
	}

	filtered := FilterBooruOutput("hatsune_miku, invalid_tag, cityscape", retrieval)
	if filtered != "hatsune_miku, cityscape" {
		t.Fatalf("unexpected filtered output: %s", filtered)
	}
}
