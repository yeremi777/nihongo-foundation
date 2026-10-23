//go:build integration

package item

import (
	"cmp"
	"context"
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
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
	// lessons holds every lesson of datasets by id.
	lessons = map[string]dataset.Lesson{}
)

// TestMain seeds the throwaway test database dbtest opens with data/.
func TestMain(m *testing.M) {
	dbtest.Main(m, func(ctx context.Context, conn *pgx.Conn) error {
		testConn = conn
		var err error
		if datasets, err = seed.Load("../../data"); err != nil {
			return err
		}
		for _, ds := range datasets {
			for _, l := range ds.Lessons {
				lessons[l.ID] = l
			}
		}
		return seed.Seed(ctx, conn, datasets)
	})
}

// facts is what a search reads of one row: its filters, its order, and the
// texts its q matches.
type facts struct {
	level, lessonID string
	sequence        int
	sources         []dataset.Source
	difficulty      dataset.Difficulty
	searched        []string
}

func kanjiFacts(k dataset.Kanji) facts {
	return facts{k.Level, k.LessonID, k.Sequence, k.Sources, "",
		slices.Concat([]string{k.Character, k.MeaningEN, k.MeaningID}, k.Onyomi, k.Kunyomi, k.Examples)}
}

func vocabularyFacts(v dataset.Vocabulary) facts {
	return facts{v.Level, v.LessonID, v.Sequence, v.Sources, "", []string{v.Word, v.Reading, v.MeaningEN, v.MeaningID}}
}

func grammarFacts(g dataset.Grammar) facts {
	searched := []string{g.Pattern, g.MeaningEN, g.MeaningID}
	if g.Reading != nil {
		searched = append(searched, *g.Reading)
	}
	return facts{g.Level, g.LessonID, g.Sequence, g.Sources, g.Difficulty, searched}
}

// foldText lowercases ASCII letters and turns katakana into hiragana.
func foldText(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z':
			return r + 'a' - 'A'
		case r >= 'ァ' && r <= 'ヶ':
			return r - 'ァ' + 'ぁ'
		}
		return r
	}, s)
}

