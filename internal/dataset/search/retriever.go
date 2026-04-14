package search

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/dataset/sqlite"
	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/utils"
)

type Retriever struct {
	repo *sqlite.Repository
}

func NewRetriever(repo *sqlite.Repository) *Retriever {
	return &Retriever{repo: repo}
}

func (r *Retriever) Retrieve(ctx context.Context, prompt string, mode domain.Mode) (domain.RetrievalResult, error) {
	termWeights := buildTermWeights(prompt)
	terms := orderedTerms(termWeights)

	charMatches, err := r.repo.FindCharactersByTerms(ctx, terms, 6)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	tagMatches, err := r.repo.FindTagsByTerms(ctx, terms, 60)
	if err != nil {
		return domain.RetrievalResult{}, err
	}
	prefixTags, err := r.repo.SearchTagsByPrefix(ctx, terms, 12)
	if err != nil {
		return domain.RetrievalResult{}, err
	}

	confirmed := make(map[string]domain.TagCandidate)
	characterTags := make(map[string]domain.TagCandidate)
	suggested := make(map[string]domain.TagCandidate)
	explicitTags := explicitTagsFromPrompt(prompt)

	for _, m := range tagMatches {
		score := scoreFromMatch(m.MatchType, m.Tag.PostCount) + termWeightBoost(termWeights, m.MatchedTerm)
		score += lexicalAlignmentScore(termWeights, m.Tag.Name) * 0.25
		current := domain.TagCandidate{
			Name:      m.Tag.Name,
			Category:  m.Tag.Category,
			PostCount: m.Tag.PostCount,
			Score:     score,
			Reason:    "matched by " + m.MatchType + " (" + m.MatchedTerm + ")",
		}
		if existing, ok := confirmed[current.Name]; !ok || existing.Score < current.Score {
			confirmed[current.Name] = current
		}
	}

	for _, c := range charMatches {
		characterConfidence := scoreFromMatch(c.MatchType, c.Character.Count) + termWeightBoost(termWeights, c.MatchedTerm)
		charCandidate := domain.TagCandidate{
			Name:      c.Character.Name,
			Category:  4,
			PostCount: c.Character.Count,
			Score:     characterConfidence + lexicalAlignmentScore(termWeights, c.Character.Name)*0.4,
			Reason:    "character " + c.MatchType + " match (" + c.MatchedTerm + ")",
		}
		characterTags[charCandidate.Name] = charCandidate

		if c.Character.CopyrightName != "" {
			cp := domain.TagCandidate{
				Name:      c.Character.CopyrightName,
				Category:  3,
				PostCount: c.Character.Count,
				Score:     scoreFromMatch("copyright", c.Character.Count) + characterConfidence*0.12,
				Reason:    "copyright inferred from character " + c.Character.Name,
			}
			if existing, ok := suggested[cp.Name]; !ok || existing.Score < cp.Score {
				suggested[cp.Name] = cp
			}
		}

		coreTags, err := r.repo.CoreTagsForCharacter(ctx, c.Character.ID, 20)
		if err != nil {
			return domain.RetrievalResult{}, err
		}
		for _, coreTag := range coreTags {
			core := domain.TagCandidate{
				Name:      coreTag.TagName,
				Category:  coreTag.Category,
				PostCount: coreTag.PostCount,
				Score:     scoreFromMatch("trigger", coreTag.PostCount) + characterConfidence*0.15 - 0.18,
				Reason:    "character core tag from " + c.Character.Name,
			}
			if existing, ok := suggested[core.Name]; !ok || existing.Score < core.Score {
				suggested[core.Name] = core
			}
		}
	}

	for _, t := range prefixTags {
		candidate := domain.TagCandidate{
			Name:      t.Name,
			Category:  t.Category,
			PostCount: t.PostCount,
			Score:     scoreFromMatch("prefix", t.PostCount) + lexicalAlignmentScore(termWeights, t.Name)*0.3 - 0.08,
			Reason:    "prefix lookup candidate",
		}
		if existing, ok := suggested[candidate.Name]; !ok || existing.Score < candidate.Score {
			suggested[candidate.Name] = candidate
		}
	}

	confirmedList := rerankCandidates(mapToSortedSlice(confirmed), termWeights, mode, "confirmed", 24, explicitTags)
	characterList := rerankCandidates(mapToSortedSlice(characterTags), termWeights, mode, "character", 8, explicitTags)
	suggestedList := rerankCandidates(mapToSortedSlice(suggested), termWeights, mode, "suggested", 32, explicitTags)

	confirmedList, rejected := applyConflicts(prompt, confirmedList)
	suggestedList, rejectedSuggested := applyConflicts(prompt, suggestedList)
	rejected = append(rejected, rejectedSuggested...)

	return domain.RetrievalResult{
		ConfirmedTags: confirmedList,
		CharacterTags: characterList,
		SuggestedTags: suggestedList,
		RejectedTags:  rejected,
		Debug: map[string]any{
			"mode":         mode,
			"terms":        terms,
			"term_weights": termWeights,
			"explicit":     explicitTags,
		},
	}, nil
}

