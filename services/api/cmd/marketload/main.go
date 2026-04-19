// Command marketload upserts daily wholesale + retail price observations
// into the market_price table from a CSV file.
//
// Usage:
//
//	marketload --file=pipelines/sources/market_prices/fixtures/dambulla-2026-04-15.csv
//	marketload --file=data/staging/dambulla/2026-04-19.csv --source-url=https://...
//
// CSV columns (header row required, order-independent):
//
//	observed_on, market_code, commodity_label, grade,
//	price_lkr_per_kg_min, price_lkr_per_kg_max, price_lkr_per_kg_avg,
//	unit, currency, sample_size, source_url, crop_slug
//
// observed_on, market_code, commodity_label are required. Numeric fields
// may be empty. Loader is idempotent — rows are upserted by
// (market_code, commodity_label, grade, observed_on).
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	path := flag.String("file", "", "path to a CSV of price observations")
	defaultSourceURL := flag.String("source-url", "",
		"override source_url for every row (used when the CSV came from one bulletin)")
	flag.Parse()

	if *path == "" {
		flag.Usage()
		return errors.New("--file is required")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	rows, err := readCSV(*path)
	if err != nil {
		return fmt.Errorf("read %s: %w", *path, err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("no rows in %s", *path)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("pgx pool: %w", err)
	}
	defer pool.Close()

	inserted := 0
	for i, r := range rows {
		if r.SourceURL == "" {
			r.SourceURL = *defaultSourceURL
		}
		if err := upsert(ctx, pool, r); err != nil {
			return fmt.Errorf("row %d (%s / %s / %s): %w",
				i+2, r.MarketCode, r.CommodityLabel, r.ObservedOn.Format("2006-01-02"), err)
		}
		inserted++
	}
	fmt.Printf("upserted %d price observations from %s\n", inserted, *path)
	return nil
}

type row struct {
	ObservedOn     time.Time
	MarketCode     string
	CommodityLabel string
	Grade          string
	CropSlug       string
	PriceMin       *float64
	PriceMax       *float64
	PriceAvg       *float64
	Unit           string
	Currency       string
	SampleSize     *int
	SourceURL      string
}

func readCSV(path string) ([]row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	idx := map[string]int{}
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	required := []string{"observed_on", "market_code", "commodity_label"}
	for _, k := range required {
		if _, ok := idx[k]; !ok {
			return nil, fmt.Errorf("missing required column %q", k)
		}
	}

	var out []row
	for line := 2; ; line++ {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}

		get := func(k string) string {
			i, ok := idx[k]
			if !ok || i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}

		observed, err := time.Parse("2006-01-02", get("observed_on"))
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid observed_on %q", line, get("observed_on"))
		}
		market := get("market_code")
		commodity := get("commodity_label")
		if market == "" || commodity == "" {
			return nil, fmt.Errorf("line %d: market_code and commodity_label are required", line)
		}

		out = append(out, row{
			ObservedOn:     observed,
			MarketCode:     market,
			CommodityLabel: commodity,
			Grade:          get("grade"),
			CropSlug:       get("crop_slug"),
			PriceMin:       parseFloat(get("price_lkr_per_kg_min")),
			PriceMax:       parseFloat(get("price_lkr_per_kg_max")),
			PriceAvg:       parseFloat(get("price_lkr_per_kg_avg")),
			Unit:           strFallback(get("unit"), "kg"),
			Currency:       strFallback(get("currency"), "LKR"),
			SampleSize:     parseInt(get("sample_size")),
			SourceURL:      get("source_url"),
		})
	}
	return out, nil
}

const upsertSQL = `
INSERT INTO market_price (
    market_code, crop_slug, commodity_label, grade, observed_on,
    price_lkr_per_kg_min, price_lkr_per_kg_max, price_lkr_per_kg_avg,
    unit, currency, sample_size, source_url, field_provenance
)
VALUES (
    $1, NULLIF($2, ''), $3, $4, $5,
    $6, $7, $8,
    $9, $10, $11, NULLIF($12, ''), $13
)
ON CONFLICT (market_code, commodity_label, grade, observed_on) DO UPDATE SET
    crop_slug = EXCLUDED.crop_slug,
    price_lkr_per_kg_min = EXCLUDED.price_lkr_per_kg_min,
    price_lkr_per_kg_max = EXCLUDED.price_lkr_per_kg_max,
    price_lkr_per_kg_avg = EXCLUDED.price_lkr_per_kg_avg,
    unit = EXCLUDED.unit,
    currency = EXCLUDED.currency,
    sample_size = EXCLUDED.sample_size,
    source_url = EXCLUDED.source_url,
    field_provenance = EXCLUDED.field_provenance
`

func upsert(ctx context.Context, pool *pgxpool.Pool, r row) error {
	prov := map[string]any{
		"loader":     "marketload",
		"loaded_at":  time.Now().UTC().Format(time.RFC3339),
		"source_url": r.SourceURL,
	}
	provJSON, _ := json.Marshal(prov)
	// The unique constraint compares grade as text — empty string and NULL
	// would otherwise be treated as distinct rows. Coerce empty to '' to
	// keep the upsert deterministic; the table stores '' as well.
	grade := r.Grade
	_, err := pool.Exec(ctx, upsertSQL,
		r.MarketCode, r.CropSlug, r.CommodityLabel, grade, r.ObservedOn,
		r.PriceMin, r.PriceMax, r.PriceAvg,
		r.Unit, r.Currency, r.SampleSize, r.SourceURL, provJSON,
	)
	return err
}

func parseFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		return nil
	}
	return &v
}

func parseInt(s string) *int {
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(strings.ReplaceAll(s, ",", ""))
	if err != nil {
		return nil
	}
	return &v
}

func strFallback(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
