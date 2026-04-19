package geo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeRepo lets us exercise the handler's contract without spinning up
// PostGIS — the spatial-join behaviour is tested separately.
type fakeRepo struct {
	res Lookup
	err error
}

func (f *fakeRepo) Lookup(_ context.Context, p Point) (Lookup, error) {
	if f.err != nil {
		return Lookup{}, f.err
	}
	out := f.res
	out.Location = p
	return out, nil
}

func newServer(repo Repository) http.Handler {
	h := NewHandler(repo)
	return h.Routes()
}

func TestLookup_MissingParams(t *testing.T) {
	srv := newServer(&fakeRepo{})

	cases := []struct {
		name, url string
		wantCode  int
	}{
		{"missing both", "/lookup", http.StatusBadRequest},
		{"missing lat", "/lookup?lng=80.6", http.StatusBadRequest},
		{"missing lng", "/lookup?lat=7.29", http.StatusBadRequest},
		{"non-numeric lat", "/lookup?lat=abc&lng=80", http.StatusBadRequest},
		{"non-numeric lng", "/lookup?lat=7&lng=xyz", http.StatusBadRequest},
		{"north of SL", "/lookup?lat=20&lng=80", http.StatusBadRequest},
		{"east of SL", "/lookup?lat=7&lng=90", http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, c.url, nil))
			if rr.Code != c.wantCode {
				t.Fatalf("want %d, got %d (%s)", c.wantCode, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestLookup_RequiresDatabase(t *testing.T) {
	srv := newServer(NewStubRepo())
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/lookup?lat=7.29&lng=80.63", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rr.Code)
	}
}

func TestLookup_NotFound(t *testing.T) {
	srv := newServer(&fakeRepo{err: ErrLocationNotFound})
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/lookup?lat=7.29&lng=80.63", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
}

func TestLookup_OK(t *testing.T) {
	rainfall := float32(2100)
	repo := &fakeRepo{
		res: Lookup{
			District: &AdminDistrict{Code: "LK-21", NameEN: "Kandy", ProvinceName: "Central"},
			AEZ: &AEZ{
				Code:               "WM3",
				ZoneGroup:          "wet",
				ElevationClass:     "mid_country",
				AvgRainfallMM:      &rainfall,
				DominantSoilGroups: []string{"red_yellow_podzolic"},
			},
		},
	}
	srv := newServer(repo)

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/lookup?lat=7.2906&lng=80.6337", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rr.Code, rr.Body.String())
	}

	var got Lookup
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Location.Lat != 7.2906 || got.Location.Lng != 80.6337 {
		t.Fatalf("location not echoed back: %+v", got.Location)
	}
	if got.District == nil || got.District.NameEN != "Kandy" {
		t.Fatalf("district missing: %+v", got.District)
	}
	if got.AEZ == nil || got.AEZ.Code != "WM3" {
		t.Fatalf("aez missing: %+v", got.AEZ)
	}
}
