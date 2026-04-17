package crops

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/platform/httpx"
)

// Handler wires crops HTTP routes to the repository.
type Handler struct {
	repo Repository
}

// NewHandler returns a crops HTTP handler.
func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Routes returns a chi sub-router for /v1/crops.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Get("/{slug}", h.get)
	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := ListFilter{
		Category: q.Get("category"),
		Query:    q.Get("q"),
		Limit:    limit,
		Offset:   offset,
	}
	items, err := h.repo.List(r.Context(), filter)
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
	c, err := h.repo.Get(r.Context(), slug)
	if errors.Is(err, ErrNotFound) {
		httpx.Problem(w, r, http.StatusNotFound, "crop-not-found", "no crop with slug "+slug)
		return
	}
	if err != nil {
		httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}
