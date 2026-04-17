package pests

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/goyama/api/internal/review"
)

// PgxRepo is the production implementation of pests.Repository.
type PgxRepo struct {
	pool *pgxpool.Pool
}

// NewPgxRepo returns a PgxRepo over the given pool.
func NewPgxRepo(pool *pgxpool.Pool) *PgxRepo { return &PgxRepo{pool: pool} }

// selectColumns is shared by list and single-get queries.
const selectColumns = `
	p.slug, p.scientific_name, p.kingdom,
	COALESCE(p.affected_crop_slugs, '{}'),
	COALESCE(p.life_stages, '{}'),
	COALESCE(p.feeding_type, '{}'),
	p.favored_conditions, p.attrs, p.field_provenance, p.status::text,
	COALESCE(p.reviewed_by, ''), p.reviewed_at, COALESCE(p.review_notes, '')
`

func scanPest(rows interface{ Scan(dest ...any) error }) (Pest, error) {
	var p Pest
	var reviewedAt *time.Time
	if err := rows.Scan(
		&p.Slug, &p.ScientificName, &p.Kingdom,
		&p.AffectedCropSlugs, &p.LifeStages, &p.FeedingType,
		&p.FavoredConditions, &p.Attrs, &p.FieldProvenance, &p.Status,
		&p.ReviewedBy, &reviewedAt, &p.ReviewNotes,
	); err != nil {
		return p, err
	}
	if reviewedAt != nil {
		p.ReviewedAt = reviewedAt.UTC().Format(time.RFC3339)
	}
	return p, nil
}

// ListByStatus returns every pest matching the status, ordered by slug.
func (r *PgxRepo) ListByStatus(ctx context.Context, status string) ([]Pest, error) {
	sql := `
SELECT ` + selectColumns + `
FROM pest p
WHERE p.status = $1::record_status
ORDER BY p.slug`
	rows, err := r.pool.Query(ctx, sql, status)
	if err != nil {
		return nil, fmt.Errorf("list pests by status: %w", err)
	}
	defer rows.Close()
	out := make([]Pest, 0, 32)
	for rows.Next() {
		p, err := scanPest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pest: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	if len(out) > 0 {
		if err := r.loadI18n(ctx, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Get returns a single pest by slug with i18n fields populated.
func (r *PgxRepo) Get(ctx context.Context, slug string) (Pest, error) {
	sql := `
SELECT ` + selectColumns + `
FROM pest p
WHERE p.slug = $1
ORDER BY p.version DESC
LIMIT 1`
	row := r.pool.QueryRow(ctx, sql, slug)
	p, err := scanPest(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Pest{}, ErrNotFound
		}
		return Pest{}, fmt.Errorf("get pest %q: %w", slug, err)
	}
	single := []Pest{p}
	if err := r.loadI18n(ctx, single); err != nil {
		return Pest{}, err
	}
	return single[0], nil
}

// loadI18n populates names / description / economic_threshold / aliases
// in place. One query per scope — cheap because translation + entity_alias
// both have (entity_type, entity_slug) indexes.
func (r *PgxRepo) loadI18n(ctx context.Context, items []Pest) error {
	if len(items) == 0 {
		return nil
	}
	slugs := make([]string, len(items))
	for i, p := range items {
		slugs[i] = p.Slug
	}

	names := map[string]map[string]string{}
	desc := map[string]map[string]string{}
	threshold := map[string]map[string]string{}
	aliases := map[string][]string{}

	tlRows, err := r.pool.Query(ctx, `
		SELECT entity_slug, field, locale, value
		FROM translation
		WHERE entity_type = 'pest' AND entity_slug = ANY($1)
		  AND field IN ('names', 'description', 'economic_threshold')
	`, slugs)
	if err != nil {
		return fmt.Errorf("load pest translations: %w", err)
	}
	defer tlRows.Close()
	for tlRows.Next() {
		var slug, field, locale, value string
		if err := tlRows.Scan(&slug, &field, &locale, &value); err != nil {
			return fmt.Errorf("scan translation: %w", err)
		}
		var bucket map[string]map[string]string
		switch field {
		case "names":
			bucket = names
		case "description":
			bucket = desc
		case "economic_threshold":
			bucket = threshold
		default:
			continue
		}
		if bucket[slug] == nil {
			bucket[slug] = map[string]string{}
		}
		bucket[slug][locale] = value
	}
	if err := tlRows.Err(); err != nil {
		return fmt.Errorf("iterate translations: %w", err)
	}

	aliasRows, err := r.pool.Query(ctx, `
		SELECT entity_slug, alias
		FROM entity_alias
		WHERE entity_type = 'pest' AND entity_slug = ANY($1)
	`, slugs)
	if err != nil {
		return fmt.Errorf("load pest aliases: %w", err)
	}
	defer aliasRows.Close()
	for aliasRows.Next() {
		var slug, alias string
		if err := aliasRows.Scan(&slug, &alias); err != nil {
			return fmt.Errorf("scan alias: %w", err)
		}
		aliases[slug] = append(aliases[slug], alias)
	}
	if err := aliasRows.Err(); err != nil {
		return fmt.Errorf("iterate aliases: %w", err)
	}

	for i := range items {
		items[i].Names = names[items[i].Slug]
		items[i].Description = desc[items[i].Slug]
		items[i].EconomicThreshold = threshold[items[i].Slug]
		items[i].Aliases = aliases[items[i].Slug]
	}
	return nil
}

// SetStatus promotes / rejects a pest, stamping the reviewer + notes.
func (r *PgxRepo) SetStatus(ctx context.Context, slug string, u review.StatusUpdate) error {
	const sql = `
UPDATE pest
SET status       = $2::record_status,
    reviewed_by  = $3,
    reviewed_at  = now(),
    review_notes = NULLIF($4, ''),
    updated_at   = now()
WHERE slug = $1`
	tag, err := r.pool.Exec(ctx, sql, slug, u.Status, u.ReviewedBy, u.Notes)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
