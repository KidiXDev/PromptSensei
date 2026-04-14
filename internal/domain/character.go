package domain

type Character struct {
	ID                      int64
	Name                    string
	NormalizedName          string
	CopyrightName           string
	NormalizedCopyrightName string
	Count                   int
	SoloCount               int
	URL                     string
}
