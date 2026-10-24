package quiz

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

var (
	ichiKanji   = dataset.Kanji{Level: "n5", Code: "k102-001", Character: "一", Onyomi: []string{"イチ", "イツ"}, Kunyomi: []string{"ひと"}, Examples: []string{"一つ"}, MeaningEN: "one", MeaningID: "satu"}
	niKanji     = dataset.Kanji{Level: "n5", Code: "k102-002", Character: "二", Onyomi: []string{"ニ"}, Kunyomi: []string{"ふた"}, Examples: []string{"二つ"}, MeaningEN: "two", MeaningID: "dua"}
	sanKanji    = dataset.Kanji{Level: "n5", Code: "k102-003", Character: "三", Onyomi: []string{"サン"}, Kunyomi: []string{"み"}, Examples: []string{"三つ"}, MeaningEN: "three", MeaningID: "tiga"}
	senKanji    = dataset.Kanji{Level: "n5", Code: "k102-004", Character: "千", Onyomi: []string{"セン"}, Examples: []string{"三千"}, MeaningEN: "thousand", MeaningID: "seribu"}
	maeKanji    = dataset.Kanji{Level: "n5", Code: "k102-005", Character: "前", Onyomi: []string{"ゼン"}, Kunyomi: []string{"まえ"}, Examples: []string{"名前"}, MeaningEN: "front, ahead", MeaningID: "depan"}
	silentKanji = dataset.Kanji{Level: "n5", Code: "k102-006", Character: "々", Onyomi: []string{}, Kunyomi: []string{}, Examples: []string{}, MeaningEN: "repeat", MeaningID: "ulang"}

	mizuWord = dataset.Vocabulary{Level: "n5", Code: "v102-001", Word: "水", Reading: "みず", PartOfSpeech: "noun", MeaningEN: "water", MeaningID: "air"}
	honWord  = dataset.Vocabulary{Level: "n5", Code: "v102-002", Word: "本", Reading: "ほん", PartOfSpeech: "noun", MeaningEN: "book", MeaningID: "buku"}
	inuWord  = dataset.Vocabulary{Level: "n5", Code: "v102-003", Word: "犬", Reading: "いぬ", PartOfSpeech: "noun", MeaningEN: "dog", MeaningID: "anjing"}

	nagaraGrammar = dataset.Grammar{Level: "n4", Code: "g201-b", Pattern: "〜ながら", Difficulty: dataset.DifficultyEasy, MeaningEN: "while doing", MeaningID: "sambil"}
	kotoGrammar   = dataset.Grammar{Level: "n4", Code: "g201-c", Pattern: "〜ことがある", Difficulty: dataset.DifficultyEasy, MeaningEN: "have done", MeaningID: "pernah"}
	hazuGrammar   = dataset.Grammar{Level: "n4", Code: "g201-d", Pattern: "〜はずだ", Difficulty: dataset.DifficultyHard, MeaningEN: "should be", MeaningID: "seharusnya"}

	poolItems = Items{
		Kanji:      []dataset.Kanji{sakiKanji, bareKanji, ichiKanji, niKanji, sanKanji, senKanji, maeKanji, silentKanji},
		Vocabulary: []dataset.Vocabulary{amaiWord, kazeWord, kareWord, chuuWord, mizuWord, honWord, inuWord},
		Grammar:    []dataset.Grammar{aidaGrammr, nagaraGrammar, kotoGrammar, hazuGrammar},
	}
)

// draftOf writes the draft of one target from the pool, as the dataset quiz does.
func draftOf(t *testing.T, section dataset.Section, target Target) Draft {
	t.Helper()
	drafts := datasetDrafts(section, poolItems, LangEN, []Target{target})
	if len(drafts) != 1 || drafts[0].Code != target.Code || drafts[0].Type != target.Type {
		t.Fatalf("drafts = %+v; want one for %s %s", drafts, target.Code, target.Type)
	}
	return drafts[0]
}