// want returns the rows of all that q matches, in search order.
func want[T any](all []T, factsOf func(T) facts, q Query) []T {
	needle := foldText(q.Text)
	exact := func(f facts) bool {
		return needle != "" && slices.ContainsFunc(f.searched, func(s string) bool { return foldText(s) == needle })
	}
	matched := []T{}
	for _, row := range all {
		f := factsOf(row)
		if (q.Level == "" || f.level == q.Level) &&
			(q.Week == 0 || lessons[f.lessonID].Week == q.Week) &&
			(q.Source == "" || slices.Contains(f.sources, q.Source)) &&
			(q.Difficulty == "" || f.difficulty == q.Difficulty) &&
			(needle == "" || slices.ContainsFunc(f.searched, func(s string) bool { return strings.Contains(foldText(s), needle) })) {
			matched = append(matched, row)
		}
	}
	slices.SortStableFunc(matched, func(a, b T) int {
		fa, fb := factsOf(a), factsOf(b)
		la, lb := lessons[fa.lessonID], lessons[fb.lessonID]
		ea, eb := exact(fa), exact(fb)
		return cmp.Or(
			-cmp.Compare(boolInt(ea), boolInt(eb)),
			cmp.Compare(slices.Index(levels, fa.level), slices.Index(levels, fb.level)),
			cmp.Compare(la.Week, lb.Week),
			cmp.Compare(la.Day, lb.Day),
			cmp.Compare(fa.sequence, fb.sequence),
		)
	})
	return matched
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func search[T any](t *testing.T, r repository[T], q Query) Page[T] {
	t.Helper()
	got, err := r.Search(context.Background(), q)
	if err != nil {
		t.Fatalf("%+v: %v", q, err)
	}
	return got
}

// assertSearches compares every query's first two pages and a page past the
// last with the rows of all it should return.
func assertSearches[T any](t *testing.T, r repository[T], all []T, factsOf func(T) facts, queries []Query) {
	t.Helper()
	for _, q := range queries {
		rows := want(all, factsOf, q)
		for _, page := range []int{1, 2, math.MaxInt} {
			q.Page, q.Size = page, 7
			got := search(t, r, q)
			start := len(rows)
			if page != math.MaxInt {
				start = min(len(rows), (page-1)*7)
			}
			wantPage := Page[T]{Items: rows[start:min(len(rows), start+7)], Page: page, Size: 7, Total: len(rows)}
			if !reflect.DeepEqual(got, wantPage) {
				t.Errorf("%s %+v: the page differs from data/: got %d items of total %d, want %d of %d", r.section.name, q, len(got.Items), got.Total, len(wantPage.Items), len(rows))
			}
		}
	}
}

func every[T any](rows func(dataset.Dataset) []T) []T {
	var all []T
	for _, ds := range datasets {
		all = append(all, rows(ds)...)
	}
	return all
}

// queries are the filters every section is searched with.
var queries = []Query{
	{},
	{Level: "n5"},
	{Level: "n4", Week: 2},
	{Level: "n3", Week: 1, Source: dataset.SourceShinkanzen},
	{Source: dataset.SourceMinnaNoNihongo},
	{Level: "n4", Source: dataset.SourceSoumatome},
	{Level: "n5", Week: 3000000000},
	{Level: "n2"},
	{Text: "せん"},
	{Text: "セン"},
	{Text: "先生"},
	{Text: "WATER"},
	{Level: "n5", Text: "日"},
	{Text: "%"},
	{Text: "_"},
	{Level: "n4", Text: "Ka"},
}

func TestSearchReturnsTheMatchingRowsOfEachSectionInSearchOrder(t *testing.T) {
	assertSearches(t, repository[dataset.Kanji]{testConn, kanji}, every(func(ds dataset.Dataset) []dataset.Kanji { return ds.Kanji }), kanjiFacts, queries)
	assertSearches(t, repository[dataset.Vocabulary]{testConn, vocabulary}, every(func(ds dataset.Dataset) []dataset.Vocabulary { return ds.Vocabulary }), vocabularyFacts, queries)
	assertSearches(t, repository[dataset.Grammar]{testConn, grammar}, every(func(ds dataset.Dataset) []dataset.Grammar { return ds.Grammar }), grammarFacts,
		append(slices.Clone(queries), Query{Difficulty: dataset.DifficultyHard}, Query{Level: "n5", Difficulty: dataset.DifficultyEasy, Text: "I-KEIYOUSHI"}))
}

func codes[T any](items []T, code func(T) string) []string {
	got := []string{}
	for _, item := range items {
		got = append(got, code(item))
	}
	return got
}

func TestSearchMatchesFoldedTextAndPutsExactMatchesFirst(t *testing.T) {
	kanjiRepo := repository[dataset.Kanji]{testConn, kanji}
	char := func(k dataset.Kanji) string { return k.Character }
	for _, text := range []string{"せん", "先生"} {
		if got := codes(search(t, kanjiRepo, Query{Level: "n5", Text: text, Page: 1, Size: 100}).Items, char); !slices.Contains(got, "先") {
			t.Errorf("n5 kanji %q: got %v, want 先 among them", text, got)
		}
	}

	hi := search(t, kanjiRepo, Query{Level: "n5", Text: "日", Page: 1, Size: 100}).Items
	if len(hi) < 2 || hi[0].Character != "日" {
		t.Fatalf("n5 kanji 日: got %v, want 日 first of several", codes(hi, char))
	}
	earlier := func(k dataset.Kanji) bool {
		lk, lh := lessons[k.LessonID], lessons[hi[0].LessonID]
		return cmp.Or(cmp.Compare(lk.Week, lh.Week), cmp.Compare(lk.Day, lh.Day), cmp.Compare(k.Sequence, hi[0].Sequence)) < 0
	}
	if !slices.ContainsFunc(hi[1:], earlier) {
		t.Error("日 is first in curriculum order anyway, so the exact-first check proves nothing")
	}

	grammarRepo := repository[dataset.Grammar]{testConn, grammar}
	got := search(t, grammarRepo, Query{Level: "n5", Text: "I-KEIYOUSHI NO KATACHI", Page: 1, Size: 100}).Items
	if len(got) == 0 || got[0].Code != "g101" {
		t.Errorf("n5 grammar by uppercase reading: got %v, want g101 first", codes(got, func(g dataset.Grammar) string { return g.Code }))
	}

	vocabularyRepo := repository[dataset.Vocabulary]{testConn, vocabulary}
	for _, v := range search(t, vocabularyRepo, Query{Text: "%", Page: 1, Size: 100}).Items {
		if !slices.ContainsFunc(vocabularyFacts(v).searched, func(s string) bool { return strings.Contains(s, "%") }) {
			t.Errorf("%%: got %s %s, which has no %% in a searched field", v.Level, v.Code)
		}
	}
}

func TestGetReturnsTheRowOfTheLevelAndCode(t *testing.T) {
	assertGets(t, repository[dataset.Kanji]{testConn, kanji}, every(func(ds dataset.Dataset) []dataset.Kanji { return ds.Kanji }),
		func(k dataset.Kanji) (string, string) { return k.Level, k.Code })
	assertGets(t, repository[dataset.Vocabulary]{testConn, vocabulary}, every(func(ds dataset.Dataset) []dataset.Vocabulary { return ds.Vocabulary }),
		func(v dataset.Vocabulary) (string, string) { return v.Level, v.Code })
	assertGets(t, repository[dataset.Grammar]{testConn, grammar}, every(func(ds dataset.Dataset) []dataset.Grammar { return ds.Grammar }),
		func(g dataset.Grammar) (string, string) { return g.Level, g.Code })
}

// assertGets looks up every row by its level and code, then a code no level
// has, and a code of another level only.
func assertGets[T any](t *testing.T, r repository[T], all []T, key func(T) (string, string)) {
	t.Helper()
	byLevel := map[string][]string{}
	for _, row := range all {
		level, code := key(row)
		byLevel[level] = append(byLevel[level], code)
		got, err := r.Get(context.Background(), level, code)
		if err != nil || !reflect.DeepEqual(got, row) {
			t.Errorf("%s %s %s: got %+v, %v", r.section.name, level, code, got, err)
		}
	}
	if _, err := r.Get(context.Background(), "n5", "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("%s n5 nope: error = %v, want ErrNotFound", r.section.name, err)
	}
	i := slices.IndexFunc(byLevel["n3"], func(code string) bool { return !slices.Contains(byLevel["n5"], code) })
	if i < 0 {
		t.Fatalf("%s: every n3 code is also an n5 code", r.section.name)
	}
	if _, err := r.Get(context.Background(), "n5", byLevel["n3"][i]); !errors.Is(err, ErrNotFound) {
		t.Errorf("%s n5 %s (an n3 code only): error = %v, want ErrNotFound", r.section.name, byLevel["n3"][i], err)
	}
}

func TestSearchRanksAnItemWithANullFieldByItsOtherFields(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() {
		if _, err := testConn.Exec(ctx, "DELETE FROM grammar WHERE code = 'g999-z'"); err != nil {
			t.Error(err)
		}
	})
	_, err := testConn.Exec(ctx, "INSERT INTO grammar ("+dataset.GrammarColumns+")"+
		" SELECT gen_random_uuid(), level, 'g999-z', lesson_id, 999, curriculum_code, pattern || '・', NULL, formula, note, example, difficulty, meaning_en, meaning_id, sources"+
		" FROM grammar WHERE level = 'n5' AND code = 'g101'")
	if err != nil {
		t.Fatal(err)
	}

	got := search(t, repository[dataset.Grammar]{testConn, grammar}, Query{Level: "n5", Text: "い形容詞の形", Page: 1, Size: 100}).Items
	order := codes(got, func(g dataset.Grammar) string { return g.Code })
	if exact, contains := slices.Index(order, "g101"), slices.Index(order, "g999-z"); exact < 0 || contains < exact {
		t.Errorf("got %v, want g101, an exact match, before g999-z, whose reading is NULL", order)
	}
}

func BenchmarkSearch(b *testing.B) {
	r := repository[dataset.Vocabulary]{testConn, vocabulary}
	for _, q := range []Query{
		{Level: "n3", Page: 1, Size: 20},
		{Level: "n3", Text: "か", Page: 1, Size: 20},
		{Text: "か", Page: 1, Size: 20},
	} {
		b.Run(q.Level+"/"+q.Text, func(b *testing.B) {
			for b.Loop() {
				if _, err := r.Search(context.Background(), q); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
