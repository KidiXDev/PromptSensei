package config

import "github.com/kidixdev/PromptSensei/internal/domain"

type Config struct {
	General  GeneralConfig  `json:"general"`
	Provider ProviderConfig `json:"provider"`
	UI       UIConfig       `json:"ui"`
	Dataset  DatasetConfig  `json:"dataset"`
}

type GeneralConfig struct {
	DefaultMode           domain.Mode `json:"default_mode"`
	DefaultOutputStyle    string      `json:"default_output_style"`
	PreferredProvider     string      `json:"preferred_provider"`
	PreferredModel        string      `json:"preferred_model"`
	StrictBooruValidation bool        `json:"strict_booru_validation"`
}

type ProviderConfig struct {
	Enabled        bool    `json:"enabled"`
	Name           string  `json:"name"`
	APIBaseURL     string  `json:"api_base_url"`
	APIKey         string  `json:"api_key"`
	Model          string  `json:"model"`
	Temperature    float64 `json:"temperature"`
	MaxTokens      int     `json:"max_tokens"`
	TimeoutSeconds int     `json:"timeout_seconds"`
}

type UIConfig struct {
	ShowDebugPanel          bool `json:"show_debug_panel"`
	ShowRetrievalCandidates bool `json:"show_retrieval_candidates"`
}

type DatasetConfig struct {
	TagCSVPath             string `json:"tag_csv_path"`
	CharacterCSVPath       string `json:"character_csv_path"`
	CachePath              string `json:"cache_path"`
	MetadataPath           string `json:"metadata_path"`
	AutoRebuildOnCSVChange bool   `json:"auto_rebuild_on_csv_change"`
	SchemaVersion          int    `json:"schema_version"`
}

func Default(paths Paths) Config {
	return Config{
		General: GeneralConfig{
			DefaultMode:           domain.ModeNatural,
			DefaultOutputStyle:    "enhanced",
			PreferredProvider:     "openai",
			PreferredModel:        "gpt-5.4-mini",
			StrictBooruValidation: false,
		},
		Provider: ProviderConfig{
			Enabled:        false,
			Name:           "openai",
			APIBaseURL:     "",
			APIKey:         "",
			Model:          "gpt-5.4-mini",
			Temperature:    0.7,
			MaxTokens:      700,
			TimeoutSeconds: 60,
		},
		UI: UIConfig{
			ShowDebugPanel:          false,
			ShowRetrievalCandidates: true,
		},
		Dataset: DatasetConfig{
			TagCSVPath:             paths.TagCSV,
			CharacterCSVPath:       paths.CharacterCSV,
			CachePath:              paths.DatasetDB,
			MetadataPath:           paths.DatasetMeta,
			AutoRebuildOnCSVChange: true,
			SchemaVersion:          1,
		},
	}
}

func (c *Config) ApplyDefaults(paths Paths) {
	d := Default(paths)

	if c.General.DefaultMode == "" {
		c.General.DefaultMode = d.General.DefaultMode
	}
	if c.General.DefaultOutputStyle == "" {
		c.General.DefaultOutputStyle = d.General.DefaultOutputStyle
	}
	if c.General.PreferredProvider == "" {
		c.General.PreferredProvider = d.General.PreferredProvider
	}
	if c.General.PreferredModel == "" {
		c.General.PreferredModel = d.General.PreferredModel
	}

	if c.Provider.Name == "" {
		c.Provider.Name = d.Provider.Name
	}
	if c.Provider.Model == "" {
		c.Provider.Model = d.Provider.Model
	}
	if c.Provider.Temperature == 0 {
		c.Provider.Temperature = d.Provider.Temperature
	}
	if c.Provider.MaxTokens == 0 {
		c.Provider.MaxTokens = d.Provider.MaxTokens
	}
	if c.Provider.TimeoutSeconds <= 0 {
		c.Provider.TimeoutSeconds = d.Provider.TimeoutSeconds
	}

	if c.Dataset.TagCSVPath == "" {
		c.Dataset.TagCSVPath = d.Dataset.TagCSVPath
	}
	if c.Dataset.CharacterCSVPath == "" {
		c.Dataset.CharacterCSVPath = d.Dataset.CharacterCSVPath
	}
	if c.Dataset.CachePath == "" {
		c.Dataset.CachePath = d.Dataset.CachePath
	}
	if c.Dataset.MetadataPath == "" {
		c.Dataset.MetadataPath = d.Dataset.MetadataPath
	}
	if c.Dataset.SchemaVersion == 0 {
		c.Dataset.SchemaVersion = d.Dataset.SchemaVersion
	}
}
