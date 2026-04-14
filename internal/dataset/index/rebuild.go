package index

import (
	"fmt"
	"os"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/utils"
)

type RebuildDecision struct {
	Needed        bool
	Reasons       []string
	TagHash       string
	CharacterHash string
}

func ShouldRebuild(paths config.DatasetPaths, metadataPath string, dbPath string, schemaVersion int) (RebuildDecision, error) {
	tagHash, err := utils.FileSHA256(paths.TagCSV)
	if err != nil {
		return RebuildDecision{}, fmt.Errorf("hash tag csv: %w", err)
	}
	charHash, err := utils.FileSHA256(paths.CharacterCSV)
	if err != nil {
		return RebuildDecision{}, fmt.Errorf("hash character csv: %w", err)
	}

	meta, err := LoadMetadata(metadataPath)
	if err != nil {
		return RebuildDecision{}, err
	}

	var reasons []string
	if _, err := os.Stat(dbPath); err != nil {
		reasons = append(reasons, "sqlite cache missing")
	}
	if meta == nil {
		reasons = append(reasons, "metadata missing")
	} else {
		if meta.SchemaVersion != schemaVersion {
			reasons = append(reasons, "schema version changed")
		}
		if meta.TagCSVHash != tagHash {
			reasons = append(reasons, "tag.csv hash changed")
		}
		if meta.CharacterCSVHash != charHash {
			reasons = append(reasons, "danbooru_character.csv hash changed")
		}
	}

	return RebuildDecision{
		Needed:        len(reasons) > 0,
		Reasons:       reasons,
		TagHash:       tagHash,
		CharacterHash: charHash,
	}, nil
}
