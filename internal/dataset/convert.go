package dataset

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	kanjiHeader      = []string{"Kanji", "On", "Kun", "English", "Indonesian", "Examples"}
	vocabularyHeader = []string{"Item ID", "Word", "Reading", "Part of speech", "English", "Indonesian", "Usage/Notes"}
	grammarHeader    = []string{"Item ID", "Pattern", "Reading", "English", "Indonesian", "Formula", "Usage/Notes", "Example"}
	comparisonHeader = []string{"Pattern A", "Pattern B", "Main Difference", "Use This When", "Example"}
	expressionHeader = []string{"Expression", "Reading", "English", "Indonesian", "Usage/Notes", "Example"}

	// skippedGrammarTables are worked examples of how one pattern forms, not study items.
	skippedGrammarTables = [][]string{
		{"Base Noun", "Explanation", "Modified Noun", "English", "Indonesian"},
		{"Verb Type", "Rule", "Example", "English", "Indonesian"},
	}

	grammarCode  = regexp.MustCompile(`^(g\d+)(-[a-z])?$`)
	mistakeStart = `<div class="grammar-mistake-card">`
	mistakeEnd   = `</div>`
	mistakeField = regexp.MustCompile(`^<p><strong>(.+?):</strong>\s*(.*?)</p>$`)
	verdictMarks = map[string]Verdict{"❌": VerdictIncorrect, "✅": VerdictCorrect}
)

// converter builds one level's dataset and enforces uniqueness across its sections.
type converter struct {
	level      string
	toc        curriculumTOC
	tocFile    File
	ds         Dataset
	codes      map[string]bool
	characters map[string]bool
}

// ConvertLevel builds one level's dataset from its source lists, checked against its curriculum TOC.
func ConvertLevel(level string, files LevelFiles) (Dataset, error) {
	if _, ok := sourceLetters[level]; !ok {
		return Dataset{}, fmt.Errorf("level %q has no source-letter table", level)
	}
	toc, err := parseTOC(level, files.TOC)
	if err != nil {
		return Dataset{}, err
	}
	c := converter{
		level: level, toc: toc, tocFile: files.TOC,
		ds:    Dataset{Level: level, Kanji: []Kanji{}, Vocabulary: []Vocabulary{}, Grammar: []Grammar{}, GrammarComparisons: []GrammarComparison{}, GrammarMistakes: []GrammarMistake{}, GrammarExpressions: []GrammarExpression{}},
		codes: map[string]bool{}, characters: map[string]bool{},
	}
	for _, step := range []struct {
		section Section
		file    File
		convert func(File, listLesson, Lesson) error
	}{
		{SectionKanji, files.Kanji, c.kanjiRows},
		{SectionVocabulary, files.Vocabulary, c.vocabularyRows},
		{SectionGrammar, files.Grammar, c.grammarRows},
	} {
		if err := c.convertList(step.section, step.file, step.convert); err != nil {
			return Dataset{}, err
		}
	}
	return c.ds, nil
}

func (c *converter) convertList(section Section, f File, rows func(File, listLesson, Lesson) error) error {
	lessons, err := parseList(f)
	if err != nil {
		return err
	}
	seen := map[weekDay]bool{}
	for _, ll := range lessons {
		lesson, err := c.lesson(section, f, ll)
		if err != nil {
			return err
		}
		seen[weekDay{ll.week.number, ll.day}] = true
		c.ds.Lessons = append(c.ds.Lessons, lesson)
		if err := rows(f, ll, lesson); err != nil {
			return err
		}
	}
	if section == SectionGrammar {
		return nil
	}
	return c.checkEveryTOCDayHasLesson(section, seen)
}

func (c *converter) checkEveryTOCDayHasLesson(section Section, seen map[weekDay]bool) error {
	days := c.toc.days[section]
	keys := make([]weekDay, 0, len(days))
	for k := range days {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b weekDay) int { return days[a].line - days[b].line })
	for _, k := range keys {
		if !seen[k] {
			return errorAt(c.tocFile, days[k].line, "%s day %s has no lesson in the %s list", section, days[k].code, section)
		}
	}
	return nil
}

