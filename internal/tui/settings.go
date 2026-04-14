package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/domain"
)

const (
	settingGeneralDefaultMode        = "general.default_mode"
	settingGeneralStrictValidation   = "general.strict_booru_validation"
	settingProviderEnabled           = "provider.enabled"
	settingProviderName              = "provider.name"
	settingProviderAPIBaseURL        = "provider.api_base_url"
	settingProviderAPIKey            = "provider.api_key"
	settingProviderModel             = "provider.model"
	settingProviderTemperature       = "provider.temperature"
	settingProviderMaxTokens         = "provider.max_tokens"
	settingProviderTimeoutSeconds    = "provider.timeout_seconds"
	settingUIShowDebugPanel          = "ui.show_debug_panel"
	settingUIShowRetrievalCandidates = "ui.show_retrieval_candidates"
	settingDatasetTagCSVPath         = "dataset.tag_csv_path"
	settingDatasetCharacterCSVPath   = "dataset.character_csv_path"
	settingDatasetCachePath          = "dataset.cache_path"
	settingDatasetMetadataPath       = "dataset.metadata_path"
	settingDatasetAutoRebuild        = "dataset.auto_rebuild_on_csv_change"
	settingDatasetSchemaVersion      = "dataset.schema_version"
)

type settingKind int

const (
	settingKindString settingKind = iota
	settingKindSecret
	settingKindBool
	settingKindEnum
	settingKindFloat
	settingKindInt
)

type settingField struct {
	key         string
	group       string
	title       string
	description string
	kind        settingKind
	enumValues  []string
}

type settingItem struct {
	field settingField
	value string
}

func (i settingItem) Title() string {
	return fmt.Sprintf("[%s] %s: %s", i.field.group, i.field.title, i.value)
}

func (i settingItem) Description() string {
	return i.field.description
}

func (i settingItem) FilterValue() string {
	return i.field.group + " " + i.field.title + " " + i.value
}

var settingsFields = []settingField{
	{key: settingGeneralDefaultMode, group: "General", title: "Default Mode", description: "Default prompting mode", kind: settingKindEnum, enumValues: []string{"natural", "booru", "hybrid"}},
	{key: settingGeneralStrictValidation, group: "General", title: "Strict Booru Validation", description: "Filter booru output to retrieved tags only", kind: settingKindBool},
	{key: settingProviderEnabled, group: "Provider", title: "Enabled", description: "Enable online provider calls", kind: settingKindBool},
	{key: settingProviderName, group: "Provider", title: "Name", description: "Active provider integration", kind: settingKindEnum, enumValues: []string{"openai", "openrouter", "nanogpt"}},
	{key: settingProviderAPIBaseURL, group: "Provider", title: "API Base URL", description: "Override API endpoint", kind: settingKindString},
	{key: settingProviderAPIKey, group: "Provider", title: "API Key", description: "Authentication token", kind: settingKindSecret},
	{key: settingProviderModel, group: "Provider", title: "Model", description: "Model identifier used for calls", kind: settingKindString},
	{key: settingProviderTemperature, group: "Provider", title: "Temperature", description: "Sampling creativity (0-2)", kind: settingKindFloat},
	{key: settingProviderMaxTokens, group: "Provider", title: "Max Tokens", description: "Completion token budget", kind: settingKindInt},
	{key: settingProviderTimeoutSeconds, group: "Provider", title: "Timeout Seconds", description: "HTTP timeout for provider requests", kind: settingKindInt},
	{key: settingUIShowDebugPanel, group: "UI", title: "Show Debug Panel", description: "Display debug details in UI views", kind: settingKindBool},
	{key: settingUIShowRetrievalCandidates, group: "UI", title: "Show Retrieval Candidates", description: "Display retrieval candidate blocks", kind: settingKindBool},
	{key: settingDatasetAutoRebuild, group: "Dataset", title: "Auto Rebuild on CSV Change", description: "Rebuild cache automatically when CSV changes", kind: settingKindBool},
	{key: settingDatasetTagCSVPath, group: "Dataset", title: "Tag CSV Path", description: "Path to tag.csv", kind: settingKindString},
	{key: settingDatasetCharacterCSVPath, group: "Dataset", title: "Character CSV Path", description: "Path to danbooru_character.csv", kind: settingKindString},
	{key: settingDatasetCachePath, group: "Dataset", title: "Cache DB Path", description: "Path to generated sqlite cache", kind: settingKindString},
	{key: settingDatasetMetadataPath, group: "Dataset", title: "Metadata Path", description: "Path to dataset metadata JSON", kind: settingKindString},
	{key: settingDatasetSchemaVersion, group: "Dataset", title: "Schema Version", description: "Dataset schema version integer", kind: settingKindInt},
}

