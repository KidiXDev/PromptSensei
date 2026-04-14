package prompting

import (
	"fmt"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/domain"
)

type AssemblyInput struct {
	Mode        domain.Mode
	SystemRules string
	UserPrompt  string
	Knowledge   []domain.KnowledgeDoc
	Retrieval   domain.RetrievalResult
	CreateMode  bool
}

type AssemblyOutput struct {
	SystemPrompt string
	UserPrompts  []string
}

func Assemble(in AssemblyInput) AssemblyOutput {
	systemParts := []string{
		strings.TrimSpace(in.SystemRules),
		coreLogic(),
		retrievalGuidance(),
		detailingInstruction(),
		weightInstruction(),
		"Mode rules: " + modeRules(in.Mode),
		"Technique stack: intent preservation, retrieval grounding, conflict avoidance, concise output shaping, character-consistent detailing.",
		outputContract(),
	}

	confirmed := tagsToLine("confirmed_tags", in.Retrieval.ConfirmedTags)
	character := tagsToLine("character_tags", in.Retrieval.CharacterTags)
	suggested := tagsToLine("suggested_tags", in.Retrieval.SuggestedTags)
	rejected := rejectedTagsToLine(in.Retrieval.RejectedTags)
	suggestedDetails := candidateDetailsLine("suggested_tag_details", in.Retrieval.SuggestedTags)
	confirmedDetails := candidateDetailsLine("confirmed_tag_details", in.Retrieval.ConfirmedTags)

	intent := "Enhance this prompt while preserving user intent."
	if in.CreateMode {
		intent = "Create a high-quality prompt from this idea."
	}

	intentPrompt := fmt.Sprintf(
		"Task intent:\n%s\n\nUser input:\n%s\n\nPrompt checks:\n%s",
		intent,
		strings.TrimSpace(in.UserPrompt),
		strings.Join(promptChecks(in.Mode), "\n"),
	)
	retrievalPrompt := fmt.Sprintf(
		"Retrieval summary:\n%s\n%s\n%s\n%s\n\nRetrieval details:\n%s\n%s\n\nImportant:\n- `suggested_tags` are optional hints, not mandatory output.\n- Keep high-confidence required identity details from matched character context unless user explicitly conflicts.\n- Suggested tags are shown because of lookup matches and character/core-tag correlation; use only when they strengthen coherence.",
		confirmed,
		character,
		suggested,
		rejected,
		confirmedDetails,
		suggestedDetails,
	)

	userPrompts := []string{
		intentPrompt,
		retrievalPrompt,
	}
	for idx, characterCtx := range in.Retrieval.Characters {
		userPrompts = append(userPrompts, buildCharacterContextPrompt(idx+1, characterCtx))
	}
	for _, doc := range in.Knowledge {
		if strings.TrimSpace(doc.Content) == "" {
			continue
		}
		userPrompts = append(userPrompts, fmt.Sprintf("Knowledge context (%s):\n%s", doc.Name, strings.TrimSpace(doc.Content)))
	}
	userPrompts = append(userPrompts,
		"Final generation rules:\n- Strict ordering: metadata -> subject -> character -> background/environment -> lighting -> composition.\n- Enrichment: If the user input is simple, you MUST add descriptive scene details (environment, lighting, mood) to achieve high-quality results.\n- Use required identity anchors for characters.\n- Return ONLY the final prompt line, no markdown, no explanation.",
	)

	return AssemblyOutput{
		SystemPrompt: strings.Join(systemParts, "\n\n"),
		UserPrompts:  userPrompts,
	}
}

func candidateDetailsLine(label string, tags []domain.TagCandidate) string {
	if len(tags) == 0 {
		return label + ": (none)"
	}
	items := make([]string, 0, len(tags))
	for _, tag := range tags {
		reason := strings.TrimSpace(tag.Reason)
		if reason == "" {
			items = append(items, tag.Name)
			continue
		}
		items = append(items, tag.Name+" <- "+reason)
	}
	return label + ": " + strings.Join(items, "; ")
}

func buildCharacterContextPrompt(index int, ctx domain.CharacterRetrievalContext) string {
	anchor := "(none)"
	if len(ctx.AnchorTags) > 0 {
		anchor = strings.Join(ctx.AnchorTags, ", ")
	}
	suggested := "(none)"
	if len(ctx.SuggestedTags) > 0 {
		suggested = strings.Join(ctx.SuggestedTags, ", ")
	}
	return fmt.Sprintf(
		"Character context #%d:\n- name: %s\n- matched_by: %s\n- matched_term: %s\n- copyright: %s\n- required_identity_anchors: %s\n- optional_character_suggestions: %s\n\nGuidance:\n- Keep required identity anchors unless they directly conflict with explicit user instructions.\n- Optional suggestions can be ignored when they reduce coherence.",
		index,
		emptyFallback(ctx.Name),
		emptyFallback(ctx.MatchType),
		emptyFallback(ctx.MatchedTerm),
		emptyFallback(ctx.CopyrightName),
		anchor,
		suggested,
	)
}

func emptyFallback(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "(none)"
	}
	return value
}

func promptChecks(mode domain.Mode) []string {
	checks := []string{
		"- Keep core subject and scene intent from user input.",
		"- Prefer high-confidence retrieved tags over speculative tags.",
		"- Remove contradictory or duplicate descriptors.",
		"- Maintain stable style and composition details.",
	}
	if mode == domain.ModeBooru {
		checks = append(checks, "- Output must remain booru tag style only.")
	}
	if mode == domain.ModeHybrid {
		checks = append(checks, "- Keep natural language concise before/around tags.")
	}
	return checks
}

func tagsToLine(label string, tags []domain.TagCandidate) string {
	if len(tags) == 0 {
		return label + ": (none)"
	}
	items := make([]string, 0, len(tags))
	for _, tag := range tags {
		items = append(items, tag.Name)
	}
	return label + ": " + strings.Join(items, ", ")
}

func rejectedTagsToLine(tags []domain.RejectedTag) string {
	if len(tags) == 0 {
		return "rejected_tags: (none)"
	}
	items := make([]string, 0, len(tags))
	for _, tag := range tags {
		items = append(items, tag.Name+" ("+tag.Reason+")")
	}
	return "rejected_tags: " + strings.Join(items, ", ")
}
