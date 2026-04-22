package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeRepo struct {
	chunks    []Chunk
	chunksErr error
	sources   map[string]Source
}

func (f *fakeRepo) ListByEntity(context.Context, string, string) ([]Chunk, error) {
	return f.chunks, f.chunksErr
}
func (f *fakeRepo) GetSource(_ context.Context, slug string) (Source, error) {
	if s, ok := f.sources[slug]; ok {
		return s, nil
	}
	return Source{}, errors.New("not found")
}

func TestByEntityHandler_AttachesSources(t *testing.T) {
	repo := &fakeRepo{
		chunks: []Chunk{
			{Slug: "c1", SourceSlug: "src-a", Body: "…", Authority: "doa_official"},
			{Slug: "c2", SourceSlug: "src-a", Body: "…", Authority: "doa_official"},
			{Slug: "c3", SourceSlug: "src-b", Body: "…", Authority: "inferred_by_analogy"},
		},
		sources: map[string]Source{
			"src-a": {Slug: "src-a", DisplayName: "DOA", Authority: "doa_official"},
			"src-b": {Slug: "src-b", DisplayName: "TNAU", Authority: "regional_authority"},
		},
	}
	r := chi.NewRouter()
	r.Get("/crops/{slug}/knowledge", New(repo).ByEntityHandler("crop"))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/crops/red-onion/knowledge", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var got struct {
		EntityType string   `json:"entity_type"`
		EntitySlug string   `json:"entity_slug"`
		Chunks     []Chunk  `json:"chunks"`
		Sources    []Source `json:"sources"`
		Count      int      `json:"count"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 3 {
		t.Fatalf("want 3 chunks, got %d", got.Count)
	}
	if len(got.Sources) != 2 {
		t.Fatalf("want 2 dedup'd sources, got %d", len(got.Sources))
	}
}

func TestJSONLRepo_LoadsFixtures(t *testing.T) {
	repo := NewJSONLRepo("../../../../corpus/seed")
	chunks, err := repo.ListByEntity(context.Background(), "crop", "red-onion")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected both DOA and TNAU red-onion chunks in fixtures, got %d", len(chunks))
	}
	// Authority diversity is a product requirement — we ingest from
	// multiple sources and label the authority honestly. Assert the
	// fixtures exercise both bands.
	var hasOfficial, hasAnalogy bool
	for _, c := range chunks {
		switch c.Authority {
		case "doa_official":
			hasOfficial = true
		case "inferred_by_analogy":
			hasAnalogy = true
		}
	}
	if !hasOfficial || !hasAnalogy {
		t.Fatalf("fixtures should include both doa_official and inferred_by_analogy authority chunks; got %+v",
			chunks)
	}

	src, err := repo.GetSource(context.Background(), "doa-afaci-red-onion-2014")
	if err != nil {
		t.Fatalf("get source: %v", err)
	}
	if src.Publisher == "" {
		t.Fatalf("expected publisher populated on the DOA source")
	}
}
