package crops

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxRepo serves crops from a Postgres database using pgx. It implements the
// Repository interface and is the production data path; the JSONLRepo stays
// as a no-dependency dev fallback.
type PgxRepo struct {
	pool *pgxpool.Pool
}

// NewPgxRepo returns a PgxRepo backed by the given pgx pool. The pool
// lifecycle is owned by the caller.
func NewPgxRepo(pool *pgxpool.Pool) *PgxRepo {
	return &PgxRepo{pool: pool}
}

// cropRow mirrors the columns we select from crop + translation + entity_alias.
// Names and description are collected via jsonb_object_agg in the SQL and
// delivered as map[string]string; aliases come in as text[].
type cropRow struct {
	Slug              string
	ScientificName    string
	Family            *string
	Category          string
	LifeCycle         string
	GrowthHabit       *string
	DefaultSeason     *string
	DurationDaysMin   *int
	DurationDaysMax   *int
	ElevationMMin     *float32
	ElevationMMax     *float32
	RainfallMMMin     *float32
	RainfallMMMax     *float32
	TemperatureCMin   *float32
	TemperatureCMax   *float32
	SoilPHMin         *float32
	SoilPHMax         *float32
	YieldKgPerAcreMin *float32
	YieldKgPerAcreMax *float32
	Status            string
	Names             map[string]string
	Aliases           []string
	Description       map[string]string
	FieldProvenance   map[string]any
}

// listSQL is parameterised so a single query handles the category and query
// filters without needing separate paths. Both filters are optional; an empty
// string is treated as "no filter".
const listSQL = `
WITH latest AS (
	SELECT DISTINCT ON (slug) slug, scientific_name, category, status
	FROM crop
	ORDER BY slug, version DESC
),
names AS (
	SELECT entity_slug AS slug,
	       jsonb_object_agg(locale, value) AS value
	FROM translation
	WHERE entity_type = 'crop' AND field = 'names'
	GROUP BY entity_slug
),
aliases AS (
	SELECT entity_slug AS slug, array_agg(DISTINCT alias ORDER BY alias) AS aliases
	FROM entity_alias
	WHERE entity_type = 'crop'
	GROUP BY entity_slug
)
SELECT l.slug,
       l.scientific_name,
       l.category,
       COALESCE(n.value, '{}'::jsonb) AS names,
       COALESCE(a.aliases, '{}') AS aliases
FROM latest l
LEFT JOIN names n   ON n.slug = l.slug
LEFT JOIN aliases a ON a.slug = l.slug
WHERE ($1::text = '' OR l.category = $1::crop_category)
  AND ($2::text = '' OR (
       l.slug ILIKE '%' || $2 || '%' OR
       l.scientific_name ILIKE '%' || $2 || '%' OR
       EXISTS (SELECT 1 FROM translation t
               WHERE t.entity_type = 'crop' AND t.entity_slug = l.slug
                 AND t.value ILIKE '%' || $2 || '%') OR
       EXISTS (SELECT 1 FROM entity_alias ea
               WHERE ea.entity_type = 'crop' AND ea.entity_slug = l.slug
                 AND ea.alias ILIKE '%' || $2 || '%')
  ))
ORDER BY l.slug
OFFSET $3 LIMIT $4
`

// List returns a page of crop summaries.
func (r *PgxRepo) List(ctx context.Context, filter ListFilter) ([]Summary, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := r.pool.Query(ctx, listSQL,
		strings.TrimSpace(filter.Category),
		strings.TrimSpace(filter.Query),
		offset,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list crops: %w", err)
	}
	defer rows.Close()

	out := make([]Summary, 0, limit)
	for rows.Next() {
		var s Summary
		var names map[string]string
		var aliases []string
		if err := rows.Scan(&s.Slug, &s.ScientificName, &s.Category, &names, &aliases); err != nil {
			return nil, fmt.Errorf("scan crop: %w", err)
		}
		s.Names = names
		s.Aliases = aliases
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate crops: %w", err)
	}
	return out, nil
}

