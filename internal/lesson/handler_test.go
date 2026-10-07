package lesson

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// fakeStore answers from the lessons and details it holds.
type fakeStore struct {
	lessons []dataset.Lesson
	details []Detail
	err     error
}

func (f fakeStore) List(_ context.Context, level string, section dataset.Section) ([]dataset.Lesson, error) {
	matched := []dataset.Lesson{}
	for _, l := range f.lessons {
		if l.Level == level && l.Section == section {
			matched = append(matched, l)
		}
	}
	return matched, f.err
}

func (f fakeStore) Get(_ context.Context, id string) (Detail, error) {
	if f.err != nil {
		return Detail{}, f.err
	}
	for _, d := range f.details {
		if d.Lesson.ID == id {
			return d, nil
		}
	}
	return Detail{}, ErrNotFound
}

func get(t *testing.T, store fakeStore, target string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
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

func lesson(id string, section dataset.Section) dataset.Lesson {
	return dataset.Lesson{ID: id, Level: "n5", Section: section, Week: 1, Day: 1,
		Title: "お名前は？", TitleEN: "Names", TitleID: "Nama", WeekTitle: "れんしゅう①", WeekTitleEN: "Practice 1"}
}

func lessonJSON(id, section string) string {
	return `{"id":"` + id + `","level":"n5","section":"` + section + `","week":1,"day":1,"title":"お名前は？","title_en":"Names","title_id":"Nama","week_title":"れんしゅう①","week_title_en":"Practice 1","week_title_id":null}`
}

var (
	kanjiLesson = Detail{
		Lesson: lesson("l-kanji", dataset.SectionKanji),
		Kanji: []dataset.Kanji{{ID: "k1", Level: "n5", Code: "k101-001", LessonID: "l-kanji", Sequence: 1, Character: "先",
			Onyomi: []string{"セン"}, Kunyomi: []string{"さき"}, Examples: []string{"先生"},
			MeaningEN: "ahead", MeaningID: "depan", Sources: []dataset.Source{dataset.SourceSoumatome}}},
	}
	vocabularyLesson = Detail{
		Lesson: lesson("l-vocabulary", dataset.SectionVocabulary),
		Vocabulary: []dataset.Vocabulary{{ID: "v1", Level: "n5", Code: "v101-001", LessonID: "l-vocabulary", Sequence: 1,
			Word: "あまい", Reading: "あまい", PartOfSpeech: "い-adjective", MeaningEN: "sweet", MeaningID: "manis",
			Sources: []dataset.Source{dataset.SourceSoumatome}}},
	}
	grammarLesson = Detail{
		Lesson: lesson("l-grammar", dataset.SectionGrammar),
		Grammar: []dataset.Grammar{{ID: "g1", Level: "n5", Code: "g101", LessonID: "l-grammar", Sequence: 1,
			CurriculumCode: "g101", Pattern: "い形容詞の形", Difficulty: dataset.DifficultyEasy,
			MeaningEN: "i-adjective forms", MeaningID: "bentuk adjektiva-i", Sources: []dataset.Source{dataset.SourceSoumatome}}},
		Comparisons: []dataset.GrammarComparison{{ID: "c1", LessonID: "l-grammar", Sequence: 1,
			PatternA: "は", PatternB: "が", Difference: "topic or subject", UseWhen: "new information", Example: "わたしが行きます。"}},
		Mistakes:    []dataset.GrammarMistake{},
		Expressions: []dataset.GrammarExpression{},
	}
)

func TestHandlerListsTheLessonsOfALevelAndSection(t *testing.T) {
	store := fakeStore{lessons: []dataset.Lesson{lesson("l-kanji", dataset.SectionKanji), lesson("l-grammar", dataset.SectionGrammar)}}

	assertBody(t, get(t, store, "/api/levels/n5/sections/kanji/lessons"), http.StatusOK, `[`+lessonJSON("l-kanji", "kanji")+`]`)
	assertBody(t, get(t, store, "/api/levels/n2/sections/kanji/lessons"), http.StatusOK, `[]`)
}

func TestHandlerRejectsAnUnknownLevelOrSection(t *testing.T) {
	store := fakeStore{lessons: []dataset.Lesson{lesson("l-kanji", dataset.SectionKanji)}}
	levelNotFound := `{"error":{"code":"level_not_found","message":"Level was not found."}}`
	sectionNotFound := `{"error":{"code":"section_not_found","message":"Section was not found."}}`

	assertBody(t, get(t, store, "/api/levels/n9/sections/kanji/lessons"), http.StatusNotFound, levelNotFound)
	assertBody(t, get(t, store, "/api/levels/N5/sections/kanji/lessons"), http.StatusNotFound, levelNotFound)
	assertBody(t, get(t, store, "/api/levels/n5/sections/phrases/lessons"), http.StatusNotFound, sectionNotFound)
	assertBody(t, get(t, store, "/api/levels/n5/sections/Kanji/lessons"), http.StatusNotFound, sectionNotFound)
}

func TestHandlerGetsALessonWithTheItemsOfItsSection(t *testing.T) {
	store := fakeStore{details: []Detail{kanjiLesson, vocabularyLesson, grammarLesson}}

	assertBody(t, get(t, store, "/api/lessons/l-kanji"), http.StatusOK,
		`{"lesson":`+lessonJSON("l-kanji", "kanji")+`,"kanji":[{"id":"k1","level":"n5","code":"k101-001","lesson_id":"l-kanji","sequence":1,"character":"先","onyomi":["セン"],"kunyomi":["さき"],"examples":["先生"],"meaning_en":"ahead","meaning_id":"depan","sources":["soumatome"]}]}`)
	assertBody(t, get(t, store, "/api/lessons/l-vocabulary"), http.StatusOK,
		`{"lesson":`+lessonJSON("l-vocabulary", "vocabulary")+`,"vocabulary":[{"id":"v1","level":"n5","code":"v101-001","lesson_id":"l-vocabulary","sequence":1,"word":"あまい","reading":"あまい","part_of_speech":"い-adjective","note":null,"meaning_en":"sweet","meaning_id":"manis","sources":["soumatome"]}]}`)
	assertBody(t, get(t, store, "/api/lessons/l-grammar"), http.StatusOK,
		`{"lesson":`+lessonJSON("l-grammar", "grammar")+`,"grammar":[{"id":"g1","level":"n5","code":"g101","lesson_id":"l-grammar","sequence":1,"curriculum_code":"g101","pattern":"い形容詞の形","reading":null,"formula":null,"note":null,"example":null,"difficulty":"easy","meaning_en":"i-adjective forms","meaning_id":"bentuk adjektiva-i","sources":["soumatome"]}],`+
			`"comparisons":[{"id":"c1","lesson_id":"l-grammar","sequence":1,"pattern_a":"は","pattern_b":"が","difference":"topic or subject","use_when":"new information","example":"わたしが行きます。"}],"mistakes":[],"expressions":[]}`)
}

func TestHandlerAnswersLessonNotFound(t *testing.T) {
	assertBody(t, get(t, fakeStore{details: []Detail{kanjiLesson}}, "/api/lessons/nope"), http.StatusNotFound,
		`{"error":{"code":"lesson_not_found","message":"Lesson was not found."}}`)
}

func TestHandlerHidesStoreErrors(t *testing.T) {
	store := fakeStore{err: errors.New("password authentication failed for user postgres")}
	internal := `{"error":{"code":"internal_error","message":"Internal server error."}}`

	assertBody(t, get(t, store, "/api/levels/n5/sections/kanji/lessons"), http.StatusInternalServerError, internal)
	assertBody(t, get(t, store, "/api/lessons/l-kanji"), http.StatusInternalServerError, internal)
}
