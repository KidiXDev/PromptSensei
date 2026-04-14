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

type CharacterRecord struct {
	Character string
	Copyright string
	Triggers  []string
	CoreTags  []string
	Count     int
	SoloCount int
	URL       string
}

func LoadCharacters(path string) ([]CharacterRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	var out []CharacterRecord
	line := 0
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("danbooru_character.csv read line %d: %w", line+1, err)
		}
		line++
		if len(rec) == 0 {
			continue
		}
		if line == 1 && strings.EqualFold(strings.TrimSpace(rec[0]), "character") {
			continue
		}
		if len(rec) < 7 {
			return nil, fmt.Errorf("danbooru_character.csv line %d: expected 7 columns", line)
		}

		charName := normalize.Tag(rec[0])
		if charName == "" {
			continue
		}

		count, err := strconv.Atoi(strings.TrimSpace(rec[4]))
		if err != nil {
			return nil, fmt.Errorf("danbooru_character.csv line %d: invalid count: %w", line, err)
		}
		soloCount, err := strconv.Atoi(strings.TrimSpace(rec[5]))
		if err != nil {
			return nil, fmt.Errorf("danbooru_character.csv line %d: invalid solo_count: %w", line, err)
		}

		copyright := normalize.Tag(rec[1])
		triggers := utils.SplitList(rec[2])
		core := utils.SplitList(rec[3])

		out = append(out, CharacterRecord{
			Character: charName,
			Copyright: copyright,
			Triggers:  triggers,
			CoreTags:  core,
			Count:     count,
			SoloCount: soloCount,
			URL:       strings.TrimSpace(rec[6]),
		})
	}

	return out, nil
}
