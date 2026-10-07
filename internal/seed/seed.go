package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// table is the full content one database table must hold after a seed.
type table struct {
	name    string
	columns []string // the first column is the primary key, id
	rows    [][]any
}

func (t table) ids() []string {
	ids := make([]string, len(t.rows))
	for i, row := range t.rows {
		ids[i] = row[0].(string)
	}
	return ids
}

// upsertSQL inserts a row or updates it in place; updated_at moves only when a value changed.
func (t table) upsertSQL() string {
	params := make([]string, len(t.columns))
	for i := range t.columns {
		params[i] = "$" + strconv.Itoa(i+1)
	}
	values := t.columns[1:]
	sets := make([]string, len(values))
	current := make([]string, len(values))
	incoming := make([]string, len(values))
	for i, c := range values {
		sets[i] = c + " = EXCLUDED." + c
		current[i] = t.name + "." + c
		incoming[i] = "EXCLUDED." + c
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (id) DO UPDATE SET %s, updated_at = now() WHERE (%s) IS DISTINCT FROM (%s)",
		t.name, strings.Join(t.columns, ", "), strings.Join(params, ", "),
		strings.Join(sets, ", "), strings.Join(current, ", "), strings.Join(incoming, ", "))
}

// Seed makes the database hold exactly the given datasets, in one transaction:
// rows are upserted by id and rows absent from the datasets are deleted.
func Seed(ctx context.Context, conn *pgx.Conn, datasets []dataset.Dataset) error {
	lessons, children, err := tables(datasets)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL DEFERRED"); err != nil {
			return err
		}
		for _, t := range children {
			if err := deleteAbsent(ctx, tx, t); err != nil {
				return err
			}
		}
		for _, t := range append([]table{lessons}, children...) {
			if err := upsert(ctx, tx, t); err != nil {
				return err
			}
		}
		return deleteAbsent(ctx, tx, lessons)
	})
}

func deleteAbsent(ctx context.Context, tx pgx.Tx, t table) error {
	if _, err := tx.Exec(ctx, "DELETE FROM "+t.name+" WHERE NOT (id = ANY($1::uuid[]))", t.ids()); err != nil {
		return fmt.Errorf("delete from %s: %w", t.name, err)
	}
	return nil
}

func upsert(ctx context.Context, tx pgx.Tx, t table) error {
	sql := t.upsertSQL()
	batch := &pgx.Batch{}
	for _, row := range t.rows {
		batch.Queue(sql, row...)
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("upsert into %s: %w", t.name, err)
	}
	return nil
}

func tables(datasets []dataset.Dataset) (table, []table, error) {
	lessons := table{name: "lesson", columns: []string{"id", "level", "section", "week", "day", "title", "title_en", "title_id", "week_title", "week_title_en", "week_title_id"}}
	kanji := table{name: "kanji", columns: []string{"id", "level", "code", "lesson_id", "sequence", "character", "onyomi", "kunyomi", "examples", "meaning_en", "meaning_id", "sources"}}
	vocabulary := table{name: "vocabulary", columns: []string{"id", "level", "code", "lesson_id", "sequence", "word", "reading", "part_of_speech", "note", "meaning_en", "meaning_id", "sources"}}
	grammar := table{name: "grammar", columns: []string{"id", "level", "code", "lesson_id", "sequence", "curriculum_code", "pattern", "reading", "formula", "note", "example", "difficulty", "meaning_en", "meaning_id", "sources"}}
	comparisons := table{name: "grammar_comparison", columns: []string{"id", "lesson_id", "sequence", "pattern_a", "pattern_b", "difference", "use_when", "example"}}
	mistakes := table{name: "grammar_mistake", columns: []string{"id", "lesson_id", "sequence", "point", "lines", "meaning_en", "meaning_id"}}
	expressions := table{name: "grammar_expression", columns: []string{"id", "lesson_id", "sequence", "expression", "reading", "meaning_en", "meaning_id", "note", "example"}}

	for _, ds := range datasets {
		for _, r := range ds.Lessons {
			lessons.rows = append(lessons.rows, []any{r.ID, r.Level, string(r.Section), r.Week, r.Day, r.Title, r.TitleEN, r.TitleID, r.WeekTitle, r.WeekTitleEN, r.WeekTitleID})
		}
		for _, r := range ds.Kanji {
			kanji.rows = append(kanji.rows, []any{r.ID, r.Level, r.Code, r.LessonID, r.Sequence, r.Character, r.Onyomi, r.Kunyomi, r.Examples, r.MeaningEN, r.MeaningID, sourceNames(r.Sources)})
		}
		for _, r := range ds.Vocabulary {
			vocabulary.rows = append(vocabulary.rows, []any{r.ID, r.Level, r.Code, r.LessonID, r.Sequence, r.Word, r.Reading, r.PartOfSpeech, r.Note, r.MeaningEN, r.MeaningID, sourceNames(r.Sources)})
		}
		for _, r := range ds.Grammar {
			grammar.rows = append(grammar.rows, []any{r.ID, r.Level, r.Code, r.LessonID, r.Sequence, r.CurriculumCode, r.Pattern, r.Reading, r.Formula, r.Note, r.Example, string(r.Difficulty), r.MeaningEN, r.MeaningID, sourceNames(r.Sources)})
		}
		for _, r := range ds.GrammarComparisons {
			comparisons.rows = append(comparisons.rows, []any{r.ID, r.LessonID, r.Sequence, r.PatternA, r.PatternB, r.Difference, r.UseWhen, r.Example})
		}
		for _, r := range ds.GrammarMistakes {
			lines, err := json.Marshal(r.Lines)
			if err != nil {
				return table{}, nil, err
			}
			mistakes.rows = append(mistakes.rows, []any{r.ID, r.LessonID, r.Sequence, r.Point, lines, r.MeaningEN, r.MeaningID})
		}
		for _, r := range ds.GrammarExpressions {
			expressions.rows = append(expressions.rows, []any{r.ID, r.LessonID, r.Sequence, r.Expression, r.Reading, r.MeaningEN, r.MeaningID, r.Note, r.Example})
		}
	}
	return lessons, []table{kanji, vocabulary, grammar, comparisons, mistakes, expressions}, nil
}

func sourceNames(sources []dataset.Source) []string {
	names := make([]string, len(sources))
	for i, s := range sources {
		names[i] = string(s)
	}
	return names
}
