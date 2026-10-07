package dataset

import (
	"regexp"
	"strconv"
	"strings"
)

type listWeek struct {
	number int
	title  string
	paren  string
	line   int
}

// listLesson is one "### Day" block of a source list with everything under it.
type listLesson struct {
	week   listWeek
	day    int
	title  string
	paren  string
	line   int
	tables []table
	body   []line
}

var (
	listWeekHeading = regexp.MustCompile(`^## Week (\d+) — (.+?) \((.+)\)$`)
	listDayHeading  = regexp.MustCompile(`^### Day (\d+) — (.+?) \((.+)\)$`)
	reviewHeading   = regexp.MustCompile(`^## .*Review Focus$`)
)

// parseList splits a source list into its lessons. Content outside a lesson
// (titles, legends, review sections) is not part of any lesson.
func parseList(f File) ([]listLesson, error) {
	var (
		lessons []listLesson
		week    *listWeek
		lesson  *listLesson
	)
	closeLesson := func() {
		if lesson != nil {
			lessons = append(lessons, *lesson)
			lesson = nil
		}
	}
	lines := linesOf(f)
	for i := 0; i < len(lines); {
		ln := lines[i]
		switch {
		case strings.HasPrefix(ln.text, "## "):
			closeLesson()
			week = nil
			if m := listWeekHeading.FindStringSubmatch(ln.text); m != nil && !reviewHeading.MatchString(ln.text) {
				n, _ := strconv.Atoi(m[1])
				week = &listWeek{number: n, title: m[2], paren: m[3], line: ln.no}
			} else if strings.HasPrefix(ln.text, "## Week") && !reviewHeading.MatchString(ln.text) {
				return nil, errorAt(f, ln.no, "week heading is not \"## Week N — <Japanese> (<English>)\"")
			}
			i++
		case strings.HasPrefix(ln.text, "### "):
			closeLesson()
			m := listDayHeading.FindStringSubmatch(ln.text)
			if m == nil {
				if strings.HasPrefix(ln.text, "### Day") {
					return nil, errorAt(f, ln.no, "day heading is not \"### Day N — <Japanese> (<English>)\"")
				}
				i++
				continue
			}
			if week == nil {
				return nil, errorAt(f, ln.no, "day heading is outside a week")
			}
			n, _ := strconv.Atoi(m[1])
			lesson = &listLesson{week: *week, day: n, title: m[2], paren: m[3], line: ln.no}
			i++
		case isTableLine(ln.text):
			t, next, err := readTable(f, lines, i)
			if err != nil {
				return nil, err
			}
			if lesson != nil {
				lesson.tables = append(lesson.tables, t)
			}
			i = next
		default:
			if lesson != nil {
				lesson.body = append(lesson.body, ln)
			}
			i++
		}
	}
	closeLesson()
	return lessons, checkLessonOrder(f, lessons)
}

func checkLessonOrder(f File, lessons []listLesson) error {
	for i := 1; i < len(lessons); i++ {
		prev, cur := lessons[i-1], lessons[i]
		if cur.week.number < prev.week.number || cur.week.number == prev.week.number && cur.day <= prev.day {
			return errorAt(f, cur.line, "week %d day %d comes after week %d day %d", cur.week.number, cur.day, prev.week.number, prev.day)
		}
	}
	return nil
}

func lessonID(level string, section Section, week, day int) string {
	return rowID(level + ":lesson:" + string(section) + ":" + strconv.Itoa(week) + ":" + strconv.Itoa(day))
}