func (c *converter) lesson(section Section, f File, ll listLesson) (Lesson, error) {
	lesson := Lesson{
		ID: lessonID(c.level, section, ll.week.number, ll.day), Level: c.level, Section: section,
		Week: ll.week.number, Day: ll.day, Title: ll.title, WeekTitle: ll.week.title,
	}
	if section == SectionGrammar {
		en, id, ok := splitEnglishIndonesian(ll.paren)
		if !ok {
			return lesson, errorAt(f, ll.line, `day heading must hold exactly one " / " between English and Indonesian`)
		}
		weekEN, weekID, ok := splitEnglishIndonesian(ll.week.paren)
		if !ok {
			return lesson, errorAt(f, ll.week.line, `week heading must hold exactly one " / " between English and Indonesian`)
		}
		lesson.TitleEN, lesson.TitleID, lesson.WeekTitleEN, lesson.WeekTitleID = en, id, weekEN, &weekID
		return lesson, nil
	}
	day, ok := c.toc.days[section][weekDay{ll.week.number, ll.day}]
	if !ok {
		return lesson, errorAt(f, ll.line, "week %d day %d has no curriculum TOC day", ll.week.number, ll.day)
	}
	lesson.TitleEN, lesson.TitleID, lesson.WeekTitleEN = ll.paren, day.titleID, ll.week.paren
	if id, ok := c.toc.weekTitleIDs[section][ll.week.number]; ok {
		lesson.WeekTitleID = &id
	}
	return lesson, nil
}

func (c *converter) claimCode(f File, row tableRow, code string) error {
	if c.codes[code] {
		return errorAt(f, row.line, "code %s repeats within %s", code, c.level)
	}
	c.codes[code] = true
	return nil
}

func (c *converter) kanjiRows(f File, ll listLesson, lesson Lesson) error {
	dayCode := c.toc.days[SectionKanji][weekDay{ll.week.number, ll.day}].code
	seq := 0
	for _, t := range ll.tables {
		if !t.is(kanjiHeader...) {
			return errorAt(f, t.line, "unexpected table in a kanji lesson")
		}
		for _, row := range t.rows {
			seq++
			character, err := required(f, row, "Kanji", t.cell(row, "Kanji"))
			if err != nil {
				return err
			}
			if utf8.RuneCountInString(character) != 1 {
				return errorAt(f, row.line, "kanji %q is not one character", character)
			}
			if c.characters[character] {
				return errorAt(f, row.line, "kanji %s repeats within %s", character, c.level)
			}
			c.characters[character] = true
			en, err := required(f, row, "English", t.cell(row, "English"))
			if err != nil {
				return err
			}
			id, err := required(f, row, "Indonesian", t.cell(row, "Indonesian"))
			if err != nil {
				return err
			}
			code := fmt.Sprintf("%s-%03d", dayCode, seq)
			if err := c.claimCode(f, row, code); err != nil {
				return err
			}
			c.ds.Kanji = append(c.ds.Kanji, Kanji{
				ID: rowID(c.level + ":kanji:" + code), Level: c.level, Code: code, LessonID: lesson.ID, Sequence: seq,
				Character: character, Onyomi: splitList(t.cell(row, "On")), Kunyomi: splitList(t.cell(row, "Kun")),
				Examples: splitList(t.cell(row, "Examples")), MeaningEN: en, MeaningID: id,
				Sources: []Source{SourceSoumatome},
			})
		}
	}
	return nil
}

