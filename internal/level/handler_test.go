package level

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

type fakeStore struct {
	levels []Level
	err    error
}

func (f fakeStore) List(context.Context) ([]Level, error) { return f.levels, f.err }

func get(t *testing.T, store fakeStore) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/levels", nil))
	return rec
}

func assertBody(t *testing.T, rec *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if rec.Code != status || rec.Body.String() != body+"\n" {
		t.Errorf("got %d %s\nwant %d %s", rec.Code, rec.Body.String(), status, body)
	}
}

func TestListAnswersEveryLevelWithItsSectionCounts(t *testing.T) {
	store := fakeStore{levels: []Level{
		{Level: "n5", Sections: []Section{
			{Section: dataset.SectionKanji, Weeks: 2, Lessons: 10, Items: 108},
			{Section: dataset.SectionGrammar, Weeks: 3, Lessons: 14, Items: 77},
		}},
		{Level: "n4", Sections: []Section{{Section: dataset.SectionVocabulary, Weeks: 1, Lessons: 5, Items: 40}}},
	}}
	assertBody(t, get(t, store), http.StatusOK, `[`+
		`{"level":"n5","sections":[{"section":"kanji","weeks":2,"lessons":10,"items":108},{"section":"grammar","weeks":3,"lessons":14,"items":77}]},`+
		`{"level":"n4","sections":[{"section":"vocabulary","weeks":1,"lessons":5,"items":40}]}]`)
}

func TestListAnswersAnEmptyArrayWhenNoLessonExists(t *testing.T) {
	assertBody(t, get(t, fakeStore{}), http.StatusOK, `[]`)
}

func TestListAnswersInternalErrorWithoutTheCause(t *testing.T) {
	rec := get(t, fakeStore{err: errors.New("db down")})
	assertBody(t, rec, http.StatusInternalServerError, `{"error":{"code":"internal_error","message":"Internal server error."}}`)
}
