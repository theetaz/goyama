package crops

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

// ErrNotFound is returned when a crop slug has no record.
var ErrNotFound = errors.New("crop not found")

// Crop is the public shape returned by the API. It's a subset of the full
// corpus schema — enough for the client to render cards and detail pages.
// The full provenance block is available via the detail endpoint.
type Crop struct {
	Slug            string            `json:"slug"`
	ScientificName  string            `json:"scientific_name,omitempty"`
	Family          string            `json:"family,omitempty"`
	Category        string            `json:"category,omitempty"`
	LifeCycle       string            `json:"life_cycle,omitempty"`
	GrowthHabit     string            `json:"growth_habit,omitempty"`
	Names           map[string]string `json:"names,omitempty"`
	Aliases         []string          `json:"aliases,omitempty"`
	DefaultSeason   string            `json:"default_season,omitempty"`
	DurationDays    *Range            `json:"duration_days,omitempty"`
	ElevationM      *Range            `json:"elevation_m,omitempty"`
	RainfallMM      *Range            `json:"rainfall_mm,omitempty"`
	TemperatureC    *Range            `json:"temperature_c,omitempty"`
	SoilPH          *Range            `json:"soil_ph,omitempty"`
	ExpectedYield   *Range            `json:"expected_yield_kg_per_acre,omitempty"`
	Description     map[string]string `json:"description,omitempty"`
	Status          string            `json:"status,omitempty"`
	FieldProvenance map[string]any    `json:"field_provenance,omitempty"`
}

// Range matches the schema's Range / IntRange object.
type Range struct {
	Min  any    `json:"min,omitempty"`
	Max  any    `json:"max,omitempty"`
	Unit string `json:"unit,omitempty"`
}

// Summary is the short card representation returned in list endpoints.
type Summary struct {
	Slug           string            `json:"slug"`
	ScientificName string            `json:"scientific_name,omitempty"`
	Category       string            `json:"category,omitempty"`
	Names          map[string]string `json:"names,omitempty"`
	Aliases        []string          `json:"aliases,omitempty"`
}

// Repository is the farmer-facing crop surface: listing, getting a single
// crop, and reading the cultivation guide for a given crop. Admin-side
// cultivation-step review operations live on CultivationStepRepo.
type Repository interface {
	List(ctx context.Context, filter ListFilter) ([]Summary, error)
	Get(ctx context.Context, slug string) (Crop, error)
	// ListCultivationSteps returns the ordered step-by-step cultivation
	// guide for a crop. Returns an empty slice when the repo has no step
	// data (e.g. the JSONL fallback).
	ListCultivationSteps(ctx context.Context, cropSlug string) ([]CultivationStep, error)
}

// CultivationStepRepo is the admin-facing cultivation-step surface.
// Separated from Repository so the review.Routes factory can bind against
// the canonical ListByStatus / Get / SetStatus shape without also
// dragging in the farmer-facing Crop type.
//
// Both JSONLRepo and PgxRepo implement both interfaces.
type CultivationStepRepo interface {
	ListByStatus(ctx context.Context, status string) ([]CultivationStep, error)
	Get(ctx context.Context, slug string) (CultivationStep, error)
	SetStatus(ctx context.Context, slug string, update review.StatusUpdate) error
}

// CultivationStep is one entry in a crop's cultivation guide. Title and body
// are delivered as locale -> string maps so the client renders the current
// i18n locale without a second round-trip.
type CultivationStep struct {
	Slug             string            `json:"slug"`
	CropSlug         string            `json:"crop_slug"`
	VarietySlug      string            `json:"variety_slug,omitempty"`
	AEZCode          string            `json:"aez_code,omitempty"`
	Season           string            `json:"season,omitempty"`
	Stage            string            `json:"stage"`
	OrderIdx         int               `json:"order_idx"`
	DayAfterPlanting *IntRange         `json:"day_after_planting,omitempty"`
	Title            map[string]string `json:"title,omitempty"`
	Body             map[string]string `json:"body,omitempty"`
	Inputs           []map[string]any  `json:"inputs,omitempty"`
	MediaSlugs       []string          `json:"media_slugs,omitempty"`
	Status           string            `json:"status,omitempty"`
	FieldProvenance  map[string]any    `json:"field_provenance,omitempty"`
	ReviewedBy       string            `json:"reviewed_by,omitempty"`
	ReviewedAt       string            `json:"reviewed_at,omitempty"`
	ReviewNotes      string            `json:"review_notes,omitempty"`
}

// IntRange matches the corpus schema IntRange object — integer min/max with
// an optional free-text unit.
type IntRange struct {
	Min  *int   `json:"min,omitempty"`
	Max  *int   `json:"max,omitempty"`
	Unit string `json:"unit,omitempty"`
}

// ListFilter is a minimal filter set used by the list endpoint.
type ListFilter struct {
	Category string
	Query    string
	Limit    int
	Offset   int
}

