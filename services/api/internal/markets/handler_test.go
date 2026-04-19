package markets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeRepo struct {
	listRes   []Price
	listErr   error
	latestRes []Price
	latestErr error

	lastFilter ListFilter
	lastMarket string
}

func (f *fakeRepo) List(_ context.Context, filter ListFilter) ([]Price, error) {
	f.lastFilter = filter
	return f.listRes, f.listErr
}

func (f *fakeRepo) Latest(_ context.Context, market string) ([]Price, error) {
	f.lastMarket = market
	return f.latestRes, f.latestErr
}

func newServer(repo Repository) http.Handler {
	return NewHandler(repo).Routes()
}

func TestList_OK(t *testing.T) {
	avg := 240.0
	repo := &fakeRepo{
		listRes: []Price{
			{
				MarketCode:     "dambulla-dec",
				CommodityLabel: "Brinjals (Long)",
				ObservedOn:     "2026-04-15",
				PriceLKRAvg:    &avg,
				Unit:           "kg",
				Currency:       "LKR",
			},
		},
	}
	srv := newServer(repo)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet,
		"/?market=dambulla-dec&since=2026-04-01&limit=10", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if repo.lastFilter.Market != "dambulla-dec" {
		t.Fatalf("market filter not applied: %+v", repo.lastFilter)
	}
	if repo.lastFilter.Since.IsZero() {
		t.Fatalf("since not parsed")
	}
	if repo.lastFilter.Limit != 10 {
		t.Fatalf("limit not applied: %d", repo.lastFilter.Limit)
	}

	var got struct {
		Items []Price `json:"items"`
		Count int     `json:"count"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Count != 1 || got.Items[0].CommodityLabel != "Brinjals (Long)" {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func TestList_BadSince(t *testing.T) {
	srv := newServer(&fakeRepo{})
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/?since=yesterday", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rr.Code)
	}
}

func TestList_StubRepo(t *testing.T) {
	srv := newServer(NewStubRepo())
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rr.Code)
	}
}

func TestLatest_NotFound(t *testing.T) {
	srv := newServer(&fakeRepo{latestErr: ErrNotFound})
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/latest/welisara-dec", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
}

func TestLatest_OK(t *testing.T) {
	avg := 180.0
	repo := &fakeRepo{
		latestRes: []Price{{
			MarketCode: "dambulla-dec", CommodityLabel: "Carrot",
			ObservedOn: "2026-04-15", PriceLKRAvg: &avg,
		}},
	}
	srv := newServer(repo)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/latest/dambulla-dec", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if repo.lastMarket != "dambulla-dec" {
		t.Fatalf("market not passed through: %q", repo.lastMarket)
	}
}
