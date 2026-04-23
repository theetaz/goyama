// Command knowledgeload upserts knowledge_source + knowledge_chunk
// rows from JSON fixtures into Postgres. Reads both sibling
// directories in one pass; sources are loaded before chunks so the
// foreign-key check in knowledge_chunk never trips.
//
// Usage:
//
//	knowledgeload --dir=corpus/seed
//
// The --dir must contain both knowledge_sources/ and knowledge_chunks/
// subdirectories. Idempotent — rerunning updates rows in place keyed
// on slug.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sourceFile struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
	Medium      string `json:"medium"`
	Publisher   string `json:"publisher"`
	Authority   string `json:"authority"`
	URL         string `json:"url"`
	Language    string `json:"language"`
	Licence     string `json:"licence"`
	PublishedAt string `json:"published_at"`
	Notes       string `json:"notes"`
}

type chunkFile struct {
	Slug               string         `json:"slug"`
	SourceSlug         string         `json:"source_slug"`
	ChunkIdx           int            `json:"chunk_idx"`
	Language           string         `json:"language"`
	Title              string         `json:"title"`
	Body               string         `json:"body"`
	BodyTranslated     map[string]any `json:"body_translated"`
	EntityRefs         []any          `json:"entity_refs"`
	Authority          string         `json:"authority"`
	AppliesToAEZCodes  []string       `json:"applies_to_aez_codes"`
	AppliesToCountries []string       `json:"applies_to_countries"`
	TopicTags          []string       `json:"topic_tags"`
	Confidence         *float64       `json:"confidence"`
	Quote              string         `json:"quote"`
	Status             string         `json:"status"`
	FieldProvenance    map[string]any `json:"field_provenance"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	dir := flag.String("dir", "", "corpus/seed directory containing knowledge_sources/ and knowledge_chunks/")
	flag.Parse()
	if *dir == "" {
		flag.Usage()
		return fmt.Errorf("--dir is required")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	sources, err := readAll[sourceFile](filepath.Join(*dir, "knowledge_sources"))
	if err != nil {
		return fmt.Errorf("read sources: %w", err)
	}
	chunks, err := readAll[chunkFile](filepath.Join(*dir, "knowledge_chunks"))
	if err != nil {
		return fmt.Errorf("read chunks: %w", err)
	}
	if len(sources) == 0 && len(chunks) == 0 {
		return fmt.Errorf("no fixtures found under %s", *dir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("pgx pool: %w", err)
	}
	defer pool.Close()

	// Sources first so the chunk FK always resolves.
	for _, s := range sources {
		if err := upsertSource(ctx, pool, s); err != nil {
			return fmt.Errorf("upsert source %s: %w", s.Slug, err)
		}
	}
	for _, c := range chunks {
		if c.Status == "" {
			c.Status = "draft"
		}
		if c.Language == "" {
			c.Language = "en"
		}
		if err := upsertChunk(ctx, pool, c); err != nil {
			return fmt.Errorf("upsert chunk %s: %w", c.Slug, err)
		}
	}
	fmt.Printf("upserted %d source(s) + %d chunk(s)\n", len(sources), len(chunks))
	return nil
}

// readAll iterates a directory of *.json files and decodes each into T.
// Generic so the same walk handles source and chunk fixtures.
func readAll[T any](dir string) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]T, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", path, err)
		}
		body, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var v T
		if err := json.Unmarshal(body, &v); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		out = append(out, v)
	}
	return out, nil
}

const upsertSourceSQL = `
INSERT INTO knowledge_source (
    slug, display_name, medium, publisher, authority, url, language,
    licence, published_at, notes, updated_at
)
VALUES ($1, $2, $3::knowledge_medium, NULLIF($4, ''), $5::authority_level,
        NULLIF($6, ''), NULLIF($7, ''),
        NULLIF($8, ''), NULLIF($9, '')::date, NULLIF($10, ''), now())
ON CONFLICT (slug) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    medium = EXCLUDED.medium,
    publisher = EXCLUDED.publisher,
    authority = EXCLUDED.authority,
    url = EXCLUDED.url,
    language = EXCLUDED.language,
    licence = EXCLUDED.licence,
    published_at = EXCLUDED.published_at,
    notes = EXCLUDED.notes,
    updated_at = now()`

func upsertSource(ctx context.Context, pool *pgxpool.Pool, s sourceFile) error {
	_, err := pool.Exec(ctx, upsertSourceSQL,
		s.Slug, s.DisplayName, s.Medium, s.Publisher, s.Authority,
		s.URL, s.Language,
		s.Licence, s.PublishedAt, s.Notes,
	)
	return err
}

const upsertChunkSQL = `
INSERT INTO knowledge_chunk (
    slug, source_slug, chunk_idx, language, title, body,
    body_translated, entity_refs, authority,
    applies_to_aez_codes, applies_to_countries, topic_tags,
    confidence, quote, status, field_provenance, updated_at
)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6,
        $7::jsonb, $8::jsonb, $9::authority_level,
        $10, $11, $12,
        $13, NULLIF($14, ''), $15::record_status, $16::jsonb, now())
ON CONFLICT (slug) DO UPDATE SET
    source_slug = EXCLUDED.source_slug,
    chunk_idx = EXCLUDED.chunk_idx,
    language = EXCLUDED.language,
    title = EXCLUDED.title,
    body = EXCLUDED.body,
    body_translated = EXCLUDED.body_translated,
    entity_refs = EXCLUDED.entity_refs,
    authority = EXCLUDED.authority,
    applies_to_aez_codes = EXCLUDED.applies_to_aez_codes,
    applies_to_countries = EXCLUDED.applies_to_countries,
    topic_tags = EXCLUDED.topic_tags,
    confidence = EXCLUDED.confidence,
    quote = EXCLUDED.quote,
    status = EXCLUDED.status,
    field_provenance = EXCLUDED.field_provenance,
    updated_at = now()`

func upsertChunk(ctx context.Context, pool *pgxpool.Pool, c chunkFile) error {
	bodyTrans, _ := json.Marshal(c.BodyTranslated)
	entityRefs, _ := json.Marshal(orEmptyArray(c.EntityRefs))
	provenance, _ := json.Marshal(c.FieldProvenance)
	countries := c.AppliesToCountries
	if len(countries) == 0 {
		countries = []string{"LK"}
	}
	_, err := pool.Exec(ctx, upsertChunkSQL,
		c.Slug, c.SourceSlug, c.ChunkIdx, c.Language, c.Title, c.Body,
		bodyTrans, entityRefs, c.Authority,
		c.AppliesToAEZCodes, countries, c.TopicTags,
		c.Confidence, c.Quote, c.Status, provenance,
	)
	return err
}

func orEmptyArray(v []any) []any {
	if v == nil {
		return []any{}
	}
	return v
}
