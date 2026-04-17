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

// cultivationStepRecord matches corpus/seed/cultivation_steps/*.json. Title
// and body are I18n strings that flow into the translation table, not
// columns on `cultivation_step` itself.
type cultivationStepRecord struct {
	Slug             string           `json:"slug"`
	Version          int              `json:"version"`
	Status           string           `json:"status"`
	CropSlug         string           `json:"crop_slug"`
	VarietySlug      string           `json:"variety_slug"`
	AEZCode          string           `json:"aez_code"`
	Season           string           `json:"season"`
	Stage            string           `json:"stage"`
	OrderIdx         int              `json:"order_idx"`
	DayAfterPlanting *jsonRange       `json:"day_after_planting"`
	Title            map[string]any   `json:"title"`
	Body             map[string]any   `json:"body"`
	Inputs           []map[string]any `json:"inputs"`
	MediaSlugs       []string         `json:"media_slugs"`
	FieldProvenance  map[string]any   `json:"field_provenance"`
}

func seedCultivationSteps(ctx context.Context, tx pgx.Tx, logger *slog.Logger, dir string) error {
	files, err := listRecords(dir)
	if err != nil {
		return err
	}
	var inserted, refreshed int
	for _, f := range files {
		record, err := readCultivationStep(f)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		isNew, err := upsertCultivationStep(ctx, tx, record)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		if isNew {
			inserted++
		} else {
			refreshed++
		}
	}
	logger.Info("seeded cultivation steps",
		slog.Int("inserted", inserted),
		slog.Int("refreshed", refreshed),
		slog.Int("total", len(files)),
	)
	return nil
}

func readCultivationStep(path string) (*cultivationStepRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec cultivationStepRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &rec, nil
}

// upsertCultivationStep writes one step across cultivation_step + translation.
// Returns true when (slug, version) was newly inserted.
func upsertCultivationStep(ctx context.Context, tx pgx.Tx, r *cultivationStepRecord) (bool, error) {
	inputsJSON, err := json.Marshal(coalesceInputs(r.Inputs))
	if err != nil {
		return false, fmt.Errorf("marshal inputs: %w", err)
	}
	provJSON, err := json.Marshal(r.FieldProvenance)
	if err != nil {
		return false, fmt.Errorf("marshal field_provenance: %w", err)
	}

	var existing bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM cultivation_step WHERE slug = $1 AND version = $2)`,
		r.Slug, r.Version,
	).Scan(&existing); err != nil {
		return false, fmt.Errorf("check step: %w", err)
	}

	stepSQL := `
INSERT INTO cultivation_step (
	slug, version, status, crop_slug, variety_slug, aez_code, season, stage,
	order_idx, dap_min, dap_max, inputs, media_slugs, field_provenance
) VALUES (
	$1, $2, $3::record_status, $4, NULLIF($5, ''), NULLIF($6, ''),
	NULLIF($7, '')::season, $8, $9, $10, $11, $12, $13, $14
)
ON CONFLICT (slug, version) DO UPDATE SET
	status = EXCLUDED.status,
	crop_slug = EXCLUDED.crop_slug,
	variety_slug = EXCLUDED.variety_slug,
	aez_code = EXCLUDED.aez_code,
	season = EXCLUDED.season,
	stage = EXCLUDED.stage,
	order_idx = EXCLUDED.order_idx,
	dap_min = EXCLUDED.dap_min,
	dap_max = EXCLUDED.dap_max,
	inputs = EXCLUDED.inputs,
	media_slugs = EXCLUDED.media_slugs,
	field_provenance = EXCLUDED.field_provenance,
	updated_at = now()
`
	dapMin, dapMax := splitInt(r.DayAfterPlanting)
	if _, err := tx.Exec(ctx, stepSQL,
		r.Slug, r.Version, r.Status, r.CropSlug, r.VarietySlug, r.AEZCode,
		r.Season, r.Stage, r.OrderIdx, dapMin, dapMax,
		inputsJSON, coalesceSlugs(r.MediaSlugs), provJSON,
	); err != nil {
		return false, fmt.Errorf("upsert step: %w", err)
	}

	// Rebuild translations for this step — rebuilding guarantees that a
	// locale removed from the record actually disappears from the DB.
	if _, err := tx.Exec(ctx,
		`DELETE FROM translation WHERE entity_type = 'cultivation_step' AND entity_slug = $1`,
		r.Slug,
	); err != nil {
		return false, fmt.Errorf("delete translations: %w", err)
	}
	if err := insertI18n(ctx, tx, "cultivation_step", r.Slug, "title", r.Title); err != nil {
		return false, err
	}
	if err := insertI18n(ctx, tx, "cultivation_step", r.Slug, "body", r.Body); err != nil {
		return false, err
	}

	return !existing, nil
}

// insertI18n writes a translation row per locale from an I18nString-shaped
// map. Keys ending in "_status" are translation-status metadata, not locale
// values, so they're skipped.
func insertI18n(ctx context.Context, tx pgx.Tx, entityType, slug, field string, i18n map[string]any) error {
	for locale, raw := range i18n {
		if locale == "" || len(locale) > 5 {
			// Skip the _status sibling keys.
			continue
		}
		value, ok := raw.(string)
		if !ok || value == "" {
			continue
		}
		status := "machine_draft"
		if statusRaw, ok := i18n[locale+"_status"]; ok {
			if s, ok := statusRaw.(string); ok && s != "" {
				status = s
			}
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO translation (entity_type, entity_slug, field, locale, value, status)
			 VALUES ($1, $2, $3, $4, $5, $6::translation_status)`,
			entityType, slug, field, locale, value, status,
		); err != nil {
			return fmt.Errorf("insert %s/%s translation: %w", field, locale, err)
		}
	}
	return nil
}

func coalesceInputs(in []map[string]any) []map[string]any {
	if in == nil {
		return []map[string]any{}
	}
	return in
}

func coalesceSlugs(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
