package search

import (
	"math"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/utils"
)

func scoreFromMatch(matchType string, postCount int) float64 {
	base := 0.0
	switch matchType {
	case "exact":
		base = 4.0
	case "alias":
		base = 3.0
	case "trigger":
		base = 3.5
	case "name":
		base = 3.8
	case "copyright":
		base = 2.7
	default:
		base = 2.0
	}
	popularity := math.Log10(float64(postCount) + 1)
	return base + popularity*0.4
}

func lexicalAlignmentScore(promptTerms map[string]float64, candidate string) float64 {
	candidateNorm := utils.NormalizeForLookup(candidate)
	if candidateNorm == "" || len(promptTerms) == 0 {
		return 0
	}

	score := 0.0
	for term, weight := range promptTerms {
		if term == candidateNorm {
			score += 0.8 + weight*0.15
			continue
		}
		if strings.Contains(candidateNorm, term) || strings.Contains(term, candidateNorm) {
			score += 0.25 + weight*0.08
		}
	}
	return score
}

func categoryPreferenceScore(mode domain.Mode, category int, bucket string) float64 {
	switch bucket {
	case "character":
		if category == 4 {
			return 0.45
		}
		return 0.05
	case "confirmed":
		if category == 0 || category == 4 {
			return 0.12
		}
		return 0
	case "suggested":
		if mode == domain.ModeNatural && category == 0 {
			return 0.05
		}
		if mode == domain.ModeBooru && (category == 0 || category == 3 || category == 4) {
			return 0.16
		}
		return 0
	default:
		return 0
	}
}

func rarityPenalty(postCount int) float64 {
	if postCount <= 0 {
		return 0.3
	}
	if postCount < 10 {
		return 0.14
	}
	if postCount < 40 {
		return 0.07
	}
	return 0
}
