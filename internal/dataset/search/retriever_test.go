package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kidixdev/PromptSensei/internal/config"
	"github.com/kidixdev/PromptSensei/internal/dataset/index"
	"github.com/kidixdev/PromptSensei/internal/dataset/sqlite"
	"github.com/kidixdev/PromptSensei/internal/domain"
)

func TestRetrieverFindsCharacterAndTags(t *testing.T) {
	dir := t.TempDir()
	tagPath := filepath.Join(dir, "tag.csv")
	charPath := filepath.Join(dir, "danbooru_character.csv")
	dbPath := filepath.Join(dir, "danbooru.db")
	metaPath := filepath.Join(dir, "dataset.meta.json")

	tagCSV := "tag,category,post_count,alternative\n1girl,0,100,\"solo female\"\nhatsune_miku,4,500,\"hatsune miku,miku\"\ntwintails,0,300,\nvocaloid,3,800,\n"
	charCSV := "character,copyright,trigger,core_tags,count,solo_count,url\nhatsune_miku,vocaloid,\"hatsune miku,miku\",\"1girl,twintails\",500,450,https://example.com\n"
	if err := os.WriteFile(tagPath, []byte(tagCSV), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(charPath, []byte(charCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := index.Build(context.Background(), config.DatasetPaths{
		TagCSV:       tagPath,
		CharacterCSV: charPath,
		DBPath:       dbPath,
		MetadataPath: metaPath,
	}, 1)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer repo.Close()

	retriever := NewRetriever(repo)
	result, err := retriever.Retrieve(context.Background(), "hatsune miku in a city at night", domain.ModeHybrid)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(result.CharacterTags) == 0 {
		t.Fatalf("expected at least 1 character tag")
	}
	if !containsTag(result.CharacterTags, "hatsune_miku") {
		t.Fatalf("expected hatsune_miku character tag")
	}
	if !containsTag(result.SuggestedTags, "twintails") {
		t.Fatalf("expected twintails suggested tag")
	}
}

func TestBuildTermWeightsIncludesPhrases(t *testing.T) {
	weights := buildTermWeights("hatsune miku, neon city street at night")
	if weights["hatsune miku"] <= 0 {
		t.Fatalf("expected phrase weight for hatsune miku")
	}
	if weights["city street at"] <= 0 {
		t.Fatalf("expected trigram term weight")
	}
	if weights["neon city street at night"] <= 0 {
		t.Fatalf("expected full phrase term weight")
	}
}

func TestApplyConflictsPrefersExplicitPromptTag(t *testing.T) {
	filtered, rejected := applyConflicts("night city skyline", []domain.TagCandidate{
		{Name: "day", Score: 4.2, PostCount: 200},
		{Name: "night", Score: 3.1, PostCount: 100},
	})
	if containsTag(filtered, "day") {
		t.Fatalf("expected day tag to be rejected due to conflict")
	}
	if !containsTag(filtered, "night") {
		t.Fatalf("expected explicit night tag to remain")
	}
	if len(rejected) == 0 {
		t.Fatalf("expected rejected tags")
	}
}

func containsTag(tags []domain.TagCandidate, name string) bool {
	for _, t := range tags {
		if t.Name == name {
			return true
		}
	}
	return false
}
