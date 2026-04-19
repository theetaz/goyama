package media

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/goyama/api/internal/review"
)

// PgxRepo serves media records from Postgres.
type PgxRepo struct {
	pool *pgxpool.Pool
}

// NewPgxRepo returns a repo over the given pool.
func NewPgxRepo(pool *pgxpool.Pool) *PgxRepo { return &PgxRepo{pool: pool} }

const baseSelect = `
SELECT slug, type::text, hosting::text,
       COALESCE(url, ''), COALESCE(external_url, ''),
       COALESCE(credit, ''), licence,
       COALESCE(related, '{}'::jsonb),
       COALESCE(tags, '{}'),
       status::text,
       COALESCE(reviewed_by, ''), reviewed_at, COALESCE(review_notes, '')
FROM media
`

// ListByEntity returns every media record attached to the given entity.
// status='' returns all statuses; otherwise filters to that one. Most
// recent first.
func (r *PgxRepo) ListByEntity(ctx context.Context, entityType, entitySlug, status string) ([]Media, error) {
	sql := baseSelect + `
WHERE related->>'entity_type' = $1
  AND related->>'entity_slug' = $2
`
	args := []any{entityType, entitySlug}
	if s := strings.TrimSpace(status); s != "" {
		sql += " AND status = $3::record_status\n"
		args = append(args, s)
	}
	sql += " ORDER BY created_at DESC, slug"

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list media: %w", err)
	}
	defer rows.Close()

	out := make([]Media, 0, 8)
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate media: %w", err)
	}
	return out, nil
}

// Get returns a single media record by slug.
func (r *PgxRepo) Get(ctx context.Context, slug string) (Media, error) {
	rows, err := r.pool.Query(ctx, baseSelect+"WHERE slug = $1 ORDER BY version DESC LIMIT 1", slug)
	if err != nil {
		return Media{}, fmt.Errorf("get media: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return Media{}, ErrNotFound
	}
	return scanMedia(rows)
}

// Attach creates a fresh external-link media record. Idempotent — if a
// record already exists for (entity, external_url), returns it.
func (r *PgxRepo) Attach(ctx context.Context, in AttachInput) (Media, error) {
	in.EntityType = strings.TrimSpace(in.EntityType)
	in.EntitySlug = strings.TrimSpace(in.EntitySlug)
	in.ExternalURL = strings.TrimSpace(in.ExternalURL)
	in.Licence = strings.TrimSpace(in.Licence)
	if in.Type == "" {
		in.Type = "image"
	}
	if in.EntityType == "" || in.EntitySlug == "" || in.ExternalURL == "" || in.Licence == "" {
		return Media{}, errors.New("entity_type, entity_slug, external_url, and licence are all required")
	}
	if _, err := url.ParseRequestURI(in.ExternalURL); err != nil {
		return Media{}, fmt.Errorf("external_url is not a valid URL: %w", err)
	}

	// Idempotency: same external URL on the same entity returns the
	// existing record verbatim. The sha1 of the URL is the slug seed —
	// stable, short, and avoids accidental collisions with other media.
	seed := sha1.Sum([]byte(in.EntityType + "|" + in.EntitySlug + "|" + in.ExternalURL))
	slug := fmt.Sprintf("%s-img-%s", in.EntitySlug, hex.EncodeToString(seed[:6]))

	if existing, err := r.Get(ctx, slug); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Media{}, err
	}

	related := map[string]any{
		"entity_type": in.EntityType,
		"entity_slug": in.EntitySlug,
	}
	relatedJSON, _ := json.Marshal(related)

	prov := map[string]any{
		"attached_by": in.CreatedBy,
		"attached_at": time.Now().UTC().Format(time.RFC3339),
		"source_url":  in.ExternalURL,
	}
	provJSON, _ := json.Marshal(prov)

	const insertSQL = `
INSERT INTO media (slug, version, status, type, hosting, external_url, credit, licence, related, tags, field_provenance)
VALUES ($1, 1, 'in_review', $2::media_type, 'external_link', $3, NULLIF($4, ''), $5, $6::jsonb, $7, $8::jsonb)
`
	if _, err := r.pool.Exec(ctx, insertSQL,
		slug, in.Type, in.ExternalURL, in.Credit, in.Licence, relatedJSON, in.Tags, provJSON,
	); err != nil {
		return Media{}, fmt.Errorf("insert media: %w", err)
	}
	return r.Get(ctx, slug)
}

// SetStatus updates review status — same lifecycle as the other admin
// queues. The review package validates the transition before calling
// here, so this is a straight UPDATE.
func (r *PgxRepo) SetStatus(ctx context.Context, slug string, u review.StatusUpdate) error {
	const sql = `
UPDATE media
SET status       = $2::record_status,
    reviewed_by  = $3,
    reviewed_at  = now(),
    review_notes = NULLIF($4, ''),
    updated_at   = now()
WHERE slug = $1`
	tag, err := r.pool.Exec(ctx, sql, slug, u.Status, u.ReviewedBy, u.Notes)
	if err != nil {
		return fmt.Errorf("update media status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanMedia(rows pgx.Row) (Media, error) {
	var (
		m          Media
		related    map[string]any
		reviewedAt *time.Time
	)
	if err := rows.Scan(
		&m.Slug, &m.Type, &m.Hosting,
		&m.URL, &m.ExternalURL,
		&m.Credit, &m.Licence,
		&related, &m.Tags,
		&m.Status,
		&m.ReviewedBy, &reviewedAt, &m.ReviewNotes,
	); err != nil {
		return m, fmt.Errorf("scan media: %w", err)
	}
	if related != nil {
		if v, ok := related["entity_type"].(string); ok {
			m.EntityType = v
		}
		if v, ok := related["entity_slug"].(string); ok {
			m.EntitySlug = v
		}
	}
	if reviewedAt != nil {
		m.ReviewedAt = reviewedAt.UTC().Format(time.RFC3339)
	}
	return m, nil
}
