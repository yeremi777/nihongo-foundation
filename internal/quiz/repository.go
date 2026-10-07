package quiz

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/yeremi777/nihongo-foundation/internal/database"
	"github.com/yeremi777/nihongo-foundation/internal/dataset"
)

// Each column list names the columns its dataset row type holds, so
// pgx.RowToStructByName can scan them.
const (
	kanjiColumns      = "id, level, code, lesson_id, sequence, character, onyomi, kunyomi, examples, meaning_en, meaning_id, sources"
	vocabularyColumns = "id, level, code, lesson_id, sequence, word, reading, part_of_speech, note, meaning_en, meaning_id, sources"
	grammarColumns    = "id, level, code, lesson_id, sequence, curriculum_code, pattern, reading, formula, note, example, difficulty, meaning_en, meaning_id, sources"
)

// Repository reads the items of a quiz's scope.
type Repository struct{ db database.Querier }

// NewRepository reads through db.
func NewRepository(db database.Querier) Repository { return Repository{db: db} }

// Items returns the scope's rows of its section, by code, or
// ErrInvalidLesson. The lessons are checked and the rows read in one
// snapshot.
func (r Repository) Items(ctx context.Context, scope Scope) (Items, error) {
	lessonIDs := make([]pgtype.UUID, 0, len(scope.LessonIDs))
	for _, id := range scope.LessonIDs {
		var u pgtype.UUID
		if err := u.Scan(id); err != nil {
			return Items{}, ErrInvalidLesson
		}
		if !slices.Contains(lessonIDs, u) {
			lessonIDs = append(lessonIDs, u)
		}
	}
	var items Items
	snapshot := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	err := pgx.BeginTxFunc(ctx, r.db, snapshot, func(tx pgx.Tx) error {
		var matched int
		err := tx.QueryRow(ctx, "SELECT count(*) FROM lesson WHERE id = ANY($1) AND level = $2 AND section = $3",
			lessonIDs, scope.Level, string(scope.Section)).Scan(&matched)
		if err != nil {
			return err
		}
		if matched != len(lessonIDs) {
			return ErrInvalidLesson
		}
		switch scope.Section {
		case dataset.SectionKanji:
			items.Kanji, err = itemsOf[dataset.Kanji](ctx, tx, "kanji", kanjiColumns, scope.Level, lessonIDs)
		case dataset.SectionVocabulary:
			items.Vocabulary, err = itemsOf[dataset.Vocabulary](ctx, tx, "vocabulary", vocabularyColumns, scope.Level, lessonIDs)
		case dataset.SectionGrammar:
			items.Grammar, err = itemsOf[dataset.Grammar](ctx, tx, "grammar", grammarColumns, scope.Level, lessonIDs)
		}
		return err
	})
	return items, err
}

// itemsOf returns the level's rows of table, only of lessonIDs when any are
// given, by code.
func itemsOf[T any](ctx context.Context, tx pgx.Tx, table, columns, level string, lessonIDs []pgtype.UUID) ([]T, error) {
	rows, err := tx.Query(ctx, "SELECT "+columns+" FROM "+table+
		" WHERE level = $1 AND (cardinality($2::uuid[]) = 0 OR lesson_id = ANY($2)) ORDER BY code COLLATE \"C\"", level, lessonIDs)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}