func targetOf(t *testing.T, all []Target, code string, typ Type) Target {
	t.Helper()
	for _, target := range all {
		if target.Code == code && target.Type == typ {
			return target
		}
	}
	t.Fatalf("no %s target for %s", typ, code)
	return Target{}
}

func TestDatasetTargetsAreTheMeaningAndReadingPairsTheRowsAnswer(t *testing.T) {
	all := datasetTargets(LangEN, poolItems, []Type{TypeMeaning, TypeReading, TypeUsage})
	var got []string
	for _, target := range all {
		got = append(got, target.Code+" "+string(target.Type))
	}
	for _, want := range []string{"k101-001 meaning", "k101-001 reading", "k102-004 reading", "v101-001 meaning", "v201-001 reading", "g201-a meaning"} {
		if !slices.Contains(got, want) {
			t.Errorf("targets lack %s", want)
		}
	}
	for _, unwanted := range []string{"k102-006 reading", "v101-001 reading", "v201-002 reading", "v301-001 reading", "v101-001 usage", "g201-a usage", "k101-001 usage"} {
		if slices.Contains(got, unwanted) {
			t.Errorf("targets hold %s", unwanted)
		}
	}
	if len(got) != len(slices.Compact(slices.Sorted(slices.Values(got)))) {
		t.Errorf("targets repeat a pair: %v", got)
	}
	if only := datasetTargets(LangEN, poolItems, []Type{TypeUsage}); len(only) != 0 {
		t.Errorf("usage targets = %d, want none", len(only))
	}
}

func TestDatasetKanjiReadingAsksOneReadingOfTheCharacter(t *testing.T) {
	for range 40 {
		target := targetOf(t, datasetTargets(LangEN, Items{Kanji: []dataset.Kanji{ichiKanji}}, []Type{TypeReading}), "k102-001", TypeReading)
		if target.Prompt != "一" || !target.Kana || !slices.Contains([]string{"イチ", "イツ", "ひと"}, target.Answer) {
			t.Fatalf("target = %+v; want prompt 一 and one of its readings as the answer", target)
		}
	}
}

func TestDatasetMeaningDistractorsAreOtherItemsMeaningsWithoutASharedTerm(t *testing.T) {
	target := targetOf(t, datasetTargets(LangEN, poolItems, []Type{TypeMeaning}), "k101-001", TypeMeaning)
	valid := []string{"number", "one", "two", "three", "thousand", "repeat"}
	for range 100 {
		d := draftOf(t, dataset.SectionKanji, target).Distractors
		if len(d) != 3 || len(slices.Compact(slices.Sorted(slices.Values(d)))) != 3 {
			t.Fatalf("distractors = %v; want 3 distinct", d)
		}
		for _, s := range d {
			if !slices.Contains(valid, s) {
				t.Fatalf("distractor %q; want a meaning of another kanji that shares no term with %q", s, target.Answer)
			}
		}
	}
}

func TestDatasetKanjiReadingDistractorsAreKanaAndNeverTheKanjisOwnReadings(t *testing.T) {
	target := Target{Code: "k101-001", Type: TypeReading, Prompt: "先", Answer: "さき", Kana: true,
		Facts: Facts{Character: "先", Onyomi: []string{"セン"}, Kunyomi: []string{"さき"}}}
	for range 100 {
		for _, s := range draftOf(t, dataset.SectionKanji, target).Distractors {
			if s == "セン" || s == "さき" || !isKana(s) {
				t.Fatalf("distractor %q; want kana that is not a reading of 先", s)
			}
		}
	}
}

func TestDatasetVocabularyReadingDistractorsAreOtherWordsKanaReadings(t *testing.T) {
	target := targetOf(t, datasetTargets(LangEN, poolItems, []Type{TypeReading}), "v201-001", TypeReading)
	valid := []string{"あまい", "みず", "ほん", "いぬ"}
	for range 100 {
		for _, s := range draftOf(t, dataset.SectionVocabulary, target).Distractors {
			if !slices.Contains(valid, s) {
				t.Fatalf("distractor %q; want the kana reading of another word", s)
			}
		}
	}
}

