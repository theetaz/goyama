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

// diseaseRecord matches corpus/seed/diseases/*.json. Names, description and
// aliases flow into translation / entity_alias rows, so they're not stored
// as columns on `disease`.
type diseaseRecord struct {
	Slug              string            `json:"slug"`
	Version           int               `json:"version"`
	Status            string            `json:"status"`
	ScientificName    string            `json:"scientific_name"`
	CausalOrganism    string            `json:"causal_organism"`
	CausalSpecies     string            `json:"causal_species"`
	AffectedCropSlugs []string          `json:"affected_crop_slugs"`
	AffectedParts     []string          `json:"affected_parts"`
	Transmission      []string          `json:"transmission"`
	FavoredConditions map[string]any    `json:"favored_conditions"`
	Severity          string            `json:"severity"`
	ConfusedWith      []string          `json:"confused_with"`
	Names             map[string]string `json:"names"`
	Aliases           []string          `json:"aliases"`
	Description       map[string]string `json:"description"`
	FieldProvenance   map[string]any    `json:"field_provenance"`
	Extras            map[string]any    `json:"-"`
}

func seedDiseases(ctx context.Context, tx pgx.Tx, logger *slog.Logger, dir string) error {
	files, err := listRecords(dir)
	if err != nil {
		return err
	}
	var inserted, refreshed int
	for _, f := range files {
		record, err := readDisease(f)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		isNew, err := upsertDisease(ctx, tx, record)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		if isNew {
			inserted++
		} else {
			refreshed++
		}
	}
	logger.Info("seeded diseases",
		slog.Int("inserted", inserted),
		slog.Int("refreshed", refreshed),
		slog.Int("total", len(files)),
	)
	return nil
}

func readDisease(path string) (*diseaseRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec diseaseRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	// Stash anything we don't map to a column in attrs so no
	// provenance-carrying field is silently dropped.
	var all map[string]any
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("decode attrs: %w", err)
	}
	for _, k := range []string{
		"slug", "version", "status", "scientific_name", "causal_organism",
		"causal_species", "affected_crop_slugs", "affected_parts",
		"transmission", "favored_conditions", "severity", "confused_with",
		"names", "aliases", "description", "field_provenance",
	} {
		delete(all, k)
	}
	rec.Extras = all
	return &rec, nil
}

// upsertDisease writes one disease row plus its translation / alias rows.
func upsertDisease(ctx context.Context, tx pgx.Tx, r *diseaseRecord) (bool, error) {
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
		`SELECT EXISTS (SELECT 1 FROM disease WHERE slug = $1 AND version = $2)`,
		r.Slug, r.Version,
	).Scan(&existing); err != nil {
		return false, fmt.Errorf("check disease: %w", err)
	}

	const sql = `
INSERT INTO disease (
	slug, version, status, scientific_name, causal_organism, causal_species,
	affected_crop_slugs, affected_parts, transmission,
	favored_conditions, severity, confused_with,
	attrs, field_provenance
) VALUES (
	$1, $2, $3::record_status, NULLIF($4, ''), $5, NULLIF($6, ''),
	$7, $8, $9,
	$10, NULLIF($11, ''), $12,
	$13, $14
)
ON CONFLICT (slug, version) DO UPDATE SET
	status = EXCLUDED.status,
	scientific_name = EXCLUDED.scientific_name,
	causal_organism = EXCLUDED.causal_organism,
	causal_species = EXCLUDED.causal_species,
	affected_crop_slugs = EXCLUDED.affected_crop_slugs,
	affected_parts = EXCLUDED.affected_parts,
	transmission = EXCLUDED.transmission,
	favored_conditions = EXCLUDED.favored_conditions,
	severity = EXCLUDED.severity,
	confused_with = EXCLUDED.confused_with,
	attrs = EXCLUDED.attrs,
	field_provenance = EXCLUDED.field_provenance,
	updated_at = now()
`
	if _, err := tx.Exec(ctx, sql,
		r.Slug, r.Version, r.Status, r.ScientificName, r.CausalOrganism, r.CausalSpecies,
		coalesceSlugs(r.AffectedCropSlugs), coalesceSlugs(r.AffectedParts), coalesceSlugs(r.Transmission),
		conditionsJSON, r.Severity, coalesceSlugs(r.ConfusedWith),
		attrsJSON, provJSON,
	); err != nil {
		return false, fmt.Errorf("upsert disease: %w", err)
	}

	// Rebuild translations + aliases so locale or alias removals actually
	// propagate into the DB.
	if _, err := tx.Exec(ctx,
		`DELETE FROM translation WHERE entity_type = 'disease' AND entity_slug = $1`,
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
			 VALUES ('disease', $1, 'names', $2, $3, 'machine_draft')`,
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
			 VALUES ('disease', $1, 'description', $2, $3, 'machine_draft')`,
			r.Slug, locale, value,
		); err != nil {
			return false, fmt.Errorf("insert description translation: %w", err)
		}
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM entity_alias WHERE entity_type = 'disease' AND entity_slug = $1`,
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
			 VALUES ('disease', $1, $2, 1.0, false)
			 ON CONFLICT (entity_type, entity_slug, alias, locale) DO NOTHING`,
			r.Slug, alias,
		); err != nil {
			return false, fmt.Errorf("insert alias: %w", err)
		}
	}

	return !existing, nil
}

func coalesceMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	return in
}
