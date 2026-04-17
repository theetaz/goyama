package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5"
)

// seedCrops walks the crops seed directory and upserts each record.
func seedCrops(ctx context.Context, tx pgx.Tx, logger *slog.Logger, dir string) error {
	files, err := listRecords(dir)
	if err != nil {
		return err
	}
	var inserted, refreshed int
	for _, f := range files {
		record, err := readCrop(f)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		isNew, err := upsertCrop(ctx, tx, record)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		if isNew {
			inserted++
		} else {
			refreshed++
		}
	}
	logger.Info("seeded crops",
		slog.Int("inserted", inserted),
		slog.Int("refreshed", refreshed),
		slog.Int("total", len(files)),
	)
	return nil
}

// cropRecord matches the JSON layout of corpus/seed/crops/*.json. Extra fields
// we don't have a direct column for land in the attrs JSONB.
type cropRecord struct {
	Slug              string            `json:"slug"`
	Version           int               `json:"version"`
	Status            string            `json:"status"`
	ScientificName    string            `json:"scientific_name"`
	Family            string            `json:"family"`
	Names             map[string]string `json:"names"`
	Description       map[string]string `json:"description"`
	Aliases           []string          `json:"aliases"`
	Category          string            `json:"category"`
	LifeCycle         string            `json:"life_cycle"`
	GrowthHabit       string            `json:"growth_habit"`
	DefaultSeason     string            `json:"default_season"`
	WaterRequirement  string            `json:"water_requirement"`
	ElevationM        *jsonRange        `json:"elevation_m"`
	RainfallMM        *jsonRange        `json:"rainfall_mm"`
	TemperatureC      *jsonRange        `json:"temperature_c"`
	SoilPH            *jsonRange        `json:"soil_ph"`
	DurationDays      *jsonRange        `json:"duration_days"`
	ExpectedYield     *jsonRange        `json:"expected_yield_kg_per_acre"`
	SeedRateKgPerAcre *float64          `json:"seed_rate_kg_per_acre"`
	SpacingCM         *jsonRange        `json:"spacing_cm"`
	FieldProvenance   map[string]any    `json:"field_provenance"`
	Extras            map[string]any    `json:"-"`
}

type jsonRange struct {
	Min  *float64 `json:"min"`
	Max  *float64 `json:"max"`
	Unit string   `json:"unit,omitempty"`
}

func readCrop(path string) (*cropRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec cropRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	// Stash the full raw JSON (minus keys we map to columns) in attrs so no
	// provenance-carrying field is silently dropped.
	var all map[string]any
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("decode attrs: %w", err)
	}
	for _, k := range []string{
		"slug", "version", "status", "scientific_name", "family", "names",
		"description", "aliases", "category", "life_cycle", "growth_habit",
		"default_season", "water_requirement", "elevation_m", "rainfall_mm",
		"temperature_c", "soil_ph", "duration_days", "expected_yield_kg_per_acre",
		"seed_rate_kg_per_acre", "field_provenance",
	} {
		delete(all, k)
	}
	rec.Extras = all
	return &rec, nil
}

// upsertCrop writes one crop record across crop + translation + entity_alias.
// Returns true if this (slug, version) was new, false if it already existed
// and was refreshed in place.
func upsertCrop(ctx context.Context, tx pgx.Tx, r *cropRecord) (bool, error) {
	attrs := map[string]any{}
	for k, v := range r.Extras {
		attrs[k] = v
	}
	if r.SpacingCM != nil {
		attrs["spacing_cm"] = r.SpacingCM
	}

	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		return false, fmt.Errorf("marshal attrs: %w", err)
	}
	provJSON, err := json.Marshal(r.FieldProvenance)
	if err != nil {
		return false, fmt.Errorf("marshal field_provenance: %w", err)
	}

	// Whether a row already exists for (slug, version). We do one query to
	// decide insert vs update — conditional-logic is cheap here and keeps
	// the SQL readable compared to ON CONFLICT across all 30 columns.
	var existing bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM crop WHERE slug = $1 AND version = $2)`,
		r.Slug, r.Version,
	).Scan(&existing); err != nil {
		return false, fmt.Errorf("check crop: %w", err)
	}

	cropSQL := `
