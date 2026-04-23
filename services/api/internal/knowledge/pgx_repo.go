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

// ListUnembedded returns slug + body of every chunk that doesn't yet
// have a content_embedding. The backfill worker iterates over this
// set; the chunks endpoint doesn't filter on it because once a chunk
// has an embedding the cosine query is preferred.
func (r *PgxRepo) ListUnembedded(ctx context.Context, limit int) ([]ChunkBody, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	const sql = `
SELECT slug, COALESCE(title, ''), body
FROM knowledge_chunk
WHERE content_embedding IS NULL
ORDER BY updated_at DESC
LIMIT $1`
	rows, err := r.pool.Query(ctx, sql, limit)
	if err != nil {
		return nil, fmt.Errorf("list unembedded chunks: %w", err)
	}
	defer rows.Close()
	out := make([]ChunkBody, 0, limit)
	for rows.Next() {
		var c ChunkBody
		if err := rows.Scan(&c.Slug, &c.Title, &c.Body); err != nil {
			return nil, fmt.Errorf("scan chunk body: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateEmbedding writes the content_embedding column for a chunk.
// pgvector accepts the literal "[v1,v2,...]" text form when the column
// type is `vector(N)`; we format that here rather than depending on a
// separate pgvector-go binding.
func (r *PgxRepo) UpdateEmbedding(ctx context.Context, slug string, vec []float32) error {
	if len(vec) == 0 {
		return fmt.Errorf("empty embedding")
	}
	const sql = `
UPDATE knowledge_chunk
SET content_embedding = $2::vector,
    updated_at = now()
WHERE slug = $1`
	tag, err := r.pool.Exec(ctx, sql, slug, vectorLiteral(vec))
	if err != nil {
		return fmt.Errorf("update embedding: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Search returns the top-k published chunks ranked by cosine
// similarity to the query vector, with optional crop / aez / country
// filters. Uses pgvector's `<=>` cosine-distance operator (smaller =
// closer); the HNSW index from migration 0010 lights up.
func (r *PgxRepo) Search(ctx context.Context, q SearchQuery) ([]SearchHit, error) {
	if len(q.QueryVector) == 0 {
		return nil, fmt.Errorf("empty query vector")
	}
	if q.Limit <= 0 || q.Limit > 50 {
		q.Limit = 8
	}

	args := []any{vectorLiteral(q.QueryVector)}
	conds := []string{"content_embedding IS NOT NULL", "status = 'published'"}
	if s := strings.TrimSpace(q.CropSlug); s != "" {
		args = append(args, fmt.Sprintf(`[{"type":"crop","slug":"%s"}]`,
			strings.ReplaceAll(s, `"`, ``)))
		conds = append(conds, fmt.Sprintf("entity_refs @> $%d::jsonb", len(args)))
	}
	if len(q.AEZCodes) > 0 {
		args = append(args, q.AEZCodes)
		conds = append(conds,
			fmt.Sprintf("(applies_to_aez_codes IS NULL OR cardinality(applies_to_aez_codes) = 0 OR applies_to_aez_codes && $%d)", len(args)))
	}
	if len(q.Countries) > 0 {
		args = append(args, q.Countries)
		conds = append(conds,
			fmt.Sprintf("(applies_to_countries IS NULL OR applies_to_countries && $%d)", len(args)))
	}
	args = append(args, q.Limit)

	sql := chunkSelect +
		"WHERE " + strings.Join(conds, " AND ") +
		"\nORDER BY content_embedding <=> $1::vector\nLIMIT $" + fmt.Sprintf("%d", len(args))

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer rows.Close()
	out := make([]SearchHit, 0, q.Limit)
	for rows.Next() {
		c, err := scanChunk(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, SearchHit{Chunk: c})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hits: %w", err)
	}
	if err := r.scoreHits(ctx, q.QueryVector, out); err != nil {
		return nil, err
	}
	return out, nil
}

// scoreHits annotates each hit with its cosine similarity (1 - cosine
// distance, range [-1, 1]). Done as a second tiny query to keep the
// generic scanChunk row reader a no-arg stable function.
func (r *PgxRepo) scoreHits(ctx context.Context, vec []float32, hits []SearchHit) error {
	if len(hits) == 0 {
		return nil
	}
	slugs := make([]string, len(hits))
	for i, h := range hits {
		slugs[i] = h.Chunk.Slug
	}
	const sql = `
SELECT slug, 1 - (content_embedding <=> $1::vector) AS score
FROM knowledge_chunk
WHERE slug = ANY($2)`
	rows, err := r.pool.Query(ctx, sql, vectorLiteral(vec), slugs)
	if err != nil {
		return fmt.Errorf("score hits: %w", err)
	}
	defer rows.Close()
	scoreBy := make(map[string]float64, len(hits))
	for rows.Next() {
		var slug string
		var score float64
		if err := rows.Scan(&slug, &score); err != nil {
			return fmt.Errorf("scan score: %w", err)
		}
		scoreBy[slug] = score
	}
	for i := range hits {
		hits[i].Score = scoreBy[hits[i].Chunk.Slug]
	}
	return nil
}

// vectorLiteral formats a Go slice as the pgvector text literal
// "[v1,v2,...,vN]". Cheap and avoids pulling in pgvector-go.
func vectorLiteral(v []float32) string {
	var sb strings.Builder
	sb.Grow(8 * len(v))
	sb.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("%g", x))
	}
	sb.WriteByte(']')
	return sb.String()
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
var _ EmbeddingRepo = (*PgxRepo)(nil)
var _ SearchRepo = (*PgxRepo)(nil)
