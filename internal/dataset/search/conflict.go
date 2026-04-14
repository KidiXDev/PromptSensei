package search

import (
	"strings"

	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/utils"
)

var conflictGroups = [][]string{
	{"1girl", "1boy", "1other"},
	{"solo", "multiple_girls", "multiple_boys"},
	{"day", "night"},
}

func applyConflicts(prompt string, input []domain.TagCandidate) ([]domain.TagCandidate, []domain.RejectedTag) {
	promptNorm := utils.NormalizeForLookup(prompt)
	explicit := map[string]struct{}{}
	for _, group := range conflictGroups {
		for _, tag := range group {
			if strings.Contains(promptNorm, utils.NormalizeForLookup(tag)) {
				explicit[tag] = struct{}{}
			}
		}
	}

	var filtered []domain.TagCandidate
	var rejected []domain.RejectedTag
	present := map[string]domain.TagCandidate{}

	for _, tag := range input {
		if _, ok := explicit[tag.Name]; ok {
			present[tag.Name] = tag
			continue
		}
		conflicted := false
		for _, group := range conflictGroups {
			inGroup := false
			for _, item := range group {
				if item == tag.Name {
					inGroup = true
					break
				}
			}
			if !inGroup {
				continue
			}
			for explicitTag := range explicit {
				for _, item := range group {
					if explicitTag == item && explicitTag != tag.Name {
						conflicted = true
						rejected = append(rejected, domain.RejectedTag{
							Name:   tag.Name,
							Reason: "conflicts with explicit prompt tag " + explicitTag,
						})
					}
				}
			}
		}
		if conflicted {
			continue
		}

		if existing, ok := present[tag.Name]; !ok || tag.Score > existing.Score {
			present[tag.Name] = tag
		}
	}

	for _, v := range present {
		filtered = append(filtered, v)
	}
	return filtered, rejected
}