INSERT INTO crop (
	slug, version, status, scientific_name, family, category, life_cycle,
	growth_habit, default_season,
	duration_days_min, duration_days_max,
	elevation_m_min, elevation_m_max,
	rainfall_mm_min, rainfall_mm_max,
	temperature_c_min, temperature_c_max,
	soil_ph_min, soil_ph_max,
	yield_kg_per_acre_min, yield_kg_per_acre_max,
	seed_rate_kg_per_acre, water_requirement,
	attrs, field_provenance
) VALUES (
	$1, $2, $3::record_status, $4, $5, $6::crop_category, $7::life_cycle,
	$8, NULLIF($9, '')::season,
	$10, $11,
	$12, $13,
	$14, $15,
	$16, $17,
	$18, $19,
	$20, $21,
	$22, NULLIF($23, '')::water_req,
	$24, $25
)
ON CONFLICT (slug, version) DO UPDATE SET
	status = EXCLUDED.status,
	scientific_name = EXCLUDED.scientific_name,
	family = EXCLUDED.family,
	category = EXCLUDED.category,
	life_cycle = EXCLUDED.life_cycle,
	growth_habit = EXCLUDED.growth_habit,
	default_season = EXCLUDED.default_season,
	duration_days_min = EXCLUDED.duration_days_min,
	duration_days_max = EXCLUDED.duration_days_max,
	elevation_m_min = EXCLUDED.elevation_m_min,
	elevation_m_max = EXCLUDED.elevation_m_max,
	rainfall_mm_min = EXCLUDED.rainfall_mm_min,
	rainfall_mm_max = EXCLUDED.rainfall_mm_max,
	temperature_c_min = EXCLUDED.temperature_c_min,
	temperature_c_max = EXCLUDED.temperature_c_max,
	soil_ph_min = EXCLUDED.soil_ph_min,
	soil_ph_max = EXCLUDED.soil_ph_max,
	yield_kg_per_acre_min = EXCLUDED.yield_kg_per_acre_min,
	yield_kg_per_acre_max = EXCLUDED.yield_kg_per_acre_max,
	seed_rate_kg_per_acre = EXCLUDED.seed_rate_kg_per_acre,
	water_requirement = EXCLUDED.water_requirement,
	attrs = EXCLUDED.attrs,
	field_provenance = EXCLUDED.field_provenance,
	updated_at = now()
`
	if _, err := tx.Exec(ctx, cropSQL,
		r.Slug, r.Version, r.Status, r.ScientificName, nullIfEmpty(r.Family),
		r.Category, r.LifeCycle,
		nullIfEmpty(r.GrowthHabit), r.DefaultSeason,
		rangeInt(r.DurationDays),
		func() *int { _, max := splitInt(r.DurationDays); return max }(),
		rangeFloat(r.ElevationM),
		func() *float32 { _, max := splitFloat(r.ElevationM); return max }(),
		rangeFloat(r.RainfallMM),
		func() *float32 { _, max := splitFloat(r.RainfallMM); return max }(),
		rangeFloat(r.TemperatureC),
		func() *float32 { _, max := splitFloat(r.TemperatureC); return max }(),
		rangeFloat(r.SoilPH),
		func() *float32 { _, max := splitFloat(r.SoilPH); return max }(),
		rangeFloat(r.ExpectedYield),
		func() *float32 { _, max := splitFloat(r.ExpectedYield); return max }(),
		float32PtrFromFloat64(r.SeedRateKgPerAcre), r.WaterRequirement,
		attrsJSON, provJSON,
	); err != nil {
		return false, fmt.Errorf("upsert crop: %w", err)
	}

	// Replace translations for this crop — we rebuild them every run so a
	// removed locale in the corpus record actually disappears from the DB.
	if _, err := tx.Exec(ctx,
		`DELETE FROM translation WHERE entity_type = 'crop' AND entity_slug = $1`,
		r.Slug,
	); err != nil {
		return false, fmt.Errorf("delete translations: %w", err)
	}
	for locale, value := range r.Names {
		if value == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO translation (entity_type, entity_slug, field, locale, value, status)
			 VALUES ('crop', $1, 'names', $2, $3, 'machine_draft')`,
			r.Slug, locale, value,
		); err != nil {
			return false, fmt.Errorf("insert names translation: %w", err)
		}
	}
	for locale, value := range r.Description {
		if value == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO translation (entity_type, entity_slug, field, locale, value, status)
			 VALUES ('crop', $1, 'description', $2, $3, 'machine_draft')`,
			r.Slug, locale, value,
		); err != nil {
			return false, fmt.Errorf("insert description translation: %w", err)
		}
	}

	// Aliases — same rebuild strategy.
	if _, err := tx.Exec(ctx,
		`DELETE FROM entity_alias WHERE entity_type = 'crop' AND entity_slug = $1`,
		r.Slug,
	); err != nil {
		return false, fmt.Errorf("delete aliases: %w", err)
	}
	for _, alias := range r.Aliases {
		if alias == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO entity_alias (entity_type, entity_slug, alias, confidence, reviewed)
			 VALUES ('crop', $1, $2, 1.0, false)
			 ON CONFLICT (entity_type, entity_slug, alias, locale) DO NOTHING`,
			r.Slug, alias,
		); err != nil {
			return false, fmt.Errorf("insert alias: %w", err)
		}
	}

	return !existing, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func rangeInt(r *jsonRange) *int {
	if r == nil || r.Min == nil {
		return nil
	}
	v := int(*r.Min)
	return &v
}

func splitInt(r *jsonRange) (*int, *int) {
	if r == nil {
		return nil, nil
	}
	var minV, maxV *int
	if r.Min != nil {
		v := int(*r.Min)
		minV = &v
	}
	if r.Max != nil {
		v := int(*r.Max)
		maxV = &v
	}
	return minV, maxV
}

func rangeFloat(r *jsonRange) *float32 {
	if r == nil || r.Min == nil {
		return nil
	}
	v := float32(*r.Min)
	return &v
}

func splitFloat(r *jsonRange) (*float32, *float32) {
	if r == nil {
		return nil, nil
	}
	var minV, maxV *float32
	if r.Min != nil {
		v := float32(*r.Min)
		minV = &v
	}
	if r.Max != nil {
		v := float32(*r.Max)
		maxV = &v
	}
	return minV, maxV
}

func float32PtrFromFloat64(p *float64) *float32 {
	if p == nil {
		return nil
	}
	v := float32(*p)
	return &v
}
