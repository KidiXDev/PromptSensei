package utils

import (
	"regexp"
	"strings"
)

var whitespaceRe = regexp.MustCompile(`\s+`)

func NormalizeForLookup(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	input = strings.ReplaceAll(input, `\(`, "(")
	input = strings.ReplaceAll(input, `\)`, ")")
	input = strings.ReplaceAll(input, "_", " ")
	input = strings.ReplaceAll(input, "-", " ")
	input = whitespaceRe.ReplaceAllString(input, " ")
	return strings.TrimSpace(input)
}

func CanonicalTag(input string) string {
	input = NormalizeForLookup(input)
	input = strings.ReplaceAll(input, " ", "_")
	return strings.Trim(input, "_")
}

func Tokenize(input string) []string {
	normalized := NormalizeForLookup(input)
	if normalized == "" {
		return nil
	}
	return strings.Fields(normalized)
}

func SplitList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		canonical := CanonicalTag(p)
		if canonical == "" {
			continue
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	return out
}
