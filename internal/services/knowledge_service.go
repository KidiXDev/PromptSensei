package services

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/domain"
)

type KnowledgeService struct {
	knowledgeDir string
}

func NewKnowledgeService(paths config.Paths) *KnowledgeService {
	return &KnowledgeService{knowledgeDir: paths.KnowledgeDir}
}

func (s *KnowledgeService) List() ([]string, error) {
	entries, err := os.ReadDir(s.knowledgeDir)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".md") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *KnowledgeService) LoadSelected(selected []string) ([]domain.KnowledgeDoc, []string) {
	if len(selected) == 0 {
		return nil, nil
	}

	var out []domain.KnowledgeDoc
	var warnings []string
	for _, name := range selected {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if filepath.Ext(name) == "" {
			name += ".md"
		}
		path := filepath.Join(s.knowledgeDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			warnings = append(warnings, "knowledge not found: "+name)
			continue
		}
		out = append(out, domain.KnowledgeDoc{
			Name:    name,
			Path:    path,
			Content: string(data),
		})
	}
	return out, warnings
}