func buildTermWeights(prompt string) map[string]float64 {
	tokens := utils.Tokenize(prompt)
	weights := map[string]float64{}
	if len(tokens) == 0 {
		return weights
	}

	add := func(v string, weight float64) {
		v = utils.NormalizeForLookup(v)
		if v == "" {
			return
		}
		if existing, ok := weights[v]; ok && existing >= weight {
			return
		}
		weights[v] = weight
	}

	phrases := splitPromptPhrases(prompt)
	for _, phrase := range phrases {
		words := utils.Tokenize(phrase)
		if len(words) >= 2 {
			add(phrase, 3.5+math.Min(1.5, float64(len(words))*0.2))
		}
	}

	for i := 0; i < len(tokens); i++ {
		add(tokens[i], 1.2)
	}
	for i := 0; i < len(tokens)-1; i++ {
		add(tokens[i]+" "+tokens[i+1], 2.0)
	}
	for i := 0; i < len(tokens)-2; i++ {
		add(tokens[i]+" "+tokens[i+1]+" "+tokens[i+2], 2.5)
	}

	for _, t := range tokens {
		add(t, 1.2)
	}

	canonicalPromptTags := utils.SplitList(prompt)
	for _, tag := range canonicalPromptTags {
		add(strings.ReplaceAll(tag, "_", " "), 3.8)
	}

	add(prompt, 2.8)
	return weights
}

func orderedTerms(weights map[string]float64) []string {
	out := make([]string, 0, len(weights))
	for term := range weights {
		out = append(out, term)
	}
	sort.Slice(out, func(i, j int) bool {
		if weights[out[i]] == weights[out[j]] {
			return out[i] < out[j]
		}
		return weights[out[i]] > weights[out[j]]
	})
	return out
}

func splitPromptPhrases(prompt string) []string {
	raw := strings.NewReplacer("\n", ",", ";", ",", "|", ",").Replace(prompt)
	chunks := strings.Split(raw, ",")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		chunk = utils.NormalizeForLookup(chunk)
		if chunk == "" {
			continue
		}
		if _, ok := seen[chunk]; ok {
			continue
		}
		seen[chunk] = struct{}{}
		out = append(out, chunk)
	}
	return out
}

func explicitTagsFromPrompt(prompt string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, tag := range utils.SplitList(prompt) {
		out[tag] = struct{}{}
	}
	return out
}

func termWeightBoost(weights map[string]float64, term string) float64 {
	term = utils.NormalizeForLookup(term)
	if term == "" {
		return 0
	}
	w := weights[term]
	if w <= 0 {
		return 0
	}
	return math.Min(1.2, w*0.28)
}

func rerankCandidates(
	input []domain.TagCandidate,
	promptTerms map[string]float64,
	mode domain.Mode,
	bucket string,
	maxCount int,
	explicitTags map[string]struct{},
) []domain.TagCandidate {
	if len(input) == 0 {
		return nil
	}

	for i := range input {
		candidate := &input[i]
		candidate.Score += lexicalAlignmentScore(promptTerms, candidate.Name)
		candidate.Score += categoryPreferenceScore(mode, candidate.Category, bucket)
		candidate.Score -= rarityPenalty(candidate.PostCount)
		if _, ok := explicitTags[candidate.Name]; ok {
			candidate.Score += 0.85
		}
	}

	sort.Slice(input, func(i, j int) bool {
		if input[i].Score == input[j].Score {
			return input[i].PostCount > input[j].PostCount
		}
		return input[i].Score > input[j].Score
	})

	if maxCount <= 0 || len(input) <= maxCount {
		return input
	}
	return input[:maxCount]
}

func mapToSortedSlice(input map[string]domain.TagCandidate) []domain.TagCandidate {
	out := make([]domain.TagCandidate, 0, len(input))
	for _, v := range input {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].PostCount > out[j].PostCount
		}
		return out[i].Score > out[j].Score
	})
	return out
}
