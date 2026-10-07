package lesson

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeremi777/nihongo-foundation/internal/database"
	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// Each column list names the columns of one table that its dataset row type
// holds, so pgx.RowToStructByName can scan them.
const (
	lessonColumns     = "id, level, section, week, day, title, title_en, title_id, week_title, week_title_en, week_title_id"
	kanjiColumns      = "id, level, code, lesson_id, sequence, character, onyomi, kunyomi, examples, meaning_en, meaning_id, sources"
	vocabularyColumns = "id, level, code, lesson_id, sequence, word, reading, part_of_speech, note, meaning_en, meaning_id, sources"
	grammarColumns    = "id, level, code, lesson_id, sequence, curriculum_code, pattern, reading, formula, note, example, difficulty, meaning_en, meaning_id, sources"
	comparisonColumns = "id, lesson_id, sequence, pattern_a, pattern_b, difference, use_when, example"
	mistakeColumns    = "id, lesson_id, sequence, point, lines, meaning_en, meaning_id"
	expressionColumns = "id, lesson_id, sequence, expression, reading, meaning_en, meaning_id, note, example"
)

// Repository reads lessons and their rows.
type Repository struct{ db database.Querier }

// NewRepository reads through db.
func NewRepository(db database.Querier) Repository { return Repository{db: db} }

// List returns the lessons of a level in one section, by week and day.
func (r Repository) List(ctx context.Context, level string, section dataset.Section) ([]dataset.Lesson, error) {
	rows, err := r.db.Query(ctx, "SELECT "+lessonColumns+" FROM lesson WHERE level = $1 AND section = $2 ORDER BY week, day", level, section)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[dataset.Lesson])
}

// Get returns the lesson with the given id and the rows of its section, or
// ErrNotFound. It reads them in one snapshot, so a seed that commits midway
// never mixes rows from before and after it.
func (r Repository) Get(ctx context.Context, id string) (Detail, error) {
	var lessonID pgtype.UUID
	if err := lessonID.Scan(id); err != nil {
		return Detail{}, ErrNotFound
	}
	var d Detail
	snapshot := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	err := pgx.BeginTxFunc(ctx, r.db, snapshot, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, "SELECT "+lessonColumns+" FROM lesson WHERE id = $1", lessonID)
		if err != nil {
			return err
		}
		if d.Lesson, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[dataset.Lesson]); err != nil {
			return err
		}
		switch d.Lesson.Section {
		case dataset.SectionKanji:
			d.Kanji, err = rowsOf[dataset.Kanji](ctx, tx, "kanji", kanjiColumns, lessonID)
		case dataset.SectionVocabulary:
			d.Vocabulary, err = rowsOf[dataset.Vocabulary](ctx, tx, "vocabulary", vocabularyColumns, lessonID)
		case dataset.SectionGrammar:
			if d.Grammar, err = rowsOf[dataset.Grammar](ctx, tx, "grammar", grammarColumns, lessonID); err != nil {
				return err
			}
			if d.Comparisons, err = rowsOf[dataset.GrammarComparison](ctx, tx, "grammar_comparison", comparisonColumns, lessonID); err != nil {
				return err
			}
			if d.Mistakes, err = rowsOf[dataset.GrammarMistake](ctx, tx, "grammar_mistake", mistakeColumns, lessonID); err != nil {
				return err
			}
			d.Expressions, err = rowsOf[dataset.GrammarExpression](ctx, tx, "grammar_expression", expressionColumns, lessonID)
		}
		return err
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	return d, err
}

// rowsOf returns the rows of table that belong to the lesson, by sequence.
func rowsOf[T any](ctx context.Context, tx pgx.Tx, table, columns string, lessonID pgtype.UUID) ([]T, error) {
	rows, err := tx.Query(ctx, "SELECT "+columns+" FROM "+table+" WHERE lesson_id = $1 ORDER BY sequence", lessonID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}
