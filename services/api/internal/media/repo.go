// Package media is the read + admin surface for media records — currently
// scoped to disease images, but the schema (and this package) is generic
// over entity_type so pest images and crop reference photos drop in
// without further changes.
//
// Storage policy (per CLAUDE.md): we link to images on their original
// host. The `external_link` hosting mode is the only one wired here. A
// future `own` hosting path lands when the R2 + Cloudflare Images
// integration is in place; until then the loader and admin UI both
// require an external URL.
package media

import (
	"context"
	"errors"

	"github.com/goyama/api/internal/review"
)

// ErrNotFound is returned when a media slug has no record.
var ErrNotFound = errors.New("media not found")

// ErrRequiresDatabase is returned by the stub repo when the API runs
// against the JSONL fallback.
var ErrRequiresDatabase = errors.New("media operations require Postgres (set DATABASE_URL)")

// Media is the public + admin shape returned by every endpoint.
type Media struct {
	Slug        string            `json:"slug"`
	Type        string            `json:"type"`          // 'image' | 'video' | 'pdf' | 'audio' | 'transcript'
	Hosting     string            `json:"hosting"`       // 'own' | 'external_link'
	URL         string            `json:"url,omitempty"` // populated when hosting='own'
	ExternalURL string            `json:"external_url,omitempty"`
	Credit      string            `json:"credit,omitempty"`
	Licence     string            `json:"licence"`
	Caption     map[string]string `json:"caption,omitempty"`
	EntityType  string            `json:"entity_type,omitempty"`
	EntitySlug  string            `json:"entity_slug,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Status      string            `json:"status"`
	ReviewedBy  string            `json:"reviewed_by,omitempty"`
	ReviewedAt  string            `json:"reviewed_at,omitempty"`
	ReviewNotes string            `json:"review_notes,omitempty"`
}

// AttachInput captures everything the admin needs to attach a new
// external image to a disease (or any reviewable entity).
type AttachInput struct {
	EntityType  string // 'disease' | 'pest' | 'crop'
	EntitySlug  string
	Type        string // defaults to 'image'
	ExternalURL string
	Credit      string
	Licence     string
	Tags        []string
	CreatedBy   string
}

// Repository is the per-entity media surface used by both the admin
// gallery and the farmer-facing image strip on disease detail pages.
type Repository interface {
	// ListByEntity returns every media record attached to the entity at
	// the requested status. Pass status='' to get everything (admin only).
	ListByEntity(ctx context.Context, entityType, entitySlug, status string) ([]Media, error)
	// Get returns a single media record by slug.
	Get(ctx context.Context, slug string) (Media, error)
	// Attach creates a fresh external-link media record in 'in_review'
	// status and returns its slug. Idempotent on (entity, external_url):
	// re-attaching the same URL returns the existing record.
	Attach(ctx context.Context, in AttachInput) (Media, error)
	// SetStatus is the review-lifecycle write — promote / reject / etc.
	// Same shape as the cultivation-step / disease admin queues.
	SetStatus(ctx context.Context, slug string, u review.StatusUpdate) error
}

// StubRepo backs the JSONL fallback — every call fails so the admin UI
// surfaces an explicit "geo / media features need Postgres" message
// rather than silently returning an empty list.
type StubRepo struct{}

// NewStubRepo returns the JSONL-mode placeholder.
func NewStubRepo() *StubRepo { return &StubRepo{} }

// ListByEntity always returns ErrRequiresDatabase.
func (*StubRepo) ListByEntity(context.Context, string, string, string) ([]Media, error) {
	return nil, ErrRequiresDatabase
}

// Get always returns ErrRequiresDatabase.
func (*StubRepo) Get(context.Context, string) (Media, error) {
	return Media{}, ErrRequiresDatabase
}

// Attach always returns ErrRequiresDatabase.
func (*StubRepo) Attach(context.Context, AttachInput) (Media, error) {
	return Media{}, ErrRequiresDatabase
}

// SetStatus always returns ErrRequiresDatabase.
func (*StubRepo) SetStatus(context.Context, string, review.StatusUpdate) error {
	return ErrRequiresDatabase
}
