package quiz

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// fakeStore answers with its items of the scope's section and keeps the
// scope it was asked for.
type fakeStore struct {
	items Items
	err   error
	got   *Scope
}

func (f fakeStore) Items(_ context.Context, scope Scope) (Items, error) {
	if f.got != nil {
		*f.got = scope
	}
	switch scope.Section {
	case dataset.SectionKanji:
		return Items{Kanji: f.items.Kanji}, f.err
	case dataset.SectionVocabulary:
		return Items{Vocabulary: f.items.Vocabulary}, f.err
	default:
		return Items{Grammar: f.items.Grammar}, f.err
	}
}

// fakeModel answers like a provider, with drafts from draft.
type fakeModel func(ctx context.Context, lang Lang, targets []Target) ([]Draft, error)

func (f fakeModel) Generate(ctx context.Context, lang Lang, targets []Target) ([]Draft, error) {
	return f(ctx, lang, targets)
}

// goodDraft is an entry that passes every rule of its target.
func goodDraft(t Target) Draft {
	d := Draft{Code: t.Code, Type: t.Type, Distractors: []string{"bitter", "salty", "sour"}, Explanation: "Karena itu."}
	switch {
	case t.Kana:
		d.Distractors = []string{"かぜをいく", "かぜをしく", "かせをひく"}
	case t.Blank:
		d.Prompt, d.Distractors = "毎日＿＿。", []string{"かう", "のむ", "みる"}
	}
	if t.Examples != nil {
		d.Prompt = t.Examples[0]
	}
	if t.Answer == "" {
		d.Answer = "せんせい"
	}
	return d
}

// model answers goodDraft for every target, changed by edit.
func model(edit func(t Target, d *Draft)) fakeModel {
	return func(_ context.Context, _ Lang, targets []Target) ([]Draft, error) {
		drafts := make([]Draft, len(targets))
		for i, t := range targets {
			drafts[i] = goodDraft(t)
			edit(t, &drafts[i])
		}
		return drafts, nil
	}
}

var unchanged = model(func(Target, *Draft) {})

var items = Items{
	Kanji:      []dataset.Kanji{sakiKanji, bareKanji},
	Vocabulary: []dataset.Vocabulary{amaiWord, kazeWord},
	Grammar:    []dataset.Grammar{aidaGrammr},
}

func post(t *testing.T, h Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	h.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/quizzes", strings.NewReader(body)))
	return rec
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code, message string) {
	t.Helper()
	want := `{"error":{"code":"` + code + `","message":"` + message + `"}}` + "\n"
	if rec.Code != status || rec.Body.String() != want {
		t.Errorf("got %d %s\nwant %d %s", rec.Code, rec.Body.String(), status, want)
	}
}

