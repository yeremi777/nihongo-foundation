package dataset

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type weekDay struct{ week, day int }

type tocDay struct {
	code    string
	titleID string
	line    int
}

type tocGrammar struct {
	difficulty Difficulty
	sources    []Source
	line       int
}

// curriculumTOC is what the source lists take from the curriculum TOC.
type curriculumTOC struct {
	days         map[Section]map[weekDay]tocDay
	weekTitleIDs map[Section]map[int]string
	grammar      map[string]tocGrammar
}

var (
	tocWeekHeading = regexp.MustCompile(`^#{3,4} Week (\d+) — .+? \*\((.+)\)\*$`)
	tocDayCode     = map[Section]*regexp.Regexp{
		SectionKanji:      regexp.MustCompile(`^k(\d)(\d{2})$`),
		SectionVocabulary: regexp.MustCompile(`^v(\d)(\d{2})$`),
	}
	tocGrammarCode = regexp.MustCompile(`^g\d+$`)
)

// sourceLetters decodes a TOC Src letter; the same letter names different books per level.
var sourceLetters = map[string]map[string][]Source{
	"n5": {"S": {SourceSoumatome}, "M": {SourceMinnaNoNihongo}, "B": {SourceSoumatome, SourceMinnaNoNihongo}},
	"n4": {"S": {SourceSoumatome}, "M": {SourceMinnaNoNihongo}, "B": {SourceSoumatome, SourceMinnaNoNihongo}},
	"n3": {"S": {SourceSoumatome}, "K": {SourceShinkanzen}, "B": {SourceSoumatome, SourceShinkanzen}},
}

var difficulties = []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard}

func parseTOC(level string, f File) (curriculumTOC, error) {
	toc := curriculumTOC{
		days:         map[Section]map[weekDay]tocDay{SectionKanji: {}, SectionVocabulary: {}},
		weekTitleIDs: map[Section]map[int]string{SectionKanji: {}, SectionVocabulary: {}},
		grammar:      map[string]tocGrammar{},
	}
	var section Section
	lines := linesOf(f)
	for i := 0; i < len(lines); {
		ln := lines[i]
		switch {
		case strings.HasPrefix(ln.text, "## "):
			section = tocSection(ln.text)
			i++
		case section != SectionGrammar && section != "" && tocWeekHeading.MatchString(ln.text):
			m := tocWeekHeading.FindStringSubmatch(ln.text)
			_, id, ok := splitEnglishIndonesian(m[2])
			if !ok {
				return toc, errorAt(f, ln.no, `week heading must hold exactly one " / " between English and Indonesian`)
			}
			week, _ := strconv.Atoi(m[1])
			toc.weekTitleIDs[section][week] = id
			i++
		case isTableLine(ln.text) && section != "":
			t, next, err := readTable(f, lines, i)
			if err != nil {
				return toc, err
			}
			if err := toc.addTable(level, section, f, t); err != nil {
				return toc, err
			}
			i = next
		default:
			i++
		}
	}
	return toc, nil
}

func tocSection(heading string) Section {
	switch {
	case strings.Contains(heading, "GRAMMAR"):
		return SectionGrammar
	case strings.Contains(heading, "VOCABULARY"):
		return SectionVocabulary
	case strings.Contains(heading, "KANJI"):
		return SectionKanji
	}
	return ""
}

func (toc curriculumTOC) addTable(level string, section Section, f File, t table) error {
	for _, row := range t.rows {
		code := row.cells[0]
		if section == SectionGrammar {
			if !tocGrammarCode.MatchString(code) {
				continue
			}
			if err := toc.addGrammar(level, f, t, row, code); err != nil {
				return err
			}
			continue
		}
		m := tocDayCode[section].FindStringSubmatch(code)
		if m == nil {
			continue
		}
		if err := toc.addDay(section, f, t, row, code, m); err != nil {
			return err
		}
	}
	return nil
}

func (toc curriculumTOC) addDay(section Section, f File, t table, row tableRow, code string, m []string) error {
	week, _ := strconv.Atoi(m[1])
	day, _ := strconv.Atoi(m[2])
	if t.cell(row, "Day") != strconv.Itoa(day) {
		return errorAt(f, row.line, "%s is day %d by its code but the Day column says %q", code, day, t.cell(row, "Day"))
	}
	if t.has("Src") && t.cell(row, "Src") != "S" {
		return errorAt(f, row.line, "source letter %q: %s days come from Soumatome only", t.cell(row, "Src"), section)
	}
	titleID, err := required(f, row, "Indonesian", t.cell(row, "Indonesian"))
	if err != nil {
		return err
	}
	key := weekDay{week, day}
	if prev, ok := toc.days[section][key]; ok {
		return errorAt(f, row.line, "%s repeats week %d day %d of %s", code, week, day, prev.code)
	}
	toc.days[section][key] = tocDay{code: code, titleID: titleID, line: row.line}
	return nil
}

func (toc curriculumTOC) addGrammar(level string, f File, t table, row tableRow, code string) error {
	if _, ok := toc.grammar[code]; ok {
		return errorAt(f, row.line, "curriculum code %s repeats", code)
	}
	difficulty := Difficulty(t.cell(row, "Diff"))
	if !slices.Contains(difficulties, difficulty) {
		return errorAt(f, row.line, "difficulty %q is not easy, medium, or hard", difficulty)
	}
	letter := t.cell(row, "Src")
	sources, ok := sourceLetters[level][letter]
	if !ok {
		return errorAt(f, row.line, "source letter %q is not defined for %s", letter, level)
	}
	toc.grammar[code] = tocGrammar{difficulty: difficulty, sources: slices.Clone(sources), line: row.line}
	return nil
}
