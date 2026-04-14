package prompting

import "github.com/kidixdev/PromptSensei/internal/domain"

func modeRules(mode domain.Mode) string {
	switch mode {
	case domain.ModeBooru:
		return "Output booru-style tags only, comma-separated, compact, non-conflicting, and valid."
	case domain.ModeHybrid:
		return "Output a hybrid prompt: concise natural language lead plus valid booru tags."
	default:
		return "Output fluent natural language prompt with optional booru tags only when they strengthen precision."
	}
}
