package item

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/yeremi777/nihongo-foundation/internal/database"
	"github.com/yeremi777/nihongo-foundation/internal/dataset"
	"github.com/yeremi777/nihongo-foundation/internal/httpx"
)

const (
	defaultSize = 20
	maxSize     = 100
	maxText     = 100
)

type store[T any] interface {
	Search(ctx context.Context, q Query) (Page[T], error)
	Get(ctx context.Context, level, code string) (T, error)
}

// Register adds the search and lookup routes of every section to mux.
func Register(mux httpx.Mux, db database.Querier) {
	route(mux, kanji, repository[dataset.Kanji]{db: db, section: kanji})
	route(mux, vocabulary, repository[dataset.Vocabulary]{db: db, section: vocabulary})
	route(mux, grammar, repository[dataset.Grammar]{db: db, section: grammar})
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
			httpx.WriteError(w, http.StatusBadRequest, "invalid_level", "Level must be n5, n4, n3, n2, or n1.")
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

// parseQuery reads a search from v, or the code and message of its first
// invalid parameter. difficulty is read only for a section filtered by it.
func parseQuery(v url.Values, difficulty bool) (q Query, code, message string) {
	q = Query{Level: v.Get("level"), Source: dataset.Source(v.Get("source")), Text: strings.TrimSpace(v.Get("q")), Page: 1, Size: defaultSize}
	if difficulty {
		q.Difficulty = dataset.Difficulty(v.Get("difficulty"))
	}
	switch {
	case q.Level != "" && !slices.Contains(levels, q.Level):
		return q, "invalid_level", "Level must be n5, n4, n3, n2, or n1."
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
