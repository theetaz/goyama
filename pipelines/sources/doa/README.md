# Source: Department of Agriculture, Sri Lanka (doa.gov.lk)

**Authority.** The Department of Agriculture (DOA) is Sri Lanka's national agricultural research, extension, and regulatory body. Content published on `doa.gov.lk` and its affiliated institute subdomains is produced by DOA researchers and extension officers.

**Why we crawl it first.** It's the single most authoritative Sri Lanka-specific source for crop profiles, variety releases, cultivation calendars, pest & disease advisories, and approved remedies.

## Crawl policy

- **User-Agent**: `CropDocBot/<version> (+https://cropdoc.lk/bot)`
- **Rate**: 1 req / sec / host (override in `config.yaml` if needed).
- **robots.txt**: respected via the shared `RobotsCache`.
- **Freshness**: weekly for crop profiles; daily for advisories.
- **Failure handling**: transport errors retry with exponential backoff (3 attempts); 4xx/5xx logged and skipped.

## Licence and redistribution

Government of Sri Lanka content is typically reusable with attribution. We:

- Extract **facts** (names, ranges, schedules, dosages) into our schema with citation.
- **Link** to the original PDF/page; we do not rehost documents.
- Keep a private snapshot in `data/raw/` for link-rot protection.
- Attribute in the corpus record's `provenance.source_url`.

Confirm licence in the per-source register before publishing; open an issue if a specific document's terms are unclear.

## Files

- `config.yaml` — seed URLs, allow/deny patterns, rate limit.
- `crawler.py` — discovers and fetches crop-profile and advisory pages.
- `extractor.py` — converts a fetched HTML page into a draft `Crop` record (JSON Schema validated).
