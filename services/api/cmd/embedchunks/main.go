// Command embedchunks backfills the content_embedding column on
// knowledge_chunk rows that don't have one yet.
//
// Usage:
//
//	embedchunks                    # picks up VOYAGE_API_KEY from env if set
//	embedchunks --limit=200
//	embedchunks --embedder=hash    # force the dev embedder even with VOYAGE_API_KEY set
//
// Idempotent — only rows where content_embedding IS NULL are touched,
// so re-running after adding new chunks just embeds the new ones.
// Switching embedders means manually clearing the column first
// (UPDATE knowledge_chunk SET content_embedding = NULL).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/goyama/api/internal/embed"
	"github.com/goyama/api/internal/knowledge"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	limit := flag.Int("limit", 100, "max chunks to embed in one run")
	embedderFlag := flag.String("embedder", "", "force a specific embedder: hash | voyage (default: hash unless VOYAGE_API_KEY is set)")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	embedder := selectEmbedder(*embedderFlag)
	fmt.Printf("using embedder: %s\n", embedder.Name())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("pgx pool: %w", err)
	}
	defer pool.Close()

	repo := knowledge.NewPgxRepo(pool)
	chunks, err := repo.ListUnembedded(ctx, *limit)
	if err != nil {
		return fmt.Errorf("list unembedded: %w", err)
	}
	if len(chunks) == 0 {
		fmt.Println("nothing to embed — all chunks already have a content_embedding")
		return nil
	}
	fmt.Printf("embedding %d chunk(s)…\n", len(chunks))

	embedded := 0
	for _, c := range chunks {
		// Title + body gives the embedder more signal than body alone
		// for short chunks; titles tend to carry the headline noun
		// phrase ("Diamondback moth control regimen").
		text := c.Title
		if c.Body != "" {
			if text != "" {
				text += "\n\n"
			}
			text += c.Body
		}
		vec, err := embedder.Embed(ctx, text)
		if err != nil {
			return fmt.Errorf("embed %s: %w", c.Slug, err)
		}
		if err := repo.UpdateEmbedding(ctx, c.Slug, vec); err != nil {
			return fmt.Errorf("update %s: %w", c.Slug, err)
		}
		fmt.Printf("  ✓ %s\n", c.Slug)
		embedded++
	}
	fmt.Printf("\nembedded %d chunk(s) with %s\n", embedded, embedder.Name())
	return nil
}

func selectEmbedder(force string) embed.Embedder {
	switch strings.ToLower(strings.TrimSpace(force)) {
	case "hash":
		return embed.NewHashEmbedder()
	case "voyage":
		key := os.Getenv("VOYAGE_API_KEY")
		if key == "" {
			fmt.Fprintln(os.Stderr, "WARNING: --embedder=voyage but VOYAGE_API_KEY is empty; falling back to hash")
			return embed.NewHashEmbedder()
		}
		return embed.NewVoyageEmbedder(key)
	default:
		return embed.FromEnv()
	}
}
