package geo

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/platform/httpx"
)

// Handler wires the geo HTTP routes to the repository.
type Handler struct {
	repo Repository
}

// NewHandler returns a geo HTTP handler.
func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// Routes returns a chi sub-router for /v1/geo.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/lookup", h.lookup)
	return r
}

// lookup serves GET /v1/geo/lookup?lat=<float>&lng=<float>.
//
// Returns 200 with a Lookup payload on success, 400 when params are
// missing or out of Sri Lanka's lat/lng envelope, 404 when no layer
// covers the point, and 503 when the API is in JSONL-only mode.
func (h *Handler) lookup(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	latStr, lngStr := q.Get("lat"), q.Get("lng")
	if latStr == "" || lngStr == "" {
		httpx.Problem(w, r, http.StatusBadRequest, "missing-coordinates",
			"both `lat` and `lng` query parameters are required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		httpx.Problem(w, r, http.StatusBadRequest, "invalid-lat",
			"lat must be a decimal number")
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		httpx.Problem(w, r, http.StatusBadRequest, "invalid-lng",
			"lng must be a decimal number")
		return
	}

	// Sri Lanka's mainland envelope (with a small margin for offshore
	// reefs). Reject obviously-bad coordinates here so we don't burn a
	// PostGIS round-trip on a typo'd request.
	if lat < 5.5 || lat > 10.2 || lng < 79.3 || lng > 82.2 {
		httpx.Problem(w, r, http.StatusBadRequest, "outside-sri-lanka",
			"coordinates must lie within Sri Lanka (lat 5.5–10.2, lng 79.3–82.2)")
		return
	}

	res, err := h.repo.Lookup(r.Context(), Point{Lat: lat, Lng: lng})
	switch {
	case errors.Is(err, ErrLocationNotFound):
		httpx.Problem(w, r, http.StatusNotFound, "location-not-covered",
			"no administrative or AEZ layer covers this point")
		return
	case errors.Is(err, ErrRequiresDatabase):
		httpx.Problem(w, r, http.StatusServiceUnavailable, "geo-disabled",
			"geo lookup requires the Postgres-backed deployment with geo layers loaded")
		return
	case err != nil:
		httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, res)
}
