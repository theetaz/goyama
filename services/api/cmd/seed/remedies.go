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

// chemicalBlock is the nested object only present on chemical-type
// remedies. Its fields get unpacked into flat columns on the remedy
// table below.
type chemicalBlock struct {
	ActiveIngredient       string            `json:"active_ingredient"`
	Concentration          string            `json:"concentration"`
	Formulation            string            `json:"formulation"`
	Dosage                 string            `json:"dosage"`
	ApplicationMethod      string            `json:"application_method"`
	Frequency              string            `json:"frequency"`
	PreHarvestIntervalDays *int              `json:"pre_harvest_interval_days"`
	ReEntryIntervalHours   *int              `json:"re_entry_interval_hours"`
	WhoHazardClass         string            `json:"who_hazard_class"`
	DoaRegistrationNo      string            `json:"doa_registration_no"`
	SafetyNotes            map[string]string `json:"safety_notes"`
}

// remedyRecord matches corpus/seed/remedies/*.json. Chemical-type
// remedies carry a nested `chemical` block; all other types leave it
// nil.
type remedyRecord struct {
	Slug                string            `json:"slug"`
	Version             int               `json:"version"`
	Status              string            `json:"status"`
	Type                string            `json:"type"`
	TargetDiseaseSlugs  []string          `json:"target_disease_slugs"`
	TargetPestSlugs     []string          `json:"target_pest_slugs"`
	ApplicableCropSlugs []string          `json:"applicable_crop_slugs"`
	Name                map[string]string `json:"name"`
	Description         map[string]string `json:"description"`
	Instructions        map[string]string `json:"instructions"`
	Chemical            *chemicalBlock    `json:"chemical"`
	Effectiveness       string            `json:"effectiveness"`
	CostTier            string            `json:"cost_tier"`
	OrganicCompatible   *bool             `json:"organic_compatible"`
	FieldProvenance     map[string]any    `json:"field_provenance"`
	Extras              map[string]any    `json:"-"`
}

func seedRemedies(ctx context.Context, tx pgx.Tx, logger *slog.Logger, dir string) error {
	files, err := listRecords(dir)
	if err != nil {
		return err
	}
	var inserted, refreshed int
	for _, f := range files {
		record, err := readRemedy(f)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		isNew, err := upsertRemedy(ctx, tx, record)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		if isNew {
			inserted++
		} else {
			refreshed++
		}
	}
	logger.Info("seeded remedies",
		slog.Int("inserted", inserted),
		slog.Int("refreshed", refreshed),
		slog.Int("total", len(files)),
	)
	return nil
}

func readRemedy(path string) (*remedyRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec remedyRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	var all map[string]any
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("decode attrs: %w", err)
	}
	for _, k := range []string{
		"slug", "version", "status", "type",
		"target_disease_slugs", "target_pest_slugs", "applicable_crop_slugs",
		"name", "description", "instructions", "chemical",
		"effectiveness", "cost_tier", "organic_compatible", "field_provenance",
	} {
		delete(all, k)
	}
	rec.Extras = all
	return &rec, nil
}

func upsertRemedy(ctx context.Context, tx pgx.Tx, r *remedyRecord) (bool, error) {
	attrsJSON, err := json.Marshal(r.Extras)
	if err != nil {
		return false, fmt.Errorf("marshal attrs: %w", err)
	}
	provJSON, err := json.Marshal(r.FieldProvenance)
	if err != nil {
		return false, fmt.Errorf("marshal field_provenance: %w", err)
	}

	var existing bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM remedy WHERE slug = $1 AND version = $2)`,
		r.Slug, r.Version,
	).Scan(&existing); err != nil {
		return false, fmt.Errorf("check remedy: %w", err)
	}

	// Unpack chemical sub-object into flat columns; non-chemical types
	// leave everything here empty / nil.
	var (
		activeIngredient, concentration, formulation string
		dosage, applicationMethod, frequency         string
		whoHazardClass, doaRegistrationNo            string
		preHarvestIntervalDays, reEntryIntervalHours *int
	)
	if r.Chemical != nil {
		activeIngredient = r.Chemical.ActiveIngredient
		concentration = r.Chemical.Concentration
		formulation = r.Chemical.Formulation
		dosage = r.Chemical.Dosage
		applicationMethod = r.Chemical.ApplicationMethod
		frequency = r.Chemical.Frequency
		whoHazardClass = r.Chemical.WhoHazardClass
		doaRegistrationNo = r.Chemical.DoaRegistrationNo
		preHarvestIntervalDays = r.Chemical.PreHarvestIntervalDays
		reEntryIntervalHours = r.Chemical.ReEntryIntervalHours
	}

	const sql = `
