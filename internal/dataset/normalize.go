package dataset

import (
	"regexp"
	"strings"
)

var (
	partOfSpeechSeparator = regexp.MustCompile(`\s*[／/;]\s*`)
	suruVerb              = regexp.MustCompile(`\bsuru-verb\b`)
	iAdjective            = regexp.MustCompile(`\bi-adj(ective)?\b`)
	naAdjective           = regexp.MustCompile(`\bna-adj(ective)?\b`)
	listSeparator         = regexp.MustCompile(`[、,]`)
)

// normalizePartOfSpeech gives one spelling to each part-of-speech term while
// keeping the text a display value.
func normalizePartOfSpeech(s string) string {
	s = partOfSpeechSeparator.ReplaceAllString(strings.TrimSpace(s), " / ")
	s = strings.Map(lowerASCII, s)
	s = suruVerb.ReplaceAllString(s, "する-verb")
	s = iAdjective.ReplaceAllString(s, "い-adjective")
	return naAdjective.ReplaceAllString(s, "な-adjective")
}

func lowerASCII(r rune) rune {
	if 'A' <= r && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

// splitList splits a reading or example cell; "—" or an empty cell is an empty list.
func splitList(cell string) []string {
	items := []string{}
	if isEmptyCell(cell) {
		return items
	}
	for _, item := range listSeparator.Split(cell, -1) {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

// optional is nil for an empty or "—" cell, and the trimmed text otherwise.
func optional(cell string) *string {
	if isEmptyCell(cell) {
		return nil
	}
	s := strings.TrimSpace(cell)
	return &s
}

func isEmptyCell(cell string) bool {
	s := strings.TrimSpace(cell)
	return s == "" || s == "—"
}
