// Package plans serves the cultivation-plan aggregate — a versioned
// (crop × AEZ × season) calendar assembled from a single source
// document. Children (activities, pest risks, economics) travel with
// the plan, so the public read endpoint returns one nested payload
// instead of forcing the client to stitch four round-trips together.
//
// Like other farmer-facing packages, plans reads from Postgres when
// available and falls back to JSONL fixtures during dev + smoke tests
// so the web client can demo the full layout with no DB wired up.
package plans

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

// ErrNotFound is returned when a plan slug has no record.
var ErrNotFound = errors.New("cultivation plan not found")

// ErrRequiresDatabase is returned by JSONL admin writes — mutating a
// plan's status in dev mode would require editing a file in version
// control, so we surface a clear 503 until the Postgres loader lands.
var ErrRequiresDatabase = errors.New("plan status changes require Postgres (set DATABASE_URL)")

// Summary is the short card representation returned by the list endpoint.
type Summary struct {
	Slug                   string            `json:"slug"`
	CropSlug               string            `json:"crop_slug"`
	Season                 string            `json:"season"`
	Authority              string            `json:"authority"`
	AEZCodes               []string          `json:"aez_codes,omitempty"`
	Title                  map[string]string `json:"title,omitempty"`
	Summary                map[string]string `json:"summary,omitempty"`
	DurationWeeks          int               `json:"duration_weeks,omitempty"`
	ExpectedYieldKgPerAcre *Range            `json:"expected_yield_kg_per_acre,omitempty"`
	SourceDocumentTitle    string            `json:"source_document_title,omitempty"`
}

// Range matches the common schema range shape.
type Range struct {
	Min  any    `json:"min,omitempty"`
	Max  any    `json:"max,omitempty"`
	Unit string `json:"unit,omitempty"`
}

// Plan is the detail payload — summary fields plus the three child
// collections + reviewer audit fields used by the admin queue.
type Plan struct {
	Summary
	VarietySlug       string         `json:"variety_slug,omitempty"`
	StartMonth        int            `json:"start_month,omitempty"`
	Status            string         `json:"status"`
	SourceDocumentURL string         `json:"source_document_url,omitempty"`
	FieldProvenance   map[string]any `json:"field_provenance,omitempty"`
	Activities        []Activity     `json:"activities"`
	PestRisks         []PestRisk     `json:"pest_risks"`
	Economics         []Economics    `json:"economics"`
	ReviewedBy        string         `json:"reviewed_by,omitempty"`
	ReviewedAt        string         `json:"reviewed_at,omitempty"`
	ReviewNotes       string         `json:"review_notes,omitempty"`
}

// Activity is one row of the week-by-week cultivation timeline.
type Activity struct {
	WeekIdx     int               `json:"week_idx"`
	OrderInWeek int               `json:"order_in_week"`
	Activity    string            `json:"activity"`
	DAPMin      *int              `json:"dap_min,omitempty"`
	DAPMax      *int              `json:"dap_max,omitempty"`
	Title       map[string]string `json:"title,omitempty"`
	Body        map[string]string `json:"body,omitempty"`
	Inputs      []map[string]any  `json:"inputs,omitempty"`
	WeatherHint string            `json:"weather_hint,omitempty"`
	MediaSlugs  []string          `json:"media_slugs,omitempty"`
}

// PestRisk is one (week, pest|disease, risk) row in the calendar.
type PestRisk struct {
	WeekIdx                int               `json:"week_idx"`
	DiseaseSlug            string            `json:"disease_slug,omitempty"`
	PestSlug               string            `json:"pest_slug,omitempty"`
	Risk                   string            `json:"risk"`
	RecommendedRemedySlugs []string          `json:"recommended_remedy_slugs,omitempty"`
	Notes                  map[string]string `json:"notes,omitempty"`
}

// Economics is one cost-revenue model for a specific (plan, year, unit_area).
type Economics struct {
	ReferenceYear                 int              `json:"reference_year"`
	UnitArea                      string           `json:"unit_area"`
	Currency                      string           `json:"currency"`
	CostLines                     []map[string]any `json:"cost_lines,omitempty"`
	TotalCostWithoutFamilyLabour  *float64         `json:"total_cost_without_family_labour,omitempty"`
	TotalCostWithFamilyLabour     *float64         `json:"total_cost_with_family_labour,omitempty"`
	YieldKg                       *float64         `json:"yield_kg,omitempty"`
	UnitPrice                     *float64         `json:"unit_price,omitempty"`
	GrossRevenue                  *float64         `json:"gross_revenue,omitempty"`
	NetRevenueWithoutFamilyLabour *float64         `json:"net_revenue_without_family_labour,omitempty"`
	NetRevenueWithFamilyLabour    *float64         `json:"net_revenue_with_family_labour,omitempty"`
}

