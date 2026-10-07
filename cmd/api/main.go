// Command api serves the curriculum's lessons and their items over HTTP.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/yeremi777/nihongo-foundation/docs"
	"github.com/yeremi777/nihongo-foundation/internal/config"
	"github.com/yeremi777/nihongo-foundation/internal/database"
	"github.com/yeremi777/nihongo-foundation/internal/httpx"
	"github.com/yeremi777/nihongo-foundation/internal/lesson"
)

// drainTimeout is how long in-flight requests may run after a stop signal.
const drainTimeout = 25 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("api failed", "err", err)
		os.Exit(1)
	}
}

// register adds every route the API serves to mux.
func register(mux httpx.Mux, db database.Querier, spec []byte) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	lesson.NewHandler(lesson.NewRepository(db)).Register(mux)
	httpx.Docs(mux, spec)
}

func run() error {
	cfg, err := config.LoadAPI(os.Getenv)
	if err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := database.OpenPool(ctx, cfg.Database.DSN())
	if err != nil {
		return err
	}
	defer pool.Close()

	mux := http.NewServeMux()
	register(mux, pool, docs.Spec())
	srv := &http.Server{
		Addr:              net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.Port)),
		Handler:           httpx.Router(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stop, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	served := make(chan error, 1)
	go func() { served <- srv.ListenAndServe() }()
	slog.Info("api listening", "url", cfg.URL, "docs", cfg.URL+"/docs")

	select {
	case err := <-served:
		return err
	case <-stop.Done():
	}
	drain, cancelDrain := context.WithTimeout(context.Background(), drainTimeout)
	defer cancelDrain()
	if err := srv.Shutdown(drain); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	slog.Info("api stopped")
	return nil
}
