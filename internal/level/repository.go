package level

import (
	"context"

	"github.com/yeremi777/nihongo-foundation/internal/database"
)

// Repository counts lessons and items per level and section.
type Repository struct{ db database.Querier }

// NewRepository reads through db.
func NewRepository(db database.Querier) Repository { return Repository{db: db} }

// List returns the levels with lessons, each with its sections, counted in one statement.
func (r Repository) List(ctx context.Context) ([]Level, error) {
	sectionOrder := make([]string, len(sections))
	for i, s := range sections {
		sectionOrder[i] = string(s)
	}
	rows, err := r.db.Query(ctx, `
		WITH item AS (
			SELECT lesson_id FROM kanji
			UNION ALL SELECT lesson_id FROM vocabulary
			UNION ALL SELECT lesson_id FROM grammar
		)
		SELECT l.level, l.section, count(DISTINCT l.week), count(DISTINCT l.id), count(item.lesson_id)
		FROM lesson l LEFT JOIN item ON item.lesson_id = l.id
		GROUP BY l.level, l.section
		ORDER BY array_position($1::text[], l.level), array_position($2::text[], l.section)`,
		levels, sectionOrder)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Level
	for rows.Next() {
		var level string
		var s Section
		if err := rows.Scan(&level, &s.Section, &s.Weeks, &s.Lessons, &s.Items); err != nil {
			return nil, err
		}
		if n := len(list); n == 0 || list[n-1].Level != level {
			list = append(list, Level{Level: level})
		}
		last := &list[len(list)-1]
		last.Sections = append(last.Sections, s)
	}
	return list, rows.Err()
}
