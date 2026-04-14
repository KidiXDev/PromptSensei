package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/dataset/sqlite"
)

func TestBuildCreatesSQLiteIndex(t *testing.T) {
	dir := t.TempDir()
	tagPath := filepath.Join(dir, "tag.csv")
	charPath := filepath.Join(dir, "danbooru_character.csv")
	dbPath := filepath.Join(dir, "danbooru.db")
	metaPath := filepath.Join(dir, "dataset.meta.json")

	tagCSV := "tag,category,post_count,alternative\n1girl,0,100,\"solo female\"\nhatsune_miku,4,500,\"hatsune miku,miku\"\ntwintails,0,300,\n"
	charCSV := "character,copyright,trigger,core_tags,count,solo_count,url\nhatsune_miku,vocaloid,\"hatsune miku,miku\",\"1girl,twintails\",500,450,https://example.com\n"
	if err := os.WriteFile(tagPath, []byte(tagCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(charPath, []byte(charCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	paths := config.DatasetPaths{
		TagCSV:       tagPath,
		CharacterCSV: charPath,
		DBPath:       dbPath,
		MetadataPath: metaPath,
	}

	meta, err := Build(context.Background(), paths, 1)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}
	if meta.Counts.Tags != 3 {
		t.Fatalf("expected 3 tags, got %d", meta.Counts.Tags)
	}
	if err := SaveMetadata(metaPath, meta); err != nil {
		t.Fatalf("save metadata: %v", err)
	}

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer repo.Close()

	counts, err := repo.Counts(context.Background())
	if err != nil {
		t.Fatalf("counts query: %v", err)
	}
	if counts.Characters != 1 {
		t.Fatalf("expected 1 character, got %d", counts.Characters)
	}
}

func TestBuildWithProgressEmitsStages(t *testing.T) {
	dir := t.TempDir()
	tagPath := filepath.Join(dir, "tag.csv")
	charPath := filepath.Join(dir, "danbooru_character.csv")
	dbPath := filepath.Join(dir, "danbooru.db")
	metaPath := filepath.Join(dir, "dataset.meta.json")

	tagCSV := "tag,category,post_count,alternative\n1girl,0,100,\n"
	charCSV := "character,copyright,trigger,core_tags,count,solo_count,url\nhatsune_miku,vocaloid,\"hatsune miku\",\"1girl\",500,450,https://example.com\n"
	if err := os.WriteFile(tagPath, []byte(tagCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(charPath, []byte(charCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	paths := config.DatasetPaths{
		TagCSV:       tagPath,
		CharacterCSV: charPath,
		DBPath:       dbPath,
		MetadataPath: metaPath,
	}

	var stages []string
	_, err := BuildWithProgress(context.Background(), paths, 1, func(p Progress) {
		stages = append(stages, p.Stage)
	})
	if err != nil {
		t.Fatalf("build with progress: %v", err)
	}

	if len(stages) == 0 {
		t.Fatalf("expected progress stages, got none")
	}
	if stages[len(stages)-1] != "done" {
		t.Fatalf("expected final stage 'done', got %q", stages[len(stages)-1])
	}
}
