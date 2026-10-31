package item

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// fakeKanjiStore answers with its levels and its rows holding the characters
// asked for.
type fakeKanjiStore struct {
	levels []string
	rows   []dataset.Kanji
	err    error
}

func (f fakeKanjiStore) Kanji(_ context.Context, characters []string) ([]string, []dataset.Kanji, error) {
	var held []dataset.Kanji
	for _, k := range f.rows {
		if slices.Contains(characters, k.Character) {
			held = append(held, k)
		}
	}
	return f.levels, held, f.err
}

// cover posts body to the coverage route, served from store.
func cover(t *testing.T, store fakeKanjiStore, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	routeCoverage(mux, store)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/kanji/coverage", strings.NewReader(body)))
	return rec
}

func kanjiRow(level, code, character string) dataset.Kanji {
	return dataset.Kanji{ID: code, Level: level, Code: code, LessonID: "l1", Sequence: 1, Character: character,
		Onyomi: []string{}, Kunyomi: []string{}, Examples: []string{}, Sources: []dataset.Source{dataset.SourceSoumatome}}
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var threeLevels = fakeKanjiStore{levels: []string{"n5", "n4", "n3"}}

func TestCoverageRejectsAnInvalidBodyThenAnInvalidText(t *testing.T) {
	invalidBody := errorJSON("invalid_body", "Body must be a JSON object with known keys.")
	invalidText := errorJSON("invalid_text", "Text must be 1 to 10000 characters.")
	cases := []struct{ name, body, want string }{
		{"array", `[]`, invalidBody},
		{"null", `null`, invalidBody},
		{"string", `"日"`, invalidBody},
		{"text not a string", `{"text":5}`, invalidBody},
		{"unknown key", `{"text":"日","txt":"日"}`, invalidBody},
		{"trailing data", `{"text":"日"}{}`, invalidBody},
		{"over 64 KiB", `{"text":"` + strings.Repeat("a", 64<<10) + `"}`, invalidBody},
		{"missing text", `{}`, invalidText},
		{"null text", `{"text":null}`, invalidText},
		{"blank text", `{"text":" \n\t"}`, invalidText},
		{"10001 characters", `{"text":"` + strings.Repeat("あ", 10001) + `"}`, invalidText},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertBody(t, cover(t, threeLevels, c.body), http.StatusBadRequest, c.want)
		})
	}
}

func TestCoverageAcceptsATextOf10000CharactersAfterTrimming(t *testing.T) {
	store := fakeKanjiStore{levels: []string{"n5"}, rows: []dataset.Kanji{kanjiRow("n5", "k101-002", "日")}}
	rec := cover(t, store, `{"text":"  `+strings.Repeat("日", 10000)+`\n"}`)
	var got Coverage
	if err := json.Unmarshal(rec.Body.Bytes(), &got); rec.Code != http.StatusOK || err != nil {
		t.Fatalf("got %d %s, want 200", rec.Code, rec.Body.String())
	}
	if want := []LevelCoverage{{Level: "n5", Known: 10000, KnownPercent: 100}}; got.Total != 10000 || !slices.Equal(got.Levels, want) {
		t.Errorf("total %d, levels %+v; want 10000 and %+v", got.Total, got.Levels, want)
	}
}

func TestCoverageAnswersEachKanjiAndTheKnownShareOfEveryLevel(t *testing.T) {
	now, day, weekday := kanjiRow("n5", "k101-001", "今"), kanjiRow("n5", "k101-002", "日"), kanjiRow("n4", "k201-001", "曜")
	store := fakeKanjiStore{levels: []string{"n5", "n4", "n3"}, rows: []dataset.Kanji{now, day, weekday, kanjiRow("n4", "k205-001", "日")}}
	assertBody(t, cover(t, store, `{"text":"今日は日曜日。降る々"}`), http.StatusOK, `{"total":6,"levels":[`+
		`{"level":"n5","known":4,"known_percent":66},{"level":"n4","known":5,"known_percent":83},{"level":"n3","known":5,"known_percent":83}],"kanji":[`+
		`{"character":"今","count":1,"item":`+jsonOf(t, now)+`},`+
		`{"character":"日","count":3,"item":`+jsonOf(t, day)+`},`+
		`{"character":"曜","count":1,"item":`+jsonOf(t, weekday)+`},`+
		`{"character":"降","count":1,"item":null}]}`)
}

func TestCoverageOfATextWithoutKanjiIsFullAtEveryLevel(t *testing.T) {
	assertBody(t, cover(t, threeLevels, `{"text":"ひらがな、カタカナ、々〇"}`), http.StatusOK, `{"total":0,"levels":[`+
		`{"level":"n5","known":0,"known_percent":100},{"level":"n4","known":0,"known_percent":100},{"level":"n3","known":0,"known_percent":100}],"kanji":[]}`)
}

func TestCoverageAnswers500WithoutTheStoreErrorCause(t *testing.T) {
	broken := fakeKanjiStore{err: errors.New("password authentication failed")}
	assertBody(t, cover(t, broken, `{"text":"日"}`), http.StatusInternalServerError, errorJSON("internal_error", "Internal server error."))
}
