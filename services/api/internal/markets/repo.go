// Package markets serves daily wholesale + retail price observations
// from Sri Lanka's Dedicated Economic Centres (DECs), starting with
// Dambulla. Storage is per-(market, commodity, date) — the Python
// importer in pipelines/sources/market_prices upserts daily; this
// package only reads.
package markets

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when no price observations match the filter.
var ErrNotFound = errors.New("no market price observations found")

// ErrRequiresDatabase signals the API is in JSONL fallback mode — market
// prices are Postgres-backed only.
var ErrRequiresDatabase = errors.New("market prices require Postgres (set DATABASE_URL)")

// Price is one (market, commodity, date) observation, returned by both
// the list and latest endpoints.
type Price struct {
	MarketCode      string   `json:"market_code"`
	CropSlug        string   `json:"crop_slug,omitempty"`
	CommodityLabel  string   `json:"commodity_label"`
	Grade           string   `json:"grade,omitempty"`
	ObservedOn      string   `json:"observed_on"` // ISO date
	PriceLKRMin     *float64 `json:"price_lkr_per_kg_min,omitempty"`
	PriceLKRMax     *float64 `json:"price_lkr_per_kg_max,omitempty"`
	PriceLKRAvg     *float64 `json:"price_lkr_per_kg_avg,omitempty"`
	Unit            string   `json:"unit,omitempty"`
	Currency        string   `json:"currency,omitempty"`
	SampleSize      *int     `json:"sample_size,omitempty"`
	SourceURL       string   `json:"source_url,omitempty"`
}

// ListFilter captures the query-string knobs supported by /v1/market-prices.
// Zero values mean "unfiltered" on each axis.
type ListFilter struct {
	Market   string
	CropSlug string
	Since    time.Time
	Until    time.Time
	Limit    int
	Offset   int
}

// Repository is the read-only price surface.
type Repository interface {
	List(ctx context.Context, filter ListFilter) ([]Price, error)
	Latest(ctx context.Context, marketCode string) ([]Price, error)
}

// StubRepo mirrors the geo + admin-queue pattern: every call fails when
// the API is in JSONL fallback mode so callers see an explicit 503.
type StubRepo struct{}

// NewStubRepo returns the JSONL-mode placeholder.
func NewStubRepo() *StubRepo { return &StubRepo{} }

// List always returns ErrRequiresDatabase.
func (*StubRepo) List(context.Context, ListFilter) ([]Price, error) {
	return nil, ErrRequiresDatabase
}

// Latest always returns ErrRequiresDatabase.
func (*StubRepo) Latest(context.Context, string) ([]Price, error) {
	return nil, ErrRequiresDatabase
}
