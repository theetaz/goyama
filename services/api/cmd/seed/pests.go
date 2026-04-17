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

// pestRecord matches corpus/seed/pests/*.json. Like disease records, the
// i18n fields (names, description, economic_threshold) and aliases flow
// into translation / entity_alias, so they're not stored as columns.
type pestRecord struct {
	Slug              string            `json:"slug"`
	Version           int               `json:"version"`
	Status            string            `json:"status"`
	ScientificName    string            `json:"scientific_name"`
	Kingdom           string            `json:"kingdom"`
	AffectedCropSlugs []string          `json:"affected_crop_slugs"`
	LifeStages        []string          `json:"life_stages"`
	FeedingType       []string          `json:"feeding_type"`
	FavoredConditions map[string]any    `json:"favored_conditions"`
	Names             map[string]string `json:"names"`
	Aliases           []string          `json:"aliases"`
	Description       map[string]string `json:"description"`
	EconomicThreshold map[string]string `json:"economic_threshold"`
	FieldProvenance   map[string]any    `json:"field_provenance"`
	Extras            map[string]any    `json:"-"`
}

func seedPests(ctx context.Context, tx pgx.Tx, logger *slog.Logger, dir string) error {
	files, err := listRecords(dir)
	if err != nil {
		return err
	}
	var inserted, refreshed int
	for _, f := range files {
		record, err := readPest(f)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		isNew, err := upsertPest(ctx, tx, record)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		if isNew {
			inserted++
		} else {
			refreshed++
		}
	}
	logger.Info("seeded pests",
		slog.Int("inserted", inserted),
		slog.Int("refreshed", refreshed),
		slog.Int("total", len(files)),
	)
	return nil
}

func readPest(path string) (*pestRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec pestRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	var all map[string]any
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("decode attrs: %w", err)
	}
	for _, k := range []string{
		"slug", "version", "status", "scientific_name", "kingdom",
		"affected_crop_slugs", "life_stages", "feeding_type",
		"favored_conditions", "names", "aliases", "description",
		"economic_threshold", "field_provenance",
	} {
		delete(all, k)
	}
	rec.Extras = all
	return &rec, nil
}

func upsertPest(ctx context.Context, tx pgx.Tx, r *pestRecord) (bool, error) {
	attrsJSON, err := json.Marshal(r.Extras)
	if err != nil {
		return false, fmt.Errorf("marshal attrs: %w", err)
	}
	conditionsJSON, err := json.Marshal(coalesceMap(r.FavoredConditions))
	if err != nil {
		return false, fmt.Errorf("marshal favored_conditions: %w", err)
	}
	provJSON, err := json.Marshal(r.FieldProvenance)
	if err != nil {
		return false, fmt.Errorf("marshal field_provenance: %w", err)
	}

	var existing bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pest WHERE slug = $1 AND version = $2)`,
		r.Slug, r.Version,
	).Scan(&existing); err != nil {
		return false, fmt.Errorf("check pest: %w", err)
	}

	const sql = `
INSERT INTO pest (
	slug, version, status, scientific_name, kingdom,
	affected_crop_slugs, life_stages, feeding_type,
	favored_conditions, attrs, field_provenance
) VALUES (
	$1, $2, $3::record_status, NULLIF($4, ''), $5,
	$6, $7, $8,
	$9, $10, $11
)
ON CONFLICT (slug, version) DO UPDATE SET
	status = EXCLUDED.status,
	scientific_name = EXCLUDED.scientific_name,
	kingdom = EXCLUDED.kingdom,
	affected_crop_slugs = EXCLUDED.affected_crop_slugs,
	life_stages = EXCLUDED.life_stages,
	feeding_type = EXCLUDED.feeding_type,
	favored_conditions = EXCLUDED.favored_conditions,
	attrs = EXCLUDED.attrs,
	field_provenance = EXCLUDED.field_provenance,
	updated_at = now()
`
	if _, err := tx.Exec(ctx, sql,
		r.Slug, r.Version, r.Status, r.ScientificName, r.Kingdom,
		coalesceSlugs(r.AffectedCropSlugs), coalesceSlugs(r.LifeStages), coalesceSlugs(r.FeedingType),
		conditionsJSON, attrsJSON, provJSON,
	); err != nil {
		return false, fmt.Errorf("upsert pest: %w", err)
	}

	// Rebuild translations (names / description / economic_threshold) and
	// aliases so removals actually propagate.
	if _, err := tx.Exec(ctx,
		`DELETE FROM translation WHERE entity_type = 'pest' AND entity_slug = $1`,
		r.Slug,
	); err != nil {
		return false, fmt.Errorf("delete translations: %w", err)
	}
	for field, values := range map[string]map[string]string{
		"names":              r.Names,
		"description":        r.Description,
		"economic_threshold": r.EconomicThreshold,
	} {
		for locale, value := range values {
			if value == "" {
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO translation (entity_type, entity_slug, field, locale, value, status)
				 VALUES ('pest', $1, $2, $3, $4, 'machine_draft')`,
				r.Slug, field, locale, value,
			); err != nil {
				return false, fmt.Errorf("insert %s translation: %w", field, err)
			}
		}
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM entity_alias WHERE entity_type = 'pest' AND entity_slug = $1`,
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
			 VALUES ('pest', $1, $2, 1.0, false)
			 ON CONFLICT (entity_type, entity_slug, alias, locale) DO NOTHING`,
			r.Slug, alias,
		); err != nil {
			return false, fmt.Errorf("insert alias: %w", err)
		}
	}

	return !existing, nil
}
