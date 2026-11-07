// Package item serves the items of each section: searched with filters and
// pages, or looked up by level and code.
package item

import (
	"errors"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// ErrNotFound reports a level and code that name no item of the section.
var ErrNotFound = errors.New("item not found")

// levels, sources, and difficulties are the values a Query accepts; levels
// is in list order.
var (
	levels       = dataset.Levels()
	sources      = []dataset.Source{dataset.SourceSoumatome, dataset.SourceShinkanzen, dataset.SourceMinnaNoNihongo}
	difficulties = dataset.Difficulties()
)

// Query is one search of a section; a zero filter field matches every item.
type Query struct {
	Level      string
	Week       int
	Source     dataset.Source
	Difficulty dataset.Difficulty
	Text       string
	Page       int
	Size       int
}

// Page is one page of a search, with the count of every match.
type Page[T any] struct {
	Items []T `json:"items"`
	Page  int `json:"page"`
	Size  int `json:"size"`
	Total int `json:"total"`
}

// TextKanji is one distinct kanji of a text, with its item; Item is nil when
// no level holds it.
type TextKanji struct {
	Character string         `json:"character"`
	Count     int            `json:"count"`
	Item      *dataset.Kanji `json:"item"`
}

// LevelCoverage counts the occurrences of kanji in a text known at a level or
// below it.
type LevelCoverage struct {
	Level        string `json:"level"`
	Known        int    `json:"known"`
	KnownPercent int    `json:"known_percent"`
}

// Coverage is the kanji coverage of a text.
type Coverage struct {
	Total  int             `json:"total"`
	Levels []LevelCoverage `json:"levels"`
	Kanji  []TextKanji     `json:"kanji"`
}

// section describes one item table: its row columns, the text and text[]
// columns a search matches, and whether it is filtered by difficulty.
type section[T any] struct {
	name       dataset.Section
	columns    string
	texts      []string
	arrays     []string
	difficulty bool
}

var (
	kanji = section[dataset.Kanji]{
		name: dataset.SectionKanji, columns: dataset.KanjiColumns,
		texts: []string{"character", "meaning_en", "meaning_id"}, arrays: []string{"onyomi", "kunyomi", "examples"},
	}
	vocabulary = section[dataset.Vocabulary]{
		name: dataset.SectionVocabulary, columns: dataset.VocabularyColumns,
		texts: []string{"word", "reading", "meaning_en", "meaning_id"},
	}
	grammar = section[dataset.Grammar]{
		name: dataset.SectionGrammar, columns: dataset.GrammarColumns,
		texts: []string{"pattern", "reading", "meaning_en", "meaning_id"}, difficulty: true,
	}
)
