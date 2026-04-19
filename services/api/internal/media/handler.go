package media

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/platform/httpx"
	"github.com/goyama/api/internal/review"
)

// Handler exposes both the admin (auth-gated) and public (read-only,
// published-only) media surfaces. The admin handler is mounted under
// /v1/admin; the public handler is mounted under /v1/diseases/{slug}/images
// so the gallery URL stays close to the disease detail.
type Handler struct {
	repo Repository
}

// New returns a media Handler.
func New(repo Repository) *Handler { return &Handler{repo: repo} }

// AdminRoutes returns a chi sub-router for /v1/admin/media. It hosts the
// status mutations + the per-entity attach endpoint shared across
// disease / pest / crop galleries.
func (h *Handler) AdminRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{slug}", h.adminGet)
	r.Patch("/{slug}", h.adminPatch)
	r.Get("/by-entity/{entity_type}/{entity_slug}", h.adminListByEntity)
	r.Post("/by-entity/{entity_type}/{entity_slug}", h.adminAttach)
	return r
}

// PublicGalleryHandler returns the published-only image strip for one
// entity. Mounted as a GET /v1/diseases/{slug}/images (and equivalent for
// pests once we expose them publicly).
func (h *Handler) PublicGalleryHandler(entityType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		items, err := h.repo.ListByEntity(r.Context(), entityType, slug, "published")
		switch {
		case errors.Is(err, ErrRequiresDatabase):
			// In JSONL mode we surface an empty gallery rather than
			// a 503 — the client cards still render usefully.
			httpx.JSON(w, http.StatusOK, map[string]any{
				"entity_type": entityType, "entity_slug": slug,
				"items": []Media{}, "count": 0,
			})
			return
		case err != nil:
			httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{
			"entity_type": entityType,
			"entity_slug": slug,
			"items":       items,
			"count":       len(items),
		})
	}
}

func (h *Handler) adminListByEntity(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entity_type")
	slug := chi.URLParam(r, "entity_slug")
	status := r.URL.Query().Get("status") // empty = all statuses
	items, err := h.repo.ListByEntity(r.Context(), entityType, slug, status)
	if writeErr(w, r, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"entity_type": entityType,
		"entity_slug": slug,
		"status":      status,
		"items":       items,
		"count":       len(items),
	})
}

type attachBody struct {
	ExternalURL string   `json:"external_url"`
	Credit      string   `json:"credit"`
	Licence     string   `json:"licence"`
	Tags        []string `json:"tags"`
	Type        string   `json:"type"`
}

func (h *Handler) adminAttach(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "entity_type")
	slug := chi.URLParam(r, "entity_slug")

	var body attachBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.Problem(w, r, http.StatusBadRequest, "invalid-json", err.Error())
		return
	}
	in := AttachInput{
		EntityType:  entityType,
		EntitySlug:  slug,
		Type:        body.Type,
		ExternalURL: body.ExternalURL,
		Credit:      body.Credit,
		Licence:     body.Licence,
		Tags:        body.Tags,
		CreatedBy:   strings.TrimSpace(r.Header.Get(review.ReviewerHeader)),
	}
	m, err := h.repo.Attach(r.Context(), in)
	if writeErr(w, r, err) {
		return
	}
	httpx.JSON(w, http.StatusCreated, m)
}

func (h *Handler) adminGet(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	m, err := h.repo.Get(r.Context(), slug)
	if errors.Is(err, ErrNotFound) {
		httpx.Problem(w, r, http.StatusNotFound, "media-not-found", "no media with slug "+slug)
		return
	}
	if writeErr(w, r, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, m)
}

type patchBody struct {
	Status      string `json:"status"`
	ReviewNotes string `json:"review_notes"`
}

func (h *Handler) adminPatch(w http.ResponseWriter, r *http.Request) {
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

	current, err := h.repo.Get(r.Context(), slug)
	if errors.Is(err, ErrNotFound) {
		httpx.Problem(w, r, http.StatusNotFound, "media-not-found", "no media with slug "+slug)
		return
	}
	if writeErr(w, r, err) {
		return
	}
	if allowed := review.ValidTransitions[current.Status]; !allowed[body.Status] && current.Status != body.Status {
		httpx.Problem(w, r, http.StatusBadRequest, "invalid-transition",
			"cannot transition from "+current.Status+" to "+body.Status)
		return
	}

	update := review.StatusUpdate{
		Status:     body.Status,
		ReviewedBy: strings.TrimSpace(r.Header.Get(review.ReviewerHeader)),
		Notes:      strings.TrimSpace(body.ReviewNotes),
	}
	if err := h.repo.SetStatus(r.Context(), slug, update); writeErr(w, r, err) {
		return
	}
	fresh, err := h.repo.Get(r.Context(), slug)
	if writeErr(w, r, err) {
		return
	}
	httpx.JSON(w, http.StatusOK, fresh)
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, ErrRequiresDatabase):
		httpx.Problem(w, r, http.StatusServiceUnavailable, "db-required",
			"media operations require DATABASE_URL")
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, r, http.StatusNotFound, "media-not-found", err.Error())
	default:
		httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
	}
	return true
}
