package item

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yeremi777/nihongo-foundation/internal/database"
	"github.com/yeremi777/nihongo-foundation/internal/dataset"
	"github.com/yeremi777/nihongo-foundation/internal/httpx"
)

const (
	defaultSize = 20
	maxSize     = 100
	maxText     = 100

	maxBodyBytes    = 64 << 10
	maxCoverageText = 10000

	invalidLevel = "Level must be n5, n4, n3, n2, or n1."
)

type store[T any] interface {
	Search(ctx context.Context, q Query) (Page[T], error)
	Get(ctx context.Context, level, code string) (T, error)
}

// Register adds the search and lookup routes of every section, and the kanji
// coverage route, to mux.
func Register(mux httpx.Mux, db database.Querier) {
	route(mux, kanji, repository[dataset.Kanji]{db: db, section: kanji})
	route(mux, vocabulary, repository[dataset.Vocabulary]{db: db, section: vocabulary})
	route(mux, grammar, repository[dataset.Grammar]{db: db, section: grammar})
	routeCoverage(mux, kanjiRepository{db: db})
}

// route adds the search and lookup routes of one section to mux.
func route[T any](mux httpx.Mux, s section[T], items store[T]) {
	path := "/api/" + string(s.name)
	mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		q, code, message := parseQuery(r.URL.Query(), s.difficulty)
		if code != "" {
			httpx.WriteError(w, http.StatusBadRequest, code, message)
			return
		}
		page, err := items.Search(r.Context(), q)
		if err != nil {
			httpx.InternalError(w, "search "+string(s.name), err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, page)
	})
	mux.HandleFunc("GET "+path+"/{level}/{code}", func(w http.ResponseWriter, r *http.Request) {
		level := r.PathValue("level")
		if !slices.Contains(levels, level) {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_level", invalidLevel)
			return
		}
		code := r.PathValue("code")
		if !isText(code) {
			httpx.WriteError(w, http.StatusNotFound, "item_not_found", "Item was not found.")
			return
		}
		item, err := items.Get(r.Context(), level, code)
		switch {
		case errors.Is(err, ErrNotFound):
			httpx.WriteError(w, http.StatusNotFound, "item_not_found", "Item was not found.")
		case err != nil:
			httpx.InternalError(w, "get "+string(s.name), err)
		default:
			httpx.WriteJSON(w, http.StatusOK, item)
		}
	})
}

type kanjiStore interface {
	Kanji(ctx context.Context, characters []string) (levels []string, rows []dataset.Kanji, err error)
}

// routeCoverage adds the kanji coverage route to mux.
func routeCoverage(mux httpx.Mux, store kanjiStore) {
	mux.HandleFunc("POST /api/kanji/coverage", func(w http.ResponseWriter, r *http.Request) {
		text, ok := decodeText(w, r)
		if !ok {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "Body must be a JSON object with known keys.")
			return
		}
		text = strings.TrimSpace(text)
		if n := utf8.RuneCountInString(text); n == 0 || n > maxCoverageText {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_text", "Text must be 1 to 10000 characters.")
			return
		}
		kanji := textKanji(text)
		characters := make([]string, len(kanji))
		for i, k := range kanji {
			characters[i] = k.Character
		}
		levels, rows, err := store.Kanji(r.Context(), characters)
		if err != nil {
			httpx.InternalError(w, "kanji coverage", err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, coverageOf(kanji, levels, rows))
	})
}

// decodeText reads the body as exactly one JSON object holding only text.
func decodeText(w http.ResponseWriter, r *http.Request) (string, bool) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	var req *struct {
		Text string `json:"text"`
	}
	if err := dec.Decode(&req); err != nil || req == nil {
		return "", false
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", false
	}
	return req.Text, true
}

// textKanji returns the distinct kanji of text with their counts, in order of
// first appearance; 々 and 〇 are not counted.
func textKanji(text string) []TextKanji {
	kanji := []TextKanji{}
	index := map[rune]int{}
	for _, r := range text {
		if !unicode.Is(unicode.Han, r) || !unicode.Is(unicode.Lo, r) {
			continue
		}
		i, seen := index[r]
		if !seen {
			i = len(kanji)
			index[r] = i
			kanji = append(kanji, TextKanji{Character: string(r)})
		}
		kanji[i].Count++
	}
	return kanji
}

// coverageOf gives each kanji the row of the lowest level holding it, and
// counts the occurrences known at each level or below it. rows and levels are
// from n5.
func coverageOf(kanji []TextKanji, levels []string, rows []dataset.Kanji) Coverage {
	held := map[string]dataset.Kanji{}
	for _, k := range rows {
		if _, ok := held[k.Character]; !ok {
			held[k.Character] = k
		}
	}
	c := Coverage{Kanji: kanji, Levels: make([]LevelCoverage, len(levels))}
	knownAt := make([]int, len(levels))
	for i, k := range kanji {
		c.Total += k.Count
		if row, ok := held[k.Character]; ok {
			kanji[i].Item = &row
			knownAt[slices.Index(levels, row.Level)] += k.Count
		}
	}
	known := 0
	for i, level := range levels {
		known += knownAt[i]
		c.Levels[i] = LevelCoverage{Level: level, Known: known, KnownPercent: 100}
		if c.Total > 0 {
			c.Levels[i].KnownPercent = known * 100 / c.Total
		}
	}
	return c
}

// parseQuery reads a search from v, or the code and message of its first
// invalid parameter. difficulty is read only for a section filtered by it.
func parseQuery(v url.Values, difficulty bool) (q Query, code, message string) {
	q = Query{Level: v.Get("level"), Source: dataset.Source(v.Get("source")), Text: strings.TrimSpace(v.Get("q")), Page: 1, Size: defaultSize}
	if difficulty {
		q.Difficulty = dataset.Difficulty(v.Get("difficulty"))
	}
	switch {
	case q.Level != "" && !slices.Contains(levels, q.Level):
		return q, "invalid_level", invalidLevel
	case !parseInt(v.Get("week"), &q.Week, 1, math.MaxInt) || q.Week != 0 && q.Level == "":
		return q, "invalid_week", "Week must be a positive integer, given with level."
	case q.Source != "" && !slices.Contains(sources, q.Source):
		return q, "invalid_source", "Source must be soumatome, shinkanzen, or minna-no-nihongo."
	case q.Difficulty != "" && !slices.Contains(difficulties, q.Difficulty):
		return q, "invalid_difficulty", "Difficulty must be easy, medium, or hard."
	case !isText(q.Text) || utf8.RuneCountInString(q.Text) > maxText:
		return q, "invalid_query", "Query must be valid text of at most 100 characters."
	case !parseInt(v.Get("page"), &q.Page, 1, math.MaxInt):
		return q, "invalid_page", "Page must be a positive integer."
	case !parseInt(v.Get("size"), &q.Size, 1, maxSize):
		return q, "invalid_size", "Size must be 1 to 100."
	}
	return q, "", ""
}

// parseInt sets *n from s when s is not empty, reporting whether s is empty
// or an integer from low to high.
func parseInt(s string, n *int, low, high int) bool {
	if s == "" {
		return true
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < low || v > high {
		return false
	}
	*n = v
	return true
}

// isText reports whether s is UTF-8 without NUL, as Postgres text requires.
func isText(s string) bool { return utf8.ValidString(s) && !strings.ContainsRune(s, 0) }
