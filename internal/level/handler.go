package level

import (
	"context"
	"net/http"

	"github.com/yeremi777/nihongo-foundation/internal/httpx"
)

type store interface {
	List(ctx context.Context) ([]Level, error)
}

// Handler serves the levels.
type Handler struct{ levels store }

// NewHandler serves levels from the given store.
func NewHandler(levels store) Handler { return Handler{levels: levels} }

// Register adds the level route to mux.
func (h Handler) Register(mux httpx.Mux) {
	mux.HandleFunc("GET /api/levels", h.list)
}

func (h Handler) list(w http.ResponseWriter, r *http.Request) {
	levels, err := h.levels.List(r.Context())
	if err != nil {
		httpx.InternalError(w, "list levels", err)
		return
	}
	if levels == nil {
		levels = []Level{}
	}
	httpx.WriteJSON(w, http.StatusOK, levels)
}
