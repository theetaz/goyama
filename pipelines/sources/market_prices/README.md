# Market price observations

Daily wholesale + retail price observations from Sri Lanka's Dedicated
Economic Centres (DECs) and HARTI bulletins. Backs `GET /v1/market-prices`
and `GET /v1/market-prices/latest/{market}`.

## Architecture

The flow is intentionally split:

1. **Acquire** the daily bulletin (PDF / Excel / web table) from the source.
   Per the polite-crawl rules in `CLAUDE.md`, this step honors `robots.txt`
   and rate limits, and stores raw artifacts under `data/raw/<source>/`
   (gitignored) for provenance.
2. **Normalise** the raw bulletin into the canonical CSV shape (one row
   per `(market, commodity, grade, observed_on)`). This is per-source
   work; the schema is shared.
3. **Load** the CSV into Postgres via the `marketload` Go command, which
   is idempotent (upserts by the unique key).

The CSV shape is the boundary contract: any tool that produces conformant
CSV (Python scraper, hand-edited spreadsheet, agronomist export) can feed
`marketload` directly.

## CSV format

Header row required, columns order-independent. Required columns:
`observed_on`, `market_code`, `commodity_label`. Optional columns:
`grade`, `crop_slug`, `price_lkr_per_kg_min`, `price_lkr_per_kg_max`,
`price_lkr_per_kg_avg`, `unit` (default `kg`), `currency` (default `LKR`),
`sample_size`, `source_url`.

```
observed_on,market_code,commodity_label,grade,price_lkr_per_kg_avg,source_url
2026-04-15,dambulla-dec,Brinjals,long,210,https://www.harti.gov.lk/
```

## Loading

After `make db-up && make db-migrate`:

```bash
make db-load-market-prices-fixtures
# or directly:
DATABASE_URL='...' go run ./services/api/cmd/marketload \
  --file=pipelines/sources/market_prices/fixtures/dambulla-2026-04-15.csv
```

Then probe:

```bash
curl 'http://localhost:8080/v1/market-prices/latest/dambulla-dec'
curl 'http://localhost:8080/v1/market-prices?market=dambulla-dec&since=2026-04-01&limit=20'
```

## Sources

| Market | Source | Format | Cadence | Licence | Status |
| --- | --- | --- | --- | --- | --- |
| Dambulla DEC | [HARTI](https://www.harti.gov.lk/) — Daily Wholesale Vegetable Price Bulletin | PDF / web table | daily | Public sector — confirm redistribution per bulletin | **scraper TODO** |
| Meegoda DEC | HARTI same bulletin family | PDF | daily | same | not started |
| Welisara DEC | HARTI same bulletin family | PDF | daily | same | not started |
| Keppetipola DEC | HARTI same bulletin family | PDF | daily | same | not started |
| Narahenpita DEC | HARTI same bulletin family | PDF | daily | same | not started |
| Pettah retail | Department of Census and Statistics — Weekly Retail Price Report | Excel | weekly | same | not started |

When wiring a real scraper:

- Register the host's `robots.txt` check date in `pipelines/sources/<source>/config.yaml`.
- Cap rate at ≤ 1 req/sec.
- Identify as `GoyamaBot/<version> (+https://goyama.lk/bot)`.
- Cache ETag / Last-Modified.
- Store raw PDFs under `data/raw/<source>/<date>/` for re-extraction.
- Emit CSV under `data/staging/<source>/<date>.csv` and call `marketload`.

## Known commodity → crop_slug mappings

The bulletin uses local commodity labels; these are mapped to canonical
crop slugs at load time. When extending: add the mapping in your
extractor, never in the loader (keeps `marketload` source-agnostic).

| Bulletin label | crop_slug |
| --- | --- |
| Brinjals | brinjal |
| Tomato | tomato |
| Carrot | carrot |
| Capsicum | capsicum |
| Bitter Gourd | bitter-gourd |
| Snake Gourd | snake-gourd |
| Okra / Ladies' Fingers | okra |
| Cabbage | cabbage |
| Cauliflower | cauliflower |
| Beetroot | beetroot |
| Pumpkin | pumpkin |
| Onion (Red) | onion |
| Garlic | garlic |
| Chilli (Green) | chilli |
