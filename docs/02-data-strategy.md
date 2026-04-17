# 02 — Data strategy

**This is the critical first step.** The platform's value is the quality of its data. Everything else — recommendations, scanner, marketplace — rides on this foundation. Build the data pipeline before building feature code.

## 0. Ground rules

- **We gather the data ourselves.** No partnerships, no content licensing deals, no paid data feeds. We crawl the public web — government portals, research institute publications, agri blogs, YouTube, forums — extract, normalize, and curate.
- **The resulting knowledge corpus is open source.** The curated canonical dataset (crops, diseases, cultivation steps, AEZ layers, derived suitability) is published under a permissive licence (CC-BY-SA 4.0 for content; ODbL for derived geospatial layers). Anyone can fork, mirror, contribute corrections, or build derivative products. The app is one consumer of the corpus; it is not the owner.
- **Respect source terms.** We link out to third-party videos and images; we do not redistribute copyrighted media. What we redistribute is *our own structured extraction* (facts, schedules, symptom descriptions, AEZ polygons we vectorized) with per-field provenance back to the source URL.
- **AI-assisted, human-reviewed.** The crawling, extraction, alias resolution, and draft writing are automated. An agronomist reviews and publishes. Nothing reaches users unreviewed.

## 1. Data domains

### 1.1 Geospatial base layers (Sri Lanka)
- **Administrative boundaries** — Province, District, DS Division, GN Division.
- **Agro-ecological zones (AEZ)** — Sri Lanka's 46 AEZs, grouped into Wet / Intermediate / Dry.
- **Elevation (DEM)** — SRTM 30m or ALOS AW3D30 clipped to Sri Lanka (public NASA/JAXA datasets).
- **Soil** — Great Soil Groups of Sri Lanka (Reddish Brown Earth, Low Humic Gley, Non-Calcic Brown, etc.).
- **Rainfall & temperature** — historical normals (1991–2020) and forecast.
- **Water bodies & irrigation schemes** — Mahaweli, Uda Walawe, Gal Oya, minor tanks.

All derived from public imagery, PDFs, or open global datasets; vectorized in-house. The vector layers are part of the open corpus.

### 1.2 Crop taxonomy
Each crop record has: scientific/Sinhala/Tamil/English names, family, life cycle, DOA-released varieties, AEZ suitability with class, elevation/rainfall/temperature/pH envelope, season, duration, spacing, seed rate, yield expectation, water & nutrient requirement, companions, rotation partners, market category.

### 1.3 Pest & disease catalog
Per entity: scientific + local names, affected crops, symptoms with ≥ 3 images per stage, causal organism, transmission, favoring conditions, remedies (cultural / biological / chemical with active ingredient and PHI), resistant varieties, commonly-confused-with list, severity.

### 1.4 Cultivation procedures
Per `(crop × AEZ × season)`: land prep, nursery, transplanting, fertilizer schedule by DAP, irrigation schedule, pest/disease calendar, harvesting, post-harvest, storage, grading.

### 1.5 Curated media
YouTube videos and blog posts linked (not rehosted), tagged to crop/step/language, with our own summary extracted.

### 1.6 Market data
Wholesale/retail commodity prices from the publicly available daily bulletins of the Dedicated Economic Centres and the Ministry of Agriculture. Scraped into a time series.

### 1.7 User-generated content
Disease scans (with fuzzed location), marketplace listings, posts, farm diaries. UGC never mixes into the open corpus without explicit user opt-in.

## 2. Crawl targets (Sri Lanka web)

Every target has a dedicated crawler + extractor. Crawlers respect `robots.txt`, set a descriptive User-Agent (`GoyamaBot/1.0 (+https://goyama.lk/bot)`), throttle to ≤ 1 req/sec per host, and cache aggressively.

### Government & research (primary, high-trust)
- **doa.gov.lk** — Department of Agriculture main portal: crop profiles, leaflets, advisories, variety releases.
- **HORDI** — Horticultural Crops Research & Development Institute — vegetables, fruits.
- **RRDI** — Rice Research & Development Institute — rice varieties, recommendations.
- **FCRDI** — Field Crops Research & Development Institute — maize, legumes, oilseeds.
- **SRI / CRI / TRI / RRI** — Sugarcane / Coconut / Tea / Rubber research institutes.
- **NRMC** — Natural Resources Management Centre — AEZ, soil maps as PDFs.
- **meteo.gov.lk** — Department of Meteorology — climate normals, station bulletins.
- **statistics.gov.lk** — DCS — production, extent, prices, agri surveys.
- **HARTI** — market bulletins, socio-economic studies.
- **Ministry of Agriculture** — policy documents, broadcasts.
- **Provincial agriculture departments** — regional advisories.
- **Universities** — Peradeniya, Ruhuna, Rajarata, Wayamba agri faculties — theses, extension pamphlets, journal articles.

