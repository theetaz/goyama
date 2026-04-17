// Package admin exposes the review-queue surface used by the internal
// web-admin app. Endpoints live under /v1/admin/* and are separated from
// the public /v1/crops routes so the public API never exposes draft records
// or status-mutation verbs.
//
// Authentication is intentionally minimal for now: a `X-Goyama-Reviewer`
// header identifies the agronomist making a change. A future PR will
// replace this with OIDC / staff-SSO; until then the admin app is expected
// to run behind a trusted network perimeter (VPN / Tailscale / localhost).
package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/crops"
	"github.com/goyama/api/internal/diseases"
	"github.com/goyama/api/internal/platform/httpx"
)

// ReviewerHeader is the HTTP header carrying the reviewer identity.
const ReviewerHeader = "X-Goyama-Reviewer"

// Handler wires the admin HTTP routes to each per-entity repository.
type Handler struct {
	crops    crops.Repository
	diseases diseases.Repository
}

// New returns an admin Handler backed by the given repositories.
func New(cropsRepo crops.Repository, diseasesRepo diseases.Repository) *Handler {
	return &Handler{crops: cropsRepo, diseases: diseasesRepo}
}

// Routes returns a chi sub-router mounted at /v1/admin.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Use(requireReviewer)
	r.Route("/cultivation-steps", func(r chi.Router) {
		r.Get("/", h.listSteps)
		r.Get("/{slug}", h.getStep)
		r.Patch("/{slug}", h.patchStep)
	})
	r.Route("/diseases", func(r chi.Router) {
		r.Get("/", h.listDiseases)
		r.Get("/{slug}", h.getDisease)
		r.Patch("/{slug}", h.patchDisease)
	})
	return r
}

// requireReviewer rejects admin requests that don't include a reviewer
// header. Read requests with an empty header still succeed but don't attach
// an identity; writes (PATCH) are blocked.
func requireReviewer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && strings.TrimSpace(r.Header.Get(ReviewerHeader)) == "" {
			httpx.Problem(w, r, http.StatusUnauthorized, "reviewer-required",
				"admin writes require the "+ReviewerHeader+" header")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) listSteps(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "draft"
	}
	items, err := h.crops.ListCultivationStepsByStatus(r.Context(), status)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status": status,
		"items":  items,
		"count":  len(items),
	})
}

func (h *Handler) getStep(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	step, err := h.crops.GetCultivationStep(r.Context(), slug)
	if errors.Is(err, crops.ErrNotFound) {
		httpx.Problem(w, r, http.StatusNotFound, "step-not-found", "no cultivation step with slug "+slug)
		return
	}
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, step)
}

type patchBody struct {
	Status string `json:"status"`
	Notes  string `json:"review_notes"`
}

// validTransitions is the set of allowed status moves per CLAUDE.md #5 —
// an agronomist can put a draft into review, accept (publish), or reject.
// Once published, a record can only be deprecated (from any state); rolling
// a published record back to draft is explicitly disallowed so an accident
// can't quietly republish old copy.
var validTransitions = map[string]map[string]bool{
	"draft":      {"in_review": true, "published": true, "rejected": true},
	"in_review":  {"published": true, "rejected": true, "draft": true},
	"published":  {"deprecated": true},
	"deprecated": {"published": true},
	"rejected":   {"draft": true},
}

func (h *Handler) patchStep(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var body patchBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Problem(w, r, http.StatusBadRequest, "invalid-json", err.Error())
		return
	}
	body.Status = strings.TrimSpace(body.Status)
	if body.Status == "" {
		httpx.Problem(w, r, http.StatusBadRequest, "status-required", "status is required")
		return
	}

	current, err := h.crops.GetCultivationStep(r.Context(), slug)
	if errors.Is(err, crops.ErrNotFound) {
		httpx.Problem(w, r, http.StatusNotFound, "step-not-found", "no cultivation step with slug "+slug)
		return
	}
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	if allowed := validTransitions[current.Status]; !allowed[body.Status] && current.Status != body.Status {
		httpx.Problem(w, r, http.StatusBadRequest, "invalid-transition",
			"cannot transition from "+current.Status+" to "+body.Status)
		return
	}

	update := crops.StatusUpdate{
		Status:     body.Status,
		ReviewedBy: strings.TrimSpace(r.Header.Get(ReviewerHeader)),
		Notes:      strings.TrimSpace(body.Notes),
	}
	if err := h.crops.SetCultivationStepStatus(r.Context(), slug, update); err != nil {
		h.writeRepoError(w, r, err)
		return
	}

	// Return the fresh record so the client can reflect the new status
	// without a second round-trip.
	step, err := h.crops.GetCultivationStep(r.Context(), slug)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, step)
}

// writeRepoError maps repo-package sentinel errors to HTTP responses. A
// missing DATABASE_URL in particular should be a 503 so the admin app can
// render a helpful hint rather than a generic 500.
func (h *Handler) writeRepoError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, crops.ErrRequiresDatabase) || errors.Is(err, diseases.ErrRequiresDatabase) {
		httpx.Problem(w, r, http.StatusServiceUnavailable, "db-required",
			"admin endpoints require DATABASE_URL to be set on the API")
		return
	}
	httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
}

// ─── disease handlers ──────────────────────────────────────────────────────

func (h *Handler) listDiseases(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "draft"
	}
	items, err := h.diseases.ListByStatus(r.Context(), status)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status": status,
		"items":  items,
		"count":  len(items),
	})
}

func (h *Handler) getDisease(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	d, err := h.diseases.Get(r.Context(), slug)
	if errors.Is(err, diseases.ErrNotFound) {
		httpx.Problem(w, r, http.StatusNotFound, "disease-not-found", "no disease with slug "+slug)
		return
	}
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (h *Handler) patchDisease(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var body patchBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Problem(w, r, http.StatusBadRequest, "invalid-json", err.Error())
		return
	}
	body.Status = strings.TrimSpace(body.Status)
	if body.Status == "" {
		httpx.Problem(w, r, http.StatusBadRequest, "status-required", "status is required")
		return
	}

	current, err := h.diseases.Get(r.Context(), slug)
	if errors.Is(err, diseases.ErrNotFound) {
		httpx.Problem(w, r, http.StatusNotFound, "disease-not-found", "no disease with slug "+slug)
		return
	}
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	if allowed := validTransitions[current.Status]; !allowed[body.Status] && current.Status != body.Status {
		httpx.Problem(w, r, http.StatusBadRequest, "invalid-transition",
			"cannot transition from "+current.Status+" to "+body.Status)
		return
	}

	update := diseases.StatusUpdate{
		Status:     body.Status,
		ReviewedBy: strings.TrimSpace(r.Header.Get(ReviewerHeader)),
		Notes:      strings.TrimSpace(body.Notes),
	}
	if err := h.diseases.SetStatus(r.Context(), slug, update); err != nil {
		h.writeRepoError(w, r, err)
		return
	}

	d, err := h.diseases.Get(r.Context(), slug)
	if err != nil {
		h.writeRepoError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}