const getSQL = `
WITH latest AS (
	SELECT *
	FROM crop
	WHERE slug = $1
	ORDER BY version DESC
	LIMIT 1
),
names AS (
	SELECT jsonb_object_agg(locale, value) AS value
	FROM translation
	WHERE entity_type = 'crop' AND entity_slug = $1 AND field = 'names'
),
description AS (
	SELECT jsonb_object_agg(locale, value) AS value
	FROM translation
	WHERE entity_type = 'crop' AND entity_slug = $1 AND field = 'description'
),
aliases AS (
	SELECT array_agg(DISTINCT alias ORDER BY alias) AS aliases
	FROM entity_alias
	WHERE entity_type = 'crop' AND entity_slug = $1
)
SELECT l.slug, l.scientific_name, l.family, l.category, l.life_cycle,
       l.growth_habit, l.default_season,
       l.duration_days_min, l.duration_days_max,
       l.elevation_m_min, l.elevation_m_max,
       l.rainfall_mm_min, l.rainfall_mm_max,
       l.temperature_c_min, l.temperature_c_max,
       l.soil_ph_min, l.soil_ph_max,
       l.yield_kg_per_acre_min, l.yield_kg_per_acre_max,
       l.status,
       COALESCE((SELECT value FROM names), '{}'::jsonb) AS names,
       COALESCE((SELECT aliases FROM aliases), '{}') AS aliases,
       COALESCE((SELECT value FROM description), '{}'::jsonb) AS description,
       l.field_provenance
FROM latest l
`

// Get returns the full record for a slug or ErrNotFound.
func (r *PgxRepo) Get(ctx context.Context, slug string) (Crop, error) {
	var row cropRow
	err := r.pool.QueryRow(ctx, getSQL, slug).Scan(
		&row.Slug, &row.ScientificName, &row.Family, &row.Category, &row.LifeCycle,
		&row.GrowthHabit, &row.DefaultSeason,
		&row.DurationDaysMin, &row.DurationDaysMax,
		&row.ElevationMMin, &row.ElevationMMax,
		&row.RainfallMMMin, &row.RainfallMMMax,
		&row.TemperatureCMin, &row.TemperatureCMax,
		&row.SoilPHMin, &row.SoilPHMax,
		&row.YieldKgPerAcreMin, &row.YieldKgPerAcreMax,
		&row.Status,
		&row.Names, &row.Aliases, &row.Description,
		&row.FieldProvenance,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Crop{}, ErrNotFound
		}
		return Crop{}, fmt.Errorf("get crop %q: %w", slug, err)
	}
	return rowToCrop(row), nil
}

func rowToCrop(r cropRow) Crop {
	c := Crop{
		Slug:            r.Slug,
		ScientificName:  r.ScientificName,
		Category:        r.Category,
		LifeCycle:       r.LifeCycle,
		Names:           r.Names,
		Aliases:         r.Aliases,
		Description:     r.Description,
		Status:          r.Status,
		FieldProvenance: r.FieldProvenance,
	}
	if r.Family != nil {
		c.Family = *r.Family
	}
	if r.GrowthHabit != nil {
		c.GrowthHabit = *r.GrowthHabit
	}
	if r.DefaultSeason != nil {
		c.DefaultSeason = *r.DefaultSeason
	}
	if r.DurationDaysMin != nil || r.DurationDaysMax != nil {
		c.DurationDays = &Range{Min: deref(r.DurationDaysMin), Max: deref(r.DurationDaysMax), Unit: "days"}
	}
	if r.ElevationMMin != nil || r.ElevationMMax != nil {
		c.ElevationM = &Range{Min: derefF(r.ElevationMMin), Max: derefF(r.ElevationMMax), Unit: "m"}
	}
	if r.RainfallMMMin != nil || r.RainfallMMMax != nil {
		c.RainfallMM = &Range{Min: derefF(r.RainfallMMMin), Max: derefF(r.RainfallMMMax), Unit: "mm/year"}
	}
	if r.TemperatureCMin != nil || r.TemperatureCMax != nil {
		c.TemperatureC = &Range{Min: derefF(r.TemperatureCMin), Max: derefF(r.TemperatureCMax), Unit: "°C"}
	}
	if r.SoilPHMin != nil || r.SoilPHMax != nil {
		c.SoilPH = &Range{Min: derefF(r.SoilPHMin), Max: derefF(r.SoilPHMax)}
	}
	if r.YieldKgPerAcreMin != nil || r.YieldKgPerAcreMax != nil {
		c.ExpectedYield = &Range{Min: derefF(r.YieldKgPerAcreMin), Max: derefF(r.YieldKgPerAcreMax), Unit: "kg/acre"}
	}
	return c
}

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func derefF(p *float32) any {
	if p == nil {
		return nil
	}
	return *p
}

