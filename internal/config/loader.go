package config

import (
	"encoding/json"
	"os"

	"github.com/kidixdev/PromptSensei/internal/utils"
)

func Load(paths Paths) (Config, error) {
	data, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		return Config{}, err
	}

	if len(data) == 0 {
		cfg := Default(paths)
		if err := Save(paths, cfg); err != nil {
			return Config{}, err
		}
		return cfg, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	cfg.ApplyDefaults(paths)
	if err := Save(paths, cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func Save(paths Paths, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return utils.WriteFileAtomic(paths.ConfigFile, data, 0o644)
}
