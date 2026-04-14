package sqlite

import (
	"context"
	"database/sql"
)

func CreateSchema(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA synchronous = NORMAL;`,
		`CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			normalized_name TEXT NOT NULL,
			category INTEGER NOT NULL,
			post_count INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS tag_aliases (
			id INTEGER PRIMARY KEY,
			tag_id INTEGER NOT NULL,
			alias TEXT NOT NULL,
			normalized_alias TEXT NOT NULL,
			FOREIGN KEY(tag_id) REFERENCES tags(id)
		);`,
		`CREATE TABLE IF NOT EXISTS characters (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			normalized_name TEXT NOT NULL,
			copyright_name TEXT,
			normalized_copyright_name TEXT,
			count INTEGER NOT NULL DEFAULT 0,
			solo_count INTEGER NOT NULL DEFAULT 0,
			url TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS character_triggers (
			id INTEGER PRIMARY KEY,
			character_id INTEGER NOT NULL,
			trigger TEXT NOT NULL,
			normalized_trigger TEXT NOT NULL,
			FOREIGN KEY(character_id) REFERENCES characters(id)
		);`,
		`CREATE TABLE IF NOT EXISTS character_core_tags (
			id INTEGER PRIMARY KEY,
			character_id INTEGER NOT NULL,
			tag_id INTEGER,
			raw_tag TEXT NOT NULL,
			normalized_raw_tag TEXT NOT NULL,
			weight REAL NOT NULL DEFAULT 1.0,
			FOREIGN KEY(character_id) REFERENCES characters(id),
			FOREIGN KEY(tag_id) REFERENCES tags(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tags_normalized_name ON tags(normalized_name);`,
		`CREATE INDEX IF NOT EXISTS idx_tag_aliases_normalized_alias ON tag_aliases(normalized_alias);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_alias_unique ON tag_aliases(tag_id, normalized_alias);`,
		`CREATE INDEX IF NOT EXISTS idx_characters_normalized_name ON characters(normalized_name);`,
		`CREATE INDEX IF NOT EXISTS idx_characters_normalized_copyright ON characters(normalized_copyright_name);`,
		`CREATE INDEX IF NOT EXISTS idx_character_triggers_normalized_trigger ON character_triggers(normalized_trigger);`,
		`CREATE INDEX IF NOT EXISTS idx_character_core_tags_normalized_raw_tag ON character_core_tags(normalized_raw_tag);`,
	}

	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
