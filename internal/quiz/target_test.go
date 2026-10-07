package quiz

import (
	"reflect"
	"slices"
	"testing"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

func ptr(s string) *string { return &s }

var (
	sakiKanji = dataset.Kanji{Level: "n5", Code: "k101-001", Character: "先", Onyomi: []string{"セン"}, Kunyomi: []string{"さき"},
		Examples: []string{"先生", "先月"}, MeaningEN: "previous, ahead", MeaningID: "sebelumnya, depan"}
	bareKanji = dataset.Kanji{Level: "n5", Code: "k101-002", Character: "号", Onyomi: []string{"ゴウ"},
		Examples: []string{}, MeaningEN: "number", MeaningID: "nomor"}

	amaiWord   = dataset.Vocabulary{Level: "n5", Code: "v101-001", Word: "あまい", Reading: "あまい", PartOfSpeech: "い-adjective", MeaningEN: "sweet", MeaningID: "manis"}
	kazeWord   = dataset.Vocabulary{Level: "n4", Code: "v201-001", Word: "風邪を引く", Reading: "かぜをひく", PartOfSpeech: "phrase", Note: ptr("風邪を引いた。"), MeaningEN: "catch a cold", MeaningID: "masuk angin"}
	kareWord   = dataset.Vocabulary{Level: "n3", Code: "v301-001", Word: "彼／彼氏", Reading: "かれ／かれし", PartOfSpeech: "noun", MeaningEN: "he; boyfriend", MeaningID: "dia; pacar"}
	chuuWord   = dataset.Vocabulary{Level: "n4", Code: "v201-002", Word: "〜中", Reading: "〜ちゅう", PartOfSpeech: "suffix", MeaningEN: "during", MeaningID: "selama"}
	aidaGrammr = dataset.Grammar{Level: "n4", Code: "g201-a", Pattern: "〜あいだに", Formula: ptr("Vる＋あいだに"), Example: ptr("寝ているあいだに。"),
		Difficulty: dataset.DifficultyMedium, MeaningEN: "while", MeaningID: "selagi"}
)

func TestTargetsPairEveryItemWithEachTypeItQualifiesFor(t *testing.T) {
	kanji := targets(LangID, Items{Kanji: []dataset.Kanji{sakiKanji, bareKanji}}, []Type{TypeMeaning, TypeReading})
	saki := Facts{Level: "n5", Character: "先", Onyomi: []string{"セン"}, Kunyomi: []string{"さき"}, Examples: []string{"先生", "先月"}, Meaning: "sebelumnya, depan"}
	want := []Target{
		{Code: "k101-001", Type: TypeMeaning, Facts: saki, Prompt: "先", Answer: "sebelumnya, depan"},
		{Code: "k101-001", Type: TypeReading, Facts: saki, Examples: []string{"先生", "先月"}, Kana: true},
		{Code: "k101-002", Type: TypeMeaning, Facts: Facts{Level: "n5", Character: "号", Onyomi: []string{"ゴウ"}, Examples: []string{}, Meaning: "nomor"}, Prompt: "号", Answer: "nomor"},
	}
	if !reflect.DeepEqual(kanji, want) {
		t.Errorf("kanji:\ngot  %+v\nwant %+v", kanji, want)
	}

	vocabulary := targets(LangEN, Items{Vocabulary: []dataset.Vocabulary{amaiWord, kazeWord, kareWord, chuuWord}}, []Type{TypeMeaning, TypeReading, TypeUsage})
	kaze := Facts{Level: "n4", Word: "風邪を引く", Reading: "かぜをひく", PartOfSpeech: "phrase", Note: "風邪を引いた。", Meaning: "catch a cold"}
	wantPairs := []string{
		"v101-001 meaning", "v101-001 usage",
		"v201-001 meaning", "v201-001 reading", "v201-001 usage",
		"v301-001 meaning",
		"v201-002 meaning",
	}
	if got := pairs(vocabulary); !slices.Equal(got, wantPairs) {
		t.Errorf("vocabulary pairs:\ngot  %v\nwant %v", got, wantPairs)
	}
	wantKaze := []Target{
		{Code: "v201-001", Type: TypeMeaning, Facts: kaze, Prompt: "風邪を引く", Answer: "catch a cold"},
		{Code: "v201-001", Type: TypeReading, Facts: kaze, Prompt: "風邪を引く", Answer: "かぜをひく", Kana: true},
		{Code: "v201-001", Type: TypeUsage, Facts: kaze, Answer: "風邪を引く", Avoid: "風邪を引く", Blank: true},
	}
	if !reflect.DeepEqual(vocabulary[2:5], wantKaze) {
		t.Errorf("風邪を引く:\ngot  %+v\nwant %+v", vocabulary[2:5], wantKaze)
	}

	grammar := targets(LangEN, Items{Grammar: []dataset.Grammar{aidaGrammr}}, []Type{TypeUsage})
	wantGrammar := []Target{{Code: "g201-a", Type: TypeUsage, Blank: true,
		Facts: Facts{Level: "n4", Pattern: "〜あいだに", Formula: "Vる＋あいだに", Example: "寝ているあいだに。", Difficulty: "medium", Meaning: "while"}}}
	if !reflect.DeepEqual(grammar, wantGrammar) {
		t.Errorf("grammar usage:\ngot  %+v\nwant %+v", grammar, wantGrammar)
	}
}

func TestSampleTakesDistinctTargetsUpToCount(t *testing.T) {
	all := targets(LangEN, Items{Vocabulary: []dataset.Vocabulary{amaiWord, kazeWord, kareWord, chuuWord}}, []Type{TypeMeaning, TypeReading, TypeUsage})

	got := pairs(sample(all, 3))
	if len(got) != 3 || len(slices.Compact(slices.Sorted(slices.Values(got)))) != 3 {
		t.Errorf("sample 3: got %v; want 3 distinct pairs", got)
	}
	for _, p := range got {
		if !slices.Contains(pairs(all), p) {
			t.Errorf("sample 3: %s is not a target", p)
		}
	}

	if got, want := slices.Sorted(slices.Values(pairs(sample(all, 10)))), slices.Sorted(slices.Values(pairs(all))); !slices.Equal(got, want) {
		t.Errorf("sample 10 of 7: got %v; want every target once", got)
	}
}

// pairs lists targets as "code type".
func pairs(targets []Target) []string {
	out := make([]string, len(targets))
	for i, t := range targets {
		out[i] = t.Code + " " + string(t.Type)
	}
	return out
}
