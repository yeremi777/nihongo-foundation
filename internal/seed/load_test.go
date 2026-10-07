package seed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var dataDir = filepath.Join("..", "..", "data")

func TestLoadReadsTheCommittedDataset(t *testing.T) {
	datasets, err := Load(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	totals := map[string]int{}
	for _, ds := range datasets {
		totals["lesson"] += len(ds.Lessons)
		totals["kanji"] += len(ds.Kanji)
		totals["vocabulary"] += len(ds.Vocabulary)
		totals["grammar"] += len(ds.Grammar)
		totals["grammar_comparison"] += len(ds.GrammarComparisons)
		totals["grammar_mistake"] += len(ds.GrammarMistakes)
		totals["grammar_expression"] += len(ds.GrammarExpressions)
	}
	want := map[string]int{
		"lesson": 209, "kanji": 642, "vocabulary": 1991, "grammar": 416,
		"grammar_comparison": 383, "grammar_mistake": 292, "grammar_expression": 20,
	}
	for table, n := range want {
		if totals[table] != n {
			t.Errorf("%s = %d rows, want %d", table, totals[table], n)
		}
	}
}

func TestLoadRejectsBrokenFiles(t *testing.T) {
	for _, tt := range []struct {
		name, file, old, new, want string
	}{
		{"item in a lesson of another section", "kanji.json",
			`"lesson_id": "` + firstLessonID(t, "kanji"), `"lesson_id": "` + firstLessonID(t, "grammar"), "is not a kanji lesson of n5"},
		{"item pointing at a missing lesson", "vocabulary.json",
			`"lesson_id": "`, `"lesson_id": "0`, "is not a vocabulary lesson of n5"},
		{"item of another level", "grammar.json",
			`"level": "n5"`, `"level": "n4"`, `has level "n4"`},
		{"unknown field", "grammar_mistakes.json",
			`"point": `, `"topic": "x", "point": `, "unknown field"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := copyLevel(t, "n5")
			path := filepath.Join(dir, "n5", tt.file)
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(b), tt.old) {
				t.Fatalf("%s lacks %q", tt.file, tt.old)
			}
			if err := os.WriteFile(path, []byte(strings.Replace(string(b), tt.old, tt.new, 1)), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err = Load(dir)
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), tt.file) {
				t.Errorf("error = %v, want it to name %s and say %q", err, tt.file, tt.want)
			}
		})
	}
}

func TestLoadRejectsAMissingFile(t *testing.T) {
	dir := copyLevel(t, "n5")
	if err := os.Remove(filepath.Join(dir, "n5", "grammar_expressions.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "grammar_expressions.json") {
		t.Errorf("error = %v, want it to name grammar_expressions.json", err)
	}
}

// firstLessonID is the id of the first n5 lesson of the section.
func firstLessonID(t *testing.T, section string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dataDir, "n5", "lessons.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lessons []struct{ ID, Section string }
	if err := json.Unmarshal(b, &lessons); err != nil {
		t.Fatal(err)
	}
	for _, l := range lessons {
		if l.Section == section {
			return l.ID
		}
	}
	t.Fatalf("no n5 %s lesson", section)
	return ""
}

// copyLevel copies one committed level into a temporary data directory.
func copyLevel(t *testing.T, level string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, level), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(dataDir, level))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dataDir, level, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, level, e.Name()), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
