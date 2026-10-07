//go:build integration

package quiz

import (
	"cmp"
	"context"
	"errors"
	"reflect"
	"slices"
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

// wantItems returns the dataset's rows of section whose lesson keep accepts,
// by code.
func wantItems(ds dataset.Dataset, section dataset.Section, keep func(lessonID string) bool) Items {
	var items Items
	switch section {
	case dataset.SectionKanji:
		items.Kanji = byCode(ds.Kanji, func(k dataset.Kanji) (string, string) { return k.Code, k.LessonID }, keep)
	case dataset.SectionVocabulary:
		items.Vocabulary = byCode(ds.Vocabulary, func(v dataset.Vocabulary) (string, string) { return v.Code, v.LessonID }, keep)
	case dataset.SectionGrammar:
		items.Grammar = byCode(ds.Grammar, func(g dataset.Grammar) (string, string) { return g.Code, g.LessonID }, keep)
	}
	return items
}

func byCode[T any](rows []T, key func(T) (code, lessonID string), keep func(string) bool) []T {
	kept := []T{}
	for _, r := range rows {
		if _, lessonID := key(r); keep(lessonID) {
			kept = append(kept, r)
		}
	}
	slices.SortFunc(kept, func(a, b T) int {
		codeA, _ := key(a)
		codeB, _ := key(b)
		return cmp.Compare(codeA, codeB)
	})
	return kept
}

func TestItemsReturnsTheSectionsRowsOfTheScope(t *testing.T) {
	repo := NewRepository(testConn)
	all := func(string) bool { return true }
	for _, ds := range datasets {
		for _, section := range []dataset.Section{dataset.SectionKanji, dataset.SectionVocabulary, dataset.SectionGrammar} {
			got, err := repo.Items(context.Background(), Scope{Level: ds.Level, Section: section})
			if err != nil {
				t.Fatal(err)
			}
			if want := wantItems(ds, section, all); !reflect.DeepEqual(got, want) {
				t.Errorf("%s %s: got %d+%d+%d rows, want %d+%d+%d", ds.Level, section,
					len(got.Kanji), len(got.Vocabulary), len(got.Grammar), len(want.Kanji), len(want.Vocabulary), len(want.Grammar))
			}

			var lessons []string
			for _, l := range ds.Lessons {
				if l.Section == section && len(lessons) < 2 {
					lessons = append(lessons, l.ID)
				}
			}
			got, err = repo.Items(context.Background(), Scope{Level: ds.Level, Section: section, LessonIDs: lessons})
			if err != nil {
				t.Fatal(err)
			}
			want := wantItems(ds, section, func(id string) bool { return slices.Contains(lessons, id) })
			if !reflect.DeepEqual(got, want) || len(want.Kanji)+len(want.Vocabulary)+len(want.Grammar) == 0 {
				t.Errorf("%s %s lessons %v: got %+v\nwant %+v", ds.Level, section, lessons, got, want)
			}
		}
	}
}

func TestItemsRejectsALessonOutsideTheLevelAndSection(t *testing.T) {
	repo := NewRepository(testConn)
	n5 := datasets[slices.IndexFunc(datasets, func(ds dataset.Dataset) bool { return ds.Level == "n5" })]
	first := func(section dataset.Section) string {
		return n5.Lessons[slices.IndexFunc(n5.Lessons, func(l dataset.Lesson) bool { return l.Section == section })].ID
	}
	kanjiLesson, grammarLesson := first(dataset.SectionKanji), first(dataset.SectionGrammar)

	for _, tt := range []struct {
		name  string
		scope Scope
	}{
		{"another section", Scope{Level: "n5", Section: dataset.SectionKanji, LessonIDs: []string{kanjiLesson, grammarLesson}}},
		{"another level", Scope{Level: "n4", Section: dataset.SectionKanji, LessonIDs: []string{kanjiLesson}}},
		{"not a UUID", Scope{Level: "n5", Section: dataset.SectionKanji, LessonIDs: []string{"nope"}}},
		{"no such lesson", Scope{Level: "n5", Section: dataset.SectionKanji, LessonIDs: []string{"00000000-0000-0000-0000-000000000000"}}},
	} {
		if _, err := repo.Items(context.Background(), tt.scope); !errors.Is(err, ErrInvalidLesson) {
			t.Errorf("%s: error = %v, want ErrInvalidLesson", tt.name, err)
		}
	}
	if _, err := repo.Items(context.Background(), Scope{Level: "n5", Section: dataset.SectionKanji, LessonIDs: []string{kanjiLesson, kanjiLesson}}); err != nil {
		t.Errorf("a repeated lesson: error = %v, want none", err)
	}
}
