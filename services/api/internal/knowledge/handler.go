package knowledge

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/platform/httpx"
)

// Handler wires the knowledge-graph HTTP routes.
type Handler struct {
	repo Repository
}

// New returns a knowledge handler.
func New(repo Repository) *Handler { return &Handler{repo: repo} }

// ByEntityHandler returns a GET handler that serves chunks for one entity.
// Mounted under each entity's public route, e.g.
// `/v1/crops/{slug}/knowledge`, `/v1/diseases/{slug}/knowledge`.
func (h *Handler) ByEntityHandler(entityType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		chunks, err := h.repo.ListByEntity(r.Context(), entityType, slug)
		if err != nil {
			httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
			return
		}
		// Attach source metadata so the client can render authority badges
		// without a second round-trip per chunk.
		bySource := map[string]Source{}
		for _, c := range chunks {
			if _, seen := bySource[c.SourceSlug]; seen {
				continue
			}
			if s, err := h.repo.GetSource(r.Context(), c.SourceSlug); err == nil {
				bySource[c.SourceSlug] = s
			}
		}
		sources := make([]Source, 0, len(bySource))
		for _, s := range bySource {
			sources = append(sources, s)
		}
		httpx.JSON(w, http.StatusOK, map[string]any{
			"entity_type": entityType,
			"entity_slug": slug,
			"chunks":      chunks,
			"sources":     sources,
			"count":       len(chunks),
		})
	}
}
