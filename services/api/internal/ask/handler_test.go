package ask

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goyama/api/internal/embed"
	"github.com/goyama/api/internal/geo"
	"github.com/goyama/api/internal/knowledge"
)

type fakeChunks struct {
	hits      []knowledge.SearchHit
	hitsErr   error
	source    knowledge.Source
	sourceErr error

	lastQuery knowledge.SearchQuery
}

func (f *fakeChunks) Search(_ context.Context, q knowledge.SearchQuery) ([]knowledge.SearchHit, error) {
	f.lastQuery = q
	return f.hits, f.hitsErr
}
func (f *fakeChunks) GetSource(_ context.Context, _ string) (knowledge.Source, error) {
	return f.source, f.sourceErr
}

type fakeGeo struct {
	res geo.Lookup
	err error
}

func (f *fakeGeo) Lookup(_ context.Context, p geo.Point) (geo.Lookup, error) {
	if f.err != nil {
		return geo.Lookup{}, f.err
	}
	out := f.res
	out.Location = p
	return out, nil
}

func newServer(chunks *fakeChunks, g geo.Repository) http.Handler {
	return New(embed.NewHashEmbedder(), chunks, g).Routes()
}

func TestAsk_MissingQuestion(t *testing.T) {
	srv := newServer(&fakeChunks{}, &fakeGeo{})
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestAsk_StubReturns503(t *testing.T) {
	chunks := &fakeChunks{hitsErr: knowledge.ErrSearchRequiresDatabase}
	srv := newServer(chunks, &fakeGeo{})
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/",
		strings.NewReader(`{"question": "tomato blight"}`)))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rr.Code)
	}
}

func TestAsk_OK_PassesFiltersThroughAndAttachesSource(t *testing.T) {
	score := 0.85
	conf := 0.95
	chunks := &fakeChunks{
		hits: []knowledge.SearchHit{
			{
				Chunk: knowledge.Chunk{
					Slug: "c1", SourceSlug: "src-doa", Body: "Mancozeb 80% WP 20 g/10 L.",
					Authority: "doa_official", Confidence: &conf, Status: "published",
					EntityRefs: []knowledge.EntityRef{{Type: "crop", Slug: "tomato"}},
				},
				Score: score,
			},
		},
		source: knowledge.Source{Slug: "src-doa", DisplayName: "HORDI", Authority: "doa_official"},
	}
	g := &fakeGeo{res: geo.Lookup{
		AEZ:      &geo.AEZ{Code: "DL-1"},
		District: &geo.AdminDistrict{NameEN: "Anuradhapura"},
	}}
	srv := newServer(chunks, g)

	body := `{"question": "How do I treat tomato early blight?", "crop": "tomato", "lat": 8.3, "lng": 80.4, "k": 5}`
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if chunks.lastQuery.CropSlug != "tomato" {
		t.Errorf("crop filter not threaded: %q", chunks.lastQuery.CropSlug)
	}
	if len(chunks.lastQuery.AEZCodes) != 1 || chunks.lastQuery.AEZCodes[0] != "DL-1" {
		t.Errorf("AEZ filter not derived from coords: %+v", chunks.lastQuery.AEZCodes)
	}
	if chunks.lastQuery.Limit != 5 {
		t.Errorf("k not threaded: %d", chunks.lastQuery.Limit)
	}

	var got AskResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 1 {
		t.Fatalf("want 1 hit, got %d", got.Count)
	}
	if got.UsedDistrict != "Anuradhapura" {
		t.Errorf("district not echoed: %q", got.UsedDistrict)
	}
	if got.Hits[0].Source == nil || got.Hits[0].Source.DisplayName != "HORDI" {
		t.Errorf("source metadata not attached: %+v", got.Hits[0].Source)
	}
	if got.Hits[0].Authority != "doa_official" {
		t.Errorf("authority not propagated: %q", got.Hits[0].Authority)
	}
	if got.Hits[0].Score != score {
		t.Errorf("score not propagated: %f", got.Hits[0].Score)
	}
}

func TestAsk_GeoFailureDoesNotBlockSearch(t *testing.T) {
	// Geo lookup failures are common in dev (no fixtures loaded);
	// the chat surface should still work, just without the AEZ
	// narrowing.
	chunks := &fakeChunks{hits: []knowledge.SearchHit{}}
	g := &fakeGeo{err: errors.New("geo unavailable")}
	srv := newServer(chunks, g)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/",
		strings.NewReader(`{"question": "X", "lat": 7.3, "lng": 80.6}`)))
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if len(chunks.lastQuery.AEZCodes) != 0 {
		t.Fatalf("expected AEZ filter to be empty when geo fails, got %+v", chunks.lastQuery.AEZCodes)
	}
}
