package domain

import (
	"fmt"
	"strings"
)

type Mode string

const (
	ModeNatural Mode = "natural"
	ModeBooru   Mode = "booru"
	ModeHybrid  Mode = "hybrid"
)

func ParseMode(raw string) (Mode, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(raw))) {
	case "", ModeNatural:
		return ModeNatural, nil
	case ModeBooru:
		return ModeBooru, nil
	case ModeHybrid:
		return ModeHybrid, nil
	default:
		return "", fmt.Errorf("unsupported mode %q", raw)
	}
}

type EnhanceRequest struct {
	Prompt         string
	Context        string
	Mode           Mode
	KnowledgeFiles []string
	StrictBooru    bool
	CreateMode     bool
}

type EnhanceResult struct {
	Output            string
	Retrieval         RetrievalResult
	SystemPrompt      string
	UserPrompt        string
	ProviderName      string
	UsedProvider      bool
	ValidationApplied bool
	ChainApplied      bool
	ChainStages       int
}
