// Package level serves the levels that have lessons, with the size of their sections.
package level

import "github.com/yeremi777/nihongo-foundation/internal/dataset"

// levels and sections are the values a Level lists, in list order.
var (
	levels   = dataset.Levels()
	sections = []dataset.Section{dataset.SectionKanji, dataset.SectionVocabulary, dataset.SectionGrammar}
)

// Section counts the weeks, lessons, and items of one section in a level.
type Section struct {
	Section dataset.Section `json:"section"`
	Weeks   int             `json:"weeks"`
	Lessons int             `json:"lessons"`
	Items   int             `json:"items"`
}

// Level is a level with the sections that have lessons.
type Level struct {
	Level    string    `json:"level"`
	Sections []Section `json:"sections"`
}
