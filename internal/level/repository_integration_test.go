//go:build integration

package level

import (
	"context"
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

// countsOf counts a section of one level from its dataset, not from the database.
func countsOf(ds dataset.Dataset, section dataset.Section) Section {
	weeks := map[int]bool{}
	got := Section{Section: section}
	for _, l := range ds.Lessons {
		if l.Section == section {
			weeks[l.Week] = true
			got.Lessons++
		}
	}
	got.Weeks = len(weeks)
	switch section {
	case dataset.SectionKanji:
		got.Items = len(ds.Kanji)
	case dataset.SectionVocabulary:
		got.Items = len(ds.Vocabulary)
	case dataset.SectionGrammar:
		got.Items = len(ds.Grammar)
	}
	return got
}

func TestListCountsEveryLevelAndSectionInListOrder(t *testing.T) {
	got, err := NewRepository(testConn).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var want []Level
	for _, level := range dataset.Levels() {
		for _, ds := range datasets {
			if ds.Level != level {
				continue
			}
			l := Level{Level: level}
			for _, section := range sections {
				l.Sections = append(l.Sections, countsOf(ds, section))
			}
			want = append(want, l)
		}
	}
	if len(want) == 0 {
		t.Fatal("the dataset has no level to compare with")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}
