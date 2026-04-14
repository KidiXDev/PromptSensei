package prompting

import (
	"strings"

	"github.com/kidixdev/PromptSensei/internal/domain"
)

var requiredQualityPrefix = []string{"masterpiece", "best quality", "newest"}

var knownQualityTags = map[string]struct{}{
	"masterpiece": {}, "best quality": {}, "high quality": {}, "ultra detailed": {}, "absurdres": {}, "highres": {}, "newest": {},
	"amazing quality": {}, "very aesthetic": {}, "new aesthetic": {},
}

func EnsureQualityPrefix(output string, mode domain.Mode) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return strings.Join(requiredQualityPrefix, ", ")
	}

	lower := strings.ToLower(output)
	hasQuality := false
	for tag := range knownQualityTags {
		if strings.Contains(lower, tag) {
			hasQuality = true
			break
		}
	}
	if !hasQuality {
		return strings.Join(requiredQualityPrefix, ", ") + ", " + output
	}
	if mode == domain.ModeBooru || mode == domain.ModeHybrid {
		return enforcePrefixOrder(output)
	}
	return output
}

func enforcePrefixOrder(output string) string {
	parts := strings.Split(output, ",")
	if len(parts) < 2 {
		return output
	}
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		trimmed = append(trimmed, part)
	}
	if len(trimmed) == 0 {
		return output
	}

	remaining := trimmed
	ordered := make([]string, 0, len(trimmed)+len(requiredQualityPrefix))
	for _, required := range requiredQualityPrefix {
		idx := -1
		for i, part := range remaining {
			if strings.EqualFold(part, required) {
				idx = i
				break
			}
		}
		if idx >= 0 {
			ordered = append(ordered, remaining[idx])
			remaining = append(remaining[:idx], remaining[idx+1:]...)
			continue
		}
		ordered = append(ordered, required)
	}
	ordered = append(ordered, remaining...)
	return strings.Join(ordered, ", ")
}
