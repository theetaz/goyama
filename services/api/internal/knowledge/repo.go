// Package knowledge serves the unstructured side of the corpus —
// knowledge_chunk rows keyed on (source, language, authority) and
// linked to canonical entities via entity_refs. This is the retrieval
// surface the future farmer-chat agent will query.
//
// For now we expose the read endpoint only (list by entity, and list
// by slug). The embedding column exists in the schema but is populated
// by a separate worker, so this repo is text-metadata only.
package knowledge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/goyama/api/internal/review"
)

// ErrNotFound is returned when a chunk slug has no record.
var ErrNotFound = errors.New("knowledge chunk not found")

// ErrRequiresDatabase is returned by JSONL admin writes. Knowledge
// chunk status changes only persist once the Postgres-backed admin
// repo is wired in.
var ErrRequiresDatabase = errors.New("knowledge chunk status changes require Postgres (set DATABASE_URL)")

// Chunk is one retrievable unit of unstructured agronomic signal.
type Chunk struct {
	Slug               string            `json:"slug"`
	SourceSlug         string            `json:"source_slug"`
	ChunkIdx           int               `json:"chunk_idx"`
	Language           string            `json:"language"`
	Title              string            `json:"title,omitempty"`
	Body               string            `json:"body"`
	BodyTranslated     map[string]string `json:"body_translated,omitempty"`
	EntityRefs         []EntityRef       `json:"entity_refs,omitempty"`
	Authority          string            `json:"authority"`
	AppliesToAEZCodes  []string          `json:"applies_to_aez_codes,omitempty"`
	AppliesToCountries []string          `json:"applies_to_countries,omitempty"`
	TopicTags          []string          `json:"topic_tags,omitempty"`
	Confidence         *float64          `json:"confidence,omitempty"`
	Quote              string            `json:"quote,omitempty"`
	Status             string            `json:"status"`
	FieldProvenance    map[string]any    `json:"field_provenance,omitempty"`
	ReviewedBy         string            `json:"reviewed_by,omitempty"`
	ReviewedAt         string            `json:"reviewed_at,omitempty"`
	ReviewNotes        string            `json:"review_notes,omitempty"`
}

// EntityRef is the bridge back to canonical records — `{type, slug}`.
type EntityRef struct {
	Type string `json:"type"`
	Slug string `json:"slug"`
}

// Source is the upstream document a chunk was extracted from.
type Source struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
	Medium      string `json:"medium"`
	Publisher   string `json:"publisher,omitempty"`
	Authority   string `json:"authority"`
	URL         string `json:"url,omitempty"`
	Language    string `json:"language,omitempty"`
	Licence     string `json:"licence,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

// Repository is the read-only knowledge surface used by the farmer-
// facing endpoints.
type Repository interface {
	ListByEntity(ctx context.Context, entityType, entitySlug string) ([]Chunk, error)
	GetSource(ctx context.Context, slug string) (Source, error)
}

// AdminRepo is the admin review-queue surface. Same shape as the other
// reviewable entities — ListByStatus + Get + SetStatus.
type AdminRepo interface {
	ListByStatus(ctx context.Context, status string) ([]Chunk, error)
	Get(ctx context.Context, slug string) (Chunk, error)
	SetStatus(ctx context.Context, slug string, u review.StatusUpdate) error
}

// JSONLRepo loads chunk and source fixtures from disk so the dev API
// has knowledge data with no DB. Two sibling directories:
//
//	<root>/knowledge_sources/*.json
//	<root>/knowledge_chunks/*.json
type JSONLRepo struct {
	root    string
	once    sync.Once
	chunks  []Chunk
	sources map[string]Source
	loadErr error
}

// NewJSONLRepo returns a repo rooted at <corpusSeedDir>.
func NewJSONLRepo(corpusSeedDir string) *JSONLRepo { return &JSONLRepo{root: corpusSeedDir} }

func (r *JSONLRepo) load() error {
	r.once.Do(func() {
		r.sources = map[string]Source{}
		if err := loadDir(filepath.Join(r.root, "knowledge_sources"), func(path string) error {
			var s Source
			if err := decodeFile(path, &s); err != nil {
				return err
			}
			r.sources[s.Slug] = s
			return nil
		}); err != nil {
			r.loadErr = err
			return
		}
		if err := loadDir(filepath.Join(r.root, "knowledge_chunks"), func(path string) error {
			var c Chunk
			if err := decodeFile(path, &c); err != nil {
				return err
			}
			r.chunks = append(r.chunks, c)
			return nil
		}); err != nil {
			r.loadErr = err
			return
		}
		// Stable order: by entity-slug-agnostic key, then chunk idx.
		sort.Slice(r.chunks, func(i, j int) bool {
			if r.chunks[i].SourceSlug != r.chunks[j].SourceSlug {
				return r.chunks[i].SourceSlug < r.chunks[j].SourceSlug
			}
			return r.chunks[i].ChunkIdx < r.chunks[j].ChunkIdx
		})
	})
	return r.loadErr
}

func loadDir(dir string, visit func(path string) error) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if err := visit(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func decodeFile(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	if err := json.NewDecoder(bufio.NewReader(f)).Decode(v); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// ListByEntity returns every chunk whose entity_refs contains
// (entityType, entitySlug). JSONL mode returns drafts too; production
// mode will filter to `published`.
func (r *JSONLRepo) ListByEntity(_ context.Context, entityType, entitySlug string) ([]Chunk, error) {
	if err := r.load(); err != nil {
		return nil, err
	}
	out := make([]Chunk, 0, 4)
	for _, c := range r.chunks {
		for _, ref := range c.EntityRefs {
			if ref.Type == entityType && ref.Slug == entitySlug {
				out = append(out, c)
				break
			}
		}
	}
	return out, nil
}

// GetSource returns the source document metadata by slug.
func (r *JSONLRepo) GetSource(_ context.Context, slug string) (Source, error) {
	if err := r.load(); err != nil {
		return Source{}, err
	}
	s, ok := r.sources[slug]
	if !ok {
		return Source{}, ErrNotFound
	}
	return s, nil
}

// ListByStatus powers the admin review queue.
func (r *JSONLRepo) ListByStatus(_ context.Context, status string) ([]Chunk, error) {
	if err := r.load(); err != nil {
		return nil, err
	}
	s := strings.TrimSpace(status)
	out := make([]Chunk, 0, 8)
	for _, c := range r.chunks {
		if s == "" || c.Status == s {
			out = append(out, c)
		}
	}
	return out, nil
}

// Get returns a chunk by slug.
func (r *JSONLRepo) Get(_ context.Context, slug string) (Chunk, error) {
	if err := r.load(); err != nil {
		return Chunk{}, err
	}
	for _, c := range r.chunks {
		if c.Slug == slug {
			return c, nil
		}
	}
	return Chunk{}, ErrNotFound
}

// SetStatus returns ErrRequiresDatabase — JSONL mode is read-only.
func (*JSONLRepo) SetStatus(context.Context, string, review.StatusUpdate) error {
	return ErrRequiresDatabase
}
