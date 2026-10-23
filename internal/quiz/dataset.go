package quiz

import (
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// datasetTypesOf holds the question types each section allows in a dataset quiz.
var datasetTypesOf = map[dataset.Section][]Type{
	dataset.SectionKanji:      {TypeMeaning, TypeReading},
	dataset.SectionVocabulary: {TypeMeaning, TypeReading},
	dataset.SectionGrammar:    {TypeMeaning},
}

// distractorCount is how many wrong choices a question holds.
const distractorCount = 3

// datasetDrafts writes a draft per target with distractors from the pool.
func datasetDrafts(section dataset.Section, pool Items, lang Lang, targets []Target) []Draft {
	drafts := make([]Draft, len(targets))
	for i, t := range targets {
		candidates := poolValues(section, pool, lang, t)
		rand.Shuffle(len(candidates), func(a, b int) { candidates[a], candidates[b] = candidates[b], candidates[a] })
		drafts[i] = Draft{Code: t.Code, Type: t.Type, Distractors: pick(t, candidates), Explanation: explanation(t)}
	}
	return drafts
}

// poolValues returns the meanings or readings of the pool's other items.
func poolValues(section dataset.Section, pool Items, lang Lang, t Target) []string {
	var values []string
	switch {
	case section == dataset.SectionKanji && t.Type == TypeMeaning:
		for _, k := range pool.Kanji {
			if k.Code != t.Code {
				values = append(values, meaning(lang, k.MeaningEN, k.MeaningID))
			}
		}
	case section == dataset.SectionKanji:
		for _, k := range pool.Kanji {
			if k.Code != t.Code {
				values = append(values, slices.Concat(k.Onyomi, k.Kunyomi)...)
			}
		}
	case section == dataset.SectionVocabulary && t.Type == TypeMeaning:
		for _, v := range pool.Vocabulary {
			if v.Code != t.Code {
				values = append(values, meaning(lang, v.MeaningEN, v.MeaningID))
			}
		}
	case section == dataset.SectionVocabulary:
		for _, v := range pool.Vocabulary {
			if v.Code != t.Code {
				values = append(values, v.Reading)
			}
		}
	default:
		for _, g := range pool.Grammar {
			if g.Code != t.Code {
				values = append(values, meaning(lang, g.MeaningEN, g.MeaningID))
			}
		}
	}
	return values
}

// pick returns the first candidates the answer allows, up to the distractor count.
func pick(t Target, candidates []string) []string {
	var picked []string
	for _, c := range candidates {
		switch {
		case len(picked) == distractorCount:
			return picked
		case c == t.Answer || slices.Contains(picked, c):
		case t.Type == TypeMeaning && sharesTerm(c, t.Answer):
		case t.Type == TypeReading && !isKana(c):
		case t.Type == TypeReading && (slices.Contains(t.Facts.Onyomi, c) || slices.Contains(t.Facts.Kunyomi, c)):
		default:
			picked = append(picked, c)
		}
	}
	return picked
}

// sharesTerm reports whether two meanings share a term.
func sharesTerm(a, b string) bool {
	return slices.ContainsFunc(terms(a), func(term string) bool { return slices.Contains(terms(b), term) })
}

func terms(meaning string) []string {
	parts := strings.FieldsFunc(strings.ToLower(meaning), func(r rune) bool { return r == ',' || r == '/' || r == ';' })
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// explanation states the answer for a target.
func explanation(t Target) string {
	text := t.Prompt + " · " + t.Answer
	if t.Type == TypeReading && t.Facts.Character != "" {
		text = t.Prompt
		if len(t.Facts.Onyomi) > 0 {
			text += " · 音 " + strings.Join(t.Facts.Onyomi, ", ")
		}
		if len(t.Facts.Kunyomi) > 0 {
			text += " · 訓 " + strings.Join(t.Facts.Kunyomi, ", ")
		}
	}
	return text
}
