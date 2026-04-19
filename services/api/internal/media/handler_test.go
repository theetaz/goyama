package media

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/goyama/api/internal/review"
)

type fakeRepo struct {
	listRes []Media
	listErr error
	getRes  Media
	getErr  error
	attached AttachInput
	attachErr error

	lastSetSlug string
	lastSetU    review.StatusUpdate
}

func (f *fakeRepo) ListByEntity(_ context.Context, et, es, status string) ([]Media, error) {
	_ = et
	_ = es
	_ = status
	return f.listRes, f.listErr
}
func (f *fakeRepo) Get(_ context.Context, _ string) (Media, error) { return f.getRes, f.getErr }
func (f *fakeRepo) Attach(_ context.Context, in AttachInput) (Media, error) {
	f.attached = in
	if f.attachErr != nil {
		return Media{}, f.attachErr
	}
	return Media{Slug: "test-img-1", ExternalURL: in.ExternalURL, Status: "in_review"}, nil
}
func (f *fakeRepo) SetStatus(_ context.Context, slug string, u review.StatusUpdate) error {
	f.lastSetSlug = slug
	f.lastSetU = u
	return nil
}

func adminServer(repo Repository) http.Handler {
	r := chi.NewRouter()
	r.Use(review.RequireReviewer)
	r.Mount("/admin/media", New(repo).AdminRoutes())
	return r
}

func TestAdminAttach_RequiresReviewer(t *testing.T) {
	repo := &fakeRepo{}
	srv := adminServer(repo)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/admin/media/by-entity/disease/rice-blast",
		strings.NewReader(`{"external_url":"https://example.org/x.jpg","licence":"CC-BY 4.0"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestAdminAttach_OK(t *testing.T) {
	repo := &fakeRepo{}
	srv := adminServer(repo)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/admin/media/by-entity/disease/rice-blast",
		strings.NewReader(`{"external_url":"https://example.org/x.jpg","licence":"CC-BY 4.0","credit":"Wikimedia"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(review.ReviewerHeader, "agronomist@example.org")
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	if repo.attached.EntityType != "disease" || repo.attached.EntitySlug != "rice-blast" {
		t.Fatalf("entity not threaded through: %+v", repo.attached)
	}
	if repo.attached.CreatedBy != "agronomist@example.org" {
		t.Fatalf("reviewer header not threaded through: %q", repo.attached.CreatedBy)
	}
}

func TestPublicGallery_StubReturnsEmpty(t *testing.T) {
	h := New(NewStubRepo()).PublicGalleryHandler("disease")
	r := chi.NewRouter()
	r.Get("/diseases/{slug}/images", h)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/diseases/rice-blast/images", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"items":[]`) {
		t.Fatalf("want empty items array, got %s", rr.Body.String())
	}
}

func TestAdminPatch_RejectsInvalidTransition(t *testing.T) {
	repo := &fakeRepo{getRes: Media{Slug: "x", Status: "published"}}
	srv := adminServer(repo)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/admin/media/x",
		strings.NewReader(`{"status":"draft"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(review.ReviewerHeader, "agronomist@example.org")
	srv.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
}
