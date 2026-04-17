# 09 — Content strategy: from corpus to query

This document defines **how content flows from the public web into a queryable system** that the apps and third parties can build on. It is the bridge between the data strategy (doc 02) and the domain model (doc 03).

## 1. Design influences

We borrow liberally from systems that have solved parts of this problem and adapt to Sri Lanka's context.

### Apps we study
| System | What we learn from it |
|---|---|
| **Plantix** (PEAT) | On-device disease classifier; crop + disease taxonomy; remedy catalog with cultural/biological/chemical tabs; user feed of scans; extension-officer partnerships. |
| **PlantNet** | Contributor-driven species catalog; confidence-surfaced results; "similar species" disambiguation; open data model. |
| **iNaturalist** | Community-curated observations; expert verification workflow; open licensing of data and images; taxonomic backbone. |
| **PictureThis / Seek** | Friendly onboarding for non-experts; progressive disclosure of technical detail. |
| **Kisan Suvidha / Kisan Call Centre (India)** | Location-aware advisories; weather + market integration; voice/low-literacy channels. |
| **Digital Green** | Video-first extension content; peer-to-peer learning loops; offline-first delivery. |
| **PlantVillage Nuru** | Low-bandwidth, on-device, Swahili-first design patterns applicable to Sinhala/Tamil. |
| **FarmLogs / Climate FieldView** | Plot-centric data model; season planner; actionable agro-alerts. |

### Research directions we draw on
- **Crop suitability modeling**: FAO land-suitability classes (S1/S2/S3/N), EcoCrop envelopes; weighted-overlay GIS methods; recent ML approaches (random forest, gradient-boosted trees on soil + climate features).
- **Plant disease classification**: ImageNet-pretrained CNNs fine-tuned on PlantVillage/PlantDoc; vision transformers; domain-adaptation from lab to field; open-set recognition; Grad-CAM for explainability.
- **Agricultural knowledge graphs**: AGROVOC (FAO), KNOMAD-KG, AgriKG. Schema design — entities (Crop, Disease, Pest, Remedy, Region, Season, Nutrient, Practice) and typed relations — is a well-trodden area; we adapt to Sri Lanka.
- **Information extraction from agricultural text**: rule-based + LLM-assisted extraction from extension pamphlets and research papers; multilingual NER for Sinhala/Tamil.
- **Multilingual low-resource NLP**: Sinhala/Tamil tokenization, machine translation quality thresholds, human-in-the-loop review pipelines.
- **Recommender explainability**: factor-attribution for end users who need to trust why a crop is suggested.

(A curated reading list is maintained in `docs/reading-list.md` — to be added as we go.)

## 2. The content lifecycle — six stages

```
  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐
  │ discover │──▶│ fetch    │──▶│ extract  │──▶│ normalize│──▶│ review   │──▶│ publish  │
  └──────────┘   └──────────┘   └──────────┘   └──────────┘   └──────────┘   └──────────┘
                                                                                   │
                                                         ┌─────────────────────────┼─────────────────────────┐
                                                         ▼                         ▼                         ▼
                                                  canonical DB              search + KG              open corpus
                                                                                                     (versioned release)
```

### 2.1 Discover
- Seed list per source type: DOA crop pages, research institute publication indexes, Met Dept bulletin archives, HARTI price bulletins, university thesis portals, curated YouTube channels, agri blog sitemaps.
- Link discovery is incremental: each crawl run emits new URLs; a dispatcher enqueues them subject to per-host rate limits.
- Freshness policy per source (daily for prices, weekly for blogs, monthly for research papers).