// reviewStepsSelect is the column list shared by the review-queue list and
// single-get queries — fewer columns than the user-facing endpoint because
// the admin surface doesn't care about title/body translations yet. They're
// assembled separately when an admin drills into one step.
const reviewStepsSelect = `
	cs.slug, cs.crop_slug, COALESCE(cs.variety_slug, ''),
	COALESCE(cs.aez_code, ''), COALESCE(cs.season::text, ''),
	cs.stage, cs.order_idx,
	cs.dap_min, cs.dap_max, cs.inputs,
	COALESCE(cs.media_slugs, '{}'), cs.status::text,
	COALESCE(cs.reviewed_by, ''), cs.reviewed_at, COALESCE(cs.review_notes, ''),
	cs.field_provenance
`

func scanReviewStep(rows interface{ Scan(dest ...any) error }) (CultivationStep, error) {
	var s CultivationStep
	var dapMin, dapMax *int
	var inputs []map[string]any
	var reviewedAt *time.Time
	var provenance map[string]any
	if err := rows.Scan(
		&s.Slug, &s.CropSlug, &s.VarietySlug,
		&s.AEZCode, &s.Season,
		&s.Stage, &s.OrderIdx,
		&dapMin, &dapMax, &inputs,
		&s.MediaSlugs, &s.Status,
		&s.ReviewedBy, &reviewedAt, &s.ReviewNotes,
		&provenance,
	); err != nil {
		return s, err
	}
	if dapMin != nil || dapMax != nil {
		s.DayAfterPlanting = &IntRange{Min: dapMin, Max: dapMax}
	}
	if reviewedAt != nil {
		s.ReviewedAt = reviewedAt.UTC().Format(time.RFC3339)
	}
	s.Inputs = inputs
	s.FieldProvenance = provenance
	return s, nil
}

// ListCultivationStepsByStatus powers the admin review queue.
func (r *PgxRepo) ListCultivationStepsByStatus(ctx context.Context, status string) ([]CultivationStep, error) {
	sql := `
SELECT ` + reviewStepsSelect + `
FROM cultivation_step cs
WHERE cs.status = $1::record_status
ORDER BY cs.crop_slug, cs.order_idx`
	rows, err := r.pool.Query(ctx, sql, status)
	if err != nil {
		return nil, fmt.Errorf("list steps by status: %w", err)
	}
	defer rows.Close()
	out := make([]CultivationStep, 0, 16)
	for rows.Next() {
		s, err := scanReviewStep(rows)
		if err != nil {
			return nil, fmt.Errorf("scan step: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate: %w", err)
	}
	return out, nil
}

// GetCultivationStep returns a single step — title / body translations are
// joined in so the admin detail panel has the full record in one round-trip.
func (r *PgxRepo) GetCultivationStep(ctx context.Context, slug string) (CultivationStep, error) {
	sql := `
WITH s AS (
	SELECT * FROM cultivation_step WHERE slug = $1
	ORDER BY version DESC LIMIT 1
),
titles AS (
	SELECT jsonb_object_agg(locale, value) AS value
	FROM translation
	WHERE entity_type = 'cultivation_step' AND entity_slug = $1 AND field = 'title'
),
bodies AS (
	SELECT jsonb_object_agg(locale, value) AS value
	FROM translation
	WHERE entity_type = 'cultivation_step' AND entity_slug = $1 AND field = 'body'
)
SELECT ` + strings.ReplaceAll(reviewStepsSelect, "cs.", "s.") + `,
       COALESCE((SELECT value FROM titles), '{}'::jsonb) AS title,
       COALESCE((SELECT value FROM bodies), '{}'::jsonb) AS body
FROM s`
	var s CultivationStep
	var dapMin, dapMax *int
	var inputs []map[string]any
	var reviewedAt *time.Time
	var provenance map[string]any
	var title, body map[string]string
	err := r.pool.QueryRow(ctx, sql, slug).Scan(
		&s.Slug, &s.CropSlug, &s.VarietySlug,
		&s.AEZCode, &s.Season,
		&s.Stage, &s.OrderIdx,
		&dapMin, &dapMax, &inputs,
		&s.MediaSlugs, &s.Status,
		&s.ReviewedBy, &reviewedAt, &s.ReviewNotes,
		&provenance,
		&title, &body,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CultivationStep{}, ErrNotFound
		}
		return CultivationStep{}, fmt.Errorf("get step %q: %w", slug, err)
	}
	if dapMin != nil || dapMax != nil {
		s.DayAfterPlanting = &IntRange{Min: dapMin, Max: dapMax}
	}
	if reviewedAt != nil {
		s.ReviewedAt = reviewedAt.UTC().Format(time.RFC3339)
	}
	s.Inputs = inputs
	s.FieldProvenance = provenance
	s.Title = title
	s.Body = body
	return s, nil
}

