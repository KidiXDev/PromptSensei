package index

import (
	"encoding/json"
	"os"
	"time"

	"github.com/kidixdev/PromptSensei/internal/dataset/sqlite"
	"github.com/kidixdev/PromptSensei/internal/utils"
)

type Metadata struct {
	TagCSVPath       string               `json:"tag_csv_path"`
	CharacterCSVPath string               `json:"character_csv_path"`
	TagCSVHash       string               `json:"tag_csv_hash"`
	CharacterCSVHash string               `json:"character_csv_hash"`
	LastRebuildUTC   string               `json:"last_rebuild_utc"`
	SchemaVersion    int                  `json:"schema_version"`
	Counts           sqlite.DatasetCounts `json:"counts"`
}

func LoadMetadata(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func SaveMetadata(path string, meta Metadata) error {
	if meta.LastRebuildUTC == "" {
		meta.LastRebuildUTC = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return utils.WriteFileAtomic(path, data, 0o644)
}