func buildSettingsItems(cfg config.Config) []list.Item {
	items := make([]list.Item, 0, len(settingsFields))
	for _, field := range settingsFields {
		items = append(items, settingItem{
			field: field,
			value: displaySettingValue(cfg, field.key),
		})
	}
	return items
}

func displaySettingValue(cfg config.Config, key string) string {
	switch key {
	case settingGeneralDefaultMode:
		return string(cfg.General.DefaultMode)
	case settingGeneralStrictValidation:
		return fmt.Sprintf("%t", cfg.General.StrictBooruValidation)
	case settingProviderEnabled:
		return fmt.Sprintf("%t", cfg.Provider.Enabled)
	case settingProviderName:
		return strings.TrimSpace(cfg.Provider.Name)
	case settingProviderAPIBaseURL:
		return strings.TrimSpace(cfg.Provider.APIBaseURL)
	case settingProviderAPIKey:
		if strings.TrimSpace(cfg.Provider.APIKey) == "" {
			return "(empty)"
		}
		return "********"
	case settingProviderModel:
		return strings.TrimSpace(cfg.Provider.Model)
	case settingProviderTemperature:
		return fmt.Sprintf("%.2f", cfg.Provider.Temperature)
	case settingProviderMaxTokens:
		return strconv.Itoa(cfg.Provider.MaxTokens)
	case settingProviderTimeoutSeconds:
		return strconv.Itoa(cfg.Provider.TimeoutSeconds)
	case settingUIShowDebugPanel:
		return fmt.Sprintf("%t", cfg.UI.ShowDebugPanel)
	case settingUIShowRetrievalCandidates:
		return fmt.Sprintf("%t", cfg.UI.ShowRetrievalCandidates)
	case settingDatasetTagCSVPath:
		return strings.TrimSpace(cfg.Dataset.TagCSVPath)
	case settingDatasetCharacterCSVPath:
		return strings.TrimSpace(cfg.Dataset.CharacterCSVPath)
	case settingDatasetCachePath:
		return strings.TrimSpace(cfg.Dataset.CachePath)
	case settingDatasetMetadataPath:
		return strings.TrimSpace(cfg.Dataset.MetadataPath)
	case settingDatasetAutoRebuild:
		return fmt.Sprintf("%t", cfg.Dataset.AutoRebuildOnCSVChange)
	case settingDatasetSchemaVersion:
		return strconv.Itoa(cfg.Dataset.SchemaVersion)
	default:
		return ""
	}
}

func rawSettingValue(cfg config.Config, key string) string {
	switch key {
	case settingProviderAPIKey:
		return cfg.Provider.APIKey
	default:
		return displaySettingValue(cfg, key)
	}
}

