package prompting

import (
	"fmt"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/domain"
)

type AssemblyInput struct {
	Mode        domain.Mode
	SystemRules string
	Persona     string
	UserPrompt  string
	Knowledge   []domain.KnowledgeDoc
	Retrieval   domain.RetrievalResult
	CreateMode  bool
}

type AssemblyOutput struct {
	SystemPrompt string
	UserPrompt   string
}

func Assemble(in AssemblyInput) AssemblyOutput {
	systemParts := []string{
		strings.TrimSpace(in.SystemRules),
		strings.TrimSpace(in.Persona),
		"Mode rules: " + modeRules(in.Mode),
		outputContract(),
	}

	for _, doc := range in.Knowledge {
		if strings.TrimSpace(doc.Content) == "" {
			continue
		}
		systemParts = append(systemParts, "Knowledge ("+doc.Name+"):\n"+strings.TrimSpace(doc.Content))
	}

	confirmed := tagsToLine("confirmed_tags", in.Retrieval.ConfirmedTags)
	character := tagsToLine("character_tags", in.Retrieval.CharacterTags)
	suggested := tagsToLine("suggested_tags", in.Retrieval.SuggestedTags)
	rejected := rejectedTagsToLine(in.Retrieval.RejectedTags)

	intent := "Enhance this prompt while preserving user intent."
	if in.CreateMode {
		intent = "Create a high-quality prompt from this idea."
	}

	userPrompt := fmt.Sprintf(
		"%s\n\nUser input:\n%s\n\nRetrieval context:\n%s\n%s\n%s\n%s",
		intent,
		strings.TrimSpace(in.UserPrompt),
		confirmed,
		character,
		suggested,
		rejected,
	)

	return AssemblyOutput{
		SystemPrompt: strings.Join(systemParts, "\n\n"),
		UserPrompt:   userPrompt,
	}
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
