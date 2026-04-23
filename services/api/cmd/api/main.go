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
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/goyama/api/internal/admin"
	"github.com/goyama/api/internal/ask"
	"github.com/goyama/api/internal/crops"
	"github.com/goyama/api/internal/diseases"
	"github.com/goyama/api/internal/embed"
	"github.com/goyama/api/internal/geo"
	"github.com/goyama/api/internal/health"
	"github.com/goyama/api/internal/knowledge"
	"github.com/goyama/api/internal/markets"
	"github.com/goyama/api/internal/media"
	"github.com/goyama/api/internal/pests"
	"github.com/goyama/api/internal/plans"
	"github.com/goyama/api/internal/platform/config"
	"github.com/goyama/api/internal/platform/httpx"
	"github.com/goyama/api/internal/remedies"
)

// repos bundles every repository the API serves so wiring stays
// tractable as the surface grows. Filled by newRepos, consumed by
// buildRouter.
type repos struct {
	crops           crops.Repository
	steps           crops.CultivationStepRepo
	diseases        diseases.Repository
	pests           pests.Repository
	remedies        remedies.Repository
	geo             geo.Repository
	markets         markets.Repository
	media           media.Repository
	plans           plans.Repository
	plansAdmin      plans.AdminRepo
	knowledge       knowledge.Repository
	knowledgeAdmin  knowledge.AdminRepo
	knowledgeSearch knowledge.SearchRepo
}

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

	rs, closeRepos, err := newRepos(cfg, log)
	if err != nil {
		return fmt.Errorf("repos: %w", err)
	}
	defer closeRepos()

	r := buildRouter(cfg, log, rs)

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

// newRepos builds every repository the API serves — both farmer-facing
// (crops) and admin-facing (cultivation-step review queue, diseases,
// pests). All share one pgx pool when DATABASE_URL is set; otherwise
// crops falls back to JSONL and the three admin repos return
// ErrRequiresDatabase from every call.
func newRepos(cfg config.Config, log *slog.Logger) (repos, func(), error) {
	// Plans and knowledge always read from the JSONL corpus for now;
	// Postgres-backed versions will land alongside the Postgres loader
	// command in a follow-up. They default to the per-seed-subdir
	// layout so the corpus_path env var points at the same directory
	// used by the other JSONL repos during dev.
	corpusSeedDir := filepath.Join(cfg.CorpusPath, "..", "..", "seed")
	plansRepo := plans.NewJSONLRepo(filepath.Join(corpusSeedDir, "cultivation_plans"))
	knowledgeRepo := knowledge.NewJSONLRepo(corpusSeedDir)

	if cfg.DatabaseURL == "" {
		log.Info("repos: using JSONL corpus (admin review queues, geo lookup, market prices, media disabled)",
			slog.String("corpus_path", cfg.CorpusPath),
		)
		return repos{
			crops:           crops.NewJSONLRepo(cfg.CorpusPath),
			steps:           crops.NewCultivationStepJSONLRepo(),
			diseases:        diseases.NewJSONLRepo(),
			pests:           pests.NewJSONLRepo(),
			remedies:        remedies.NewJSONLRepo(),
			geo:             geo.NewStubRepo(),
			markets:         markets.NewStubRepo(),
			media:           media.NewStubRepo(),
			plans:           plansRepo,
			plansAdmin:      plansRepo,
			knowledge:       knowledgeRepo,
			knowledgeAdmin:  knowledgeRepo,
			knowledgeSearch: knowledge.NewSearchStub(),
		}, func() {}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return repos{}, nil, fmt.Errorf("pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return repos{}, nil, fmt.Errorf("ping db: %w", err)
	}
	log.Info("repos: using Postgres")
	plansPgx := plans.NewPgxRepo(pool)
	knowledgePgx := knowledge.NewPgxRepo(pool)
	return repos{
		crops:           crops.NewPgxRepo(pool),
		steps:           crops.NewCultivationStepPgxRepo(pool),
		diseases:        diseases.NewPgxRepo(pool),
		pests:           pests.NewPgxRepo(pool),
		remedies:        remedies.NewPgxRepo(pool),
		geo:             geo.NewPgxRepo(pool),
		markets:         markets.NewPgxRepo(pool),
		media:           media.NewPgxRepo(pool),
		plans:           plansPgx,
		plansAdmin:      plansPgx,
		knowledge:       knowledgePgx,
		knowledgeAdmin:  knowledgePgx,
		knowledgeSearch: knowledgePgx,
	}, pool.Close, nil
}

func buildRouter(cfg config.Config, log *slog.Logger, rs repos) http.Handler {
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
	cropsH := crops.NewHandler(rs.crops)
	diseasesH := diseases.NewHandler(rs.diseases)
	pestsH := pests.NewHandler(rs.pests)
	remediesH := remedies.NewHandler(rs.remedies)
	geoH := geo.NewHandler(rs.geo)
	marketsH := markets.NewHandler(rs.markets)
	mediaH := media.New(rs.media)
	plansH := plans.New(rs.plans)
	knowledgeH := knowledge.New(rs.knowledge)
	// The chat-agent handler shares the same embedder used by
	// cmd/embedchunks at backfill time. The selection happens once at
	// startup so the cosine math stays consistent across the request
	// pipeline.
	embedder := embed.FromEnv()
	log.Info("ask: embedder selected", slog.String("name", embedder.Name()))
	askH := ask.New(embedder, rs.knowledgeSearch, rs.geo)
	adminH := admin.New(rs.steps, rs.diseases, rs.pests, rs.remedies, rs.plansAdmin, rs.knowledgeAdmin, mediaH)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", healthH.Get)
		r.Mount("/crops", cropsH.Routes())
		r.Get("/crops/{slug}/cultivation-plans", plansH.ByCropHandler())
		r.Get("/crops/{slug}/knowledge", knowledgeH.ByEntityHandler("crop"))
		r.Mount("/diseases", diseasesH.Routes())
		r.Get("/diseases/{slug}/images", mediaH.PublicGalleryHandler("disease"))
		r.Get("/diseases/{slug}/knowledge", knowledgeH.ByEntityHandler("disease"))
		r.Mount("/pests", pestsH.Routes())
		r.Get("/pests/{slug}/images", mediaH.PublicGalleryHandler("pest"))
		r.Get("/pests/{slug}/knowledge", knowledgeH.ByEntityHandler("pest"))
		r.Mount("/remedies", remediesH.Routes())
		r.Mount("/cultivation-plans", plansH.Routes())
		r.Mount("/geo", geoH.Routes())
		r.Mount("/market-prices", marketsH.Routes())
		r.Mount("/ask", askH.Routes())
		r.Mount("/admin", adminH.Routes())
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
