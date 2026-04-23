package knowledge

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/goyama/api/internal/review"
)

// PgxRepo serves knowledge_source + knowledge_chunk from Postgres.
// Implements both the public Repository (read-only, published-only)
// and the AdminRepo (status lifecycle) interfaces so the same
// instance handles both paths.
type PgxRepo struct {
	pool *pgxpool.Pool
}

// NewPgxRepo returns a Postgres-backed knowledge repo.
func NewPgxRepo(pool *pgxpool.Pool) *PgxRepo { return &PgxRepo{pool: pool} }

const chunkSelect = `
SELECT slug, source_slug, COALESCE(chunk_idx, 0),
       language, COALESCE(title, ''), body,
       COALESCE(body_translated, '{}'::jsonb),
       COALESCE(entity_refs, '[]'::jsonb),
       authority::text,
       COALESCE(applies_to_aez_codes, '{}'),
       COALESCE(applies_to_countries, '{LK}'),
       COALESCE(topic_tags, '{}'),
       confidence,
       COALESCE(quote, ''),
       status::text,
       COALESCE(field_provenance, '{}'::jsonb),
       COALESCE(reviewed_by, ''), reviewed_at, COALESCE(review_notes, '')
FROM knowledge_chunk
`

func scanChunk(rows pgx.Row) (Chunk, error) {
	var (
		c          Chunk
		entityRefs []map[string]any
		reviewedAt *time.Time
	)
	if err := rows.Scan(
		&c.Slug, &c.SourceSlug, &c.ChunkIdx,
		&c.Language, &c.Title, &c.Body,
		&c.BodyTranslated,
		&entityRefs,
		&c.Authority,
		&c.AppliesToAEZCodes,
		&c.AppliesToCountries,
		&c.TopicTags,
		&c.Confidence,
		&c.Quote,
		&c.Status,
		&c.FieldProvenance,
		&c.ReviewedBy, &reviewedAt, &c.ReviewNotes,
	); err != nil {
		return c, fmt.Errorf("scan chunk: %w", err)
	}
	if reviewedAt != nil {
		c.ReviewedAt = reviewedAt.UTC().Format(time.RFC3339)
	}
	c.EntityRefs = make([]EntityRef, 0, len(entityRefs))
	for _, ref := range entityRefs {
		var er EntityRef
		if v, ok := ref["type"].(string); ok {
			er.Type = v
		}
		if v, ok := ref["slug"].(string); ok {
			er.Slug = v
		}
		c.EntityRefs = append(c.EntityRefs, er)
	}
	return c, nil
}

// ListByEntity returns published-only chunks attached to one entity.
// Filters with the JSONB containment operator on entity_refs so the
// expression-index in migration 0010 lights up.
func (r *PgxRepo) ListByEntity(ctx context.Context, entityType, entitySlug string) ([]Chunk, error) {
	const sql = chunkSelect + `
WHERE entity_refs @> $1::jsonb
  AND status = 'published'
ORDER BY source_slug, chunk_idx`
	containment := fmt.Sprintf(`[{"type":"%s","slug":"%s"}]`,
		strings.ReplaceAll(entityType, `"`, ``),
		strings.ReplaceAll(entitySlug, `"`, ``),
	)
	rows, err := r.pool.Query(ctx, sql, containment)
	if err != nil {
		return nil, fmt.Errorf("list chunks by entity: %w", err)
	}
	defer rows.Close()
	out := make([]Chunk, 0, 4)
	for rows.Next() {
		c, err := scanChunk(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetSource reads a knowledge_source row by slug.
func (r *PgxRepo) GetSource(ctx context.Context, slug string) (Source, error) {
	const sql = `
SELECT slug, display_name, medium::text,
       COALESCE(publisher, ''),
       authority::text,
       COALESCE(url, ''),
       COALESCE(language, ''),
       COALESCE(licence, ''),
       published_at
FROM knowledge_source
WHERE slug = $1`
	var (
		s           Source
		publishedAt *time.Time
	)
	err := r.pool.QueryRow(ctx, sql, slug).Scan(
		&s.Slug, &s.DisplayName, &s.Medium,
		&s.Publisher,
		&s.Authority,
		&s.URL,
		&s.Language,
		&s.Licence,
		&publishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Source{}, ErrNotFound
		}
		return Source{}, fmt.Errorf("get source: %w", err)
	}
	if publishedAt != nil {
		s.PublishedAt = publishedAt.UTC().Format("2006-01-02")
	}
	return s, nil
}

// ListByStatus powers the admin queue. Returns chunks at the given
// status across all sources / entities; empty status means all.
func (r *PgxRepo) ListByStatus(ctx context.Context, status string) ([]Chunk, error) {
	sql := chunkSelect
	args := []any{}
	if s := strings.TrimSpace(status); s != "" {
		sql += "WHERE status = $1::record_status\n"
		args = append(args, s)
	}
	sql += "ORDER BY updated_at DESC, slug"
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list chunks by status: %w", err)
	}
	defer rows.Close()
	out := make([]Chunk, 0, 16)
	for rows.Next() {
		c, err := scanChunk(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Get returns a single chunk by slug for the admin detail page.
func (r *PgxRepo) Get(ctx context.Context, slug string) (Chunk, error) {
	const sql = chunkSelect + "WHERE slug = $1"
	row := r.pool.QueryRow(ctx, sql, slug)
	c, err := scanChunk(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Chunk{}, ErrNotFound
		}
		return Chunk{}, err
	}
	return c, nil
}

// SetStatus updates a chunk's review status.
func (r *PgxRepo) SetStatus(ctx context.Context, slug string, u review.StatusUpdate) error {
	const sql = `
UPDATE knowledge_chunk
SET status = $2::record_status,
    reviewed_by = $3,
    reviewed_at = now(),
    review_notes = NULLIF($4, ''),
    updated_at = now()
WHERE slug = $1`
	tag, err := r.pool.Exec(ctx, sql, slug, u.Status, u.ReviewedBy, u.Notes)
	if err != nil {
		return fmt.Errorf("update chunk status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

var _ Repository = (*PgxRepo)(nil)
var _ AdminRepo = (*PgxRepo)(nil)
