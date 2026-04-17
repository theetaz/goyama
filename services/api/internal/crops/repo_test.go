package crops

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	content := `{"slug":"rice","scientific_name":"Oryza sativa","category":"field_crop","names":{"en":"Rice","si":"Sahal"}}
{"slug":"brinjal","scientific_name":"Solanum melongena","category":"vegetable","names":{"en":"Brinjal","si":"Vambatu","ta":"Kathirikkai"},"aliases":["eggplant"]}
`
	path := filepath.Join(dir, "crops.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestJSONLRepo_List(t *testing.T) {
	repo := NewJSONLRepo(writeFixture(t))

	items, err := repo.List(context.Background(), ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	if items[0].Slug != "brinjal" {
		t.Errorf("want brinjal first (alphabetical), got %q", items[0].Slug)
	}
}

func TestJSONLRepo_List_FilterByCategory(t *testing.T) {
	repo := NewJSONLRepo(writeFixture(t))

	items, err := repo.List(context.Background(), ListFilter{Category: "vegetable"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].Slug != "brinjal" {
		t.Errorf("want only brinjal, got %+v", items)
	}
}

func TestJSONLRepo_List_FilterByQuery(t *testing.T) {
	repo := NewJSONLRepo(writeFixture(t))

	items, err := repo.List(context.Background(), ListFilter{Query: "vambatu"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].Slug != "brinjal" {
		t.Errorf("vambatu → brinjal match failed; got %+v", items)
	}
}

func TestJSONLRepo_Get(t *testing.T) {
	repo := NewJSONLRepo(writeFixture(t))

	c, err := repo.Get(context.Background(), "rice")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if c.ScientificName != "Oryza sativa" {
		t.Errorf("want Oryza sativa, got %q", c.ScientificName)
	}

	if _, err := repo.Get(context.Background(), "nope"); err != ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
