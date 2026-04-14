package domain

type RetrievalResult struct {
	ConfirmedTags []TagCandidate
	CharacterTags []TagCandidate
	SuggestedTags []TagCandidate
	RejectedTags  []RejectedTag
	Characters    []CharacterRetrievalContext
	Debug         map[string]any
}

type CharacterRetrievalContext struct {
	Name          string
	MatchType     string
	MatchedTerm   string
	CopyrightName string
	AnchorTags    []string
	SuggestedTags []string
}
