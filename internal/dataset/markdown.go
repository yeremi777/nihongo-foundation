package dataset

import (
	"fmt"
	"regexp"
	"strings"
)

// File is one source markdown file; Name is used in error messages.
type File struct {
	Name string
	Text string
}

// LevelFiles are the four source files of one level.
type LevelFiles struct {
	TOC        File
	Kanji      File
	Vocabulary File
	Grammar    File
}

type line struct {
	no   int
	text string
}

type tableRow struct {
	line  int
	cells []string
}

type table struct {
	header []string
	line   int
	rows   []tableRow
}

// cell is the row's value in the named column, or "" when the table has no such column.
func (t table) cell(row tableRow, column string) string {
	for i, name := range t.header {
		if name == column {
			return row.cells[i]
		}
	}
	return ""
}

func (t table) has(column string) bool {
	for _, name := range t.header {
		if name == column {
			return true
		}
	}
	return false
}

func (t table) is(header ...string) bool {
	if len(t.header) != len(header) {
		return false
	}
	for i := range header {
		if t.header[i] != header[i] {
			return false
		}
	}
	return true
}

var separatorCell = regexp.MustCompile(`^:?-+:?$`)

func linesOf(f File) []line {
	texts := strings.Split(strings.ReplaceAll(f.Text, "\r\n", "\n"), "\n")
	lines := make([]line, len(texts))
	for i, text := range texts {
		lines[i] = line{no: i + 1, text: strings.TrimRight(text, " \t")}
	}
	return lines
}

func isTableLine(text string) bool {
	return strings.HasPrefix(text, "|")
}

func splitRow(text string) []string {
	inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(text), "|"), "|")
	cells := strings.Split(inner, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		if !separatorCell.MatchString(c) {
			return false
		}
	}
	return true
}

// tables groups consecutive table lines into tables, starting at index i.
// It returns the table and the index of the first line after it.
func readTable(f File, lines []line, i int) (table, int, error) {
	t := table{header: splitRow(lines[i].text), line: lines[i].no}
	for i++; i < len(lines) && isTableLine(lines[i].text); i++ {
		cells := splitRow(lines[i].text)
		if isSeparator(cells) {
			continue
		}
		if len(cells) != len(t.header) {
			return table{}, i, errorAt(f, lines[i].no, "row has %d cells, its header has %d", len(cells), len(t.header))
		}
		t.rows = append(t.rows, tableRow{line: lines[i].no, cells: cells})
	}
	return t, i, nil
}

func errorAt(f File, no int, format string, args ...any) error {
	return fmt.Errorf("%s:%d: %s", f.Name, no, fmt.Sprintf(format, args...))
}

// splitEnglishIndonesian splits "English / Indonesian", which must hold exactly one " / ".
func splitEnglishIndonesian(s string) (en, id string, ok bool) {
	parts := strings.Split(s, " / ")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func required(f File, row tableRow, column, value string) (string, error) {
	if isEmptyCell(value) {
		return "", errorAt(f, row.line, "%s is empty", column)
	}
	return strings.TrimSpace(value), nil
}
