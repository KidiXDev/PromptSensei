package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/kidixdev/PromptSensei/internal/domain"
	"github.com/kidixdev/PromptSensei/internal/utils"
)

type Repository struct {
	db *sql.DB
}

type DatasetCounts struct {
	Tags              int64
	TagAliases        int64
	Characters        int64
	CharacterTriggers int64
	CharacterCoreTags int64
}

type TagMatch struct {
	Tag         domain.Tag
	MatchType   string
	MatchedTerm string
}

type CharacterMatch struct {
	Character   domain.Character
	MatchType   string
	MatchedTerm string
}

type CharacterCoreTag struct {
	TagName    string
	Category   int
	PostCount  int
	Normalized string
}

func Open(path string) (*Repository, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) DB() *sql.DB {
	return r.db
}

func (r *Repository) Counts(ctx context.Context) (DatasetCounts, error) {
	return countRows(ctx, r.db)
}

func (r *Repository) FindTagsByTerms(ctx context.Context, terms []string, limit int) ([]TagMatch, error) {
	const q = `
SELECT t.id, t.name, t.normalized_name, t.category, t.post_count, 'exact' AS match_type
FROM tags t
WHERE t.normalized_name = ?
UNION ALL
SELECT t.id, t.name, t.normalized_name, t.category, t.post_count, 'alias' AS match_type
FROM tags t
JOIN tag_aliases a ON a.tag_id = t.id
WHERE a.normalized_alias = ?
ORDER BY post_count DESC
LIMIT ?`

	seen := map[int64]TagMatch{}
	for _, term := range terms {
		term = utils.NormalizeForLookup(term)
		if term == "" {
			continue
		}
		rows, err := r.db.QueryContext(ctx, q, term, term, limit)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var m TagMatch
			if err := rows.Scan(&m.Tag.ID, &m.Tag.Name, &m.Tag.NormalizedName, &m.Tag.Category, &m.Tag.PostCount, &m.MatchType); err != nil {
				_ = rows.Close()
				return nil, err
			}
			m.MatchedTerm = term
			if existing, ok := seen[m.Tag.ID]; !ok || existing.Tag.PostCount < m.Tag.PostCount {
				seen[m.Tag.ID] = m
			}
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}

	out := make([]TagMatch, 0, len(seen))
	for _, m := range seen {
		out = append(out, m)
	}
	return out, nil
}

func (r *Repository) SearchTagsByPrefix(ctx context.Context, terms []string, limit int) ([]domain.Tag, error) {
	const q = `
SELECT id, name, normalized_name, category, post_count
FROM tags
WHERE normalized_name LIKE ?
ORDER BY post_count DESC
LIMIT ?`

	seen := map[int64]domain.Tag{}
	for _, term := range terms {
		term = strings.TrimSpace(utils.NormalizeForLookup(term))
		if len(term) < 3 {
			continue
		}
		pattern := term + "%"
		rows, err := r.db.QueryContext(ctx, q, pattern, limit)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var tag domain.Tag
			if err := rows.Scan(&tag.ID, &tag.Name, &tag.NormalizedName, &tag.Category, &tag.PostCount); err != nil {
				_ = rows.Close()
				return nil, err
			}
			seen[tag.ID] = tag
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}

	out := make([]domain.Tag, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	return out, nil
}

func (r *Repository) FindCharactersByTerms(ctx context.Context, terms []string, limit int) ([]CharacterMatch, error) {
	const q = `
SELECT c.id, c.name, c.normalized_name, c.copyright_name, c.normalized_copyright_name, c.count, c.solo_count, c.url, match_type
FROM (
	SELECT id, 'name' AS match_type FROM characters WHERE normalized_name = ?
	UNION ALL
	SELECT character_id AS id, 'trigger' AS match_type FROM character_triggers WHERE normalized_trigger = ?
	UNION ALL
	SELECT id, 'copyright' AS match_type FROM characters WHERE normalized_copyright_name = ?
) matched
JOIN characters c ON c.id = matched.id
ORDER BY c.count DESC
LIMIT ?`

	seen := map[int64]CharacterMatch{}
	for _, term := range terms {
		term = utils.NormalizeForLookup(term)
		if term == "" {
			continue
		}
		rows, err := r.db.QueryContext(ctx, q, term, term, term, limit)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var m CharacterMatch
			if err := rows.Scan(
				&m.Character.ID,
				&m.Character.Name,
				&m.Character.NormalizedName,
				&m.Character.CopyrightName,
				&m.Character.NormalizedCopyrightName,
				&m.Character.Count,
				&m.Character.SoloCount,
				&m.Character.URL,
				&m.MatchType,
			); err != nil {
				_ = rows.Close()
				return nil, err
			}
			m.MatchedTerm = term
			if existing, ok := seen[m.Character.ID]; !ok || existing.Character.Count < m.Character.Count {
				seen[m.Character.ID] = m
			}
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}

	out := make([]CharacterMatch, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	return out, nil
}

func (r *Repository) CoreTagsForCharacter(ctx context.Context, characterID int64, limit int) ([]CharacterCoreTag, error) {
	const q = `
SELECT cct.raw_tag, cct.normalized_raw_tag, COALESCE(t.category, 0), COALESCE(t.post_count, 0)
FROM character_core_tags cct
LEFT JOIN tags t ON t.id = cct.tag_id
WHERE cct.character_id = ?
ORDER BY t.post_count DESC, cct.raw_tag ASC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, q, characterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CharacterCoreTag
	for rows.Next() {
		var c CharacterCoreTag
		if err := rows.Scan(&c.TagName, &c.Normalized, &c.Category, &c.PostCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func EnsureOpenable(path string) error {
	repo, err := Open(path)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	return repo.Close()
}
