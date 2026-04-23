# Market price observations

Daily wholesale + retail price observations from Sri Lanka's Dedicated
Economic Centres (DECs) and HARTI bulletins. Backs `GET /v1/market-prices`
and `GET /v1/market-prices/latest/{market}`.

## Architecture

The flow is intentionally split:

1. **Acquire** the daily bulletin (PDF / Excel / web table) from the source.
   Per the polite-crawl rules in `CLAUDE.md`, this step honors `robots.txt`
   and rate limits, and stores raw artifacts under `data/raw/<source>/`
   (gitignored) for provenance. For HARTI, this is
   `goyama crawl market_prices` — the [`HartiDailyPricesCrawler`](crawler.py)
   scrapes the `daily-price.php` index and downloads every dated English
   bulletin PDF that isn't already in the raw zone.
2. **Normalise** the raw bulletin into the canonical CSV shape (one row
   per `(market, commodity, grade, observed_on)`). This is per-source
   work; the schema is shared. For HARTI, this is
   `goyama market-prices normalize <bulletin.pdf> --observed-on=<date>` —
   [`HartiBulletinNormalizer`](normalizer.py) extracts tabular text with
   `pdfminer.six`, segments by DEC header, and regex-parses rows into
   [`NormalizedRow`](normalizer.py) instances.
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

## End-to-end runbook (HARTI)

One-shot flow for a single day:

```bash
# 1. Acquire — download the dated bulletin PDF(s) from HARTI.
uv run goyama crawl market_prices --limit 50

# 2. Normalise — parse a specific bulletin into CSV under data/staging/.
uv run goyama market-prices normalize \
    ../data/raw/market_prices/<sha256>.bin \
    --observed-on 2026-04-15

# 3. Load — upsert into Postgres.
DATABASE_URL='postgres://goyama:goyama@localhost:54320/goyama?sslmode=disable' \
    go run ./services/api/cmd/marketload \
    --file=data/staging/market_prices/2026-04-15.csv
```

The crawler respects `robots.txt`, identifies as `GoyamaBot/…`, and caps
throughput at 0.5 req/sec (see `config.yaml`). Re-running is safe — the
raw store dedupes by content hash and `marketload` upserts.

## Sources

| Market | Source | Format | Cadence | Licence | Status |
| --- | --- | --- | --- | --- | --- |
| Dambulla DEC | [HARTI](https://www.harti.gov.lk/) — Daily Food Commodities Bulletin | PDF | daily | Public sector — confirm redistribution per bulletin | **scraper live (pending real-PDF calibration)** |
| Meegoda DEC | HARTI same bulletin family | PDF | daily | same | covered by HARTI scraper |
| Welisara DEC | HARTI same bulletin family | PDF | daily | same | covered by HARTI scraper |
| Keppetipola DEC | HARTI same bulletin family | PDF | daily | same | covered by HARTI scraper |
| Narahenpita DEC | HARTI same bulletin family | PDF | daily | same | covered by HARTI scraper |
| Pettah retail | Department of Census and Statistics — Weekly Retail Price Report | Excel | weekly | same | not started |

### Calibrating the HARTI parser against a real bulletin

The row regex in [`normalizer.py`](normalizer.py) was tuned against the
2024–2026 HARTI layout based on the public bulletin description; once a
real PDF lands in the raw zone, verify that:

- Every `_MARKET_HEADERS` section produces the expected number of rows.
- No rows are silently dropped because the grade cell uses an unexpected
  token (e.g. `Grade I`, `Premium`). Extend the grade alternatives in
  `_ROW_RE` as needed.
- The commodity labels fall into `COMMODITY_TO_CROP_SLUG`. Unresolved
  labels write `crop_slug=""` — the row still lands, but downstream joins
  onto canonical crops miss. Add new mappings rather than guessing on
  the reader side.

Run `uv run pytest tests/test_market_prices.py` after any regex change
and commit any new real-PDF fixtures under `pipelines/tests/fixtures/market_prices/`.

## Scraper expectations for any new market source

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
