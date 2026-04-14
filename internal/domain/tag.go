package domain

type Tag struct {
	ID             int64
	Name           string
	NormalizedName string
	Category       int
	PostCount      int
}

type TagCandidate struct {
	Name      string
	Category  int
	PostCount int
	Score     float64
	Reason    string
}

type RejectedTag struct {
	Name   string
	Reason string
}