INSERT INTO remedy (
	slug, version, status, type,
	target_disease_slugs, target_pest_slugs, applicable_crop_slugs,
	active_ingredient, concentration, formulation, doa_registration_no,
	dosage, application_method, frequency,
	pre_harvest_interval_days, re_entry_interval_hours, who_hazard_class,
	effectiveness, cost_tier, organic_compatible,
	attrs, field_provenance
) VALUES (
	$1, $2, $3::record_status, $4::remedy_type,
	$5, $6, $7,
	NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''),
	NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''),
	$15, $16, NULLIF($17, ''),
	NULLIF($18, ''), NULLIF($19, ''), $20,
	$21, $22
)
ON CONFLICT (slug, version) DO UPDATE SET
	status = EXCLUDED.status,
	type = EXCLUDED.type,
	target_disease_slugs = EXCLUDED.target_disease_slugs,
	target_pest_slugs = EXCLUDED.target_pest_slugs,
	applicable_crop_slugs = EXCLUDED.applicable_crop_slugs,
	active_ingredient = EXCLUDED.active_ingredient,
	concentration = EXCLUDED.concentration,
	formulation = EXCLUDED.formulation,
	doa_registration_no = EXCLUDED.doa_registration_no,
	dosage = EXCLUDED.dosage,
	application_method = EXCLUDED.application_method,
	frequency = EXCLUDED.frequency,
	pre_harvest_interval_days = EXCLUDED.pre_harvest_interval_days,
	re_entry_interval_hours = EXCLUDED.re_entry_interval_hours,
	who_hazard_class = EXCLUDED.who_hazard_class,
	effectiveness = EXCLUDED.effectiveness,
	cost_tier = EXCLUDED.cost_tier,
	organic_compatible = EXCLUDED.organic_compatible,
	attrs = EXCLUDED.attrs,
	field_provenance = EXCLUDED.field_provenance,
	updated_at = now()
`
	if _, err := tx.Exec(ctx, sql,
		r.Slug, r.Version, r.Status, r.Type,
		coalesceSlugs(r.TargetDiseaseSlugs), coalesceSlugs(r.TargetPestSlugs), coalesceSlugs(r.ApplicableCropSlugs),
		activeIngredient, concentration, formulation, doaRegistrationNo,
		dosage, applicationMethod, frequency,
		preHarvestIntervalDays, reEntryIntervalHours, whoHazardClass,
		r.Effectiveness, r.CostTier, r.OrganicCompatible,
		attrsJSON, provJSON,
	); err != nil {
		return false, fmt.Errorf("upsert remedy: %w", err)
	}

	// Rebuild translations for name / description / instructions /
	// safety_notes. safety_notes comes off the chemical sub-object for
	// chemical-type remedies only.
	if _, err := tx.Exec(ctx,
		`DELETE FROM translation WHERE entity_type = 'remedy' AND entity_slug = $1`,
		r.Slug,
	); err != nil {
		return false, fmt.Errorf("delete translations: %w", err)
	}
	fields := map[string]map[string]string{
		"name":         r.Name,
		"description":  r.Description,
		"instructions": r.Instructions,
	}
	if r.Chemical != nil && r.Chemical.SafetyNotes != nil {
		fields["safety_notes"] = r.Chemical.SafetyNotes
	}
	for field, values := range fields {
		for locale, value := range values {
			if value == "" {
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO translation (entity_type, entity_slug, field, locale, value, status)
				 VALUES ('remedy', $1, $2, $3, $4, 'machine_draft')`,
				r.Slug, field, locale, value,
			); err != nil {
				return false, fmt.Errorf("insert %s translation: %w", field, err)
			}
		}
	}

	return !existing, nil
}
