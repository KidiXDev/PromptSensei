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
		Persona:     "persona",
		UserPrompt:  "miku in city",
		Knowledge: []domain.KnowledgeDoc{
			{Name: "Anima.md", Content: "knowledge text"},
		},
		Retrieval: domain.RetrievalResult{
			ConfirmedTags: []domain.TagCandidate{{Name: "hatsune_miku"}},
			SuggestedTags: []domain.TagCandidate{{Name: "cityscape"}},
		},
	})

	if !strings.Contains(out.SystemPrompt, "base system") {
		t.Fatalf("system prompt missing base rules")
	}
	if !strings.Contains(out.SystemPrompt, "Knowledge (Anima.md)") {
		t.Fatalf("system prompt missing knowledge block")
	}
	if !strings.Contains(out.UserPrompt, "confirmed_tags: hatsune_miku") {
		t.Fatalf("user prompt missing confirmed tags")
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
