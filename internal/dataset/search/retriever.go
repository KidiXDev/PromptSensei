package search

import (
	"context"
	"sort"

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
	terms := buildTerms(prompt)

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

	for _, m := range tagMatches {
		score := scoreFromMatch(m.MatchType, m.Tag.PostCount)
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
		charCandidate := domain.TagCandidate{
			Name:      c.Character.Name,
			Category:  4,
			PostCount: c.Character.Count,
			Score:     scoreFromMatch(c.MatchType, c.Character.Count),
			Reason:    "character " + c.MatchType + " match (" + c.MatchedTerm + ")",
		}
		characterTags[charCandidate.Name] = charCandidate

		if c.Character.CopyrightName != "" {
			cp := domain.TagCandidate{
				Name:      c.Character.CopyrightName,
				Category:  3,
				PostCount: c.Character.Count,
				Score:     scoreFromMatch("copyright", c.Character.Count),
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
				Score:     scoreFromMatch("trigger", coreTag.PostCount) - 0.2,
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
			Score:     scoreFromMatch("prefix", t.PostCount),
			Reason:    "prefix lookup candidate",
		}
		if existing, ok := suggested[candidate.Name]; !ok || existing.Score < candidate.Score {
			suggested[candidate.Name] = candidate
		}
	}

	confirmedList := mapToSortedSlice(confirmed)
	characterList := mapToSortedSlice(characterTags)
	suggestedList := mapToSortedSlice(suggested)

	confirmedList, rejected := applyConflicts(prompt, confirmedList)
	suggestedList, rejectedSuggested := applyConflicts(prompt, suggestedList)
	rejected = append(rejected, rejectedSuggested...)

	return domain.RetrievalResult{
		ConfirmedTags: confirmedList,
		CharacterTags: characterList,
		SuggestedTags: suggestedList,
		RejectedTags:  rejected,
		Debug: map[string]any{
			"mode":  mode,
			"terms": terms,
		},
	}, nil
}

func buildTerms(prompt string) []string {
	tokens := utils.Tokenize(prompt)
	if len(tokens) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	add := func(v string) {
		v = utils.NormalizeForLookup(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
	}

	for _, t := range tokens {
		add(t)
	}
	for i := 0; i < len(tokens)-1; i++ {
		add(tokens[i] + " " + tokens[i+1])
	}
	add(prompt)

	out := make([]string, 0, len(seen))
	for term := range seen {
		out = append(out, term)
	}
	sort.Strings(out)
	return out
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