func TestDatasetDraftHasFewerThanThreeDistractorsWhenThePoolRunsOut(t *testing.T) {
	small := Items{Kanji: []dataset.Kanji{sakiKanji, bareKanji}}
	target := targetOf(t, datasetTargets(LangEN, small, []Type{TypeMeaning}), "k101-001", TypeMeaning)
	d := datasetDrafts(dataset.SectionKanji, small, LangEN, []Target{target})[0]
	if !slices.Equal(d.Distractors, []string{"number"}) {
		t.Errorf("distractors = %v, want [number]", d.Distractors)
	}
}

func TestDatasetExplanationIsTheRowsOwnFacts(t *testing.T) {
	kanji := datasetTargets(LangEN, poolItems, []Type{TypeMeaning, TypeReading})
	vocabulary := datasetTargets(LangEN, poolItems, []Type{TypeReading})
	reading := Target{Code: "k101-001", Type: TypeReading, Prompt: "先", Answer: "さき", Kana: true, Facts: Facts{Character: "先", Onyomi: []string{"セン"}, Kunyomi: []string{"さき"}}}
	onlyOn := Target{Code: "k101-002", Type: TypeReading, Prompt: "号", Answer: "ゴウ", Kana: true, Facts: Facts{Character: "号", Onyomi: []string{"ゴウ"}}}
	for _, tt := range []struct {
		section dataset.Section
		target  Target
		want    string
	}{
		{dataset.SectionKanji, targetOf(t, kanji, "k101-001", TypeMeaning), "先 · previous, ahead"},
		{dataset.SectionKanji, reading, "先 · 音 セン · 訓 さき"},
		{dataset.SectionKanji, onlyOn, "号 · 音 ゴウ"},
		{dataset.SectionVocabulary, targetOf(t, vocabulary, "v201-001", TypeReading), "風邪を引く · かぜをひく"},
	} {
		if got := draftOf(t, tt.section, tt.target).Explanation; got != tt.want {
			t.Errorf("%s %s: explanation = %q, want %q", tt.target.Code, tt.target.Type, got, tt.want)
		}
	}
}

func TestDatasetHandlerRejectsInvalidRequestsInOrder(t *testing.T) {
	h := NewHandler(fakeStore{items: poolItems}, nil, time.Second)
	for _, tt := range []struct{ body, code, message string }{
		{`not json`, "invalid_body", "Body must be a JSON object with known keys."},
		{`{"level":"n5","section":"kanji","lang":"en","extra":1}`, "invalid_body", "Body must be a JSON object with known keys."},
		{`{}`, "invalid_level", "Level must be n5, n4, n3, n2, or n1."},
		{`{"level":"n5","section":"Kanji"}`, "invalid_section", "Section must be kanji, vocabulary, or grammar."},
		{`{"level":"n5","section":"kanji","lang":"ja","count":0}`, "invalid_lang", "Lang must be en or id."},
		{`{"level":"n5","section":"kanji","lang":"en","types":["usage"],"count":0}`, "invalid_type", "Every type must be a question type of the section."},
		{`{"level":"n5","section":"vocabulary","lang":"en","types":["meaning","usage"]}`, "invalid_type", "Every type must be a question type of the section."},
		{`{"level":"n5","section":"grammar","lang":"en","types":["reading"]}`, "invalid_type", "Every type must be a question type of the section."},
		{`{"level":"n5","section":"kanji","lang":"en","count":11}`, "invalid_count", "Count must be 1 to 10."},
	} {
		t.Run(tt.code, func(t *testing.T) {
			assertError(t, postTo(t, h, "/api/quizzes/dataset", tt.body), http.StatusBadRequest, tt.code, tt.message)
		})
	}

	invalidLesson := NewHandler(fakeStore{err: ErrInvalidLesson}, nil, time.Second)
	assertError(t, postTo(t, invalidLesson, "/api/quizzes/dataset", `{"level":"n5","section":"kanji","lang":"en","lesson_ids":["nope"]}`),
		http.StatusBadRequest, "invalid_lesson", "Every lesson must be a lesson of the level and section.")

	kanaOnly := NewHandler(fakeStore{items: Items{Vocabulary: []dataset.Vocabulary{amaiWord}}}, nil, time.Second)
	assertError(t, postTo(t, kanaOnly, "/api/quizzes/dataset", `{"level":"n5","section":"vocabulary","lang":"en","types":["reading"]}`),
		http.StatusNotFound, "items_not_found", "No item matches the request.")
}

