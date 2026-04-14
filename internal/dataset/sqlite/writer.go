package sqlite

import (
	"context"
	"database/sql"

	csvloader "github.com/kidixdev/PromptSensei/internal/dataset/csv"
	"github.com/kidixdev/PromptSensei/internal/dataset/normalize"
)

type ProgressFunc func(done int, total int)

func InsertTags(ctx context.Context, tx *sql.Tx, tags []csvloader.TagRecord, onProgress ProgressFunc) (map[string]int64, error) {
	insertTagStmt, err := tx.PrepareContext(ctx, `
INSERT INTO tags(name, normalized_name, category, post_count)
VALUES(?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
	normalized_name = excluded.normalized_name,
	category = excluded.category,
	post_count = excluded.post_count`)
	if err != nil {
		return nil, err
	}
	defer insertTagStmt.Close()

	insertAliasStmt, err := tx.PrepareContext(ctx, `
INSERT INTO tag_aliases(tag_id, alias, normalized_alias)
VALUES(?, ?, ?)
ON CONFLICT DO NOTHING`)
	if err != nil {
		return nil, err
	}
	defer insertAliasStmt.Close()

	selectIDStmt, err := tx.PrepareContext(ctx, `SELECT id FROM tags WHERE name = ?`)
	if err != nil {
		return nil, err
	}
	defer selectIDStmt.Close()

	tagIDs := make(map[string]int64, len(tags))
	for i, tag := range tags {
		normalized := normalize.Lookup(tag.Tag)
		if _, err := insertTagStmt.ExecContext(ctx, tag.Tag, normalized, tag.Category, tag.PostCount); err != nil {
			return nil, err
		}

		var tagID int64
		if err := selectIDStmt.QueryRowContext(ctx, tag.Tag).Scan(&tagID); err != nil {
			return nil, err
		}
		tagIDs[tag.Tag] = tagID

		for _, alias := range tag.Alternatives {
			if _, err := insertAliasStmt.ExecContext(ctx, tagID, alias, normalize.Lookup(alias)); err != nil {
				return nil, err
			}
		}

		if onProgress != nil && ((i+1)%250 == 0 || i == len(tags)-1) {
			onProgress(i+1, len(tags))
		}
	}

	return tagIDs, nil
}

func InsertCharacters(ctx context.Context, tx *sql.Tx, characters []csvloader.CharacterRecord, tagIDs map[string]int64, onProgress ProgressFunc) error {
	insertCharacterStmt, err := tx.PrepareContext(ctx, `
INSERT INTO characters(name, normalized_name, copyright_name, normalized_copyright_name, count, solo_count, url)
VALUES(?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
	normalized_name = excluded.normalized_name,
	copyright_name = excluded.copyright_name,
	normalized_copyright_name = excluded.normalized_copyright_name,
	count = excluded.count,
	solo_count = excluded.solo_count,
	url = excluded.url`)
	if err != nil {
		return err
	}
	defer insertCharacterStmt.Close()

	selectCharacterIDStmt, err := tx.PrepareContext(ctx, `SELECT id FROM characters WHERE name = ?`)
	if err != nil {
		return err
	}
	defer selectCharacterIDStmt.Close()

	insertTriggerStmt, err := tx.PrepareContext(ctx, `
INSERT INTO character_triggers(character_id, trigger, normalized_trigger)
VALUES(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insertTriggerStmt.Close()

	insertCoreTagStmt, err := tx.PrepareContext(ctx, `
INSERT INTO character_core_tags(character_id, tag_id, raw_tag, normalized_raw_tag, weight)
VALUES(?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insertCoreTagStmt.Close()

	for i, c := range characters {
		if _, err := insertCharacterStmt.ExecContext(
			ctx,
			c.Character,
			normalize.Lookup(c.Character),
			c.Copyright,
			normalize.Lookup(c.Copyright),
			c.Count,
			c.SoloCount,
			c.URL,
		); err != nil {
			return err
		}

		var characterID int64
		if err := selectCharacterIDStmt.QueryRowContext(ctx, c.Character).Scan(&characterID); err != nil {
			return err
		}

		for _, trigger := range c.Triggers {
			if _, err := insertTriggerStmt.ExecContext(ctx, characterID, trigger, normalize.Lookup(trigger)); err != nil {
				return err
			}
		}

		for _, rawTag := range c.CoreTags {
			var tagID any
			if id, ok := tagIDs[rawTag]; ok {
				tagID = id
			} else {
				tagID = nil
			}
			if _, err := insertCoreTagStmt.ExecContext(ctx, characterID, tagID, rawTag, normalize.Lookup(rawTag), 1.0); err != nil {
				return err
			}
		}

		if onProgress != nil && ((i+1)%100 == 0 || i == len(characters)-1) {
			onProgress(i+1, len(characters))
		}
	}
	return nil
}
