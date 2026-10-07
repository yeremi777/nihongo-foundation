//go:build integration

package lesson

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/yeremi777/nihongo-foundation/internal/database/dbtest"
	"github.com/yeremi777/nihongo-foundation/internal/dataset"
	"github.com/yeremi777/nihongo-foundation/internal/seed"
)

var (
	testConn *pgx.Conn
	// datasets is the committed dataset the test database is seeded with.
	datasets []dataset.Dataset
)

// TestMain seeds the throwaway test database dbtest opens with data/.
func TestMain(m *testing.M) {
	dbtest.Main(m, func(ctx context.Context, conn *pgx.Conn) error {
		testConn = conn
		var err error
		if datasets, err = seed.Load("../../data"); err != nil {
			return err
		}
		return seed.Seed(ctx, conn, datasets)
	})
}

// where keeps the rows keep accepts, in order, never returning nil.
func where[T any](rows []T, keep func(T) bool) []T {
	kept := []T{}
	for _, r := range rows {
		if keep(r) {
			kept = append(kept, r)
		}
	}
	return kept
}

func TestListReturnsTheLessonsOfALevelAndSectionInWeekAndDayOrder(t *testing.T) {
	repo := NewRepository(testConn)
	for _, ds := range datasets {
		for _, section := range sections {
			got, err := repo.List(context.Background(), ds.Level, section)
			if err != nil {
				t.Fatal(err)
			}
			want := where(ds.Lessons, func(l dataset.Lesson) bool { return l.Section == section })
			if len(want) == 0 {
				t.Fatalf("%s %s: the dataset has no lessons to compare with", ds.Level, section)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s %s: got %d lessons, want the %d of data/%s/lessons.json in order", ds.Level, section, len(got), len(want), ds.Level)
			}
		}
	}

	none, err := repo.List(context.Background(), "n2", dataset.SectionKanji)
	if err != nil || none == nil || len(none) != 0 {
		t.Errorf("n2 kanji: got %v, %v; want an empty list", none, err)
	}
}

func TestGetReturnsEveryLessonWithExactlyItsRows(t *testing.T) {
	repo := NewRepository(testConn)
	for _, ds := range datasets {
		for _, l := range ds.Lessons {
			want := Detail{Lesson: l}
			switch l.Section {
			case dataset.SectionKanji:
				want.Kanji = where(ds.Kanji, func(k dataset.Kanji) bool { return k.LessonID == l.ID })
			case dataset.SectionVocabulary:
				want.Vocabulary = where(ds.Vocabulary, func(v dataset.Vocabulary) bool { return v.LessonID == l.ID })
			case dataset.SectionGrammar:
				want.Grammar = where(ds.Grammar, func(g dataset.Grammar) bool { return g.LessonID == l.ID })
				want.Comparisons = where(ds.GrammarComparisons, func(c dataset.GrammarComparison) bool { return c.LessonID == l.ID })
				want.Mistakes = where(ds.GrammarMistakes, func(m dataset.GrammarMistake) bool { return m.LessonID == l.ID })
				want.Expressions = where(ds.GrammarExpressions, func(e dataset.GrammarExpression) bool { return e.LessonID == l.ID })
			}
			got, err := repo.Get(context.Background(), l.ID)
			if err != nil {
				t.Fatalf("%s: %v", l.ID, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s %s week %d day %d:\ngot  %+v\nwant %+v", ds.Level, l.Section, l.Week, l.Day, got, want)
			}
		}
	}
}

func TestGetAnswersErrNotFoundForAnIDThatNamesNoLesson(t *testing.T) {
	repo := NewRepository(testConn)
	for _, id := range []string{"nope", "", "00000000-0000-0000-0000-000000000000"} {
		if _, err := repo.Get(context.Background(), id); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%q): error = %v, want ErrNotFound", id, err)
		}
	}
}
