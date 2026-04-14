package app

import (
	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/services"
)

type Runtime struct {
	Paths        config.Paths
	Config       config.Config
	Dataset      *services.DatasetService
	Instructions *services.InstructionService
	Knowledge    *services.KnowledgeService
	Providers    *services.ProviderService
	Prompt       *services.PromptService
}
