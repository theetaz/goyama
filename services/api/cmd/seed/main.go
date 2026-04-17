// Command seed loads canonical corpus records from corpus/seed/ into the
// Postgres instance pointed at by DATABASE_URL.
//
// Each entity type is split across the canonical tables defined in
// packages/schema/migrations/0001_init.sql — for crops, scalar columns go to
// `crop`, i18n `names` and `description` go to `translation`, and `aliases`
// go to `entity_alias`. Cultivation steps split the same way: scalar fields
// on `cultivation_step`, `title` and `body` into `translation`.
// The entire load is idempotent: running it twice leaves the DB unchanged.
//
// Usage:
//
//	DATABASE_URL=postgres://goyama:goyama@localhost:5432/goyama?sslmode=disable \
//	    go run ./cmd/seed
//
// By default the loader walks ../../corpus/seed relative to the services/api
// working directory; override with --corpus-root.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	corpusRoot := flag.String("corpus-root", "../../corpus/seed", "path to the corpus seed directory")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := seedCrops(ctx, tx, logger, filepath.Join(*corpusRoot, "crops")); err != nil {
		return fmt.Errorf("seed crops: %w", err)
	}

	// Cultivation steps depend on crops existing, so run them second within
	// the same transaction.
	stepsDir := filepath.Join(*corpusRoot, "cultivation_steps")
	if _, err := os.Stat(stepsDir); err == nil {
		if err := seedCultivationSteps(ctx, tx, logger, stepsDir); err != nil {
			return fmt.Errorf("seed cultivation steps: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// listRecords returns sorted *.json paths under dir, or an error if dir has
// no records at all (prevents silent no-op loads).
func listRecords(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", dir, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no records under %s", dir)
	}
	sort.Strings(files)
	return files, nil
}
