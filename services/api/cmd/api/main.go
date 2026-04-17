// Command api starts the Goyama HTTP API.
//
// Reads config from the environment via internal/platform/config, wires the
// HTTP router with chi, attaches structured logging + request IDs, and serves
// crops from the local corpus JSONL release (Postgres wiring lands next).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/goyama/api/internal/crops"
	"github.com/goyama/api/internal/health"
	"github.com/goyama/api/internal/platform/config"
	"github.com/goyama/api/internal/platform/httpx"
)

// version is overridden via -ldflags at build time.
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := newLogger(cfg.LogLevel)
	slog.SetDefault(log)

	r := buildRouter(cfg, log)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("api starting",
			slog.String("addr", srv.Addr),
			slog.String("env", cfg.Env),
			slog.String("corpus_path", cfg.CorpusPath),
			slog.String("version", version),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case s := <-stop:
		log.Info("shutdown signal", slog.String("signal", s.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Info("api stopped")
	return nil
}

func buildRouter(cfg config.Config, log *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(httpx.RequestIDMiddleware)
	r.Use(httpx.AccessLog(log))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CorsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	healthH := health.New(version)
	cropsRepo := crops.NewJSONLRepo(cfg.CorpusPath)
	cropsH := crops.NewHandler(cropsRepo)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", healthH.Get)
		r.Mount("/crops", cropsH.Routes())
	})

	// Root-level redirect to docs in non-prod.
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{
			"service": "goyama-api",
			"version": version,
			"docs":    "/v1/health",
		})
	})

	return r
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(h)
}
