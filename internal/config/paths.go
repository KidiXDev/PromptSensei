package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvConfigDir = "PROMPTSENSEI_CONFIG_DIR"
	AppDirName   = "prompt-sensei"
)

type Paths struct {
	RootDir        string
	ConfigFile     string
	InstructionDir string
	PersonaFile    string
	SystemFile     string
	KnowledgeDir   string
	SystemDir      string
	TagCSV         string
	CharacterCSV   string
	DatasetDB      string
	DatasetMeta    string
}

func ResolvePaths() (Paths, error) {
	if override := os.Getenv(EnvConfigDir); override != "" {
		return BuildPaths(override), nil
	}

	if userProfile := strings.TrimSpace(os.Getenv("USERPROFILE")); userProfile != "" {
		return BuildPaths(filepath.Join(userProfile, ".config", AppDirName)), nil
	}

	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return BuildPaths(filepath.Join(home, ".config", AppDirName)), nil
	}

	userCfg, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}

	return BuildPaths(filepath.Join(userCfg, AppDirName)), nil
}

func BuildPaths(root string) Paths {
	instructionDir := filepath.Join(root, "instruction")
	knowledgeDir := filepath.Join(root, "knowledge")
	systemDir := filepath.Join(root, "system")

	return Paths{
		RootDir:        root,
		ConfigFile:     filepath.Join(root, "config.cfg"),
		InstructionDir: instructionDir,
		PersonaFile:    filepath.Join(instructionDir, "persona.md"),
		SystemFile:     filepath.Join(instructionDir, "system.md"),
		KnowledgeDir:   knowledgeDir,
		SystemDir:      systemDir,
		TagCSV:         filepath.Join(systemDir, "tag.csv"),
		CharacterCSV:   filepath.Join(systemDir, "danbooru_character.csv"),
		DatasetDB:      filepath.Join(systemDir, "danbooru.db"),
		DatasetMeta:    filepath.Join(systemDir, "dataset.meta.json"),
	}
}
