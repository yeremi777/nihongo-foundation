// Package seed loads the committed JSON dataset into Postgres.
package seed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

var levelDir = regexp.MustCompile(`^n[1-5]$`)

// Load reads every level of the dataset under dir and checks that each row's
// level and lesson agree before anything reaches the database.
func Load(dir string) ([]dataset.Dataset, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var datasets []dataset.Dataset
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "source" {
			continue
		}
		if !levelDir.MatchString(e.Name()) {
			return nil, fmt.Errorf("%s: %q is not a level directory", dir, e.Name())
		}
		ds, err := loadLevel(filepath.Join(dir, e.Name()), e.Name())
		if err != nil {
			return nil, err
		}
		datasets = append(datasets, ds)
	}
	if len(datasets) == 0 {
		return nil, fmt.Errorf("%s: no level directories", dir)
	}
	return datasets, nil
}

func loadLevel(dir, level string) (dataset.Dataset, error) {
	ds := dataset.Dataset{Level: level}
	for _, f := range []struct {
		name string
		dst  any
	}{
		{"lessons.json", &ds.Lessons},
		{"kanji.json", &ds.Kanji},
		{"vocabulary.json", &ds.Vocabulary},
		{"grammar.json", &ds.Grammar},
		{"grammar_comparisons.json", &ds.GrammarComparisons},
		{"grammar_mistakes.json", &ds.GrammarMistakes},
		{"grammar_expressions.json", &ds.GrammarExpressions},
	} {
		if err := decodeFile(filepath.Join(dir, f.name), f.dst); err != nil {
			return ds, err
		}
	}
	return ds, checkLevel(dir, ds)
}

func decodeFile(path string, dst any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if dec.More() {
		return fmt.Errorf("%s: unexpected data after the array", path)
	}
	return nil
}

func checkLevel(dir string, ds dataset.Dataset) error {
	lessons := map[string]dataset.Section{}
	for i, l := range ds.Lessons {
		if l.Level != ds.Level {
			return rowError(dir, "lessons.json", i, l.ID, "has level %q", l.Level)
		}
		lessons[l.ID] = l.Section
	}
	check := func(file string, section dataset.Section, i int, id, level, lessonID string) error {
		if level != ds.Level {
			return rowError(dir, file, i, id, "has level %q", level)
		}
		if lessons[lessonID] != section {
			return rowError(dir, file, i, id, "lesson_id %s is not a %s lesson of %s", lessonID, section, ds.Level)
		}
		return nil
	}
	for i, r := range ds.Kanji {
		if err := check("kanji.json", dataset.SectionKanji, i, r.ID, r.Level, r.LessonID); err != nil {
			return err
		}
	}
	for i, r := range ds.Vocabulary {
		if err := check("vocabulary.json", dataset.SectionVocabulary, i, r.ID, r.Level, r.LessonID); err != nil {
			return err
		}
	}
	for i, r := range ds.Grammar {
		if err := check("grammar.json", dataset.SectionGrammar, i, r.ID, r.Level, r.LessonID); err != nil {
			return err
		}
	}
	for i, r := range ds.GrammarComparisons {
		if err := check("grammar_comparisons.json", dataset.SectionGrammar, i, r.ID, ds.Level, r.LessonID); err != nil {
			return err
		}
	}
	for i, r := range ds.GrammarMistakes {
		if err := check("grammar_mistakes.json", dataset.SectionGrammar, i, r.ID, ds.Level, r.LessonID); err != nil {
			return err
		}
	}
	for i, r := range ds.GrammarExpressions {
		if err := check("grammar_expressions.json", dataset.SectionGrammar, i, r.ID, ds.Level, r.LessonID); err != nil {
			return err
		}
	}
	return nil
}

func rowError(dir, file string, i int, id, format string, args ...any) error {
	return fmt.Errorf("%s: row %d (id %s): %s", filepath.Join(dir, file), i+1, id, fmt.Sprintf(format, args...))
}