func (c *converter) vocabularyRows(f File, ll listLesson, lesson Lesson) error {
	dayCode := c.toc.days[SectionVocabulary][weekDay{ll.week.number, ll.day}].code
	seq := 0
	for _, t := range ll.tables {
		if !t.is(vocabularyHeader...) {
			return errorAt(f, t.line, "unexpected table in a vocabulary lesson")
		}
		for _, row := range t.rows {
			seq++
			fields, err := requiredCells(f, t, row, "Item ID", "Word", "Reading", "Part of speech", "English", "Indonesian")
			if err != nil {
				return err
			}
			code := fields[0]
			if !strings.HasPrefix(code, dayCode+"-") {
				return errorAt(f, row.line, "code %s does not belong to lesson %s", code, dayCode)
			}
			if err := c.claimCode(f, row, code); err != nil {
				return err
			}
			c.ds.Vocabulary = append(c.ds.Vocabulary, Vocabulary{
				ID: rowID(c.level + ":vocabulary:" + code), Level: c.level, Code: code, LessonID: lesson.ID, Sequence: seq,
				Word: fields[1], Reading: fields[2], PartOfSpeech: normalizePartOfSpeech(fields[3]),
				Note: optional(t.cell(row, "Usage/Notes")), MeaningEN: fields[4], MeaningID: fields[5],
				Sources: []Source{SourceSoumatome},
			})
		}
	}
	return nil
}

func (c *converter) grammarRows(f File, ll listLesson, lesson Lesson) error {
	seq, comparisonSeq, expressionSeq := 0, 0, 0
	for _, t := range ll.tables {
		switch {
		case t.is(grammarHeader...):
			for _, row := range t.rows {
				seq++
				g, err := c.grammarRow(f, t, row, lesson, seq)
				if err != nil {
					return err
				}
				c.ds.Grammar = append(c.ds.Grammar, g)
			}
		case t.is(comparisonHeader...):
			for _, row := range t.rows {
				comparisonSeq++
				fields, err := requiredCells(f, t, row, comparisonHeader...)
				if err != nil {
					return err
				}
				c.ds.GrammarComparisons = append(c.ds.GrammarComparisons, GrammarComparison{
					ID: c.lessonRowID("grammar_comparison", lesson, comparisonSeq), LessonID: lesson.ID, Sequence: comparisonSeq,
					PatternA: fields[0], PatternB: fields[1], Difference: fields[2], UseWhen: fields[3], Example: fields[4],
				})
			}
		case t.is(expressionHeader...):
			for _, row := range t.rows {
				expressionSeq++
				fields, err := requiredCells(f, t, row, "Expression", "English", "Indonesian")
				if err != nil {
					return err
				}
				c.ds.GrammarExpressions = append(c.ds.GrammarExpressions, GrammarExpression{
					ID: c.lessonRowID("grammar_expression", lesson, expressionSeq), LessonID: lesson.ID, Sequence: expressionSeq,
					Expression: fields[0], Reading: optional(t.cell(row, "Reading")), MeaningEN: fields[1], MeaningID: fields[2],
					Note: optional(t.cell(row, "Usage/Notes")), Example: optional(t.cell(row, "Example")),
				})
			}
		case slices.ContainsFunc(skippedGrammarTables, func(h []string) bool { return t.is(h...) }):
		default:
			return errorAt(f, t.line, "unexpected table in a grammar lesson")
		}
	}
	return c.mistakeCards(f, ll, lesson)
}

func (c *converter) grammarRow(f File, t table, row tableRow, lesson Lesson, seq int) (Grammar, error) {
	fields, err := requiredCells(f, t, row, "Item ID", "Pattern", "English", "Indonesian")
	if err != nil {
		return Grammar{}, err
	}
	code := fields[0]
	m := grammarCode.FindStringSubmatch(code)
	if m == nil {
		return Grammar{}, errorAt(f, row.line, "code %q is not g<number> or g<number>-<letter>", code)
	}
	curriculum, ok := c.toc.grammar[m[1]]
	if !ok {
		return Grammar{}, errorAt(f, row.line, "curriculum code %s is not in the curriculum TOC", m[1])
	}
	if err := c.claimCode(f, row, code); err != nil {
		return Grammar{}, err
	}
	return Grammar{
		ID: rowID(c.level + ":grammar:" + code), Level: c.level, Code: code, LessonID: lesson.ID, Sequence: seq,
		CurriculumCode: m[1], Pattern: fields[1], Reading: optional(t.cell(row, "Reading")),
		Formula: optional(t.cell(row, "Formula")), Note: optional(t.cell(row, "Usage/Notes")),
		Example: optional(t.cell(row, "Example")), Difficulty: curriculum.difficulty,
		MeaningEN: fields[2], MeaningID: fields[3], Sources: slices.Clone(curriculum.sources),
	}, nil
}

