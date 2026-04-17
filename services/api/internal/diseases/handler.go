package diseases

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/platform/httpx"
)

// Handler serves the farmer-facing /v1/diseases endpoints. Always filters
// to status='published' so draft / in-review records never leak out —
// even a direct slug hit returns 404 if the record isn't published.
type Handler struct {
	repo Repository
}

// NewHandler returns a farmer-facing Handler.
func NewHandler(repo Repository) *Handler { return &Handler{repo: repo} }

// Routes returns a chi sub-router for /v1/diseases.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Get("/{slug}", h.get)
	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListByStatus(r.Context(), "published")
	if err != nil {
		httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	d, err := h.repo.Get(r.Context(), slug)
	if errors.Is(err, ErrNotFound) {
		httpx.Problem(w, r, http.StatusNotFound, "disease-not-found",
			"no disease with slug "+slug)
		return
	}
	if err != nil {
		httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
		return
	}
	// Hide non-published records behind a 404 — the farmer surface
	// should never reveal draft content, even with a correct slug.
	if d.Status != "published" {
		httpx.Problem(w, r, http.StatusNotFound, "disease-not-found",
			"no published disease with slug "+slug)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}
