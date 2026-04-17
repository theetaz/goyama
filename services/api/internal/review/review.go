// Package review hosts the generic promotion-lifecycle surface that
// underpins every admin review queue.
//
// Three near-identical handler triples across cultivation_step, disease,
// and pest gave us the signal to abstract: the variation between them
// is which repo holds the data, which typed struct comes back, and a
// small string label for error messages. Everything else — reviewer
// auth, valid-status-transition table, JSON shape, sentinel-to-HTTP
// mapping — is uniform and lives here.
//
// Adding a fourth entity (e.g. remedies) is now three steps:
//  1. Implement Repo[T] on the new package's repo.
//  2. Provide a StatusFn that reads Status off the entity.
//  3. Mount with admin.Handler: one Entity[T] struct.
package review

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/platform/httpx"
)

// ReviewerHeader is the HTTP header that carries the reviewer identity
// on admin writes. Header-based auth is temporary — see CLAUDE.md and
// the admin-package doc.
const ReviewerHeader = "X-Goyama-Reviewer"

// StatusUpdate is the canonical audit payload for a status change.
// Moved here from each entity package so all three (and future ones)
// share the exact same struct.
type StatusUpdate struct {
	Status     string
	ReviewedBy string
	Notes      string
}

// Repo is the uniform read + status-mutation interface every reviewable
// entity satisfies. Return the typed domain object; the review package
// treats T opaquely apart from the StatusFn on the Entity.
type Repo[T any] interface {
	ListByStatus(ctx context.Context, status string) ([]T, error)
	Get(ctx context.Context, slug string) (T, error)
	SetStatus(ctx context.Context, slug string, u StatusUpdate) error
}

// Entity bundles the per-entity plumbing Routes[T] needs beyond the Repo:
// a human label for error messages, a status accessor, and the sentinel
// errors to map to 404 / 503.
type Entity[T any] struct {
	Repo          Repo[T]
	Label         string         // e.g. "cultivation step", "disease", "pest"
	StatusFn      func(T) string // reads current status from a T for transition validation
	NotFoundErr   error          // maps to 404 via errors.Is
	RequiresDBErr error          // maps to 503 via errors.Is
}

// ValidTransitions is the set of allowed status moves. Enforced server-
// side by the PATCH handler; the web-admin UI mirrors it to gate which
// buttons it draws.
//
// Once published, a record can only be deprecated. Rolling a published
// record back to draft is explicitly disallowed — otherwise an accident
// on the review screen could quietly republish old copy.
var ValidTransitions = map[string]map[string]bool{
	"draft":      {"in_review": true, "published": true, "rejected": true},
	"in_review":  {"published": true, "rejected": true, "draft": true},
	"published":  {"deprecated": true},
	"deprecated": {"published": true},
	"rejected":   {"draft": true},
}

// RequireReviewer rejects writes that don't include a reviewer header.
// GET requests pass through — read-only introspection doesn't need an
// identity stamped on it.
func RequireReviewer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && strings.TrimSpace(r.Header.Get(ReviewerHeader)) == "" {
			httpx.Problem(w, r, http.StatusUnauthorized, "reviewer-required",
				"admin writes require the "+ReviewerHeader+" header")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type patchBody struct {
	Status string `json:"status"`
	Notes  string `json:"review_notes"`
}

// Routes returns a chi sub-router that serves GET / · GET /{slug} ·
// PATCH /{slug} for the given entity. Caller mounts it under the
// entity-specific path (e.g. /cultivation-steps).
func Routes[T any](e Entity[T]) chi.Router {
	r := chi.NewRouter()
	r.Get("/", listHandler(e))
	r.Get("/{slug}", getHandler(e))
	r.Patch("/{slug}", patchHandler(e))
	return r
}

func listHandler[T any](e Entity[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		if status == "" {
			status = "draft"
		}
		items, err := e.Repo.ListByStatus(r.Context(), status)
		if err != nil {
			writeRepoError(w, r, err, e)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]any{
			"status": status,
			"items":  items,
			"count":  len(items),
		})
	}
}

func getHandler[T any](e Entity[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "slug")
		item, err := e.Repo.Get(r.Context(), slug)
		if errors.Is(err, e.NotFoundErr) {
			httpx.Problem(w, r, http.StatusNotFound, e.Label+"-not-found",
				"no "+e.Label+" with slug "+slug)
			return
		}
		if err != nil {
			writeRepoError(w, r, err, e)
			return
		}
		httpx.JSON(w, http.StatusOK, item)
	}
}

func patchHandler[T any](e Entity[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		current, err := e.Repo.Get(r.Context(), slug)
		if errors.Is(err, e.NotFoundErr) {
			httpx.Problem(w, r, http.StatusNotFound, e.Label+"-not-found",
				"no "+e.Label+" with slug "+slug)
			return
		}
		if err != nil {
			writeRepoError(w, r, err, e)
			return
		}
		currentStatus := e.StatusFn(current)
		if allowed := ValidTransitions[currentStatus]; !allowed[body.Status] && currentStatus != body.Status {
			httpx.Problem(w, r, http.StatusBadRequest, "invalid-transition",
				"cannot transition from "+currentStatus+" to "+body.Status)
			return
		}

		update := StatusUpdate{
			Status:     body.Status,
			ReviewedBy: strings.TrimSpace(r.Header.Get(ReviewerHeader)),
			Notes:      strings.TrimSpace(body.Notes),
		}
		if err := e.Repo.SetStatus(r.Context(), slug, update); err != nil {
			writeRepoError(w, r, err, e)
			return
		}

		// Return the fresh record so the client can reflect the new
		// status without a second round-trip.
		fresh, err := e.Repo.Get(r.Context(), slug)
		if err != nil {
			writeRepoError(w, r, err, e)
			return
		}
		httpx.JSON(w, http.StatusOK, fresh)
	}
}

func writeRepoError[T any](w http.ResponseWriter, r *http.Request, err error, e Entity[T]) {
	if e.RequiresDBErr != nil && errors.Is(err, e.RequiresDBErr) {
		httpx.Problem(w, r, http.StatusServiceUnavailable, "db-required",
			"admin endpoints require DATABASE_URL to be set on the API")
		return
	}
	httpx.Problem(w, r, http.StatusInternalServerError, "internal-error", err.Error())
}
