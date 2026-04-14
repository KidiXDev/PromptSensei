package csv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTagsAndCharacters(t *testing.T) {
	dir := t.TempDir()

	tagPath := filepath.Join(dir, "tag.csv")
	tagCSV := "tag,category,post_count,alternative\n1girl,0,100,\"solo female,1girls\"\n"
	if err := os.WriteFile(tagPath, []byte(tagCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	charPath := filepath.Join(dir, "danbooru_character.csv")
	charCSV := "character,copyright,trigger,core_tags,count,solo_count,url\nhatsune_miku,vocaloid,\"hatsune miku,miku\",\"1girl,twintails\",100,80,https://example.com\n"
	if err := os.WriteFile(charPath, []byte(charCSV), 0o644); err != nil {
		t.Fatal(err)
	}

	tags, err := LoadTags(tagPath)
	if err != nil {
		t.Fatalf("load tags: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
	if tags[0].Tag != "1girl" {
		t.Fatalf("unexpected tag name: %s", tags[0].Tag)
	}
	if len(tags[0].Alternatives) != 2 {
		t.Fatalf("expected 2 alternatives, got %d", len(tags[0].Alternatives))
	}

	chars, err := LoadCharacters(charPath)
	if err != nil {
		t.Fatalf("load characters: %v", err)
	}
	if len(chars) != 1 {
		t.Fatalf("expected 1 character, got %d", len(chars))
	}
	if chars[0].Character != "hatsune_miku" {
		t.Fatalf("unexpected character name: %s", chars[0].Character)
	}
	if len(chars[0].Triggers) != 2 {
		t.Fatalf("expected 2 triggers, got %d", len(chars[0].Triggers))
	}
}
