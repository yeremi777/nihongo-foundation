//go:build integration

package seed

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/yeremi777/nihongo-foundation/internal/database/dbtest"
	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

var testConn *pgx.Conn

// TestMain runs the tests against the throwaway test database dbtest opens.
func TestMain(m *testing.M) {
	dbtest.Main(m, func(_ context.Context, conn *pgx.Conn) error {
		testConn = conn
		return nil
	})
}

var tableNames = []string{"lesson", "kanji", "vocabulary", "grammar", "grammar_comparison", "grammar_mistake", "grammar_expression"}

func emptyTables(t *testing.T) {
	t.Helper()
	if _, err := testConn.Exec(context.Background(), "TRUNCATE "+strings.Join(tableNames, ", ")); err != nil {
		t.Fatal(err)
	}
}

func counts(t *testing.T) map[string]int {
	t.Helper()
	got := map[string]int{}
	for _, table := range tableNames {
		var n int
		if err := testConn.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		got[table] = n
	}
	return got
}

func loadData(t *testing.T) []dataset.Dataset {
	t.Helper()
	datasets, err := Load(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	return datasets
}

func seed(t *testing.T, datasets []dataset.Dataset) {
	t.Helper()
	if err := Seed(context.Background(), testConn, datasets); err != nil {
		t.Fatal(err)
	}
}

type stamp struct {
	id        string
	updatedAt time.Time
}

func stamps(t *testing.T, table string) []stamp {
	t.Helper()
	rows, err := testConn.Query(context.Background(), "SELECT id::text, updated_at FROM "+table+" ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (stamp, error) {
		var s stamp
		return s, r.Scan(&s.id, &s.updatedAt)
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSeedLoadsEveryRow(t *testing.T) {
	emptyTables(t)
	seed(t, loadData(t))

	want := map[string]int{
		"lesson": 209, "kanji": 642, "vocabulary": 1991, "grammar": 416,
		"grammar_comparison": 383, "grammar_mistake": 292, "grammar_expression": 20,
	}
	if got := counts(t); !mapsEqual(got, want) {
		t.Errorf("counts = %v, want %v", got, want)
	}

	var line string
	err := testConn.QueryRow(context.Background(),
		`SELECT lines->0->>'label' FROM grammar_mistake WHERE lines->0->>'verdict' = 'incorrect' LIMIT 1`).Scan(&line)
	if err != nil || line == "" {
		t.Errorf("mistake lines are not queryable jsonb: %q, %v", line, err)
	}
}

func TestSeedKeepsLessonReferencesWithinLevelAndSection(t *testing.T) {
	emptyTables(t)
	seed(t, loadData(t))

	for _, table := range []string{"kanji", "vocabulary", "grammar"} {
		assertZero(t, fmt.Sprintf(
			"SELECT count(*) FROM %s i JOIN lesson l ON l.id = i.lesson_id WHERE l.section <> '%s' OR l.level <> i.level", table, table))
	}
	for _, table := range []string{"grammar_comparison", "grammar_mistake", "grammar_expression"} {
		assertZero(t, fmt.Sprintf("SELECT count(*) FROM %s c JOIN lesson l ON l.id = c.lesson_id WHERE l.section <> 'grammar'", table))
	}
}

func TestSeedTwiceKeepsIDsAndTimestamps(t *testing.T) {
	emptyTables(t)
	datasets := loadData(t)
	seed(t, datasets)
	before := map[string][]stamp{}
	for _, table := range tableNames {
		before[table] = stamps(t, table)
	}

	seed(t, datasets)

	for _, table := range tableNames {
		if after := stamps(t, table); !slices.Equal(before[table], after) {
			t.Errorf("%s ids or updated_at changed on an identical re-seed", table)
		}
	}
}

func TestSeedUpdatesOnlyChangedRows(t *testing.T) {
	emptyTables(t)
	datasets := loadData(t)
	seed(t, datasets)
	before := stamps(t, "kanji")

	changed := datasets[0].Kanji[0]
	datasets[0].Kanji[0].MeaningEN = "changed meaning"
	seed(t, datasets)

	var meaning string
	if err := testConn.QueryRow(context.Background(), "SELECT meaning_en FROM kanji WHERE id = $1", changed.ID).Scan(&meaning); err != nil {
		t.Fatal(err)
	}
	if meaning != "changed meaning" {
		t.Errorf("meaning_en = %q, want the new value", meaning)
	}
	moved := 0
	for i, s := range stamps(t, "kanji") {
		if s.updatedAt != before[i].updatedAt {
			moved++
			if s.id != changed.ID {
				t.Errorf("kanji %s updated_at moved but its row did not change", s.id)
			}
		}
	}
	if moved != 1 {
		t.Errorf("%d kanji rows got a new updated_at, want 1", moved)
	}
}

func TestSeedDeletesRowsRemovedFromTheDataset(t *testing.T) {
	emptyTables(t)
	datasets := loadData(t)
	seed(t, datasets)
	full := counts(t)

	removedVocabulary := datasets[0].Vocabulary[0].ID
	datasets[0].Vocabulary = datasets[0].Vocabulary[1:]
	removedMistake := datasets[0].GrammarMistakes[0].ID
	datasets[0].GrammarMistakes = datasets[0].GrammarMistakes[1:]
	seed(t, datasets)

	got := counts(t)
	if got["vocabulary"] != full["vocabulary"]-1 || got["grammar_mistake"] != full["grammar_mistake"]-1 {
		t.Errorf("counts after removal = %v, want one fewer vocabulary and grammar_mistake row than %v", got, full)
	}
	assertZero(t, fmt.Sprintf("SELECT count(*) FROM vocabulary WHERE id = '%s'", removedVocabulary))
	assertZero(t, fmt.Sprintf("SELECT count(*) FROM grammar_mistake WHERE id = '%s'", removedMistake))
}

func TestSeedAcceptsReorderedRows(t *testing.T) {
	emptyTables(t)
	datasets := loadData(t)
	seed(t, datasets)

	v := datasets[0].Vocabulary
	if v[0].LessonID != v[1].LessonID {
		t.Fatal("first two vocabulary rows are not in one lesson")
	}
	v[0].Sequence, v[1].Sequence = v[1].Sequence, v[0].Sequence
	g := datasets[0].Grammar
	g[0].Sequence, g[1].Sequence = g[1].Sequence, g[0].Sequence

	if err := Seed(context.Background(), testConn, datasets); err != nil {
		t.Fatalf("re-seed with two rows swapped: %v", err)
	}
}

func TestSeedFailureChangesNothing(t *testing.T) {
	emptyTables(t)
	datasets := loadData(t)
	seed(t, datasets)
	before := counts(t)
	original := datasets[0].Kanji[0]

	datasets[0].Kanji[0].MeaningEN = "written before the failure"
	datasets[0].Vocabulary = datasets[0].Vocabulary[1:]
	datasets[len(datasets)-1].Grammar[0].Difficulty = "extreme"

	if err := Seed(context.Background(), testConn, datasets); err == nil {
		t.Fatal("seed with an invalid difficulty succeeded, want an error")
	}
	if got := counts(t); !mapsEqual(got, before) {
		t.Errorf("counts after a failed seed = %v, want %v", got, before)
	}
	var meaning string
	if err := testConn.QueryRow(context.Background(), "SELECT meaning_en FROM kanji WHERE id = $1", original.ID).Scan(&meaning); err != nil {
		t.Fatal(err)
	}
	if meaning != original.MeaningEN {
		t.Errorf("kanji meaning_en = %q after a failed seed, want %q", meaning, original.MeaningEN)
	}
}

func assertZero(t *testing.T, query string) {
	t.Helper()
	var n int
	if err := testConn.QueryRow(context.Background(), query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%s = %d, want 0", query, n)
	}
}

func mapsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
