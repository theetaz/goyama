package markets

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/platform/httpx"
)

// Handler wires the markets HTTP routes to the repository.
type Handler struct {
	repo Repository
}

// NewHandler returns a markets HTTP handler.
func NewHandler(repo Repository) *Handler { return &Handler{repo: repo} }

// Routes returns a chi sub-router for /v1/market-prices.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Get("/latest/{market}", h.latest)
	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	filter := ListFilter{
		Market:   q.Get("market"),
		CropSlug: q.Get("crop"),
		Limit:    limit,
		Offset:   offset,
	}
	if s := q.Get("since"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			httpx.Problem(w, r, http.StatusBadRequest, "invalid-since",
				"`since` must be ISO date YYYY-MM-DD")
			return
		}
		filter.Since = t
	}
	if u := q.Get("until"); u != "" {
		t, err := time.Parse("2006-01-02", u)
		if err != nil {
			httpx.Problem(w, r, http.StatusBadRequest, "invalid-until",
				"`until` must be ISO date YYYY-MM-DD")
			return
		}
		filter.Until = t
	}

	items, err := h.repo.List(r.Context(), filter)
	switch {
	case errors.Is(err, ErrRequiresDatabase):
		httpx.Problem(w, r, http.StatusServiceUnavailable, "markets-disabled",
			"market prices require the Postgres-backed deployment")
		return
	case err != nil:
		httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items": items,
		"count": len(items),
	})
}

func (h *Handler) latest(w http.ResponseWriter, r *http.Request) {
	market := chi.URLParam(r, "market")
	if market == "" {
		httpx.Problem(w, r, http.StatusBadRequest, "missing-market", "market code is required")
		return
	}
	items, err := h.repo.Latest(r.Context(), market)
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Problem(w, r, http.StatusNotFound, "no-prices",
			"no observations found for market "+market)
		return
	case errors.Is(err, ErrRequiresDatabase):
		httpx.Problem(w, r, http.StatusServiceUnavailable, "markets-disabled",
			"market prices require the Postgres-backed deployment")
		return
	case err != nil:
		httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"market": market,
		"items":  items,
		"count":  len(items),
	})
}
