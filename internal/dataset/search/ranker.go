package search

import "math"

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
