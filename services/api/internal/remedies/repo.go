// Package remedies serves the Remedy entity over the admin review surface.
// Remedies are the highest-stakes entity per CLAUDE.md #5 — chemical
// dosages, pre-harvest intervals, and WHO hazard class live here, so the
// "must pass agronomist review before publishing" gate matters most.
package remedies

import (
	"context"
	"errors"

	"github.com/goyama/api/internal/review"
)

// ErrNotFound is returned when a remedy slug has no record.
var ErrNotFound = errors.New("remedy not found")

// ErrRequiresDatabase mirrors the other admin-package sentinels.
var ErrRequiresDatabase = errors.New("operation requires Postgres (set DATABASE_URL)")

// Remedy is the public shape returned by the admin endpoints. Fields
// below `Chemical` capture the bits specific to chemical remedies —
// biological / cultural / resistant-variety types leave those nil.
type Remedy struct {
	Slug                 string            `json:"slug"`
	Type                 string            `json:"type"`
	TargetDiseaseSlugs   []string          `json:"target_disease_slugs,omitempty"`
	TargetPestSlugs      []string          `json:"target_pest_slugs,omitempty"`
	ApplicableCropSlugs  []string          `json:"applicable_crop_slugs,omitempty"`
	ActiveIngredient     string            `json:"active_ingredient,omitempty"`
	Concentration        string            `json:"concentration,omitempty"`
	Formulation          string            `json:"formulation,omitempty"`
	DoaRegistrationNo    string            `json:"doa_registration_no,omitempty"`
	Dosage               string            `json:"dosage,omitempty"`
	ApplicationMethod    string            `json:"application_method,omitempty"`
	Frequency            string            `json:"frequency,omitempty"`
	PreHarvestIntervalD  *int              `json:"pre_harvest_interval_days,omitempty"`
	ReEntryIntervalHours *int              `json:"re_entry_interval_hours,omitempty"`
	WhoHazardClass       string            `json:"who_hazard_class,omitempty"`
	Effectiveness        string            `json:"effectiveness,omitempty"`
	CostTier             string            `json:"cost_tier,omitempty"`
	OrganicCompatible    *bool             `json:"organic_compatible,omitempty"`
	Name                 map[string]string `json:"name,omitempty"`
	Description          map[string]string `json:"description,omitempty"`
	Instructions         map[string]string `json:"instructions,omitempty"`
	SafetyNotes          map[string]string `json:"safety_notes,omitempty"`
	Attrs                map[string]any    `json:"attrs,omitempty"`
	Status               string            `json:"status,omitempty"`
	FieldProvenance      map[string]any    `json:"field_provenance,omitempty"`
	ReviewedBy           string            `json:"reviewed_by,omitempty"`
	ReviewedAt           string            `json:"reviewed_at,omitempty"`
	ReviewNotes          string            `json:"review_notes,omitempty"`
}

// Repository satisfies review.Repo[Remedy] — the admin routes mount via
// the generic review.Routes factory.
type Repository interface {
	ListByStatus(ctx context.Context, status string) ([]Remedy, error)
	Get(ctx context.Context, slug string) (Remedy, error)
	SetStatus(ctx context.Context, slug string, update review.StatusUpdate) error
}

// JSONLRepo returns ErrRequiresDatabase from every admin method so a
// misconfigured deploy fails loudly.
type JSONLRepo struct{}

// NewJSONLRepo returns the placeholder repo.
func NewJSONLRepo() *JSONLRepo { return &JSONLRepo{} }

// ListByStatus always returns ErrRequiresDatabase.
func (*JSONLRepo) ListByStatus(context.Context, string) ([]Remedy, error) {
	return nil, ErrRequiresDatabase
}

// Get always returns ErrRequiresDatabase.
func (*JSONLRepo) Get(context.Context, string) (Remedy, error) {
	return Remedy{}, ErrRequiresDatabase
}

// SetStatus always returns ErrRequiresDatabase.
func (*JSONLRepo) SetStatus(context.Context, string, review.StatusUpdate) error {
	return ErrRequiresDatabase
}
