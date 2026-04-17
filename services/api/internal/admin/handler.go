// Package admin exposes the review-queue surface used by the internal
// web-admin app. Endpoints live under /v1/admin/* and are separated from
// the public /v1/crops routes so the public API never exposes draft records
// or status-mutation verbs.
//
// Authentication is intentionally minimal for now: a `X-Goyama-Reviewer`
// header identifies the agronomist making a change. A future PR will
// replace this with OIDC / staff-SSO; until then the admin app is expected
// to run behind a trusted network perimeter (VPN / Tailscale / localhost).
//
// The actual list / get / patch surface is generic and lives in
// internal/review. This file just wires each canonical entity into the
// router — adding a fourth entity (remedies, cultivars, AEZs) is one
// Entity[T] value and one r.Mount call.
package admin

import (
	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/crops"
	"github.com/goyama/api/internal/diseases"
	"github.com/goyama/api/internal/pests"
	"github.com/goyama/api/internal/review"
)

// Handler wires the admin HTTP routes to each per-entity repository via
// the generic review.Routes factory.
type Handler struct {
	steps    crops.CultivationStepRepo
	diseases diseases.Repository
	pests    pests.Repository
}

// New returns an admin Handler backed by the given repositories.
func New(stepsRepo crops.CultivationStepRepo, diseasesRepo diseases.Repository, pestsRepo pests.Repository) *Handler {
	return &Handler{steps: stepsRepo, diseases: diseasesRepo, pests: pestsRepo}
}

// Routes returns a chi sub-router mounted at /v1/admin.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(review.RequireReviewer)

	r.Mount("/cultivation-steps", review.Routes(review.Entity[crops.CultivationStep]{
		Repo:          h.steps,
		Label:         "cultivation-step",
		StatusFn:      func(s crops.CultivationStep) string { return s.Status },
		NotFoundErr:   crops.ErrNotFound,
		RequiresDBErr: crops.ErrRequiresDatabase,
	}))

	r.Mount("/diseases", review.Routes(review.Entity[diseases.Disease]{
		Repo:          h.diseases,
		Label:         "disease",
		StatusFn:      func(d diseases.Disease) string { return d.Status },
		NotFoundErr:   diseases.ErrNotFound,
		RequiresDBErr: diseases.ErrRequiresDatabase,
	}))

	r.Mount("/pests", review.Routes(review.Entity[pests.Pest]{
		Repo:          h.pests,
		Label:         "pest",
		StatusFn:      func(p pests.Pest) string { return p.Status },
		NotFoundErr:   pests.ErrNotFound,
		RequiresDBErr: pests.ErrRequiresDatabase,
	}))

	return r
}