func TestDatasetHandlerAnswersWithNoProviderWhileTheAIQuizStaysUnavailable(t *testing.T) {
	h := NewHandler(fakeStore{items: poolItems}, nil, time.Second)
	got, dropped := decode(t, postTo(t, h, "/api/quizzes/dataset", `{"level":"n5","section":"kanji","lang":"en"}`))
	if len(got) != 5 || dropped != 0 {
		t.Errorf("got %d questions and %d dropped; want 5 and 0", len(got), dropped)
	}
	assertError(t, post(t, h, `{"level":"n5","section":"kanji","lang":"en"}`),
		http.StatusServiceUnavailable, "quiz_unavailable", "Quiz provider is not configured.")
}

func TestDatasetHandlerWritesEveryFieldFromTheRows(t *testing.T) {
	h := NewHandler(fakeStore{items: poolItems}, nil, time.Second)
	for _, section := range []string{"kanji", "vocabulary", "grammar"} {
		got, dropped := decode(t, postTo(t, h, "/api/quizzes/dataset", `{"level":"n4","section":"`+section+`","lang":"id","count":10}`))
		if len(got) == 0 || dropped != 0 {
			t.Fatalf("%s: got %d questions and %d dropped", section, len(got), dropped)
		}
		for _, q := range got {
			if len(slices.Compact(slices.Sorted(slices.Values(q.Choices)))) != 4 || !slices.Contains(q.Choices, q.Answer) || q.Explanation == "" || q.Prompt == "" {
				t.Errorf("%s %s %s: %+v; want 4 distinct choices that hold the answer, a prompt, and an explanation", section, q.Code, q.Type, q)
			}
			if q.Type == "reading" && slices.ContainsFunc(q.Choices, func(s string) bool { return !isKana(s) }) {
				t.Errorf("%s %s: reading choices %v are not all kana", section, q.Code, q.Choices)
			}
		}
	}

	var meaning seen
	for _, q := range func() []seen {
		got, _ := decode(t, postTo(t, h, "/api/quizzes/dataset", `{"level":"n5","section":"kanji","lang":"id","types":["meaning"],"count":10}`))
		return got
	}() {
		if q.Code == "k101-001" {
			meaning = q
		}
	}
	if meaning.Prompt != "先" || meaning.Answer != "sebelumnya, depan" || meaning.Explanation != "先 · sebelumnya, depan" {
		t.Errorf("先 meaning in id = %+v; want the row's character and meaning_id", meaning)
	}
}

func TestDatasetHandlerDrawsDistractorsFromTheWholeSectionWhateverTheLessons(t *testing.T) {
	var scope Scope
	h := NewHandler(fakeStore{items: poolItems, got: &scope}, nil, time.Second)
	decode(t, postTo(t, h, "/api/quizzes/dataset", `{"level":"n5","section":"kanji","lang":"en","lesson_ids":["l-1"]}`))
	if want := (Scope{Level: "n5", Section: dataset.SectionKanji}); !reflectEqual(scope, want) {
		t.Errorf("pool scope = %+v, want %+v", scope, want)
	}
}

func TestDatasetHandlerCountsTargetsWithoutEnoughDistractorsAsDropped(t *testing.T) {
	h := NewHandler(fakeStore{items: Items{Kanji: []dataset.Kanji{sakiKanji, bareKanji}}}, nil, time.Second)
	got, dropped := decode(t, postTo(t, h, "/api/quizzes/dataset", `{"level":"n5","section":"kanji","lang":"en","types":["meaning"]}`))
	if len(got) != 0 || dropped != 2 {
		t.Errorf("got %d questions and %d dropped; want 0 and 2", len(got), dropped)
	}
}
