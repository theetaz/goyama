package health

import (
	"net/http"
	"time"

	"github.com/goyama/api/internal/platform/httpx"
)

// Handler serves /v1/health.
type Handler struct {
	started time.Time
	version string
}

// New returns a health handler with current process start time + build version.
func New(version string) *Handler {
	return &Handler{started: time.Now(), version: version}
}

// Get returns a small JSON payload describing service liveness.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"version":    h.version,
		"uptime_sec": int(time.Since(h.started).Seconds()),
		"time":       time.Now().UTC().Format(time.RFC3339),
	})
}