### Agri media & blogs
- **govithena.lk**, **govipola.lk**, **agrimin.gov.lk** newsrooms — local agri journalism.
- Independent Sri Lankan agri blogs in Sinhala and Tamil.
- Co-op and NGO extension content (Oxfam, CARE, Practical Action Sri Lanka where content is public).

### Video
- **YouTube Data API** — discover Sri Lankan agri channels (e.g. official DOA channel, "Govi Gurukam"-style programmes, Rupavahini agri slots, Sirasa agri segments, grower vlogs). For each channel: pull metadata, transcripts (auto-captions where available), thumbnails. We store references and our own summaries, not the videos.
- Flag channels as "primary" (official), "secondary" (reputable educators), "community" (growers).

### Community knowledge
- Public Facebook groups' public posts (read via Graph API where ToS permits), forum threads, Reddit (r/srilanka, r/gardening) — used only as lead-generation for topics to research; never copied into the corpus verbatim.

### Global baselines (fill gaps, never override Sri Lanka-specific data)
- **FAO ECOCROP** — global crop envelopes (CC-BY).
- **Wikidata / Wikipedia** — taxonomic backbone (CC-BY-SA).
- **GBIF**, **iNaturalist** — species occurrence and images under compatible licences.
- **PlantVillage** — baseline disease images for the scanner (research licence).

Sri Lanka-specific sources always take precedence when they disagree with a global baseline. The provenance record makes the conflict explicit.

## 3. Crawling & extraction pipeline

```
 scheduler (cron)
      │
      ▼
 per-source crawler ──▶ raw zone (object storage, versioned by content hash)
                                │
                                ▼
                          extractor
                              │
           ┌──────────────────┼───────────────────┐
           ▼                  ▼                   ▼
   HTML/PDF → text     media → thumbnails    tables → structured
           │                  │                   │
           └──────────────────┼───────────────────┘
                              ▼
                   LLM-assisted parser
                    (structured extraction)
                              │
                              ▼
                    staging (typed, validated)
                              │
                              ▼
               entity resolution + alias matching
                              │
                              ▼
                    agronomist review queue
                              │
                              ▼
                  canonical store (published)
                              │
           ┌──────────────────┼───────────────────┐
           ▼                  ▼                   ▼
   search index       knowledge graph        open-corpus
                                              exporter
```

### Crawlers
- One worker per source, each a small TypeScript/Python module implementing `fetch()`, `discover()`, `fingerprint()`.
- Polite: rate-limited, caches `ETag`/`Last-Modified`, respects `robots.txt`.
- Output: raw blob + metadata `(source, url, fetched_at, sha256, content_type, lang)` in the raw zone. Immutable.

### Extractors
- Per content type: HTML (Readability + custom selectors), PDF (pdfminer + layout-aware parsing for tables), images (captions + OCR for embedded text in Sinhala/Tamil via Tesseract with `sin` + `tam` trained data), YouTube (Data API metadata + `yt-dlp` subtitles where licence permits).
- Language detection + translation to the other two languages (LLM-assisted with human QA).

### LLM-assisted structured extraction
- Pass extracted text through an LLM with a strict JSON schema prompt ("extract fertilizer schedule as a list of `{dap_min, dap_max, fertilizer, amount_kg_per_acre}`").
- Every extracted field carries `source_url`, `extractor_version`, `model_version`, `confidence`.
- Low-confidence or schema-violating outputs are flagged.
- The LLM is a parser, never the source of truth. Numbers and dosages must trace back to a citation.

### Entity resolution
- Crops and diseases have many aliases. A curated alias table bootstrapped from Wikidata, then expanded by fuzzy-match + embedding similarity.
- New strings land in a review queue with suggested matches; the agronomist accepts, edits, or rejects.

