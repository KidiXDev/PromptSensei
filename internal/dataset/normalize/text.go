package normalize

import "github.com/kidixdev/PromptSensei/internal/utils"

func Lookup(input string) string {
	return utils.NormalizeForLookup(input)
}

func Tag(input string) string {
	return utils.CanonicalTag(input)
}