### 2.2 Fetch
- Polite crawler framework: `robots.txt`, UA identity, rate limiting, retry with backoff, ETag/Last-Modified caching.
- Output: raw blob + headers + fetch metadata, stored immutably in the raw zone keyed by `sha256(content)`.
- **Audio/video subpipeline**:
  - YouTube via Data API (metadata) + `yt-dlp` (subtitles / audio — only where the video's licence allows; otherwise metadata + link only).
  - Audio extracted as 16 kHz mono WAV, transcribed with Whisper large-v3 (language auto-detect), per-segment confidence retained.
  - Transcripts are treated as regular text input to the extractor, with `source_type=video_transcript` and a link back to the timestamped segment.

### 2.3 Extract
- **Per content-type extractors**: HTML (Readability + source-specific selectors), PDF (layout-aware, table-aware), image (captions + OCR via Tesseract `sin`+`tam`+`eng`), transcript (segmentation + speaker turns).
- **LLM-assisted structured extraction**: a tight JSON-schema prompt per target record type (crop, disease, remedy, cultivation step, price series). Prompts are versioned in `pipelines/prompts/`.
- **Citation rule**: every numeric value (dosage, DAP, rate per acre, temperature range) must come with a verbatim source quote and URL anchor. If the LLM cannot produce one, the field is `null` with `reason=no_citation`.
- Extractors are **additive**: multiple sources can contribute fields to the same canonical record; conflicts are resolved at the normalize step.

### 2.4 Normalize
- **Entity resolution**: map extracted mentions ("vambatu", "brinjal", "Solanum melongena") to canonical crop IDs using the alias table (bootstrapped from Wikidata, curated over time).
- **Unit harmonization**: convert to canonical units (kg/acre for seed rate, mm for rainfall, °C for temperature, days for duration). Store originals for audit.
- **Conflict resolution**: when multiple sources disagree on a field, prefer (1) Sri Lanka-specific official (DOA / research institutes) > (2) Sri Lanka-specific academic > (3) Sri Lanka-specific community > (4) global baseline. A `conflicts` record stores all values with sources so reviewers can override.
- **Multilingual normalization**: text is stored per locale. English is the canonical authoring locale initially; Sinhala and Tamil are drafted by machine translation and flagged for human review.
- Output: a draft canonical record with full provenance.

### 2.5 Review
- Agronomists work from a review dashboard that shows:
  - The draft record.
  - Every field's source URL, quote, confidence, model version.
  - Any conflicting values from other sources.
  - A diff against any existing published version.
- Actions: accept, edit, request more sources, reject.
- Bulk-accept for trivially identical re-extractions.
- **Hard gates**: chemical active ingredients + dosages + PHI, disease-remedy pairs, resistant-variety claims, and any numeric figure shown to end users in a recommendation context cannot publish without explicit reviewer sign-off.

### 2.6 Publish
Once accepted:
1. A new version is appended to the canonical store (append-only; never mutate history).
2. Search index is reindexed for the affected record.
3. Knowledge graph edges are upserted.
4. Embeddings (`pgvector`) are recomputed for semantic search.
5. The nightly corpus exporter picks up the change in the next release.

## 3. Ingestion targets — how the corpus lands in the database

Each canonical entity has one or more of these representations, chosen for the query patterns it serves:

| Representation | Purpose | Backing store |
|---|---|---|
| Relational row (with JSONB for variable attributes) | Authoritative record, transactional writes, schema enforcement | Postgres |
| Geospatial geometry | Spatial queries ("what AEZ is this point in?") | PostGIS |
| Full-text index (multilingual) | Keyword search | Meilisearch |
| Dense embedding | Semantic / natural-language search | pgvector |
| Graph node + edges | Traversal ("crops suitable here that are resistant to this disease") | Apache AGE (Cypher over Postgres) |
| Open-corpus file | Portable, forkable release | GitHub repo + CDN (JSON, JSONL, GeoJSON, Parquet) |

The canonical Postgres row is authoritative. Every other representation is a **projection**. A change-data-capture job (Postgres logical replication + a lightweight dispatcher) keeps projections in sync so we never have silent drift.

## 4. Query patterns the system must serve

The content is only as good as the questions it answers. We design for these user intents from day one:

### 4.1 Identity lookups
- "What is this crop / disease / pest / remedy?"
- "What are the local names of X?"
- **Served by**: direct relational lookup + full-text + alias table.

### 4.2 Suitability & planning
- "What can I grow at this location, this season, for this effort level?"
- "Is rice variety BG 352 suitable for Anuradhapura?"
- "When should I plant brinjal in Kandy for the Yala season?"
- **Served by**: geospatial reverse-geocode → AEZ/soil/elevation context → relational filter on crop envelopes → scoring → graph traversal for rotation/companion sanity checks.

### 4.3 Diagnosis
- "What disease is this? [image]" → model prediction → linked disease record.
- "My brinjal leaves have yellow spots with a fuzzy underside. What is it?" → symptom-based retrieval (text + embedding) → ranked disease candidates.
- "What else looks like this to rule out?" → `confusion_with` edges in the graph.
- **Served by**: ML service (scanner) + symptom full-text/embedding search + KG `confusion_with` traversal.

### 4.4 Remediation
- "How do I treat late blight on tomato here?"
- "What's a chemical remedy approved in Sri Lanka, and what's the PHI?"
- "What's an organic alternative?"
- **Served by**: relational join disease → remedy, filtered by remedy type, keyed to DOA-approved active ingredients.

### 4.5 Procedural ("how-to")
- "Step-by-step, how do I grow chili?"
- "What fertilizer schedule at 30 / 45 / 60 DAP?"
- **Served by**: `cultivation_step` table, ordered by DAP and stage, enriched with media refs.

### 4.6 Context & markets
- "What's the price of tomato at Dambulla this week?"
- "Is the price of X trending up?"
- **Served by**: market time series in Postgres with materialized weekly aggregates.

### 4.7 Open-ended natural language (RAG)
- "A neighbor says to spray Mancozeb but only in the morning — why?"
- "What's the traditional pest management for bitter gourd in the Dry Zone?"
- **Served by**: retrieval over the corpus (full-text + semantic) → LLM answer **constrained to cite retrieved records** → answer rendered with the same provenance chips users see elsewhere. No freeform LLM answers without citations. If retrieval returns nothing confident, the app says "I don't have enough information about this yet" rather than inventing.

## 5. Semantic layer & embeddings

- **Record-level embeddings**: a single `content_embedding` vector per published record, computed from its concatenated canonical fields (name + scientific name + description + key attributes + locale).
- **Passage embeddings**: long-form content (cultivation steps, research summaries) is split into ~512-token passages; each passage gets its own vector, linked back to the parent record.
- **Multilingual model**: a single model that handles English, Sinhala, Tamil (e.g., multilingual-e5-large or bge-multilingual-gemma2) to keep cross-lingual retrieval honest.
- **Query embedding** cached for the session; retrieval returns record IDs with similarity scores; the API enriches with the canonical record.

## 6. Knowledge-graph projection

The graph is not a separate source of truth; it is a derived view that makes traversal queries cheap. The projection runs after each publish event and upserts:

- Nodes: `(:Crop {id})`, `(:Disease {id})`, `(:Remedy {id})`, `(:AEZ {id})`, `(:Variety {id})`, etc.
- Edges from relational foreign keys and join tables: `SUITABLE_IN`, `AFFECTED_BY`, `TREATED_BY`, `RESISTANT_TO`, `COMPANION_OF`, `ROTATES_WITH`, `GROWS_IN`, `CONFUSED_WITH`.

Example Cypher (via Apache AGE):

```sql
SELECT * FROM ag_catalog.cypher('cropdoc', $$
  MATCH (c:Crop)-[s:SUITABLE_IN]->(a:AEZ {code: 'IL1a'}),
        (c)-[:GROWS_IN]->(:Season {name: 'yala'})
  WHERE NOT (c)-[:AFFECTED_BY]->(:Disease {slug: 'late_blight'})
  RETURN c.slug, s.class
  ORDER BY s.class
$$) AS (slug agtype, class agtype);
```

## 7. Multilingual corpus rollout

- **Phase A (English-first)** — all canonical authoring is in English. Sinhala and Tamil fields populated by machine translation and shown in-app with a "machine translation" badge until reviewed.
- **Phase B (Sinhala parity)** — human review sweep across the 30 priority crops and 60 priority diseases in Sinhala. Badge removed once reviewed.
- **Phase C (Tamil parity)** — same sweep for Tamil.
- **Phase D (authored-in-SI/TA)** — reviewers can author directly in Sinhala or Tamil when the source was in that language (e.g., a DOA leaflet in Sinhala). Translations back to English are then drafted.

Release-gate rule: the app never blocks a user on a missing translation; it falls back to English with a visible language chip.

## 8. Governance of the open corpus

- **Public repo** (`cropdoc/corpus`) tracks versioned releases. Each release is a tag and a manifest.
- **CHANGELOG.md** in the corpus repo is human-readable.
- **Issues & PRs** are how the public contributes corrections. A PR with a source URL + a clear field change is reviewed by an agronomist; accepted changes become the next release.
- **Monthly release cadence** once `v0.1` ships; major versions when schema changes.
- **Deprecation policy**: fields are deprecated for two releases before removal; data consumers see a machine-readable deprecation notice in the release manifest.
- **Attribution**: downstream users must credit the corpus per CC-BY-SA; we publish a single-line attribution snippet they can copy.

## 9. Integrity & anti-misinformation

- Every user-facing fact on the app is linked to its provenance chip; tapping shows the source.
- If a record is retracted (e.g., a remedy whose registration is withdrawn), the app surfaces a "superseded" banner and links to the replacement.
- We do not redistribute content whose licence is ambiguous. When in doubt, link out.
- We log every LLM extraction with prompt + model version; if a model's quality regresses, we can re-extract affected records from the raw zone.

## 10. What this means for engineering, this week

- Lock `packages/schema/` (JSON Schema + SQL migrations) for the top-level entities in doc 03.
- Stand up the raw-zone object storage and the crawler framework in `pipelines/core/`.
- Build the first DOA crawler + extractor end-to-end, producing 10 draft crop records.
- Wire the agronomist review UI to the draft store (no publish yet).
- Set up the open-corpus repo (empty tagged `v0.0.0`) and the nightly exporter skeleton.
