package item

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// fakeStore answers with its items and keeps the query it was asked for.
type fakeStore[T any] struct {
	items []T
	err   error
	got   *Query
}

func (f fakeStore[T]) Search(_ context.Context, q Query) (Page[T], error) {
	if f.got != nil {
		*f.got = q
	}
	return Page[T]{Items: f.items, Page: q.Page, Size: q.Size, Total: len(f.items)}, f.err
}

func (f fakeStore[T]) Get(_ context.Context, level, code string) (T, error) {
	var zero T
	if f.err != nil {
		return zero, f.err
	}
	if level == "n5" && code == "k101-001" && len(f.items) > 0 {
		return f.items[0], nil
	}
	return zero, ErrNotFound
}

var saki = dataset.Kanji{ID: "k1", Level: "n5", Code: "k101-001", LessonID: "l1", Sequence: 1, Character: "先",
	Onyomi: []string{"セン"}, Kunyomi: []string{"さき"}, Examples: []string{"先生"},
	MeaningEN: "ahead", MeaningID: "depan", Sources: []dataset.Source{dataset.SourceSoumatome}}

const sakiJSON = `{"id":"k1","level":"n5","code":"k101-001","lesson_id":"l1","sequence":1,"character":"先",` +
	`"onyomi":["セン"],"kunyomi":["さき"],"examples":["先生"],"meaning_en":"ahead","meaning_id":"depan","sources":["soumatome"]}`

// serve registers the kanji and grammar routes on fake stores and answers target.
func serve(t *testing.T, kanjiStore fakeStore[dataset.Kanji], grammarStore fakeStore[dataset.Grammar], target string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	route(mux, kanji, kanjiStore)
	route(mux, grammar, grammarStore)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func assertBody(t *testing.T, rec *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if rec.Code != status || rec.Body.String() != body+"\n" {
		t.Errorf("got %d %s\nwant %d %s", rec.Code, rec.Body.String(), status, body)
	}
}

func errorJSON(code, message string) string {
	return `{"error":{"code":"` + code + `","message":"` + message + `"}}`
}

func TestSearchAnswersThePageOfTheStore(t *testing.T) {
	rec := serve(t, fakeStore[dataset.Kanji]{items: []dataset.Kanji{saki}}, fakeStore[dataset.Grammar]{}, "/api/kanji?q=%E3%81%9B%E3%82%93")
	assertBody(t, rec, http.StatusOK, `{"items":[`+sakiJSON+`],"page":1,"size":20,"total":1}`)
}

func TestSearchPassesTheFiltersAndDefaultsPageAndSize(t *testing.T) {
	var got Query
	serve(t, fakeStore[dataset.Kanji]{}, fakeStore[dataset.Grammar]{got: &got},
		"/api/grammar?level=n5&week=2&source=minna-no-nihongo&difficulty=hard&q=+%E3%81%A6+&ignored=x")
	want := Query{Level: "n5", Week: 2, Source: dataset.SourceMinnaNoNihongo, Difficulty: dataset.DifficultyHard, Text: "て", Page: 1, Size: 20}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	serve(t, fakeStore[dataset.Kanji]{got: &got}, fakeStore[dataset.Grammar]{}, "/api/kanji?difficulty=hard&page=3&size=100")
	if want := (Query{Page: 3, Size: 100}); got != want {
		t.Errorf("kanji ignores difficulty: got %+v, want %+v", got, want)
	}
}

func TestSearchRejectsAnInvalidParameterInTableOrder(t *testing.T) {
	long := ""
	for range 101 {
		long += "あ"
	}
	cases := []struct{ target, code, message string }{
		{"/api/grammar?level=n9&week=0&source=x&difficulty=x&page=0&size=0", "invalid_level", "Level must be n5, n4, n3, n2, or n1."},
		{"/api/grammar?week=1", "invalid_week", "Week must be a positive integer, given with level."},
		{"/api/grammar?level=n5&week=0", "invalid_week", "Week must be a positive integer, given with level."},
		{"/api/grammar?level=n5&week=one&source=x", "invalid_week", "Week must be a positive integer, given with level."},
		{"/api/grammar?source=x&difficulty=x", "invalid_source", "Source must be soumatome, shinkanzen, or minna-no-nihongo."},
		{"/api/grammar?difficulty=x&q=" + long, "invalid_difficulty", "Difficulty must be easy, medium, or hard."},
		{"/api/grammar?q=" + long + "&page=0", "invalid_query", "Query must be valid text of at most 100 characters."},
		{"/api/grammar?q=a%00b", "invalid_query", "Query must be valid text of at most 100 characters."},
		{"/api/grammar?q=%FF", "invalid_query", "Query must be valid text of at most 100 characters."},
		{"/api/grammar?page=0&size=0", "invalid_page", "Page must be a positive integer."},
		{"/api/grammar?page=x", "invalid_page", "Page must be a positive integer."},
		{"/api/grammar?size=0", "invalid_size", "Size must be 1 to 100."},
		{"/api/grammar?size=101", "invalid_size", "Size must be 1 to 100."},
		{"/api/kanji?size=x", "invalid_size", "Size must be 1 to 100."},
	}
	for _, c := range cases {
		t.Run(c.target, func(t *testing.T) {
			assertBody(t, serve(t, fakeStore[dataset.Kanji]{}, fakeStore[dataset.Grammar]{}, c.target), http.StatusBadRequest, errorJSON(c.code, c.message))
		})
	}
}

func TestSearchAcceptsAQueryOfExactly100Characters(t *testing.T) {
	q := ""
	for range 100 {
		q += "あ"
	}
	if rec := serve(t, fakeStore[dataset.Kanji]{}, fakeStore[dataset.Grammar]{}, "/api/kanji?q="+q); rec.Code != http.StatusOK {
		t.Errorf("got %d %s, want 200", rec.Code, rec.Body.String())
	}
}

func TestGetAnswersTheItemOfTheLevelAndCode(t *testing.T) {
	store := fakeStore[dataset.Kanji]{items: []dataset.Kanji{saki}}
	assertBody(t, serve(t, store, fakeStore[dataset.Grammar]{}, "/api/kanji/n5/k101-001"), http.StatusOK, sakiJSON)
	assertBody(t, serve(t, store, fakeStore[dataset.Grammar]{}, "/api/kanji/n5/nope"), http.StatusNotFound, errorJSON("item_not_found", "Item was not found."))
	assertBody(t, serve(t, store, fakeStore[dataset.Grammar]{}, "/api/kanji/n9/k101-001"), http.StatusBadRequest, errorJSON("invalid_level", "Level must be n5, n4, n3, n2, or n1."))
}

func TestAStoreErrorAnswers500WithoutItsCause(t *testing.T) {
	broken := fakeStore[dataset.Kanji]{err: errors.New("password authentication failed")}
	want := errorJSON("internal_error", "Internal server error.")
	assertBody(t, serve(t, broken, fakeStore[dataset.Grammar]{}, "/api/kanji"), http.StatusInternalServerError, want)
	assertBody(t, serve(t, broken, fakeStore[dataset.Grammar]{}, "/api/kanji/n5/k101-001"), http.StatusInternalServerError, want)
}

func TestGetAnswersNotFoundForACodeThatIsNotText(t *testing.T) {
	broken := fakeStore[dataset.Kanji]{err: errors.New("invalid byte sequence for encoding \"UTF8\"")}
	for _, code := range []string{"%FF", "k101%00"} {
		assertBody(t, serve(t, broken, fakeStore[dataset.Grammar]{}, "/api/kanji/n5/"+code), http.StatusNotFound, errorJSON("item_not_found", "Item was not found."))
	}
}
