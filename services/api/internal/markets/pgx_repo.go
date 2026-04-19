package markets

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxRepo serves market prices from Postgres.
type PgxRepo struct {
	pool *pgxpool.Pool
}

// NewPgxRepo returns a repo over the given pool.
func NewPgxRepo(pool *pgxpool.Pool) *PgxRepo { return &PgxRepo{pool: pool} }

const baseSelect = `
SELECT market_code, COALESCE(crop_slug, ''), commodity_label,
       COALESCE(grade, ''), observed_on,
       price_lkr_per_kg_min, price_lkr_per_kg_max, price_lkr_per_kg_avg,
       COALESCE(unit, 'kg'), COALESCE(currency, 'LKR'),
       sample_size, COALESCE(source_url, '')
FROM market_price
`

// List returns price observations matching the filter, ordered by
// observed_on DESC, market_code, commodity_label.
func (r *PgxRepo) List(ctx context.Context, f ListFilter) ([]Price, error) {
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	var (
		conds []string
		args  []any
	)
	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if m := strings.TrimSpace(f.Market); m != "" {
		add("market_code = $%d", m)
	}
	if c := strings.TrimSpace(f.CropSlug); c != "" {
		add("crop_slug = $%d", c)
	}
	if !f.Since.IsZero() {
		add("observed_on >= $%d", f.Since)
	}
	if !f.Until.IsZero() {
		add("observed_on <= $%d", f.Until)
	}

	sql := baseSelect
	if len(conds) > 0 {
		sql += "WHERE " + strings.Join(conds, " AND ") + "\n"
	}
	sql += "ORDER BY observed_on DESC, market_code, commodity_label\n"
	args = append(args, offset, limit)
	sql += fmt.Sprintf("OFFSET $%d LIMIT $%d", len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list prices: %w", err)
	}
	defer rows.Close()

	out := make([]Price, 0, limit)
	for rows.Next() {
		p, err := scanPrice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prices: %w", err)
	}
	return out, nil
}

const latestSQL = baseSelect + `
WHERE market_code = $1
  AND observed_on = (
      SELECT MAX(observed_on) FROM market_price WHERE market_code = $1
  )
ORDER BY commodity_label
`

// Latest returns every price observation from the most recent date for
// which the given market has any data. Returns ErrNotFound if the market
// has no observations at all.
func (r *PgxRepo) Latest(ctx context.Context, marketCode string) ([]Price, error) {
	rows, err := r.pool.Query(ctx, latestSQL, marketCode)
	if err != nil {
		return nil, fmt.Errorf("latest prices: %w", err)
	}
	defer rows.Close()

	out := make([]Price, 0, 32)
	for rows.Next() {
		p, err := scanPrice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prices: %w", err)
	}
	if len(out) == 0 {
		return nil, ErrNotFound
	}
	return out, nil
}

func scanPrice(rows interface{ Scan(dest ...any) error }) (Price, error) {
	var (
		p                Price
		observed         time.Time
		minP, maxP, avgP *float64
		sampleSize       *int
	)
	if err := rows.Scan(
		&p.MarketCode, &p.CropSlug, &p.CommodityLabel,
		&p.Grade, &observed,
		&minP, &maxP, &avgP,
		&p.Unit, &p.Currency,
		&sampleSize, &p.SourceURL,
	); err != nil {
		return p, fmt.Errorf("scan price: %w", err)
	}
	p.ObservedOn = observed.Format("2006-01-02")
	p.PriceLKRMin = minP
	p.PriceLKRMax = maxP
	p.PriceLKRAvg = avgP
	p.SampleSize = sampleSize
	return p, nil
}
