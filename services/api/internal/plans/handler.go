package plans

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/platform/httpx"
)

// Handler wires plans HTTP routes.
type Handler struct {
	repo Repository
}

// New returns a plans HTTP handler.
func New(repo Repository) *Handler { return &Handler{repo: repo} }

// Routes returns a chi sub-router for /v1/cultivation-plans.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{slug}", h.get)
	return r
}

// ByCropHandler returns a GET handler mounted under the crops route at
// /v1/crops/{slug}/cultivation-plans. Kept next to the crop detail path
// so the URL structure mirrors the UI.
func (h *Handler) ByCropHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cropSlug := chi.URLParam(r, "slug")
		items, err := h.repo.ListByCrop(r.Context(), cropSlug)
		if err != nil {
			httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{
			"crop_slug": cropSlug,
			"items":     items,
			"count":     len(items),
		})
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	p, err := h.repo.Get(r.Context(), slug)
	if errors.Is(err, ErrNotFound) {
		httpx.Problem(w, r, http.StatusNotFound, "plan-not-found", "no cultivation plan with slug "+slug)
		return
	}
	if err != nil {
		httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}
