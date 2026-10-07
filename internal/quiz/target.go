// Package quiz builds quizzes: AI-written questions, each anchored to one
// dataset item and checked against it.
package quiz

import (
	"math/rand/v2"
	"slices"
	"strings"
	"unicode"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// Type is a question type: what a question asks of its item.
type Type string

const (
	TypeMeaning Type = "meaning"
	TypeReading Type = "reading"
	TypeUsage   Type = "usage"
)

// typesOf holds the question types each section allows; a section missing from it is invalid.
var typesOf = map[dataset.Section][]Type{
	dataset.SectionKanji:      {TypeMeaning, TypeReading},
	dataset.SectionVocabulary: {TypeMeaning, TypeReading, TypeUsage},
	dataset.SectionGrammar:    {TypeMeaning, TypeUsage},
}

// Lang is the language of meanings and explanations.
type Lang string

const (
	LangEN Lang = "en"
	LangID Lang = "id"
)

// Items are the rows of one section in a quiz's scope.
type Items struct {
	Kanji      []dataset.Kanji
	Vocabulary []dataset.Vocabulary
	Grammar    []dataset.Grammar
}

// Facts are what the provider is told about a target's item.
type Facts struct {
	Level        string   `json:"level"`
	Character    string   `json:"character,omitempty"`
	Onyomi       []string `json:"onyomi,omitempty"`
	Kunyomi      []string `json:"kunyomi,omitempty"`
	Examples     []string `json:"examples,omitempty"`
	Word         string   `json:"word,omitempty"`
	Pattern      string   `json:"pattern,omitempty"`
	Reading      string   `json:"reading,omitempty"`
	PartOfSpeech string   `json:"part_of_speech,omitempty"`
	Formula      string   `json:"formula,omitempty"`
	Note         string   `json:"note,omitempty"`
	Example      string   `json:"example,omitempty"`
	Difficulty   string   `json:"difficulty,omitempty"`
	Meaning      string   `json:"meaning"`
}

// Target is one item and question type a question is asked for, with the
// rules that question is held to. Prompt and Answer are set when the row
// holds them; the model writes them otherwise.
type Target struct {
	Code   string
	Type   Type
	Facts  Facts
	Prompt string
	Answer string
	// Examples are the only prompts allowed, when set.
	Examples []string
	// Avoid is text the prompt must not contain.
	Avoid string
	// Kana requires kana-only choices.
	Kana bool
	// Blank requires one ＿＿ in the prompt and Japanese choices.
	Blank bool
}

// targets pairs every item with each of the types it qualifies for, item by
// item, in the order of types.
func targets(lang Lang, items Items, types []Type) []Target {
	var out []Target
	add := func(t Target) {
		if slices.Contains(types, t.Type) {
			out = append(out, t)
		}
	}
	for _, k := range items.Kanji {
		facts := Facts{Level: k.Level, Character: k.Character, Onyomi: k.Onyomi, Kunyomi: k.Kunyomi, Examples: k.Examples,
			Meaning: meaning(lang, k.MeaningEN, k.MeaningID)}
		add(Target{Code: k.Code, Type: TypeMeaning, Facts: facts, Prompt: k.Character, Answer: facts.Meaning})
		if len(k.Examples) > 0 {
			add(Target{Code: k.Code, Type: TypeReading, Facts: facts, Examples: k.Examples, Kana: true})
		}
	}
	for _, v := range items.Vocabulary {
		facts := Facts{Level: v.Level, Word: v.Word, Reading: v.Reading, PartOfSpeech: v.PartOfSpeech, Note: deref(v.Note),
			Meaning: meaning(lang, v.MeaningEN, v.MeaningID)}
		add(Target{Code: v.Code, Type: TypeMeaning, Facts: facts, Prompt: v.Word, Answer: facts.Meaning})
		if hasKanji(v.Word) && isKana(v.Reading) {
			add(Target{Code: v.Code, Type: TypeReading, Facts: facts, Prompt: v.Word, Answer: v.Reading, Kana: true})
		}
		if hasJapanese(v.Word) && !strings.ContainsAny(v.Word, "／（(〜") {
			add(Target{Code: v.Code, Type: TypeUsage, Facts: facts, Answer: v.Word, Avoid: v.Word, Blank: true})
		}
	}
	for _, g := range items.Grammar {
		facts := Facts{Level: g.Level, Pattern: g.Pattern, Reading: deref(g.Reading), Formula: deref(g.Formula), Note: deref(g.Note),
			Example: deref(g.Example), Difficulty: string(g.Difficulty), Meaning: meaning(lang, g.MeaningEN, g.MeaningID)}
		add(Target{Code: g.Code, Type: TypeMeaning, Facts: facts, Prompt: g.Pattern, Answer: facts.Meaning})
		add(Target{Code: g.Code, Type: TypeUsage, Facts: facts, Blank: true})
	}
	return out
}

// sample returns up to count of the targets, drawn uniformly at random
// without repeats.
func sample(targets []Target, count int) []Target {
	shuffled := slices.Clone(targets)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return shuffled[:min(count, len(shuffled))]
}

func meaning(lang Lang, en, id string) string {
	if lang == LangID {
		return id
	}
	return en
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func hasKanji(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return unicode.Is(unicode.Han, r) })
}

// isKana reports a non-empty string of hiragana, katakana, and ー only.
func isKana(s string) bool {
	return s != "" && !strings.ContainsFunc(s, func(r rune) bool {
		return !unicode.In(r, unicode.Hiragana, unicode.Katakana) && r != 'ー'
	})
}

// hasJapanese reports a string holding kana or kanji.
func hasJapanese(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return unicode.In(r, unicode.Hiragana, unicode.Katakana, unicode.Han) })
}
