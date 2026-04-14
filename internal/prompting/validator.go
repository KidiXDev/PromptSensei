package prompting

import (
	"strings"

	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/utils"
)

func FilterBooruOutput(output string, retrieval domain.RetrievalResult) string {
	allowed := map[string]struct{}{}
	for _, bucket := range [][]domain.TagCandidate{
		retrieval.ConfirmedTags,
		retrieval.CharacterTags,
		retrieval.SuggestedTags,
	} {
		for _, tag := range bucket {
			allowed[tag.Name] = struct{}{}
		}
	}

	parts := strings.Split(output, ",")
	var filtered []string
	seen := map[string]struct{}{}
	for _, part := range parts {
		tag := utils.CanonicalTag(part)
		if tag == "" {
			continue
		}
		if _, ok := allowed[tag]; !ok {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		filtered = append(filtered, tag)
	}
	return strings.Join(filtered, ", ")
}
