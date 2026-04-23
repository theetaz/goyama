// Command planload upserts cultivation_plan + child rows from JSON
// fixtures into Postgres. One file per plan; the aggregate is loaded
// transactionally so a half-loaded plan never appears in the review
// queue.
//
// Usage:
//
//	planload --dir=corpus/seed/cultivation_plans
//	planload --file=corpus/seed/cultivation_plans/red-onion-dry-zone-maha.json
//
// Idempotent — re-running on the same fixture file is safe; child
// rows are wiped and re-inserted inside the same transaction so a
// shrunken activity list doesn't leave orphans.
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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type planFile struct {
	Slug                string         `json:"slug"`
	Version             int            `json:"version"`
	Status              string         `json:"status"`
	Authority           string         `json:"authority"`
	CropSlug            string         `json:"crop_slug"`
	VarietySlug         string         `json:"variety_slug"`
	Season              string         `json:"season"`
	AEZCodes            []string       `json:"aez_codes"`
	Title               map[string]any `json:"title"`
	Summary             map[string]any `json:"summary"`
	StartMonth          *int           `json:"start_month"`
	DurationWeeks       *int           `json:"duration_weeks"`
	YieldMin            *float64       `json:"expected_yield_kg_per_acre_min"`
	YieldMax            *float64       `json:"expected_yield_kg_per_acre_max"`
	SourceDocumentURL   string         `json:"source_document_url"`
	SourceDocumentTitle string         `json:"source_document_title"`
	FieldProvenance     map[string]any `json:"field_provenance"`
	Activities          []activityFile `json:"activities"`
	PestRisks           []pestRiskFile `json:"pest_risks"`
	Economics           []econFile     `json:"economics"`
}

type activityFile struct {
	WeekIdx     int            `json:"week_idx"`
	OrderInWeek int            `json:"order_in_week"`
	Activity    string         `json:"activity"`
	DAPMin      *int           `json:"dap_min"`
	DAPMax      *int           `json:"dap_max"`
	Title       map[string]any `json:"title"`
	Body        map[string]any `json:"body"`
	Inputs      []any          `json:"inputs"`
	WeatherHint string         `json:"weather_hint"`
	MediaSlugs  []string       `json:"media_slugs"`
}

type pestRiskFile struct {
	WeekIdx                int            `json:"week_idx"`
	DiseaseSlug            string         `json:"disease_slug"`
	PestSlug               string         `json:"pest_slug"`
	Risk                   string         `json:"risk"`
	RecommendedRemedySlugs []string       `json:"recommended_remedy_slugs"`
	Notes                  map[string]any `json:"notes"`
}