// JSONLRepo loads the crops.jsonl bundle on first access and holds it in
// memory. Thread-safe. Swap for a pgxRepo once the DB is wired.
type JSONLRepo struct {
	corpusDir string
	once      sync.Once
	loaded    []Crop
	bySlug    map[string]Crop
	loadErr   error
}

// NewJSONLRepo returns a repo that reads crops.jsonl from the given corpus
// release directory.
func NewJSONLRepo(corpusDir string) *JSONLRepo {
	return &JSONLRepo{corpusDir: corpusDir}
}

func (r *JSONLRepo) load() error {
	r.once.Do(func() {
		path := filepath.Join(r.corpusDir, "crops.jsonl")
		f, err := os.Open(path)
		if err != nil {
			r.loadErr = fmt.Errorf("open %s: %w", path, err)
			return
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		// crops.jsonl has long records with embedded prose — raise the buffer.
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

		bySlug := make(map[string]Crop)
		var all []Crop
		line := 0
		for scanner.Scan() {
			line++
			raw := scanner.Bytes()
			if len(raw) == 0 {
				continue
			}
			var c Crop
			if err := json.Unmarshal(raw, &c); err != nil {
				r.loadErr = fmt.Errorf("decode line %d: %w", line, err)
				return
			}
			all = append(all, c)
			bySlug[c.Slug] = c
		}
		if err := scanner.Err(); err != nil {
			r.loadErr = fmt.Errorf("scan: %w", err)
			return
		}
		sort.Slice(all, func(i, j int) bool { return all[i].Slug < all[j].Slug })
		r.loaded = all
		r.bySlug = bySlug
	})
	return r.loadErr
}

// List returns a page of crop summaries filtered by category and query text.
func (r *JSONLRepo) List(_ context.Context, filter ListFilter) ([]Summary, error) {
	if err := r.load(); err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(filter.Query))
	cat := strings.TrimSpace(filter.Category)

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	out := make([]Summary, 0, limit)
	skipped := 0
	for _, c := range r.loaded {
		if cat != "" && c.Category != cat {
			continue
		}
		if q != "" && !matches(c, q) {
			continue
		}
		if skipped < offset {
			skipped++
			continue
		}
		if len(out) >= limit {
			break
		}
		out = append(out, Summary{
			Slug:           c.Slug,
			ScientificName: c.ScientificName,
			Category:       c.Category,
			Names:          c.Names,
			Aliases:        c.Aliases,
		})
	}
	return out, nil
}

// Get returns the full record for a slug or ErrNotFound.
func (r *JSONLRepo) Get(_ context.Context, slug string) (Crop, error) {
	if err := r.load(); err != nil {
		return Crop{}, err
	}
	c, ok := r.bySlug[slug]
	if !ok {
		return Crop{}, ErrNotFound
	}
	return c, nil
}

// ListCultivationSteps returns an empty slice for the JSONL fallback — the
// released JSONL bundle doesn't carry cultivation steps yet. Postgres mode
// serves the real data.
func (r *JSONLRepo) ListCultivationSteps(_ context.Context, _ string) ([]CultivationStep, error) {
	return []CultivationStep{}, nil
}

// Admin-facing operations are database-backed only. The JSONL fallback
// surfaces them as explicit errors so an admin endpoint never silently
// no-ops when DATABASE_URL is missing.
var ErrRequiresDatabase = errors.New("operation requires Postgres (set DATABASE_URL)")

// CultivationStepJSONLRepo satisfies CultivationStepRepo by failing every
// call — admin ops can't run against the JSONL corpus bundle.
type CultivationStepJSONLRepo struct{}

// NewCultivationStepJSONLRepo returns the placeholder admin repo.
func NewCultivationStepJSONLRepo() *CultivationStepJSONLRepo { return &CultivationStepJSONLRepo{} }

// ListByStatus always returns ErrRequiresDatabase.
func (*CultivationStepJSONLRepo) ListByStatus(context.Context, string) ([]CultivationStep, error) {
	return nil, ErrRequiresDatabase
}

// Get always returns ErrRequiresDatabase.
func (*CultivationStepJSONLRepo) Get(context.Context, string) (CultivationStep, error) {
	return CultivationStep{}, ErrRequiresDatabase
}

// SetStatus always returns ErrRequiresDatabase.
func (*CultivationStepJSONLRepo) SetStatus(context.Context, string, review.StatusUpdate) error {
	return ErrRequiresDatabase
}

func matches(c Crop, q string) bool {
	if strings.Contains(strings.ToLower(c.Slug), q) {
		return true
	}
	if strings.Contains(strings.ToLower(c.ScientificName), q) {
		return true
	}
	for _, v := range c.Names {
		if strings.Contains(strings.ToLower(v), q) {
			return true
		}
	}
	for _, a := range c.Aliases {
		if strings.Contains(strings.ToLower(a), q) {
			return true
		}
	}
	return false
}
