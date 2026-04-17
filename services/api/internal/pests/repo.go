// Package pests serves the Pest entity over the admin review surface.
// Mirrors the diseases package shape so a future generalisation across
// canonical review entities is a mechanical refactor.
package pests

import (
	"context"
	"errors"

	"github.com/goyama/api/internal/review"
)

// ErrNotFound is returned when a pest slug has no record.
var ErrNotFound = errors.New("pest not found")

// ErrRequiresDatabase mirrors the other admin-package sentinels.
var ErrRequiresDatabase = errors.New("operation requires Postgres (set DATABASE_URL)")

// Pest is the public shape returned by the admin endpoints.
type Pest struct {
	Slug              string            `json:"slug"`
	ScientificName    string            `json:"scientific_name,omitempty"`
	Kingdom           string            `json:"kingdom"`
	AffectedCropSlugs []string          `json:"affected_crop_slugs,omitempty"`
	LifeStages        []string          `json:"life_stages,omitempty"`
	FeedingType       []string          `json:"feeding_type,omitempty"`
	FavoredConditions map[string]any    `json:"favored_conditions,omitempty"`
	Names             map[string]string `json:"names,omitempty"`
	Aliases           []string          `json:"aliases,omitempty"`
	Description       map[string]string `json:"description,omitempty"`
	EconomicThreshold map[string]string `json:"economic_threshold,omitempty"`
	Status            string            `json:"status,omitempty"`
	Attrs             map[string]any    `json:"attrs,omitempty"`
	FieldProvenance   map[string]any    `json:"field_provenance,omitempty"`
	ReviewedBy        string            `json:"reviewed_by,omitempty"`
	ReviewedAt        string            `json:"reviewed_at,omitempty"`
	ReviewNotes       string            `json:"review_notes,omitempty"`
}

// Repository is the read + status-mutation surface for pests. Satisfies
// review.Repo[Pest] so it can be mounted via the generic review.Routes
// factory.
type Repository interface {
	ListByStatus(ctx context.Context, status string) ([]Pest, error)
	Get(ctx context.Context, slug string) (Pest, error)
	SetStatus(ctx context.Context, slug string, update review.StatusUpdate) error
}

// JSONLRepo is a no-op placeholder that returns ErrRequiresDatabase for
// every method so a deploy without DATABASE_URL fails loudly.
type JSONLRepo struct{}

// NewJSONLRepo returns the placeholder repo.
func NewJSONLRepo() *JSONLRepo { return &JSONLRepo{} }

// ListByStatus always returns ErrRequiresDatabase.
func (*JSONLRepo) ListByStatus(context.Context, string) ([]Pest, error) {
	return nil, ErrRequiresDatabase
}

// Get always returns ErrRequiresDatabase.
func (*JSONLRepo) Get(context.Context, string) (Pest, error) {
	return Pest{}, ErrRequiresDatabase
}

// SetStatus always returns ErrRequiresDatabase.
func (*JSONLRepo) SetStatus(context.Context, string, review.StatusUpdate) error {
	return ErrRequiresDatabase
}
