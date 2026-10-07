// Package lesson serves the curriculum: the lessons of a level in one
// section, and one lesson with its items.
package lesson

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// ErrNotFound reports a lesson id that names no lesson.
var ErrNotFound = errors.New("lesson not found")

// Detail is one lesson with the rows of its section, each in sequence order.
// Only the fields of the lesson's section are filled; a grammar lesson fills
// Grammar, Comparisons, Mistakes, and Expressions.
type Detail struct {
	Lesson      dataset.Lesson
	Kanji       []dataset.Kanji
	Vocabulary  []dataset.Vocabulary
	Grammar     []dataset.Grammar
	Comparisons []dataset.GrammarComparison
	Mistakes    []dataset.GrammarMistake
	Expressions []dataset.GrammarExpression
}

// MarshalJSON writes the lesson and the keys of its section only.
func (d Detail) MarshalJSON() ([]byte, error) {
	switch d.Lesson.Section {
	case dataset.SectionKanji:
		return json.Marshal(struct {
			Lesson dataset.Lesson  `json:"lesson"`
			Kanji  []dataset.Kanji `json:"kanji"`
		}{d.Lesson, d.Kanji})
	case dataset.SectionVocabulary:
		return json.Marshal(struct {
			Lesson     dataset.Lesson       `json:"lesson"`
			Vocabulary []dataset.Vocabulary `json:"vocabulary"`
		}{d.Lesson, d.Vocabulary})
	case dataset.SectionGrammar:
		return json.Marshal(struct {
			Lesson      dataset.Lesson              `json:"lesson"`
			Grammar     []dataset.Grammar           `json:"grammar"`
			Comparisons []dataset.GrammarComparison `json:"comparisons"`
			Mistakes    []dataset.GrammarMistake    `json:"mistakes"`
			Expressions []dataset.GrammarExpression `json:"expressions"`
		}{d.Lesson, d.Grammar, d.Comparisons, d.Mistakes, d.Expressions})
	}
	return nil, fmt.Errorf("lesson %s has unknown section %q", d.Lesson.ID, d.Lesson.Section)
}
