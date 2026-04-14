package services

import (
	"os"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/config"
)

type InstructionService struct {
	paths config.Paths
}

func NewInstructionService(paths config.Paths) *InstructionService {
	return &InstructionService{paths: paths}
}

func (s *InstructionService) LoadPersona() (string, error) {
	data, err := os.ReadFile(s.paths.PersonaFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (s *InstructionService) LoadSystem() (string, error) {
	data, err := os.ReadFile(s.paths.SystemFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
