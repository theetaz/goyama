# Goyama

**An open, Sri Lanka-first knowledge base and companion apps for farmers.**

Goyama builds a public, structured, multilingual knowledge corpus of Sri Lankan agriculture — crops, varieties, diseases, pests, remedies, cultivation calendars, agro-ecological zones, and market context — and a set of mobile and web applications that put that knowledge in the hands of growers.

> **Status:** early development. Corpus `v0.0` — schema definition and ingestion pipeline in progress. Contributors welcome.

## What's in this repository

- **`docs/`** — planning documents. Start with [docs/01-vision-and-scope.md](docs/01-vision-and-scope.md).
- **`corpus/`** *(coming soon)* — the public knowledge corpus: JSON/JSONL/GeoJSON/Parquet snapshots of crops, diseases, remedies, AEZ polygons, with per-field provenance.
- **`packages/`** *(coming soon)* — shared code: API client, domain types, i18n, knowledge-graph utilities.
- **`apps/`** *(coming soon)* — `web`, `mobile`, `admin` (CMS), `api` (backend).
- **`pipelines/`** *(coming soon)* — crawlers, extractors, normalizers, model training jobs.

## What we're building — in one paragraph

A data-first platform. The core asset is a **structured knowledge corpus of Sri Lankan agriculture**, gathered by crawling public sources (Department of Agriculture and research institutes, universities, open datasets, agri media, and curated YouTube content), extracted into a strict schema with full provenance, reviewed by agronomists, and released under an open licence. On top of the corpus we build a mobile and web app for farmers (professional and hobbyist): location-aware crop recommendations, cultivation guides, a disease/pest scanner, a marketplace, and a community feed. The corpus is the public good; the apps are how we make it useful.

## Guiding principles

- **Data first, code second.** Nothing is hardcoded. Content lives in the corpus; the apps render it.
- **Sri Lanka specific.** 46 agro-ecological zones, Maha/Yala seasons, Wet/Intermediate/Dry zone distinctions, local varieties (BG/BW/AT rice, local brinjal cultivars, etc.), trilingual content.
- **Geospatial at the core.** Every recommendation is bound to a location (AEZ, elevation, soil, rainfall). No generic advice.
- **Gathered by us, open to all.** We crawl and extract from public sources — no proprietary data feeds. The curated corpus is released under CC-BY-SA 4.0 (content), ODbL (geodata), and CC-BY 4.0 (images we produce).
- **Human-reviewed.** Agronomists gate every published record. Chemical dosages, PHI, and disease-remedy pairs never publish without review.
- **Privacy by default.** Personal data (plots, scans, messages, listings) is private. Only opt-in aggregates ever enter the public corpus.
- **Offline-friendly mobile.** Core reference content and the disease model ship on-device.

## Corpus at a glance

Each published record carries:

- A **canonical identity** (slug, scientific name, local names in Sinhala / Tamil / English).
- **Structured fields** typed against a schema ([docs/03-domain-model.md](docs/03-domain-model.md)).
- **Per-field provenance**: `source_url`, quote, extraction confidence, reviewer, review timestamp.
- **Versioning**: append-only history with a public changelog.
- **Licence**: per record, honoring upstream terms.

## Geo lookup (Phase 0 exit endpoint)

`GET /v1/geo/lookup?lat=<float>&lng=<float>` resolves any Sri Lanka
coordinate into its administrative + agro-ecological envelope:

```jsonc
{
  "location":   { "lat": 7.2906, "lng": 80.6337 },
  "district":   { "code": "LK-21", "name_en": "Kandy",  "province_name": "Central" },
  "ds_division":{ "code": "21-12", "name_en": "Kandy Four Gravets" },
  "aez":        { "code": "WM3", "zone_group": "wet", "elevation_class": "mid_country",
                  "avg_rainfall_mm": 2100, "dominant_soil_groups": ["red_yellow_podzolic"] }
}
```

Local dev:

```bash
make db-up
make db-migrate
make db-load-geo-fixtures   # loads simplified dev polygons (NOT real boundaries)
cd services/api && DATABASE_URL='postgres://goyama:goyama@localhost:54320/goyama?sslmode=disable' make run

curl 'http://localhost:8080/v1/geo/lookup?lat=7.29&lng=80.63'
```

For real Sri Lanka boundary data and the canonical AEZ map, see
[pipelines/geo/README.md](pipelines/geo/README.md).