// Repository is the farmer-facing plans surface.
type Repository interface {
	ListByCrop(ctx context.Context, cropSlug string) ([]Summary, error)
	Get(ctx context.Context, slug string) (Plan, error)
}

// AdminRepo is the admin review-queue surface. Mirrors the shape the
// generic review.Routes factory expects — ListByStatus returns the
// full Plan rather than a summary so the agronomist can see every
// activity / pest risk / economics row they're about to promote.
type AdminRepo interface {
	ListByStatus(ctx context.Context, status string) ([]Plan, error)
	Get(ctx context.Context, slug string) (Plan, error)
	SetStatus(ctx context.Context, slug string, u review.StatusUpdate) error
}

// JSONLRepo reads cultivation plan fixtures from corpus/seed/cultivation_plans
// so the dev API has data with no DB. Each plan is one JSON file; we load
// all files on first access and hold them in memory.
type JSONLRepo struct {
	dir     string
	once    sync.Once
	loaded  []Plan
	bySlug  map[string]Plan
	loadErr error
}

// NewJSONLRepo returns a repo that reads plan JSON files from the given
// directory on first use.
func NewJSONLRepo(dir string) *JSONLRepo { return &JSONLRepo{dir: dir} }

func (r *JSONLRepo) load() error {
	r.once.Do(func() {
		entries, err := os.ReadDir(r.dir)
		if err != nil {
			if os.IsNotExist(err) {
				r.bySlug = map[string]Plan{}
				return
			}
			r.loadErr = fmt.Errorf("read dir %s: %w", r.dir, err)
			return
		}
		bySlug := make(map[string]Plan)
		var all []Plan
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".json") {
				continue
			}
			f, err := os.Open(filepath.Join(r.dir, name))
			if err != nil {
				r.loadErr = fmt.Errorf("open %s: %w", name, err)
				return
			}
			var p Plan
			if err := json.NewDecoder(bufio.NewReader(f)).Decode(&p); err != nil {
				f.Close()
				r.loadErr = fmt.Errorf("decode %s: %w", name, err)
				return
			}
			f.Close()
			if p.Authority == "" {
				p.Authority = "doa_official"
			}
			bySlug[p.Slug] = p
			all = append(all, p)
		}
		sort.Slice(all, func(i, j int) bool { return all[i].Slug < all[j].Slug })
		r.loaded = all
		r.bySlug = bySlug
	})
	return r.loadErr
}

// ListByCrop returns published-or-draft plans for a crop. In JSONL mode
// we return every plan we parsed — production mode filters by status.
func (r *JSONLRepo) ListByCrop(_ context.Context, cropSlug string) ([]Summary, error) {
	if err := r.load(); err != nil {
		return nil, err
	}
	out := make([]Summary, 0, 4)
	for _, p := range r.loaded {
		if p.CropSlug != cropSlug {
			continue
		}
		out = append(out, p.Summary)
	}
	return out, nil
}

// Get returns the full plan by slug.
func (r *JSONLRepo) Get(_ context.Context, slug string) (Plan, error) {
	if err := r.load(); err != nil {
		return Plan{}, err
	}
	p, ok := r.bySlug[slug]
	if !ok {
		return Plan{}, ErrNotFound
	}
	return p, nil
}

// ListByStatus powers the admin review queue. JSONL mode can filter
// in-memory plans by the status field authored into the fixture — the
// agronomist can preview the queue locally even without a database.
func (r *JSONLRepo) ListByStatus(_ context.Context, status string) ([]Plan, error) {
	if err := r.load(); err != nil {
		return nil, err
	}
	s := strings.TrimSpace(status)
	out := make([]Plan, 0, 4)
	for _, p := range r.loaded {
		if s == "" || p.Status == s {
			out = append(out, p)
		}
	}
	return out, nil
}

// SetStatus returns ErrRequiresDatabase — JSONL mode is read-only.
// Status promotion will succeed once the Postgres-backed AdminPgxRepo
// is wired in alongside the plan loader.
func (*JSONLRepo) SetStatus(context.Context, string, review.StatusUpdate) error {
	return ErrRequiresDatabase
}
