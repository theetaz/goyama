package plans

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// fakeRepo lets us exercise the handler contract without filesystem I/O.
type fakeRepo struct {
	listRes []Summary
	listErr error
	getRes  Plan
	getErr  error

	lastListCrop string
	lastGetSlug  string
}

func (f *fakeRepo) ListByCrop(_ context.Context, cropSlug string) ([]Summary, error) {
	f.lastListCrop = cropSlug
	return f.listRes, f.listErr
}
func (f *fakeRepo) Get(_ context.Context, slug string) (Plan, error) {
	f.lastGetSlug = slug
	return f.getRes, f.getErr
}

func TestGet_OK(t *testing.T) {
	repo := &fakeRepo{getRes: Plan{Summary: Summary{Slug: "x", CropSlug: "red-onion", Season: "maha"}, Status: "draft"}}
	h := New(repo).Routes()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var got Plan
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.CropSlug != "red-onion" {
		t.Fatalf("unexpected payload: %+v", got)
	}
	if repo.lastGetSlug != "x" {
		t.Fatalf("slug not threaded: %q", repo.lastGetSlug)
	}
}

func TestGet_NotFound(t *testing.T) {
	h := New(&fakeRepo{getErr: ErrNotFound}).Routes()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
}

func TestByCropHandler(t *testing.T) {
	repo := &fakeRepo{listRes: []Summary{
		{Slug: "red-onion-dry-zone-maha", CropSlug: "red-onion", Season: "maha"},
	}}
	r := chi.NewRouter()
	r.Get("/crops/{slug}/cultivation-plans", New(repo).ByCropHandler())

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/crops/red-onion/cultivation-plans", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if repo.lastListCrop != "red-onion" {
		t.Fatalf("crop not threaded: %q", repo.lastListCrop)
	}
	var got struct {
		CropSlug string    `json:"crop_slug"`
		Items    []Summary `json:"items"`
		Count    int       `json:"count"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.CropSlug != "red-onion" || got.Count != 1 {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func TestJSONLRepo_LoadsFixture(t *testing.T) {
	// Points at the corpus seed directory in this repo — the red-onion
	// fixture is authored by the ingestion agent (me) and committed
	// alongside the schema.
	repo := NewJSONLRepo("../../../../corpus/seed/cultivation_plans")
	plans, err := repo.ListByCrop(context.Background(), "red-onion")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(plans) == 0 {
		t.Fatal("expected at least one red-onion plan fixture")
	}

	p, err := repo.Get(context.Background(), "red-onion-dry-zone-maha")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if p.Authority != "doa_official" {
		t.Fatalf("expected authority=doa_official, got %q", p.Authority)
	}
	if len(p.Activities) == 0 || len(p.PestRisks) == 0 || len(p.Economics) == 0 {
		t.Fatalf("fixture missing children: %d activities, %d risks, %d economics",
			len(p.Activities), len(p.PestRisks), len(p.Economics))
	}
}
