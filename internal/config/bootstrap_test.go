package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapCreatesDefaultStructure(t *testing.T) {
	root := t.TempDir()
	paths := BuildPaths(filepath.Join(root, "prompt-sensei"))

	if err := Bootstrap(paths); err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	required := []string{
		paths.RootDir,
		paths.ConfigFile,
		paths.SystemFile,
		paths.TagCSV,
		paths.CharacterCSV,
	}
	for _, p := range required {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s to exist: %v", p, err)
		}
	}
}

func TestResolvePathsPrefersUserProfileDotConfig(t *testing.T) {
	t.Setenv(EnvConfigDir, "")
	t.Setenv("USERPROFILE", `C:\Users\Sensei`)

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}

	expected := filepath.Join(`C:\Users\Sensei`, ".config", AppDirName)
	if paths.RootDir != expected {
		t.Fatalf("expected root %s, got %s", expected, paths.RootDir)
	}
}