## Market prices (Phase 1 deliverable)

`GET /v1/market-prices?market=<code>&since=<date>` returns daily wholesale
or retail price observations from Sri Lanka's Dedicated Economic Centres,
starting with Dambulla. `GET /v1/market-prices/latest/{market}` returns
every commodity from the most recent observation date.

```bash
make db-load-market-prices-fixtures
curl 'http://localhost:8080/v1/market-prices/latest/dambulla-dec'
```

CSV importer + sourcing notes for HARTI bulletins:
[pipelines/sources/market_prices/README.md](pipelines/sources/market_prices/README.md).

## Cultivation plans + knowledge graph

The corpus carries a versioned knowledge graph: structured `cultivation_plan`
aggregates (per crop × AEZ × season) with child activities / pest risks /
economics, plus unstructured `knowledge_chunk` rows (for video transcripts,
cross-regional advisory notes, research-paper excerpts). Every record
carries an `authority_level` so DOA-validated guidance renders distinctly
from "promising practice from Tamil Nadu, not yet locally validated".

```bash
make db-load-cultivation-plans   # upserts every JSON fixture under corpus/seed/cultivation_plans/
make db-load-knowledge           # upserts knowledge_source + knowledge_chunk fixtures
curl 'http://localhost:8080/v1/crops/red-onion/cultivation-plans'
curl 'http://localhost:8080/v1/cultivation-plans/red-onion-dry-zone-maha'
curl 'http://localhost:8080/v1/crops/tomato/knowledge'
```

Admins review drafts at `/review-plans` and `/review-knowledge` on the
web-admin app; promoting to `published` is what surfaces them on the
farmer crop detail page.

## Quick links

- [Vision & scope](docs/01-vision-and-scope.md)
- [Data strategy](docs/02-data-strategy.md) *(how we gather, extract, review, and release)*
- [Domain model & knowledge graph](docs/03-domain-model.md)
- [System architecture](docs/04-architecture.md) — **Go API + Vite/React client + Vite/React admin + Expo mobile**
- [Feature specifications](docs/05-features.md) — map-first UX, social feed integration
- [ML: disease & pest scanner](docs/06-ml-disease-scanner.md)
- [Marketplace & community](docs/07-marketplace-community.md)
- [Roadmap](docs/08-roadmap.md)
- [Content strategy: corpus → database → query](docs/09-content-strategy.md)
- [UI / UX principles](docs/10-ui-ux-principles.md) — farmer-friendly design language
- [Backend API design (Go)](docs/11-backend-api-design.md)
- [Contributing](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)

## Tech stack (locked decisions)

| Layer | Choice |
|---|---|
| Backend API | **Go 1.22+** · chi · pgx · sqlc · OpenAPI 3.1 |
| DB | Postgres 16 + PostGIS + pgvector |
| Web (client) | **Vite + React 18 + TypeScript + TanStack Query + Shadcn UI + Tailwind** → `app.goyama.lk` |
| Web (admin) | Same stack, **separate app + theme** → `admin.goyama.lk` |
| Mobile | **Expo + React Native + TypeScript + TanStack Query + NativeWind** |
| Maps | MapLibre GL (web + native) with self-hosted vector tiles |
| Object storage | Cloudflare R2 + Cloudflare Images |
| Data ingestion | Python pipelines (already built; see `pipelines/`) |

## Contributing

We welcome contributions of all kinds: bug reports, schema suggestions, crawler improvements, translation help, and — most of all — **data corrections**. If you spot an error in the corpus, open an issue or a PR against the `corpus/` directory. See [CONTRIBUTING.md](CONTRIBUTING.md).

Agronomists, extension officers, and researchers: we want your eyes on the review queue. Reach out via GitHub Discussions.

## Licences

- **Code** — [MIT](LICENSE)
- **Documentation & corpus content** — CC-BY-SA 4.0
- **Geodata we publish** — ODbL
- **Images we produce** — CC-BY 4.0
- **Third-party media** — we link out; we do not redistribute.

## Acknowledgements

This project would not be possible without the decades of public work done by Sri Lanka's Department of Agriculture, its research institutes (HORDI, RRDI, FCRDI, SRI, CRI, TRI, RRI), the Natural Resources Management Centre, the Department of Meteorology, the Department of Census and Statistics, HARTI, and the country's agricultural universities. We build on their publications and link back to them from every extracted field.
