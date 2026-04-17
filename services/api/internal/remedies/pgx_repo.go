package remedies

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/goyama/api/internal/review"
)

// PgxRepo is the production implementation of remedies.Repository.
type PgxRepo struct {
	pool *pgxpool.Pool
}

// NewPgxRepo returns a PgxRepo over the given pool.
func NewPgxRepo(pool *pgxpool.Pool) *PgxRepo { return &PgxRepo{pool: pool} }

// selectColumns is shared by list and single-get queries. The chemical-
// remedy-specific columns are included and come back nullable; non-
// chemical remedies leave them NULL.
const selectColumns = `
	r.slug, r.type::text,
	COALESCE(r.target_disease_slugs, '{}'),
	COALESCE(r.target_pest_slugs, '{}'),
	COALESCE(r.applicable_crop_slugs, '{}'),
	COALESCE(r.active_ingredient, ''),
	COALESCE(r.concentration, ''),
	COALESCE(r.formulation, ''),
	COALESCE(r.doa_registration_no, ''),
	COALESCE(r.dosage, ''),
	COALESCE(r.application_method, ''),
	COALESCE(r.frequency, ''),
	r.pre_harvest_interval_days,
	r.re_entry_interval_hours,
	COALESCE(r.who_hazard_class, ''),
	COALESCE(r.effectiveness, ''),
	COALESCE(r.cost_tier, ''),
	r.organic_compatible,
	r.attrs, r.field_provenance, r.status::text,
	COALESCE(r.reviewed_by, ''), r.reviewed_at, COALESCE(r.review_notes, '')
`

func scanRemedy(rows interface{ Scan(dest ...any) error }) (Remedy, error) {
	var r Remedy
	var reviewedAt *time.Time
	if err := rows.Scan(
		&r.Slug, &r.Type,
		&r.TargetDiseaseSlugs, &r.TargetPestSlugs, &r.ApplicableCropSlugs,
		&r.ActiveIngredient, &r.Concentration, &r.Formulation, &r.DoaRegistrationNo,
		&r.Dosage, &r.ApplicationMethod, &r.Frequency,
		&r.PreHarvestIntervalD, &r.ReEntryIntervalHours,
		&r.WhoHazardClass, &r.Effectiveness, &r.CostTier, &r.OrganicCompatible,
		&r.Attrs, &r.FieldProvenance, &r.Status,
		&r.ReviewedBy, &reviewedAt, &r.ReviewNotes,
	); err != nil {
		return r, err
	}
	if reviewedAt != nil {
		r.ReviewedAt = reviewedAt.UTC().Format(time.RFC3339)
	}
	return r, nil
}

// ListByStatus returns every remedy matching the status, ordered by slug.
func (r *PgxRepo) ListByStatus(ctx context.Context, status string) ([]Remedy, error) {
	sql := `
SELECT ` + selectColumns + `
FROM remedy r
WHERE r.status = $1::record_status
ORDER BY r.slug`
	rows, err := r.pool.Query(ctx, sql, status)
	if err != nil {
		return nil, fmt.Errorf("list remedies by status: %w", err)
	}
	defer rows.Close()
	out := make([]Remedy, 0, 16)
	for rows.Next() {
		rem, err := scanRemedy(rows)
		if err != nil {
			return nil, fmt.Errorf("scan remedy: %w", err)
		}
		out = append(out, rem)
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

// Get returns a single remedy by slug with i18n fields populated.
func (r *PgxRepo) Get(ctx context.Context, slug string) (Remedy, error) {
	sql := `
SELECT ` + selectColumns + `
FROM remedy r
WHERE r.slug = $1
ORDER BY r.version DESC
LIMIT 1`
	row := r.pool.QueryRow(ctx, sql, slug)
	rem, err := scanRemedy(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Remedy{}, ErrNotFound
		}
		return Remedy{}, fmt.Errorf("get remedy %q: %w", slug, err)
	}
	single := []Remedy{rem}
	if err := r.loadI18n(ctx, single); err != nil {
		return Remedy{}, err
	}
	return single[0], nil
}

// loadI18n populates name / description / instructions / safety_notes in
// place. Remedies carry four i18n fields — one more than pests — so the
// safety-notes quote on a chemical record is immediately available to
// the review UI without a follow-up round-trip.
func (r *PgxRepo) loadI18n(ctx context.Context, items []Remedy) error {
	if len(items) == 0 {
		return nil
	}
	slugs := make([]string, len(items))
	for i, rem := range items {
		slugs[i] = rem.Slug
	}

	buckets := map[string]map[string]map[string]string{
		"name":         {},
		"description":  {},
		"instructions": {},
		"safety_notes": {},
	}

	rows, err := r.pool.Query(ctx, `
		SELECT entity_slug, field, locale, value
		FROM translation
		WHERE entity_type = 'remedy' AND entity_slug = ANY($1)
		  AND field IN ('name', 'description', 'instructions', 'safety_notes')
	`, slugs)
	if err != nil {
		return fmt.Errorf("load remedy translations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var slug, field, locale, value string
		if err := rows.Scan(&slug, &field, &locale, &value); err != nil {
			return fmt.Errorf("scan translation: %w", err)
		}
		bucket, ok := buckets[field]
		if !ok {
			continue
		}
		if bucket[slug] == nil {
			bucket[slug] = map[string]string{}
		}
		bucket[slug][locale] = value
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate translations: %w", err)
	}

	for i := range items {
		items[i].Name = buckets["name"][items[i].Slug]
		items[i].Description = buckets["description"][items[i].Slug]
		items[i].Instructions = buckets["instructions"][items[i].Slug]
		items[i].SafetyNotes = buckets["safety_notes"][items[i].Slug]
	}
	return nil
}

// SetStatus promotes / rejects a remedy, stamping the reviewer + notes.
func (r *PgxRepo) SetStatus(ctx context.Context, slug string, u review.StatusUpdate) error {
	const sql = `
UPDATE remedy
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
