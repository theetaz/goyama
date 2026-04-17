// Package diseases serves the Disease entity over the admin review surface.
// Mirrors the crops package but only implements the admin-facing operations
// today (list-by-status, get, set-status) — the farmer-facing disease
// endpoints will land with the scanner / advisory workstreams.
package diseases

import (
	"context"
	"errors"

	"github.com/goyama/api/internal/review"
)

// ErrNotFound is returned when a disease slug has no record.
var ErrNotFound = errors.New("disease not found")

// ErrRequiresDatabase mirrors the crops-package sentinel and is returned
// by the JSONL fallback for admin operations that can only be satisfied by
// Postgres.
var ErrRequiresDatabase = errors.New("operation requires Postgres (set DATABASE_URL)")

// Disease is the public shape returned by the admin endpoints. Names,
// aliases, and description come back as locale -> string maps so the
// review UI renders en/si/ta in one round-trip without a follow-up join.
type Disease struct {
	Slug              string            `json:"slug"`
	ScientificName    string            `json:"scientific_name,omitempty"`
	CausalOrganism    string            `json:"causal_organism"`
	CausalSpecies     string            `json:"causal_species,omitempty"`
	Severity          string            `json:"severity,omitempty"`
	AffectedCropSlugs []string          `json:"affected_crop_slugs,omitempty"`
	AffectedParts     []string          `json:"affected_parts,omitempty"`
	Transmission      []string          `json:"transmission,omitempty"`
	ConfusedWith      []string          `json:"confused_with,omitempty"`
	FavoredConditions map[string]any    `json:"favored_conditions,omitempty"`
	Names             map[string]string `json:"names,omitempty"`
	Aliases           []string          `json:"aliases,omitempty"`
	Description       map[string]string `json:"description,omitempty"`
	Status            string            `json:"status,omitempty"`
	Attrs             map[string]any    `json:"attrs,omitempty"`
	FieldProvenance   map[string]any    `json:"field_provenance,omitempty"`
	ReviewedBy        string            `json:"reviewed_by,omitempty"`
	ReviewedAt        string            `json:"reviewed_at,omitempty"`
	ReviewNotes       string            `json:"review_notes,omitempty"`
}

// Repository is the read + status-mutation surface for diseases. It
// satisfies review.Repo[Disease] — the admin routes are mounted via
// the generic review.Routes factory.
type Repository interface {
	ListByStatus(ctx context.Context, status string) ([]Disease, error)
	Get(ctx context.Context, slug string) (Disease, error)
	SetStatus(ctx context.Context, slug string, update review.StatusUpdate) error
}

// JSONLRepo is a no-op placeholder that returns ErrRequiresDatabase for
// every method. Production API + admin boot with DATABASE_URL set; this
// exists so local dev without Postgres has a clear error path.
type JSONLRepo struct{}

// NewJSONLRepo returns the placeholder repo.
func NewJSONLRepo() *JSONLRepo { return &JSONLRepo{} }

// ListByStatus always returns ErrRequiresDatabase.
func (*JSONLRepo) ListByStatus(context.Context, string) ([]Disease, error) {
	return nil, ErrRequiresDatabase
}

// Get always returns ErrRequiresDatabase.
func (*JSONLRepo) Get(context.Context, string) (Disease, error) {
	return Disease{}, ErrRequiresDatabase
}

// SetStatus always returns ErrRequiresDatabase.
func (*JSONLRepo) SetStatus(context.Context, string, review.StatusUpdate) error {
	return ErrRequiresDatabase
}
