package quiz

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/yeremi777/nihongo-foundation/internal/dataset"
	"github.com/yeremi777/nihongo-foundation/internal/httpx"
)

const (
	maxBodyBytes = 64 << 10
	defaultCount = 5
	maxCount     = 10
	// writeMargin is added to the provider timeout for the route's write
	// deadline.
	writeMargin = 5 * time.Second
)

// ErrInvalidLesson reports a lesson id that is not a lesson of the scope's
// level and section.
var ErrInvalidLesson = errors.New("lesson is not of the level and section")

var levels = []string{"n5", "n4", "n3", "n2", "n1"}

// Scope is the items a quiz draws on; no LessonIDs means the whole section.
type Scope struct {
	Level     string
	Section   dataset.Section
	LessonIDs []string
}

type store interface {
	Items(ctx context.Context, scope Scope) (Items, error)
}

// Generator writes one draft per target.
type Generator interface {
	Generate(ctx context.Context, lang Lang, targets []Target) ([]Draft, error)
}

// Handler serves quizzes.
type Handler struct {
	items     store
	generator Generator
	timeout   time.Duration
}

// NewHandler serves quizzes from the store's items, written by generator
// within timeout; a nil generator makes quizzes unavailable.
func NewHandler(items store, generator Generator, timeout time.Duration) Handler {
	return Handler{items: items, generator: generator, timeout: timeout}
}

// Register adds the quiz route to mux.
func (h Handler) Register(mux httpx.Mux) {
	mux.HandleFunc("POST /api/quizzes", h.create)
}

type request struct {
	Level     string          `json:"level"`
	Section   dataset.Section `json:"section"`
	Lang      Lang            `json:"lang"`
	LessonIDs []string        `json:"lesson_ids"`
	Types     []Type          `json:"types"`
	Count     *int            `json:"count"`
}

func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	if h.generator == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "quiz_unavailable", "Quiz provider is not configured.")
		return
	}
	req, ok := decodeRequest(w, r)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "Body must be a JSON object with known keys.")
		return
	}
	types, known := typesOf[req.Section]
	switch {
	case !slices.Contains(levels, req.Level):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_level", "Level must be n5, n4, n3, n2, or n1.")
		return
	case !known:
		httpx.WriteError(w, http.StatusBadRequest, "invalid_section", "Section must be kanji, vocabulary, or grammar.")
		return
	case req.Lang != LangEN && req.Lang != LangID:
		httpx.WriteError(w, http.StatusBadRequest, "invalid_lang", "Lang must be en or id.")
		return
	case slices.ContainsFunc(req.Types, func(t Type) bool { return !slices.Contains(types, t) }):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_type", "Every type must be a question type of the section.")
		return
	case req.Count != nil && (*req.Count < 1 || *req.Count > maxCount):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_count", "Count must be 1 to 10.")
		return
	}
	if len(req.Types) > 0 {
		types = req.Types
	}
	count := defaultCount
	if req.Count != nil {
		count = *req.Count
	}

	items, err := h.items.Items(r.Context(), Scope{Level: req.Level, Section: req.Section, LessonIDs: req.LessonIDs})
	switch {
	case errors.Is(err, ErrInvalidLesson):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_lesson", "Every lesson must be a lesson of the level and section.")
		return
	case err != nil:
		httpx.InternalError(w, "quiz scope", err)
		return
	}
	chosen := sample(targets(req.Lang, items, types), count)
	if len(chosen) == 0 {
		httpx.WriteError(w, http.StatusNotFound, "items_not_found", "No item matches the request.")
		return
	}

	// The server's write timeout is shorter than a provider call.
	err = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(h.timeout + writeMargin))
	if err != nil && !errors.Is(err, http.ErrNotSupported) {
		httpx.InternalError(w, "quiz write deadline", err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	drafts, err := h.generator.Generate(ctx, req.Lang, chosen)
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		slog.Error("generate quiz", "err", err)
		httpx.WriteError(w, http.StatusGatewayTimeout, "quiz_generation_timeout", "Quiz generation timed out.")
		return
	case err != nil:
		slog.Error("generate quiz", "err", err)
		httpx.WriteError(w, http.StatusBadGateway, "quiz_generation_failed", "Quiz generation failed.")
		return
	}
	quiz := assemble(chosen, drafts)
	if len(quiz.Questions) == 0 {
		slog.Error("generate quiz", "err", "no usable question", "dropped", quiz.Dropped)
		httpx.WriteError(w, http.StatusBadGateway, "quiz_generation_failed", "Quiz generation failed.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, quiz)
}

// decodeRequest reads the body as exactly one JSON object with known keys.
func decodeRequest(w http.ResponseWriter, r *http.Request) (request, bool) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	var req *request
	if err := dec.Decode(&req); err != nil || req == nil {
		return request{}, false
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return request{}, false
	}
	return *req, true
}
