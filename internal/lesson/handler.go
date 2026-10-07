package lesson

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
	"github.com/yeremi777/nihongo-foundation/internal/httpx"
)

var (
	levels   = []string{"n5", "n4", "n3", "n2", "n1"}
	sections = []dataset.Section{dataset.SectionKanji, dataset.SectionVocabulary, dataset.SectionGrammar}
)

type store interface {
	List(ctx context.Context, level string, section dataset.Section) ([]dataset.Lesson, error)
	Get(ctx context.Context, id string) (Detail, error)
}

// Handler serves lessons and their items.
type Handler struct{ lessons store }

// NewHandler serves lessons from the given store.
func NewHandler(lessons store) Handler { return Handler{lessons: lessons} }

// Register adds the lesson routes to mux.
func (h Handler) Register(mux httpx.Mux) {
	mux.HandleFunc("GET /api/levels/{level}/sections/{section}/lessons", h.list)
	mux.HandleFunc("GET /api/lessons/{lesson_id}", h.get)
}

func (h Handler) list(w http.ResponseWriter, r *http.Request) {
	level, section := r.PathValue("level"), dataset.Section(r.PathValue("section"))
	if !slices.Contains(levels, level) {
		httpx.WriteError(w, http.StatusNotFound, "level_not_found", "Level was not found.")
		return
	}
	if !slices.Contains(sections, section) {
		httpx.WriteError(w, http.StatusNotFound, "section_not_found", "Section was not found.")
		return
	}
	lessons, err := h.lessons.List(r.Context(), level, section)
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