// seen is a response question with its choices sorted and its answer
// replaced by the choice it points at.
type seen struct {
	Code, Type, Prompt, Answer, Explanation string
	Choices                                 []string
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) ([]seen, int) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d %s; want 200", rec.Code, rec.Body.String())
	}
	var body struct {
		Questions []struct {
			Code        string   `json:"code"`
			Type        string   `json:"type"`
			Prompt      string   `json:"prompt"`
			Choices     []string `json:"choices"`
			Answer      int      `json:"answer"`
			Explanation string   `json:"explanation"`
		} `json:"questions"`
		Dropped int `json:"dropped"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	out := make([]seen, len(body.Questions))
	for i, q := range body.Questions {
		if len(q.Choices) != 4 || q.Answer < 0 || q.Answer > 3 {
			t.Fatalf("%s %s: choices %v, answer %d; want 4 choices and an index", q.Code, q.Type, q.Choices, q.Answer)
		}
		out[i] = seen{Code: q.Code, Type: q.Type, Prompt: q.Prompt, Answer: q.Choices[q.Answer], Explanation: q.Explanation,
			Choices: slices.Sorted(slices.Values(q.Choices))}
	}
	return out, body.Dropped
}

func TestHandlerRejectsInvalidRequestsInOrder(t *testing.T) {
	h := NewHandler(fakeStore{items: items}, unchanged, time.Second)
	for _, tt := range []struct{ body, code, message string }{
		{`not json`, "invalid_body", "Body must be a JSON object with known keys."},
		{`[]`, "invalid_body", "Body must be a JSON object with known keys."},
		{`null`, "invalid_body", "Body must be a JSON object with known keys."},
		{`{"level":"n5","section":"kanji","lang":"en","extra":1}`, "invalid_body", "Body must be a JSON object with known keys."},
		{`{"level":5}`, "invalid_body", "Body must be a JSON object with known keys."},
		{`{"level":"n5","section":"kanji","lang":"en","count":1.5}`, "invalid_body", "Body must be a JSON object with known keys."},
		{`{"level":"n5","section":"kanji","lang":"en"} {}`, "invalid_body", "Body must be a JSON object with known keys."},
		{`{"level":"n5","section":"kanji","lang":"en","note":"` + strings.Repeat("a", 64<<10) + `"}`, "invalid_body", "Body must be a JSON object with known keys."},
		{`{}`, "invalid_level", "Level must be n5, n4, n3, n2, or n1."},
		{`{"level":"N5","section":"x"}`, "invalid_level", "Level must be n5, n4, n3, n2, or n1."},
		{`{"level":"n5","section":"Kanji"}`, "invalid_section", "Section must be kanji, vocabulary, or grammar."},
		{`{"level":"n5","section":"kanji","lang":"ja","count":0}`, "invalid_lang", "Lang must be en or id."},
		{`{"level":"n5","section":"kanji","lang":"en","types":["usage"],"count":0}`, "invalid_type", "Every type must be a question type of the section."},
		{`{"level":"n5","section":"grammar","lang":"en","types":["meaning","reading"]}`, "invalid_type", "Every type must be a question type of the section."},
		{`{"level":"n5","section":"kanji","lang":"en","count":0}`, "invalid_count", "Count must be 1 to 10."},
		{`{"level":"n5","section":"kanji","lang":"en","count":11}`, "invalid_count", "Count must be 1 to 10."},
	} {
		t.Run(tt.code, func(t *testing.T) {
			assertError(t, post(t, h, tt.body), http.StatusBadRequest, tt.code, tt.message)
		})
	}

	invalidLesson := NewHandler(fakeStore{err: ErrInvalidLesson}, unchanged, time.Second)
	assertError(t, post(t, invalidLesson, `{"level":"n5","section":"kanji","lang":"en","lesson_ids":["nope"]}`),
		http.StatusBadRequest, "invalid_lesson", "Every lesson must be a lesson of the level and section.")

	kanaOnly := NewHandler(fakeStore{items: Items{Vocabulary: []dataset.Vocabulary{amaiWord}}}, unchanged, time.Second)
	assertError(t, post(t, kanaOnly, `{"level":"n5","section":"vocabulary","lang":"en","types":["reading"]}`),
		http.StatusNotFound, "items_not_found", "No item matches the request.")
}

func TestHandlerAsksForTheScopeAndCountRequested(t *testing.T) {
	var scope Scope
	var asked []Target
	var askedLang Lang
	gen := fakeModel(func(ctx context.Context, lang Lang, targets []Target) ([]Draft, error) {
		asked, askedLang = targets, lang
		return unchanged(ctx, lang, targets)
	})
	h := NewHandler(fakeStore{items: items, got: &scope}, gen, time.Second)

	got, dropped := decode(t, post(t, h, `{"level":"n4","section":"vocabulary","lang":"id","lesson_ids":["l-1","l-2"],"types":["usage"]}`))
	if want := (Scope{Level: "n4", Section: dataset.SectionVocabulary, LessonIDs: []string{"l-1", "l-2"}}); !reflectEqual(scope, want) {
		t.Errorf("scope: got %+v, want %+v", scope, want)
	}
	if askedLang != LangID || len(asked) != 2 || len(got) != 2 || dropped != 0 {
		t.Errorf("usage: asked %d targets in %q, got %d questions and %d dropped; want 2 in id, 2 and 0", len(asked), askedLang, len(got), dropped)
	}
	for _, q := range got {
		if q.Type != "usage" {
			t.Errorf("%s: type %s, want only usage", q.Code, q.Type)
		}
	}

	if got, _ := decode(t, post(t, h, `{"level":"n5","section":"kanji","lang":"en","types":[]}`)); len(got) != 3 {
		t.Errorf("kanji, no count: got %d questions; want all 3 targets, under the default of 5", len(got))
	}
	if got, _ := decode(t, post(t, h, `{"level":"n5","section":"kanji","lang":"en","count":2}`)); len(got) != 2 {
		t.Errorf("count 2: got %d questions", len(got))
	}
}

func reflectEqual(a, b Scope) bool {
	return a.Level == b.Level && a.Section == b.Section && slices.Equal(a.LessonIDs, b.LessonIDs)
}

func TestHandlerWritesRowFieldsAndKeepsModelFieldsWhereTheRowHasNone(t *testing.T) {
	overriding := model(func(t Target, d *Draft) {
		if t.Prompt != "" {
			d.Prompt = "ignored"
		}
		if t.Answer != "" {
			d.Answer = "ignored"
		}
	})
	h := NewHandler(fakeStore{items: items}, overriding, time.Second)
	byKey := func(qs []seen) map[string]seen {
		m := map[string]seen{}
		for _, q := range qs {
			m[q.Code+" "+q.Type] = q
		}
		return m
	}

	kanji, _ := decode(t, post(t, h, `{"level":"n5","section":"kanji","lang":"id"}`))
	vocabulary, _ := decode(t, post(t, h, `{"level":"n4","section":"vocabulary","lang":"en"}`))
	grammar, _ := decode(t, post(t, h, `{"level":"n4","section":"grammar","lang":"en"}`))
	got := byKey(slices.Concat(kanji, vocabulary, grammar))

	for key, want := range map[string]seen{
		"k101-001 meaning": {Prompt: "先", Answer: "sebelumnya, depan", Choices: []string{"bitter", "salty", "sebelumnya, depan", "sour"}},
		"k101-001 reading": {Prompt: "先生", Answer: "せんせい", Choices: []string{"かせをひく", "かぜをいく", "かぜをしく", "せんせい"}},
		"v201-001 reading": {Prompt: "風邪を引く", Answer: "かぜをひく", Choices: []string{"かせをひく", "かぜをいく", "かぜをしく", "かぜをひく"}},
		"v201-001 usage":   {Prompt: "毎日＿＿。", Answer: "風邪を引く", Choices: []string{"かう", "のむ", "みる", "風邪を引く"}},
		"g201-a meaning":   {Prompt: "〜あいだに", Answer: "while", Choices: []string{"bitter", "salty", "sour", "while"}},
		"g201-a usage":     {Prompt: "毎日＿＿。", Answer: "せんせい", Choices: []string{"かう", "せんせい", "のむ", "みる"}},
	} {
		q, ok := got[key]
		if !ok {
			t.Errorf("%s: no question", key)
			continue
		}
		if q.Prompt != want.Prompt || q.Answer != want.Answer || !slices.Equal(q.Choices, want.Choices) || q.Explanation != "Karena itu." {
			t.Errorf("%s:\ngot  %+v\nwant %+v", key, q, want)
		}
	}
	if len(got) != 10 {
		t.Errorf("got %d questions over the three sections; want 10", len(got))
	}
}

func TestHandlerDropsEntriesThatBreakARule(t *testing.T) {
	for _, tt := range []struct {
		name    string
		section string
		target  string // "code type" of the broken entry
		edit    func(d *Draft)
	}{
		{"two distractors", "vocabulary", "v101-001 meaning", func(d *Draft) { d.Distractors = d.Distractors[:2] }},
		{"blank distractor", "vocabulary", "v101-001 meaning", func(d *Draft) { d.Distractors[1] = "  " }},
		{"distractor is the answer", "vocabulary", "v101-001 meaning", func(d *Draft) { d.Distractors[2] = "sweet" }},
		{"repeated distractor", "vocabulary", "v101-001 meaning", func(d *Draft) { d.Distractors[2] = d.Distractors[0] }},
		{"no explanation", "kanji", "k101-001 meaning", func(d *Draft) { d.Explanation = " " }},
		{"romaji reading distractor", "vocabulary", "v201-001 reading", func(d *Draft) { d.Distractors[0] = "kaze" }},
		{"kanji reading prompt outside examples", "kanji", "k101-001 reading", func(d *Draft) { d.Prompt = "先週" }},
		{"kanji reading answer in kanji", "kanji", "k101-001 reading", func(d *Draft) { d.Answer = "先生" }},
		{"usage without a blank", "vocabulary", "v101-001 usage", func(d *Draft) { d.Prompt = "毎日。" }},
		{"usage with two blanks", "vocabulary", "v101-001 usage", func(d *Draft) { d.Prompt = "＿＿と＿＿。" }},
		{"usage distractor in English", "vocabulary", "v101-001 usage", func(d *Draft) { d.Distractors[0] = "drink" }},
		{"usage prompt holds the word", "vocabulary", "v101-001 usage", func(d *Draft) { d.Prompt = "あまい＿＿。" }},
		{"grammar usage without an answer", "grammar", "g201-a usage", func(d *Draft) { d.Answer = "" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gen := model(func(t Target, d *Draft) {
				if t.Code+" "+string(t.Type) == tt.target {
					tt.edit(d)
				}
			})
			got, dropped := decode(t, post(t, NewHandler(fakeStore{items: items}, gen, time.Second),
				`{"level":"n5","section":"`+tt.section+`","lang":"en","count":10}`))
			for _, q := range got {
				if q.Code+" "+q.Type == tt.target {
					t.Errorf("%s was kept: %+v", tt.target, q)
				}
			}
			if dropped != 1 || len(got) == 0 {
				t.Errorf("got %d questions and %d dropped; want the others kept and 1 dropped", len(got), dropped)
			}
		})
	}
}

func TestHandlerCountsMissingAndRepeatedEntriesAsDroppedAndIgnoresStrays(t *testing.T) {
	gen := fakeModel(func(_ context.Context, _ Lang, targets []Target) ([]Draft, error) {
		var drafts []Draft
		for _, t := range targets {
			switch t.Type {
			case TypeMeaning:
				drafts = append(drafts, goodDraft(t))
			case TypeReading:
				drafts = append(drafts, goodDraft(t), goodDraft(t))
			}
		}
		return append(drafts, Draft{Code: "k999-999", Type: TypeMeaning, Distractors: []string{"a", "b", "c"}, Explanation: "x"}), nil
	})
	got, dropped := decode(t, post(t, NewHandler(fakeStore{items: items}, gen, time.Second), `{"level":"n4","section":"vocabulary","lang":"en"}`))
	if keys := questionKeys(got); !slices.Equal(keys, []string{"v101-001 meaning", "v201-001 meaning"}) || dropped != 3 {
		t.Errorf("got %v and %d dropped; want the 2 meanings kept and reading and 2 usages dropped", keys, dropped)
	}
}

func questionKeys(qs []seen) []string {
	keys := make([]string, len(qs))
	for i, q := range qs {
		keys[i] = q.Code + " " + q.Type
	}
	return slices.Sorted(slices.Values(keys))
}

func TestHandlerAnswersProviderFailuresWithoutTheirCause(t *testing.T) {
	failing := fakeModel(func(context.Context, Lang, []Target) ([]Draft, error) {
		return nil, errors.New("401 invalid api key sk-secret")
	})
	allBroken := model(func(_ Target, d *Draft) { d.Explanation = "" })
	slow := fakeModel(func(ctx context.Context, _ Lang, _ []Target) ([]Draft, error) {
		<-ctx.Done()
		return nil, errors.New("request aborted: sk-secret")
	})
	request := `{"level":"n5","section":"kanji","lang":"en"}`

	assertError(t, post(t, NewHandler(fakeStore{items: items}, failing, time.Second), request),
		http.StatusBadGateway, "quiz_generation_failed", "Quiz generation failed.")
	assertError(t, post(t, NewHandler(fakeStore{items: items}, allBroken, time.Second), request),
		http.StatusBadGateway, "quiz_generation_failed", "Quiz generation failed.")
	assertError(t, post(t, NewHandler(fakeStore{items: items}, slow, 20*time.Millisecond), request),
		http.StatusGatewayTimeout, "quiz_generation_timeout", "Quiz generation timed out.")
	assertError(t, post(t, NewHandler(fakeStore{items: items}, nil, time.Second), `not json`),
		http.StatusServiceUnavailable, "quiz_unavailable", "Quiz provider is not configured.")
	assertError(t, post(t, NewHandler(fakeStore{err: errors.New("connection refused")}, unchanged, time.Second), request),
		http.StatusInternalServerError, "internal_error", "Internal server error.")
}

func TestHandlerOutlastsTheServerWriteTimeoutWithinItsOwn(t *testing.T) {
	slow := fakeModel(func(ctx context.Context, lang Lang, targets []Target) ([]Draft, error) {
		time.Sleep(150 * time.Millisecond)
		return unchanged(ctx, lang, targets)
	})
	mux := http.NewServeMux()
	NewHandler(fakeStore{items: items}, slow, time.Second).Register(mux)
	srv := httptest.NewUnstartedServer(mux)
	srv.Config.WriteTimeout = 50 * time.Millisecond
	srv.Start()
	defer srv.Close()

	res, err := http.Post(srv.URL+"/api/quizzes", "application/json", strings.NewReader(`{"level":"n5","section":"kanji","lang":"en"}`))
	if err != nil {
		t.Fatalf("a quiz slower than the write timeout got no answer: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("got %d, want 200", res.StatusCode)
	}
}