type econFile struct {
	ReferenceYear                 int            `json:"reference_year"`
	UnitArea                      string         `json:"unit_area"`
	Currency                      string         `json:"currency"`
	CostLines                     []any          `json:"cost_lines"`
	TotalCostWithoutFamilyLabour  *float64       `json:"total_cost_without_family_labour"`
	TotalCostWithFamilyLabour     *float64       `json:"total_cost_with_family_labour"`
	YieldKg                       *float64       `json:"yield_kg"`
	UnitPrice                     *float64       `json:"unit_price"`
	GrossRevenue                  *float64       `json:"gross_revenue"`
	NetRevenueWithoutFamilyLabour *float64       `json:"net_revenue_without_family_labour"`
	NetRevenueWithFamilyLabour    *float64       `json:"net_revenue_with_family_labour"`
	FieldProvenance               map[string]any `json:"field_provenance"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	dir := flag.String("dir", "", "directory of plan JSON fixtures (mutually exclusive with --file)")
	file := flag.String("file", "", "single plan JSON fixture")
	flag.Parse()

	if (*dir == "" && *file == "") || (*dir != "" && *file != "") {
		flag.Usage()
		return fmt.Errorf("exactly one of --dir or --file is required")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	files, err := collectFiles(*dir, *file)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no fixtures found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("pgx pool: %w", err)
	}
	defer pool.Close()

	loaded := 0
	for _, path := range files {
		p, err := readPlan(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := upsertPlan(ctx, pool, p); err != nil {
			return fmt.Errorf("upsert %s: %w", p.Slug, err)
		}
		loaded++
		fmt.Printf("loaded %s (v%d, %d activities, %d risks, %d economics)\n",
			p.Slug, p.Version, len(p.Activities), len(p.PestRisks), len(p.Economics))
	}
	fmt.Printf("\nupserted %d plan(s)\n", loaded)
	return nil
}

func collectFiles(dir, file string) ([]string, error) {
	if file != "" {
		return []string{file}, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}

func readPlan(path string) (planFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return planFile{}, err
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil {
		return planFile{}, err
	}
	var p planFile
	if err := json.Unmarshal(body, &p); err != nil {
		return planFile{}, fmt.Errorf("decode: %w", err)
	}
	if p.Version == 0 {
		p.Version = 1
	}
	if p.Status == "" {
		p.Status = "draft"
	}
	if p.Authority == "" {
		p.Authority = "doa_official"
	}
	return p, nil
}

// upsertPlan writes the header + child rows transactionally. Children
// are wiped and re-inserted to keep the loader idempotent under
// shrinking lists (e.g. removing an activity).
func upsertPlan(ctx context.Context, pool *pgxpool.Pool, p planFile) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	titleJSON, _ := json.Marshal(p.Title)
	summaryJSON, _ := json.Marshal(p.Summary)
	provenanceJSON, _ := json.Marshal(p.FieldProvenance)

	const upsertHeader = `
INSERT INTO cultivation_plan (
    slug, version, status, authority, crop_slug, variety_slug, season,
    aez_codes, title, summary, start_month, duration_weeks,
    expected_yield_kg_per_acre_min, expected_yield_kg_per_acre_max,
    source_document_url, source_document_title, field_provenance,
    updated_at
)
VALUES (
    $1, $2, $3::record_status, $4::authority_level, $5, NULLIF($6, ''), $7::season,
    $8, $9::jsonb, $10::jsonb, $11, $12,
    $13, $14,
    NULLIF($15, ''), NULLIF($16, ''), $17::jsonb,
    now()
)
ON CONFLICT (slug, version) DO UPDATE SET
    status = EXCLUDED.status,
    authority = EXCLUDED.authority,
    crop_slug = EXCLUDED.crop_slug,
    variety_slug = EXCLUDED.variety_slug,
    season = EXCLUDED.season,
    aez_codes = EXCLUDED.aez_codes,
    title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    start_month = EXCLUDED.start_month,
    duration_weeks = EXCLUDED.duration_weeks,
    expected_yield_kg_per_acre_min = EXCLUDED.expected_yield_kg_per_acre_min,
    expected_yield_kg_per_acre_max = EXCLUDED.expected_yield_kg_per_acre_max,
    source_document_url = EXCLUDED.source_document_url,
    source_document_title = EXCLUDED.source_document_title,
    field_provenance = EXCLUDED.field_provenance,
    updated_at = now()`

	if _, err := tx.Exec(ctx, upsertHeader,
		p.Slug, p.Version, p.Status, p.Authority,
		p.CropSlug, p.VarietySlug, p.Season,
		p.AEZCodes, titleJSON, summaryJSON,
		nilIntZero(p.StartMonth), nilIntZero(p.DurationWeeks),
		p.YieldMin, p.YieldMax,
		p.SourceDocumentURL, p.SourceDocumentTitle, provenanceJSON,
	); err != nil {
		return fmt.Errorf("upsert plan header: %w", err)
	}

	// Replace children — same (slug, version) edge => wipe then insert.
	for _, table := range []string{"cultivation_activity", "cultivation_pest_risk", "cultivation_economics"} {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf("DELETE FROM %s WHERE plan_slug = $1 AND plan_version = $2", table),
			p.Slug, p.Version,
		); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}

	for _, a := range p.Activities {
		titleJSON, _ := json.Marshal(a.Title)
		bodyJSON, _ := json.Marshal(a.Body)
		inputsJSON, _ := json.Marshal(orEmptyArray(a.Inputs))
		const sql = `
INSERT INTO cultivation_activity (
    plan_slug, plan_version, week_idx, order_in_week, activity,
    dap_min, dap_max, title, body, inputs, weather_hint, media_slugs
)
VALUES ($1, $2, $3, $4, $5::activity_type, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb,
        NULLIF($11, '')::weather_expectation, $12)`
		if _, err := tx.Exec(ctx, sql,
			p.Slug, p.Version, a.WeekIdx, a.OrderInWeek, a.Activity,
			a.DAPMin, a.DAPMax,
			titleJSON, bodyJSON, inputsJSON,
			a.WeatherHint, a.MediaSlugs,
		); err != nil {
			return fmt.Errorf("insert activity W%d %s: %w", a.WeekIdx, a.Activity, err)
		}
	}

	for _, r := range p.PestRisks {
		notesJSON, _ := json.Marshal(r.Notes)
		const sql = `
INSERT INTO cultivation_pest_risk (
    plan_slug, plan_version, week_idx, disease_slug, pest_slug, risk,
    recommended_remedy_slugs, notes
)
VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6::risk_level, $7, $8::jsonb)`
		if _, err := tx.Exec(ctx, sql,
			p.Slug, p.Version, r.WeekIdx,
			r.DiseaseSlug, r.PestSlug, r.Risk,
			r.RecommendedRemedySlugs, notesJSON,
		); err != nil {
			return fmt.Errorf("insert pest risk W%d: %w", r.WeekIdx, err)
		}
	}

	for _, e := range p.Economics {
		costJSON, _ := json.Marshal(orEmptyArray(e.CostLines))
		provJSON, _ := json.Marshal(e.FieldProvenance)
		const sql = `
INSERT INTO cultivation_economics (
    plan_slug, plan_version, reference_year, unit_area, currency,
    cost_lines,
    total_cost_without_family_labour, total_cost_with_family_labour,
    yield_kg, unit_price, gross_revenue,
    net_revenue_without_family_labour, net_revenue_with_family_labour,
    field_provenance
)
VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), 'acre'), COALESCE(NULLIF($5, ''), 'LKR'),
        $6::jsonb,
        $7, $8,
        $9, $10, $11,
        $12, $13,
        $14::jsonb)`
		if _, err := tx.Exec(ctx, sql,
			p.Slug, p.Version, e.ReferenceYear, e.UnitArea, e.Currency,
			costJSON,
			e.TotalCostWithoutFamilyLabour, e.TotalCostWithFamilyLabour,
			e.YieldKg, e.UnitPrice, e.GrossRevenue,
			e.NetRevenueWithoutFamilyLabour, e.NetRevenueWithFamilyLabour,
			provJSON,
		); err != nil {
			return fmt.Errorf("insert economics %d: %w", e.ReferenceYear, err)
		}
	}

	return tx.Commit(ctx)
}

func nilIntZero(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func orEmptyArray(v []any) []any {
	if v == nil {
		return []any{}
	}
	return v
}
