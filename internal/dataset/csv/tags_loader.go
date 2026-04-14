package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/kidixdev/PromptSensei/internal/dataset/normalize"
	"github.com/kidixdev/PromptSensei/internal/utils"
)

type TagRecord struct {
	Tag          string
	Category     int
	PostCount    int
	Alternatives []string
}

func LoadTags(path string) ([]TagRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	var out []TagRecord
	line := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tag.csv read line %d: %w", line+1, err)
		}
		line++
		if len(rec) == 0 {
			continue
		}
		if line == 1 && strings.EqualFold(strings.TrimSpace(rec[0]), "tag") {
			continue
		}
		if len(rec) < 3 {
			return nil, fmt.Errorf("tag.csv line %d: expected at least 3 columns", line)
		}

		category, err := strconv.Atoi(strings.TrimSpace(rec[1]))
		if err != nil {
			return nil, fmt.Errorf("tag.csv line %d: invalid category: %w", line, err)
		}
		postCount, err := strconv.Atoi(strings.TrimSpace(rec[2]))
		if err != nil {
			return nil, fmt.Errorf("tag.csv line %d: invalid post_count: %w", line, err)
		}

		tag := normalize.Tag(rec[0])
		if tag == "" {
			continue
		}

		var alternatives []string
		if len(rec) > 3 {
			alternatives = utils.SplitList(rec[3])
		}

		out = append(out, TagRecord{
			Tag:          tag,
			Category:     category,
			PostCount:    postCount,
			Alternatives: alternatives,
		})
	}

	return out, nil
}