### Agronomist review
- Draft records are `status=draft` with the full extraction trail visible: every field's source URL, quote, and model confidence.
- Reviewer can edit, accept, request more sources, or reject. Bulk-approve for trivially correct records.
- **No chemical dosage, PHI, or disease-remedy pair is published without explicit agronomist approval.** This is a hard gate.

### Open-corpus exporter
- A nightly job snapshots the canonical, published records into machine-readable dumps (JSON, JSONL, GeoJSON, Parquet) with full provenance.
- Published to a public GitHub repo (`goyama/corpus`) and mirrored on a static CDN. Versioned; each release tagged.
- Anyone can `git clone` the knowledge base and use it.

## 4. Data quality

- **Provenance on every field** — `source_id`, `source_url`, `quote`, `confidence`, `reviewed_by`, `reviewed_at`. The API surfaces it.
- **Versioning** — append-only canonical; updates create new versions. Changelog in the CMS and in the public corpus.
- **Audit tests in CI** — every published crop has ≥ 1 AEZ suitability; every disease has ≥ 3 images; every chemical remedy names an active ingredient and PHI; bilingual coverage ≥ 90%.
- **Cross-source consistency checks** — for fields we've extracted from multiple sources, a discrepancy report surfaces contradictions.
- **Community corrections** — public issue tracker on the corpus repo; PRs accepted (human-reviewed) to fix errors, add aliases, or contribute local-variety data.

## 5. Licensing & ethics

- **Corpus licence** — **CC-BY-SA 4.0** for text content, **ODbL** for geodata, **CC-BY 4.0** for images we produce ourselves. Per-record licence stored, because we honor upstream licences for anything we redistribute (most government content is CC-BY or public-domain; we verify per source).
- **No redistribution of third-party media** — videos and photos stay with the original host; we store links + our own summary + a permalink archive (for link-rot protection, we keep a private Wayback-style snapshot but do not serve it publicly).
- **User data separate from open corpus** — scans, listings, messages, posts, and plot polygons are private. Aggregated, anonymised, opt-in signals (e.g. disease-pressure heatmap at GN-division grain) may be published; raw UGC never is.
- **Transparent bot identity** — crawler UA identifies us and links to a page describing what we do and how to opt out.

## 6. First 90 days — concrete plan

| Week | Deliverable |
|---|---|
| 1–2 | Canonical schemas locked. Raw/staging/canonical stores up. One lead agronomist onboarded (reviewer of record). Crawler framework + one working crawler (DOA) end-to-end. |
| 3–4 | Geospatial layers ingested and vectorized (admin, AEZ, soil, elevation, rainfall). Reverse-geocode API returns the full context for any Sri Lanka coordinate. |
| 5–6 | Crawlers for HORDI, RRDI, FCRDI, Met Dept, DCS live. LLM-assisted extraction for HTML + PDF. Alias tables seeded. |
| 7–8 | 30 priority crops drafted from extractions (rice, 10 priority veg, 10 priority fruits, 5 spices, maize, green gram, soybean). Agronomist review in progress. |
| 9–10 | 60 priority diseases drafted with ≥ 3 images each (sourced via iNaturalist / GBIF / open research repositories; Sri Lanka-specific image collection queued for the in-house shoot programme in doc 06). |
| 11–12 | Market price daily scraper live. Weather proxy live. Cultivation procedure drafts for 30 crops × 2 seasons. Read-only data API exposed. Admin CMS usable end-to-end. **First public corpus release tagged `v0.1`.** |

## 7. The "AI does the gathering" operating model

This system is designed for an AI agent (the one building this platform) to run continuously:

1. **Discover** — given a crop or disease slug, the agent searches the crawl targets, finds relevant pages/videos, enqueues them.
2. **Extract** — for each fetched resource, the agent runs the extractor + LLM parser, produces a draft record with provenance.
3. **Reconcile** — the agent merges extractions across sources, flags disagreements, suggests an authoritative value with a rationale.
4. **Enrich** — the agent backfills translations (SI/TA/EN), generates plain-language summaries, tags media.
5. **Queue for review** — the agent presents the reviewer with the draft, the source quotes, and its proposed resolution.
6. **Learn** — agronomist edits become training signal: the agent tunes its extraction prompts and priors for future runs.

The human role collapses to **review, correction, and prioritization** — not transcription. This is what makes the corpus feasible at Sri Lanka's full scope within one team.