// SetCultivationStepStatus promotes or rejects a step. The status enum is
// validated by Postgres itself; an invalid transition will come back as a
// pg error that the handler maps to a 400.
func (r *PgxRepo) SetCultivationStepStatus(ctx context.Context, slug string, u StatusUpdate) error {
	const sql = `
UPDATE cultivation_step
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

const cultivationStepsSQL = `
WITH latest AS (
	SELECT DISTINCT ON (slug) *
	FROM cultivation_step
	WHERE crop_slug = $1
	ORDER BY slug, version DESC
),
titles AS (
	SELECT entity_slug AS slug,
	       jsonb_object_agg(locale, value) AS value
	FROM translation
	WHERE entity_type = 'cultivation_step' AND field = 'title'
	GROUP BY entity_slug
),
bodies AS (
	SELECT entity_slug AS slug,
	       jsonb_object_agg(locale, value) AS value
	FROM translation
	WHERE entity_type = 'cultivation_step' AND field = 'body'
	GROUP BY entity_slug
)
SELECT l.slug, l.crop_slug, COALESCE(l.variety_slug, ''), COALESCE(l.aez_code, ''),
       COALESCE(l.season::text, ''), l.stage, l.order_idx,
       l.dap_min, l.dap_max, l.inputs, COALESCE(l.media_slugs, '{}'),
       l.status::text,
       COALESCE(t.value, '{}'::jsonb) AS title,
       COALESCE(b.value, '{}'::jsonb) AS body,
       l.field_provenance
FROM latest l
LEFT JOIN titles t ON t.slug = l.slug
LEFT JOIN bodies b ON b.slug = l.slug
ORDER BY l.order_idx
`

// ListCultivationSteps returns every cultivation step attached to the crop,
// ordered by order_idx. Titles and bodies come back as locale -> string
// maps so the client can render its current i18n locale directly.
func (r *PgxRepo) ListCultivationSteps(ctx context.Context, cropSlug string) ([]CultivationStep, error) {
	rows, err := r.pool.Query(ctx, cultivationStepsSQL, cropSlug)
	if err != nil {
		return nil, fmt.Errorf("list cultivation steps: %w", err)
	}
	defer rows.Close()

	out := make([]CultivationStep, 0, 8)
	for rows.Next() {
		var s CultivationStep
		var dapMin, dapMax *int
		var title, body map[string]string
		var inputs []map[string]any
		var provenance map[string]any
		if err := rows.Scan(
			&s.Slug, &s.CropSlug, &s.VarietySlug, &s.AEZCode,
			&s.Season, &s.Stage, &s.OrderIdx,
			&dapMin, &dapMax, &inputs, &s.MediaSlugs,
			&s.Status, &title, &body, &provenance,
		); err != nil {
			return nil, fmt.Errorf("scan cultivation step: %w", err)
		}
		if dapMin != nil || dapMax != nil {
			s.DayAfterPlanting = &IntRange{Min: dapMin, Max: dapMax}
		}
		s.Title = title
		s.Body = body
		s.Inputs = inputs
		s.FieldProvenance = provenance
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cultivation steps: %w", err)
	}
	return out, nil
}
