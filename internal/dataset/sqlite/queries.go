package sqlite

import (
	"context"
	"database/sql"
)

func countRows(ctx context.Context, db *sql.DB) (DatasetCounts, error) {
	queries := map[string]*int64{}
	var counts DatasetCounts
	queries["SELECT COUNT(1) FROM tags"] = &counts.Tags
	queries["SELECT COUNT(1) FROM tag_aliases"] = &counts.TagAliases
	queries["SELECT COUNT(1) FROM characters"] = &counts.Characters
	queries["SELECT COUNT(1) FROM character_triggers"] = &counts.CharacterTriggers
	queries["SELECT COUNT(1) FROM character_core_tags"] = &counts.CharacterCoreTags

	for q, target := range queries {
		if err := db.QueryRowContext(ctx, q).Scan(target); err != nil {
			return DatasetCounts{}, err
		}
	}
	return counts, nil
}
