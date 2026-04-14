package config

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/kidixdev/PromptSensei/assets"
	"github.com/kidixdev/PromptSensei/internal/utils"
)

func Bootstrap(paths Paths) error {
	dirs := []string{
		paths.RootDir,
		paths.InstructionDir,
		paths.KnowledgeDir,
		paths.SystemDir,
	}

	for _, dir := range dirs {
		if err := utils.EnsureDir(dir); err != nil {
			return err
		}
	}

	if err := installDefaults(paths); err != nil {
		return err
	}

	if !utils.FileExists(paths.ConfigFile) {
		cfg := Default(paths)
		if err := Save(paths, cfg); err != nil {
			return err
		}
	}

	return nil
}

func installDefaults(paths Paths) error {
	return fs.WalkDir(assets.DefaultsFS, "defaults", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(path, "defaults/")
		dst := mapDefaultPath(paths, rel)
		if utils.FileExists(dst) {
			return nil
		}

		data, err := assets.DefaultsFS.ReadFile(path)
		if err != nil {
			return err
		}

		return utils.WriteFileAtomic(dst, data, 0o644)
	})
}

func mapDefaultPath(paths Paths, rel string) string {
	switch rel {
	case "config.cfg":
		return paths.ConfigFile
	default:
		return filepath.Join(paths.RootDir, rel)
	}
}
