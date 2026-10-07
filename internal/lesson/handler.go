package lesson

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
	"github.com/yeremi777/nihongo-foundation/internal/httpx"
)

type store interface {
	List(ctx context.Context, filter Filter) ([]dataset.Lesson, error)
	Get(ctx context.Context, id string) (Detail, error)
}

// Handler serves lessons and their items.
type Handler struct{ lessons store }

// NewHandler serves lessons from the given store.
func NewHandler(lessons store) Handler { return Handler{lessons: lessons} }

// Register adds the lesson routes to mux.
func (h Handler) Register(mux httpx.Mux) {
	mux.HandleFunc("GET /api/lessons", h.list)
	mux.HandleFunc("GET /api/lessons/{lesson_id}", h.get)
}

func (h Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := Filter{Level: q.Get("level"), Section: dataset.Section(q.Get("section"))}
	if filter.Level != "" && !slices.Contains(levels, filter.Level) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_level", "Level must be n5, n4, n3, n2, or n1.")
		return
	}
	if filter.Section != "" && !slices.Contains(sections, filter.Section) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_section", "Section must be kanji, vocabulary, or grammar.")
		return
	}
	lessons, err := h.lessons.List(r.Context(), filter)
	if err != nil {
		httpx.InternalError(w, "list lessons", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, lessons)
}

func (h Handler) get(w http.ResponseWriter, r *http.Request) {
	detail, err := h.lessons.Get(r.Context(), r.PathValue("lesson_id"))
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "lesson_not_found", "Lesson was not found.")
	case err != nil:
		httpx.InternalError(w, "get lesson", err)
	default:
		httpx.WriteJSON(w, http.StatusOK, detail)
	}
}
