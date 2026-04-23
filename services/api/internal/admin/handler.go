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
	"github.com/goyama/api/internal/knowledge"
	"github.com/goyama/api/internal/media"
	"github.com/goyama/api/internal/pests"
	"github.com/goyama/api/internal/plans"
	"github.com/goyama/api/internal/remedies"
	"github.com/goyama/api/internal/review"
)

// Handler wires the admin HTTP routes to each per-entity repository via
// the generic review.Routes factory.
type Handler struct {
	steps     crops.CultivationStepRepo
	diseases  diseases.Repository
	pests     pests.Repository
	remedies  remedies.Repository
	plans     plans.AdminRepo
	knowledge knowledge.AdminRepo
	media     *media.Handler
}

// New returns an admin Handler backed by the given repositories.
func New(
	stepsRepo crops.CultivationStepRepo,
	diseasesRepo diseases.Repository,
	pestsRepo pests.Repository,
	remediesRepo remedies.Repository,
	plansRepo plans.AdminRepo,
	knowledgeRepo knowledge.AdminRepo,
	mediaH *media.Handler,
) *Handler {
	return &Handler{
		steps:     stepsRepo,
		diseases:  diseasesRepo,
		pests:     pestsRepo,
		remedies:  remediesRepo,
		plans:     plansRepo,
		knowledge: knowledgeRepo,
		media:     mediaH,
	}
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

	r.Mount("/remedies", review.Routes(review.Entity[remedies.Remedy]{
		Repo:          h.remedies,
		Label:         "remedy",
		StatusFn:      func(rem remedies.Remedy) string { return rem.Status },
		NotFoundErr:   remedies.ErrNotFound,
		RequiresDBErr: remedies.ErrRequiresDatabase,
	}))

	r.Mount("/cultivation-plans", review.Routes(review.Entity[plans.Plan]{
		Repo:          h.plans,
		Label:         "cultivation-plan",
		StatusFn:      func(p plans.Plan) string { return p.Status },
		NotFoundErr:   plans.ErrNotFound,
		RequiresDBErr: plans.ErrRequiresDatabase,
	}))

	r.Mount("/knowledge-chunks", review.Routes(review.Entity[knowledge.Chunk]{
		Repo:          h.knowledge,
		Label:         "knowledge-chunk",
		StatusFn:      func(c knowledge.Chunk) string { return c.Status },
		NotFoundErr:   knowledge.ErrNotFound,
		RequiresDBErr: knowledge.ErrRequiresDatabase,
	}))

	// Media doesn't fit the generic review.Routes shape — its admin
	// surface adds an attach endpoint and is queried per-entity rather
	// than by status. Mount the bespoke handler instead.
	r.Mount("/media", h.media.AdminRoutes())

	return r
}
