package domain

type RetrievalResult struct {
	ConfirmedTags []TagCandidate
	CharacterTags []TagCandidate
	SuggestedTags []TagCandidate
	RejectedTags  []RejectedTag
	Debug         map[string]any
}
