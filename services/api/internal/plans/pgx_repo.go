package plans

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

// PgxRepo serves cultivation plans from Postgres for both the farmer-
// facing `Repository` surface and the admin `AdminRepo` surface. The
// aggregate is reconstituted by joining the three child tables with
// `json_agg` so one round-trip returns a full Plan including all
// activities, pest risks, and economics rows.
type PgxRepo struct {
	pool *pgxpool.Pool
}

// NewPgxRepo returns a Postgres-backed plans repo.
func NewPgxRepo(pool *pgxpool.Pool) *PgxRepo { return &PgxRepo{pool: pool} }

// ListByCrop returns the published-or-draft summary set for one crop.
// Kept lean — the farmer-facing list doesn't need the children.
func (r *PgxRepo) ListByCrop(ctx context.Context, cropSlug string) ([]Summary, error) {
	const sql = `
SELECT DISTINCT ON (slug)
    slug, crop_slug, season::text, authority::text,
    COALESCE(aez_codes, '{}'),
    COALESCE(title, '{}'::jsonb), COALESCE(summary, '{}'::jsonb),
    COALESCE(duration_weeks, 0),
    expected_yield_kg_per_acre_min, expected_yield_kg_per_acre_max,
    COALESCE(source_document_title, '')
FROM cultivation_plan
WHERE crop_slug = $1 AND status = 'published'
ORDER BY slug, version DESC`
	rows, err := r.pool.Query(ctx, sql, cropSlug)
	if err != nil {
		return nil, fmt.Errorf("list plans by crop: %w", err)
	}
	defer rows.Close()
	out := make([]Summary, 0, 4)
	for rows.Next() {
		s, err := scanSummary(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plan summaries: %w", err)
	}
	return out, nil
}

// ListByStatus powers the admin review queue. Returns the full Plan
// aggregate for each row so the review page can show every activity
// the reviewer is about to promote.
func (r *PgxRepo) ListByStatus(ctx context.Context, status string) ([]Plan, error) {
	const sql = `
SELECT slug
FROM cultivation_plan
WHERE ($1 = '' OR status = $1::record_status)
ORDER BY updated_at DESC, slug`
	rows, err := r.pool.Query(ctx, sql, strings.TrimSpace(status))
	if err != nil {
		return nil, fmt.Errorf("list plans by status: %w", err)
	}
	slugs := make([]string, 0, 16)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan plan slug: %w", err)
		}
		slugs = append(slugs, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plans: %w", err)
	}

	out := make([]Plan, 0, len(slugs))
	for _, slug := range slugs {
		p, err := r.Get(ctx, slug)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// Get returns the latest version of a plan with every child row
// reconstituted into the aggregate.
func (r *PgxRepo) Get(ctx context.Context, slug string) (Plan, error) {
	const headerSQL = `
SELECT slug, version, status::text, authority::text,
       crop_slug, COALESCE(variety_slug, ''), season::text,
       COALESCE(aez_codes, '{}'),
       COALESCE(title, '{}'::jsonb),
       COALESCE(summary, '{}'::jsonb),
       COALESCE(start_month, 0),
       COALESCE(duration_weeks, 0),
       expected_yield_kg_per_acre_min, expected_yield_kg_per_acre_max,
       COALESCE(source_document_url, ''), COALESCE(source_document_title, ''),
       COALESCE(field_provenance, '{}'::jsonb),
       COALESCE(reviewed_by, ''), reviewed_at, COALESCE(review_notes, '')
FROM cultivation_plan
WHERE slug = $1
ORDER BY version DESC
LIMIT 1`

	var (
		p          Plan
		version    int
		ageMin     *float32
		ageMax     *float32
		titleMap   map[string]string
		summaryMap map[string]string
		aezCodes   []string
		provenance map[string]any
		reviewedAt *time.Time
	)

	err := r.pool.QueryRow(ctx, headerSQL, slug).Scan(
		&p.Slug, &version, &p.Status, &p.Authority,
		&p.CropSlug, &p.VarietySlug, &p.Season,
		&aezCodes,
		&titleMap, &summaryMap,
		&p.StartMonth,
		&p.DurationWeeks,
		&ageMin, &ageMax,
		&p.SourceDocumentURL, &p.SourceDocumentTitle,
		&provenance,
		&p.ReviewedBy, &reviewedAt, &p.ReviewNotes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Plan{}, ErrNotFound
		}
		return Plan{}, fmt.Errorf("get plan %q: %w", slug, err)
	}
	// Promoted-field assignment via the embedded Summary struct so the
	// embedded `Summary map[string]string` field doesn't collide with
	// the local `Summary` *struct*.
	p.Summary.AEZCodes = aezCodes
	p.Summary.Title = titleMap
	p.Summary.Summary = summaryMap
	p.FieldProvenance = provenance
	if reviewedAt != nil {
		p.ReviewedAt = reviewedAt.UTC().Format(time.RFC3339)
	}
	if ageMin != nil || ageMax != nil {
		p.Summary.ExpectedYieldKgPerAcre = &Range{Unit: "kg/acre"}
		if ageMin != nil {
			p.Summary.ExpectedYieldKgPerAcre.Min = *ageMin
		}
		if ageMax != nil {
			p.Summary.ExpectedYieldKgPerAcre.Max = *ageMax
		}
	}

	// Children — fetch each list and attach. Three short queries beats
	// a single jsonb_agg monster from a readability standpoint, and
	// the plan detail endpoint isn't on a hot path.
	activities, err := r.activities(ctx, slug, version)
	if err != nil {
		return Plan{}, err
	}
	p.Activities = activities

	risks, err := r.pestRisks(ctx, slug, version)
	if err != nil {
		return Plan{}, err
	}
	p.PestRisks = risks

	econ, err := r.economics(ctx, slug, version)
	if err != nil {
		return Plan{}, err
	}
	p.Economics = econ
	return p, nil
}

func (r *PgxRepo) activities(ctx context.Context, slug string, version int) ([]Activity, error) {
	const sql = `
SELECT week_idx, COALESCE(order_in_week, 0), activity::text,
       dap_min, dap_max,
       COALESCE(title, '{}'::jsonb), COALESCE(body, '{}'::jsonb),
       COALESCE(inputs, '[]'::jsonb),
       COALESCE(weather_hint::text, ''),
       COALESCE(media_slugs, '{}')
FROM cultivation_activity
WHERE plan_slug = $1 AND plan_version = $2
ORDER BY week_idx, order_in_week, activity`
	rows, err := r.pool.Query(ctx, sql, slug, version)
	if err != nil {
		return nil, fmt.Errorf("list activities: %w", err)
	}
	defer rows.Close()
	out := make([]Activity, 0, 8)
	for rows.Next() {
		var a Activity
		var title, body map[string]string
		var inputs []map[string]any
		if err := rows.Scan(
			&a.WeekIdx, &a.OrderInWeek, &a.Activity,
			&a.DAPMin, &a.DAPMax,
			&title, &body,
			&inputs,
			&a.WeatherHint,
			&a.MediaSlugs,
		); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		a.Title = title
		a.Body = body
		a.Inputs = inputs
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PgxRepo) pestRisks(ctx context.Context, slug string, version int) ([]PestRisk, error) {
	const sql = `
SELECT week_idx, COALESCE(disease_slug, ''), COALESCE(pest_slug, ''),
       risk::text,
       COALESCE(recommended_remedy_slugs, '{}'),
       COALESCE(notes, '{}'::jsonb)
FROM cultivation_pest_risk
WHERE plan_slug = $1 AND plan_version = $2
ORDER BY week_idx, disease_slug, pest_slug`
	rows, err := r.pool.Query(ctx, sql, slug, version)
	if err != nil {
		return nil, fmt.Errorf("list pest risks: %w", err)
	}
	defer rows.Close()
	out := make([]PestRisk, 0, 8)
	for rows.Next() {
		var p PestRisk
		var notes map[string]string
		if err := rows.Scan(
			&p.WeekIdx, &p.DiseaseSlug, &p.PestSlug,
			&p.Risk,
			&p.RecommendedRemedySlugs,
			&notes,
		); err != nil {
			return nil, fmt.Errorf("scan pest risk: %w", err)
		}
		p.Notes = notes
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PgxRepo) economics(ctx context.Context, slug string, version int) ([]Economics, error) {
	const sql = `
SELECT reference_year, unit_area, currency,
       COALESCE(cost_lines, '[]'::jsonb),
       total_cost_without_family_labour, total_cost_with_family_labour,
       yield_kg, unit_price,
       gross_revenue,
       net_revenue_without_family_labour, net_revenue_with_family_labour
FROM cultivation_economics
WHERE plan_slug = $1 AND plan_version = $2
ORDER BY reference_year DESC`
	rows, err := r.pool.Query(ctx, sql, slug, version)
	if err != nil {
		return nil, fmt.Errorf("list economics: %w", err)
	}
	defer rows.Close()
	out := make([]Economics, 0, 1)
	for rows.Next() {
		var e Economics
		var cost []map[string]any
		if err := rows.Scan(
			&e.ReferenceYear, &e.UnitArea, &e.Currency,
			&cost,
			&e.TotalCostWithoutFamilyLabour, &e.TotalCostWithFamilyLabour,
			&e.YieldKg, &e.UnitPrice,
			&e.GrossRevenue,
			&e.NetRevenueWithoutFamilyLabour, &e.NetRevenueWithFamilyLabour,
		); err != nil {
			return nil, fmt.Errorf("scan economics: %w", err)
		}
		e.CostLines = cost
		out = append(out, e)
	}
	return out, rows.Err()
}

// SetStatus promotes, rejects, or deprecates the latest version of a
// plan. review.Routes validates the transition before calling here, so
// a failed UPDATE means the plan vanished between list and write.
func (r *PgxRepo) SetStatus(ctx context.Context, slug string, u review.StatusUpdate) error {
	const sql = `
UPDATE cultivation_plan
SET status = $2::record_status,
    reviewed_by = $3,
    reviewed_at = now(),
    review_notes = NULLIF($4, ''),
    updated_at = now()
WHERE slug = $1
  AND version = (SELECT MAX(version) FROM cultivation_plan WHERE slug = $1)`
	tag, err := r.pool.Exec(ctx, sql, slug, u.Status, u.ReviewedBy, u.Notes)
	if err != nil {
		return fmt.Errorf("update plan status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanSummary reads a row from the ListByCrop query into a Summary.
// The shared helper exists so a future ListByAEZ or ListBySeason can
// reuse the same scan code.
func scanSummary(rows pgx.Row) (Summary, error) {
	var s Summary
	var aez []string
	var title, summary map[string]string
	var dur int
	var ageMin, ageMax *float32
	if err := rows.Scan(
		&s.Slug, &s.CropSlug, &s.Season, &s.Authority,
		&aez,
		&title, &summary,
		&dur,
		&ageMin, &ageMax,
		&s.SourceDocumentTitle,
	); err != nil {
		return Summary{}, fmt.Errorf("scan summary: %w", err)
	}
	s.AEZCodes = aez
	s.Title = title
	s.Summary = summary
	s.DurationWeeks = dur
	if ageMin != nil || ageMax != nil {
		s.ExpectedYieldKgPerAcre = &Range{Unit: "kg/acre"}
		if ageMin != nil {
			s.ExpectedYieldKgPerAcre.Min = *ageMin
		}
		if ageMax != nil {
			s.ExpectedYieldKgPerAcre.Max = *ageMax
		}
	}
	return s, nil
}

var _ Repository = (*PgxRepo)(nil)
var _ AdminRepo = (*PgxRepo)(nil)
