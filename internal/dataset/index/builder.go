package index

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kidixdev/PromptSensei/internal/config"
	csvloader "github.com/kidixdev/PromptSensei/internal/dataset/csv"
	"github.com/kidixdev/PromptSensei/internal/dataset/sqlite"
	"github.com/kidixdev/PromptSensei/internal/utils"
)

func Build(ctx context.Context, paths config.DatasetPaths, schemaVersion int) (Metadata, error) {
	return BuildWithProgress(ctx, paths, schemaVersion, nil)
}

func BuildWithProgress(ctx context.Context, paths config.DatasetPaths, schemaVersion int, progress ProgressFunc) (Metadata, error) {
	report(progress, "load_tags", 0, 0, "loading tag.csv")
	tags, err := csvloader.LoadTags(paths.TagCSV)
	if err != nil {
		return Metadata{}, fmt.Errorf("load tags: %w", err)
	}
	report(progress, "load_tags", len(tags), len(tags), "tag.csv loaded")

	report(progress, "load_characters", 0, 0, "loading danbooru_character.csv")
	characters, err := csvloader.LoadCharacters(paths.CharacterCSV)
	if err != nil {
		return Metadata{}, fmt.Errorf("load characters: %w", err)
	}
	report(progress, "load_characters", len(characters), len(characters), "character csv loaded")

	if err := utils.EnsureDir(filepath.Dir(paths.DBPath)); err != nil {
		return Metadata{}, err
	}

	tmpDBPath := paths.DBPath + ".tmp"
	_ = os.Remove(tmpDBPath)

	repo, err := sqlite.Open(tmpDBPath)
	if err != nil {
		return Metadata{}, err
	}
	db := repo.DB()

	report(progress, "create_schema", 0, 0, "creating sqlite schema")
	if err := sqlite.CreateSchema(ctx, db); err != nil {
		_ = repo.Close()
		return Metadata{}, err
	}
	report(progress, "create_schema", 1, 1, "sqlite schema ready")

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		_ = repo.Close()
		return Metadata{}, err
	}

	report(progress, "insert_tags", 0, len(tags), "indexing tags")
	tagIDs, err := sqlite.InsertTags(ctx, tx, tags, func(done int, total int) {
		report(progress, "insert_tags", done, total, "indexing tags")
	})
	if err != nil {
		_ = tx.Rollback()
		_ = repo.Close()
		return Metadata{}, err
	}
	report(progress, "insert_characters", 0, len(characters), "indexing characters")
	if err := sqlite.InsertCharacters(ctx, tx, characters, tagIDs, func(done int, total int) {
		report(progress, "insert_characters", done, total, "indexing characters")
	}); err != nil {
		_ = tx.Rollback()
		_ = repo.Close()
		return Metadata{}, err
	}

	report(progress, "commit", 0, 0, "committing transaction")
	if err := tx.Commit(); err != nil {
		_ = repo.Close()
		return Metadata{}, err
	}
	report(progress, "commit", 1, 1, "transaction committed")

	report(progress, "count_rows", 0, 0, "counting indexed rows")
	counts, err := repo.Counts(ctx)
	if err != nil {
		_ = repo.Close()
		return Metadata{}, err
	}
	if err := repo.Close(); err != nil {
		return Metadata{}, err
	}

	report(progress, "swap_db", 0, 0, "replacing cache database")
	if err := replaceDBAtomically(tmpDBPath, paths.DBPath); err != nil {
		return Metadata{}, err
	}
	report(progress, "swap_db", 1, 1, "cache database replaced")

	report(progress, "hash_csv", 0, 0, "computing csv hashes")
	tagHash, err := utils.FileSHA256(paths.TagCSV)
	if err != nil {
		return Metadata{}, err
	}
	charHash, err := utils.FileSHA256(paths.CharacterCSV)
	if err != nil {
		return Metadata{}, err
	}

	meta := Metadata{
		TagCSVPath:       paths.TagCSV,
		CharacterCSVPath: paths.CharacterCSV,
		TagCSVHash:       tagHash,
		CharacterCSVHash: charHash,
		LastRebuildUTC:   time.Now().UTC().Format(time.RFC3339),
		SchemaVersion:    schemaVersion,
		Counts:           counts,
	}
	report(progress, "done", 1, 1, "dataset rebuild completed")
	return meta, nil
}

func replaceDBAtomically(tmpPath string, targetPath string) error {
	backupPath := targetPath + ".bak"
	_ = os.Remove(backupPath)

	if utils.FileExists(targetPath) {
		if err := os.Rename(targetPath, backupPath); err != nil {
			return fmt.Errorf("backup old db: %w", err)
		}
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		if utils.FileExists(backupPath) {
			_ = os.Rename(backupPath, targetPath)
		}
		return fmt.Errorf("replace db: %w", err)
	}

	_ = os.Remove(backupPath)
	return nil
}