func applySettingValue(cfg *config.Config, field settingField, raw string) error {
	raw = strings.TrimSpace(raw)

	switch field.key {
	case settingGeneralDefaultMode:
		mode, err := domain.ParseMode(raw)
		if err != nil {
			return err
		}
		cfg.General.DefaultMode = mode
	case settingGeneralStrictValidation:
		v, err := parseBool(raw)
		if err != nil {
			return err
		}
		cfg.General.StrictBooruValidation = v
	case settingProviderEnabled:
		v, err := parseBool(raw)
		if err != nil {
			return err
		}
		cfg.Provider.Enabled = v
	case settingProviderName:
		next := normalizeProviderName(raw)
		prevProvider := normalizeProviderName(cfg.Provider.Name)
		prevBase := strings.TrimSpace(cfg.Provider.APIBaseURL)
		cfg.Provider.Name = next
		if prevBase == "" || prevBase == defaultAPIBase(prevProvider) {
			cfg.Provider.APIBaseURL = defaultAPIBase(next)
		}
	case settingProviderAPIBaseURL:
		cfg.Provider.APIBaseURL = raw
	case settingProviderAPIKey:
		cfg.Provider.APIKey = raw
	case settingProviderModel:
		cfg.Provider.Model = raw
	case settingProviderTemperature:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("temperature must be numeric")
		}
		if v < 0 || v > 2 {
			return fmt.Errorf("temperature must be between 0 and 2")
		}
		cfg.Provider.Temperature = v
	case settingProviderMaxTokens:
		v, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("max tokens must be integer")
		}
		if v < 64 {
			return fmt.Errorf("max tokens must be at least 64")
		}
		cfg.Provider.MaxTokens = v
	case settingProviderTimeoutSeconds:
		v, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("timeout seconds must be integer")
		}
		if v < 5 {
			return fmt.Errorf("timeout must be at least 5 seconds")
		}
		cfg.Provider.TimeoutSeconds = v
	case settingUIShowDebugPanel:
		v, err := parseBool(raw)
		if err != nil {
			return err
		}
		cfg.UI.ShowDebugPanel = v
	case settingUIShowRetrievalCandidates:
		v, err := parseBool(raw)
		if err != nil {
			return err
		}
		cfg.UI.ShowRetrievalCandidates = v
	case settingDatasetTagCSVPath:
		cfg.Dataset.TagCSVPath = raw
	case settingDatasetCharacterCSVPath:
		cfg.Dataset.CharacterCSVPath = raw
	case settingDatasetCachePath:
		cfg.Dataset.CachePath = raw
	case settingDatasetMetadataPath:
		cfg.Dataset.MetadataPath = raw
	case settingDatasetAutoRebuild:
		v, err := parseBool(raw)
		if err != nil {
			return err
		}
		cfg.Dataset.AutoRebuildOnCSVChange = v
	case settingDatasetSchemaVersion:
		v, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("schema version must be integer")
		}
		if v <= 0 {
			return fmt.Errorf("schema version must be > 0")
		}
		cfg.Dataset.SchemaVersion = v
	default:
		return fmt.Errorf("unsupported setting %s", field.key)
	}
	return nil
}

func cycleSettingValue(cfg *config.Config, field settingField, direction int) (bool, error) {
	if direction == 0 {
		direction = 1
	}
	switch field.kind {
	case settingKindBool:
		current := strings.EqualFold(displaySettingValue(*cfg, field.key), "true")
		next := !current
		return true, applySettingValue(cfg, field, strconv.FormatBool(next))
	case settingKindEnum:
		options := field.enumValues
		if len(options) == 0 {
			return false, nil
		}
		current := strings.ToLower(strings.TrimSpace(displaySettingValue(*cfg, field.key)))
		idx := 0
		for i := range options {
			if options[i] == current {
				idx = i
				break
			}
		}
		if direction > 0 {
			idx = (idx + 1) % len(options)
		} else {
			idx = (idx - 1 + len(options)) % len(options)
		}
		return true, applySettingValue(cfg, field, options[idx])
	default:
		return false, nil
	}
}

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "y", "on":
		return true, nil
	case "false", "0", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("expected boolean value (true/false)")
	}
}

func normalizeProviderName(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "openai":
		return "openai"
	case "openrouter":
		return "openrouter"
	case "nanogpt", "nano-gpt":
		return "nanogpt"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func defaultAPIBase(provider string) string {
	switch normalizeProviderName(provider) {
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "nanogpt":
		return "https://nano-gpt.com/api/v1"
	default:
		return "https://api.openai.com/v1"
	}
}
