package services

import (
	"os"
	"regexp"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/config"
)

var commentRegex = regexp.MustCompile(`(?s)<!--.*?-->`)

type InstructionService struct {
	paths config.Paths
}

func NewInstructionService(paths config.Paths) *InstructionService {
	return &InstructionService{paths: paths}
}

func (s *InstructionService) LoadSystem() (string, error) {
	data, err := os.ReadFile(s.paths.SystemFile)
	if err != nil {
		return "", err
	}
	content := string(data)
	content = commentRegex.ReplaceAllString(content, "")
	return strings.TrimSpace(content), nil
}