func (c *converter) mistakeCards(f File, ll listLesson, lesson Lesson) error {
	var (
		card  *GrammarMistake
		start int
		seq   int
	)
	for _, ln := range ll.body {
		text := strings.TrimSpace(ln.text)
		switch {
		case text == mistakeStart:
			card, start = &GrammarMistake{LessonID: lesson.ID, Lines: []MistakeLine{}}, ln.no
		case card != nil && text == mistakeEnd:
			if err := checkMistakeCard(f, start, card); err != nil {
				return err
			}
			seq++
			card.ID, card.Sequence = c.lessonRowID("grammar_mistake", lesson, seq), seq
			c.ds.GrammarMistakes = append(c.ds.GrammarMistakes, *card)
			card = nil
		case card != nil:
			if err := addMistakeField(f, ln, card); err != nil {
				return err
			}
		}
	}
	if card != nil {
		return errorAt(f, start, "mistake card is not closed")
	}
	return nil
}

// addMistakeField reads one "<p><strong>Label:</strong> text</p>" line of a mistake card.
// A label starting with ❌ or ✅ is a judged sentence; the rest of the label is kept as written.
func addMistakeField(f File, ln line, card *GrammarMistake) error {
	m := mistakeField.FindStringSubmatch(strings.TrimSpace(ln.text))
	if m == nil {
		return errorAt(f, ln.no, "unexpected line in a mistake card")
	}
	label, value := m[1], strings.TrimSpace(m[2])
	if value == "" {
		return errorAt(f, ln.no, "mistake card %s is empty", label)
	}
	mark, rest, _ := strings.Cut(label, " ")
	if verdict, ok := verdictMarks[mark]; ok && rest != "" {
		card.Lines = append(card.Lines, MistakeLine{Verdict: verdict, Label: rest, Sentence: value})
		return nil
	}
	var field *string
	switch label {
	case "Point":
		field = &card.Point
	case "ID":
		field = &card.MeaningID
	case "EN":
		if card.MeaningEN != nil {
			return errorAt(f, ln.no, "mistake card has a second EN")
		}
		card.MeaningEN = &value
		return nil
	default:
		return errorAt(f, ln.no, "unknown mistake card label %q", label)
	}
	if *field != "" {
		return errorAt(f, ln.no, "mistake card has a second %s", label)
	}
	*field = value
	return nil
}

func checkMistakeCard(f File, start int, card *GrammarMistake) error {
	switch {
	case card.Point == "":
		return errorAt(f, start, "mistake card has no Point")
	case card.MeaningID == "":
		return errorAt(f, start, "mistake card has no ID")
	case !slices.ContainsFunc(card.Lines, func(l MistakeLine) bool { return l.Verdict == VerdictCorrect }):
		return errorAt(f, start, "mistake card has no ✅ sentence")
	}
	return nil
}

func (c *converter) lessonRowID(table string, lesson Lesson, seq int) string {
	return rowID(fmt.Sprintf("%s:%s:%d:%d:%s", c.level, table, lesson.Week, lesson.Day, strconv.Itoa(seq)))
}

func requiredCells(f File, t table, row tableRow, columns ...string) ([]string, error) {
	values := make([]string, len(columns))
	for i, column := range columns {
		v, err := required(f, row, column, t.cell(row, column))
		if err != nil {
			return nil, err
		}
		values[i] = v
	}
	return values, nil
}
