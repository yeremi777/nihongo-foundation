package item

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/yeremi777/nihongo-foundation/internal/database"
)

// katakana and hiragana pair each katakana letter, ァ to ヶ, with its
// hiragana, so translate folds one into the other.
var katakana, hiragana = kana('ァ'), kana('ぁ')

func kana(first rune) string {
	var b strings.Builder
	for r := first; r <= first+'ヶ'-'ァ'; r++ {
		b.WriteRune(r)
	}
	return b.String()
}

// fold lowercases ASCII letters and turns katakana into hiragana.
func fold(sql string) string {
	// translate is slow, so it runs only on text holding katakana.
	return "CASE WHEN " + sql + " ~ '[ァ-ヶ]' THEN translate(lower(" + sql + "), @katakana, @hiragana) ELSE lower(" + sql + ") END"
}

// repository reads the items of one section.
type repository[T any] struct {
	db      database.Querier
	section section[T]
}

// Search returns the page of items q matches and their count, read in one
// snapshot.
func (r repository[T]) Search(ctx context.Context, q Query) (Page[T], error) {
	// An offset past the largest int lies past every table.
	offset := math.MaxInt
	if q.Page-1 <= math.MaxInt/q.Size {
		offset = (q.Page - 1) * q.Size
	}
	args := pgx.NamedArgs{
		"level": q.Level, "week": q.Week, "source": string(q.Source), "difficulty": string(q.Difficulty), "text": q.Text,
		"katakana": katakana, "hiragana": hiragana, "levels": levels, "size": q.Size, "offset": offset,
	}
	from, exact := r.from()
	page := Page[T]{Page: q.Page, Size: q.Size}
	snapshot := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	err := pgx.BeginTxFunc(ctx, r.db, snapshot, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, "SELECT count(*)"+from, args).Scan(&page.Total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, "SELECT "+r.section.columns+from+
			" ORDER BY "+exact+" DESC, array_position(@levels::text[], level), week, day, sequence LIMIT @size OFFSET @offset", args)
		if err != nil {
			return err
		}
		page.Items, err = pgx.CollectRows(rows, pgx.RowToStructByName[T])
		return err
	})
	return page, err
}

// from returns the FROM and WHERE clauses of a search, and the condition an
// exact match of its text meets.
func (r repository[T]) from() (from, exact string) {
	var contains, equals []string
	for _, c := range r.section.texts {
		contains = append(contains, "strpos("+fold(c)+", needle) > 0")
		equals = append(equals, fold(c)+" = needle")
	}
	for _, c := range r.section.arrays {
		contains = append(contains, "EXISTS (SELECT 1 FROM unnest("+c+") e WHERE strpos("+fold("e")+", needle) > 0)")
		equals = append(equals, "needle IN (SELECT "+fold("e")+" FROM unnest("+c+") e)")
	}
	from = " FROM " + string(r.section.name) +
		" JOIN (SELECT id AS lesson, week, day FROM lesson) l ON l.lesson = lesson_id" +
		" CROSS JOIN (SELECT " + fold("@text") + " AS needle) n" +
		" WHERE (@level = '' OR level = @level) AND (@week::bigint = 0 OR week = @week::bigint)" +
		" AND (@source = '' OR @source = ANY(sources))"
	if r.section.difficulty {
		from += " AND (@difficulty = '' OR difficulty = @difficulty)"
	}
	from += " AND (needle = '' OR " + strings.Join(contains, " OR ") + ")"
	return from, "(needle <> '' AND (" + strings.Join(equals, " OR ") + ")) IS TRUE"
}

// Get returns the item of the level and code, or ErrNotFound.
func (r repository[T]) Get(ctx context.Context, level, code string) (T, error) {
	rows, err := r.db.Query(ctx, "SELECT "+r.section.columns+" FROM "+string(r.section.name)+" WHERE level = $1 AND code = $2", level, code)
	if err != nil {
		var zero T
		return zero, err
	}
	item, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return item, ErrNotFound
	}
	return item, err
}
