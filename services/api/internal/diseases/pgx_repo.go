package diseases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxRepo is the production implementation of diseases.Repository.
type PgxRepo struct {
	pool *pgxpool.Pool
}

// NewPgxRepo returns a PgxRepo over the given pool. Ownership stays with
// the caller.
func NewPgxRepo(pool *pgxpool.Pool) *PgxRepo { return &PgxRepo{pool: pool} }

// selectColumns is the column list shared by the list and single-get
// queries. `confused_with` is coalesced to an empty array so a missing
// column value doesn't blow up pgx's array scan.
const selectColumns = `
	d.slug, d.scientific_name, d.causal_organism, COALESCE(d.causal_species, ''),
	COALESCE(d.severity, ''),
	COALESCE(d.affected_crop_slugs, '{}'),
	COALESCE(d.affected_parts, '{}'),
	COALESCE(d.transmission, '{}'),
	COALESCE(d.confused_with, '{}'),
	d.favored_conditions, d.attrs, d.field_provenance, d.status::text,
	COALESCE(d.reviewed_by, ''), d.reviewed_at, COALESCE(d.review_notes, '')
`

func scanDisease(rows interface{ Scan(dest ...any) error }) (Disease, error) {
	var d Disease
	var reviewedAt *time.Time
	if err := rows.Scan(
		&d.Slug, &d.ScientificName, &d.CausalOrganism, &d.CausalSpecies,
		&d.Severity,
		&d.AffectedCropSlugs, &d.AffectedParts, &d.Transmission, &d.ConfusedWith,
		&d.FavoredConditions, &d.Attrs, &d.FieldProvenance, &d.Status,
		&d.ReviewedBy, &reviewedAt, &d.ReviewNotes,
	); err != nil {
		return d, err
	}
	if reviewedAt != nil {
		d.ReviewedAt = reviewedAt.UTC().Format(time.RFC3339)
	}
	return d, nil
}

// ListByStatus returns every disease matching the given status, ordered by
// slug. Intended for the admin review queue — the farmer-facing disease
// surface will come later and likely filter to status='published'.
func (r *PgxRepo) ListByStatus(ctx context.Context, status string) ([]Disease, error) {
	sql := `
SELECT ` + selectColumns + `
FROM disease d
WHERE d.status = $1::record_status
ORDER BY d.slug`
	rows, err := r.pool.Query(ctx, sql, status)
	if err != nil {
		return nil, fmt.Errorf("list diseases by status: %w", err)
	}
	defer rows.Close()
	out := make([]Disease, 0, 32)
	for rows.Next() {
		d, err := scanDisease(rows)
		if err != nil {
			return nil, fmt.Errorf("scan disease: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	// Enrich with names / aliases / description via a single follow-up
	// query so the list page can show the en title without a second hit
	// per row.
	if len(out) > 0 {
		if err := r.loadI18n(ctx, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Get returns a single disease by slug with i18n fields populated.
func (r *PgxRepo) Get(ctx context.Context, slug string) (Disease, error) {
	sql := `
SELECT ` + selectColumns + `
FROM disease d
WHERE d.slug = $1
ORDER BY d.version DESC
LIMIT 1`
	row := r.pool.QueryRow(ctx, sql, slug)
	d, err := scanDisease(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Disease{}, ErrNotFound
		}
		return Disease{}, fmt.Errorf("get disease %q: %w", slug, err)
	}
	// loadI18n mutates the slice in place, so we pass a 1-element slice
	// and return its [0] element with the i18n fields populated.
	single := []Disease{d}
	if err := r.loadI18n(ctx, single); err != nil {
		return Disease{}, err
	}
	return single[0], nil
}

// loadI18n populates Names, Aliases, and Description in place for the
// supplied diseases. One query per field — cheap because the translation
// table is indexed on (entity_type, entity_slug).
func (r *PgxRepo) loadI18n(ctx context.Context, items []Disease) error {
	if len(items) == 0 {
		return nil
	}
	slugs := make([]string, len(items))
	for i, d := range items {
		slugs[i] = d.Slug
	}

	namesByEntity := map[string]map[string]string{}
	descByEntity := map[string]map[string]string{}
	aliasesByEntity := map[string][]string{}

	tlRows, err := r.pool.Query(ctx, `
		SELECT entity_slug, field, locale, value
		FROM translation
		WHERE entity_type = 'disease' AND entity_slug = ANY($1)
		  AND field IN ('names', 'description')
	`, slugs)
	if err != nil {
		return fmt.Errorf("load disease translations: %w", err)
	}
	defer tlRows.Close()
	for tlRows.Next() {
		var slug, field, locale, value string
		if err := tlRows.Scan(&slug, &field, &locale, &value); err != nil {
			return fmt.Errorf("scan translation: %w", err)
		}
		switch field {
		case "names":
			if namesByEntity[slug] == nil {
				namesByEntity[slug] = map[string]string{}
			}
			namesByEntity[slug][locale] = value
		case "description":
			if descByEntity[slug] == nil {
				descByEntity[slug] = map[string]string{}
			}
			descByEntity[slug][locale] = value
		}
	}
	if err := tlRows.Err(); err != nil {
		return fmt.Errorf("iterate translations: %w", err)
	}

	aliasRows, err := r.pool.Query(ctx, `
		SELECT entity_slug, alias
		FROM entity_alias
		WHERE entity_type = 'disease' AND entity_slug = ANY($1)
	`, slugs)
	if err != nil {
		return fmt.Errorf("load disease aliases: %w", err)
	}
	defer aliasRows.Close()
	for aliasRows.Next() {
		var slug, alias string
		if err := aliasRows.Scan(&slug, &alias); err != nil {
			return fmt.Errorf("scan alias: %w", err)
		}
		aliasesByEntity[slug] = append(aliasesByEntity[slug], alias)
	}
	if err := aliasRows.Err(); err != nil {
		return fmt.Errorf("iterate aliases: %w", err)
	}

	for i := range items {
		items[i].Names = namesByEntity[items[i].Slug]
		items[i].Description = descByEntity[items[i].Slug]
		items[i].Aliases = aliasesByEntity[items[i].Slug]
	}
	return nil
}

// SetStatus promotes / rejects a disease, stamping the reviewer identity
// and optional notes on the row. Returns ErrNotFound if no row matched.
func (r *PgxRepo) SetStatus(ctx context.Context, slug string, u StatusUpdate) error {
	const sql = `
UPDATE disease
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
