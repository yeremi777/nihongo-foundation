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
	"github.com/yeremi777/nihongo-foundation/internal/quiz"
)

// drainTimeout is how long in-flight requests may run after a stop signal.
const drainTimeout = 25 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("api failed", "err", err)
		os.Exit(1)
	}
}

// register adds every route the API serves to mux. Quizzes are written by
// generator within aiTimeout; a nil generator makes them unavailable.
func register(mux httpx.Mux, db database.Querier, spec []byte, generator quiz.Generator, aiTimeout time.Duration) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	lesson.NewHandler(lesson.NewRepository(db)).Register(mux)
	quiz.NewHandler(quiz.NewRepository(db), generator, aiTimeout).Register(mux)
	httpx.Docs(mux, spec)
}

// generatorFor returns Mock when ai lists mock, else the chain of listed
// providers, or nil when none is usable. A provider without API key or model
// is skipped with a warning.
func generatorFor(ai config.AI) quiz.Generator {
	var chain quiz.Chain
	for _, name := range ai.Providers {
		var chat config.Chat
		switch name {
		case "mock":
			return quiz.Mock{}
		case "openrouter":
			chat = ai.OpenRouter
		case "opencode_zen":
			chat = ai.OpenCodeZen
		}
		if chat.APIKey == "" || chat.Model == "" {
			slog.Warn("quiz provider skipped: no API key or model", "provider", name)
			continue
		}
		chain = append(chain, quiz.NewChat(chat.Name, chat.ServerURL, chat.Model, chat.APIKey, chat.Headers))
	}
	if len(chain) == 0 {
		slog.Warn("no quiz provider is usable; quizzes answer quiz_unavailable", "AI_PROVIDERS", ai.Providers)
		return nil
	}
	return chain
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
	register(mux, pool, docs.Spec(), generatorFor(cfg.AI), cfg.AI.Timeout)
	srv := &http.Server{
		Addr:              net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.Port)),
		Handler:           httpx.Router(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Bind before logging, so "api listening" is printed only once the port
	// is ours.
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	stop, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()
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
