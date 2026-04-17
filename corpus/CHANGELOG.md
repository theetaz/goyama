# Corpus changelog

All notable changes to the Goyama open corpus. The format follows [Keep a Changelog](https://keepachangelog.com/) and semantic versioning.

## [v0.0.1-drafts] — 2026-04-17

### Added
- **First pre-release drafts bundle** tagged as `v0.0.1-drafts` under `corpus/releases/v0.0.1-drafts/`:
  - `crops.jsonl` (98), `crop_varieties.jsonl` (30), `diseases.jsonl` (34), `pests.jsonl` (28), `remedies.jsonl` (11).
  - `manifest.json` with per-bundle sha256 checksums and licence metadata.
  - `sources.json` copied verbatim from the seed register.
  - `README.md` describing review workflow and known limitations.
- **Corpus exporter** (`pipelines/core/exporter.py` + `goyama export` CLI command) — deterministic JSONL serialisation sorted by slug, sha256 manifest, and a `--include-draft` flag for pre-release drafts bundles.
- **Exporter tests** (3 new unit tests) covering draft inclusion, published-only default, and licence/timestamp manifest fields. 12/12 tests green.

### Notes
- Drafts bundle requires agronomist review before promotion to `v0.1.0`. Chemical remedy PHI values, disease-remedy pairings, and user-facing numeric recommendations are all subject to the CLAUDE.md hard-gate rule.

## [Unreleased]

### Added
- Canonical JSON Schemas for Crop, CropVariety, AEZ, Disease, Pest, Symptom, Remedy, CultivationStep, Media, and Provenance.
- Postgres migrations (`packages/schema/migrations/0001_init.sql`, `0002_graph.sql`) covering relational, geospatial, vector, and graph layers.
- Crawling + extraction framework under `pipelines/core/` with polite fetcher, robots.txt handling, per-host rate limiting, and a raw-zone append-only store.
- First source integration: Department of Agriculture (`pipelines/sources/doa/`) with crawler, LLM-assisted extractor, and the `crop_extraction_v1` prompt.
- `goyama` CLI (`goyama sources list | crawl | extract | validate`).
- **Seed corpus drafts** under `corpus/seed/`, gathered live from authoritative Sri Lankan sources (DOA, HORDI, RRDI, FCRDI, FRDI, Ministry of Agriculture, Department of Export Agriculture, HARTI, Sri Lanka Biodiversity CHM, plus FAO baselines) with full per-field provenance and schema-validated:
  - **98 crops** — rice + 40 vegetables (standard + indigenous leafy greens + specialty: brinjal, tomato, okra, bitter-gourd, cabbage, capsicum, carrot, big onion, cucumber, snake-gourd, pumpkin, luffa, leeks, beetroot, potato, sweet-potato, yard-long-bean, radish, cassava, snap-bean, winged-bean, cauliflower, knol-khol, innala, kiriala, kohila, oyster mushroom, chilli, ela-batu, kekiri, lettuce, thumba-karawila, curry-leaf, elephant-foot-yam, gotukola, mukunuwenna, kankun, ash-plantain, taro, nivithi, kathurumurunga, thampala, sarana, thibbatu) + 8 field crops (groundnut, sesame, soybean, maize, finger-millet, mung-bean, cowpea, pigeon-pea) + 14 fruits (mango, banana, pineapple, passion-fruit, rambutan, guava, dragon-fruit, mangosteen, durian, sweet-orange, papaya, lime, jackfruit, breadfruit + wood-apple, beli, nelli, sapodilla, soursop, sugar-apple, grape) + 5 spices (cinnamon, black-pepper, cardamom, ginger, turmeric + betel, vanilla, arecanut) + 4 plantation crops (tea, coconut, rubber, cocoa, coffee, cashew, gliricidia) + 4 medicinal (aralu, bulu, kumbuk, thebu) + 2 ornamentals (anthurium, orchid) + others (sugarcane, tobacco, lotus).
  - **30 varieties** — 20 rice (14 modern RRDI + 6 traditional heirlooms: Suwandel, Kalu Heenati, Rath Suwandel, Pachchaperumal, Madathawalu, Sudu Heenati) + 4 banana (Kolikuttu, Ambul, Seeni, Anamalu) + 2 brinjal (Thinnaweli Purple, Amanda F1) + 1 tomato (Thilina) + 1 king coconut + 2 tea clone series (TRI 2000, TRI 5000).
  - **34 plant diseases** covering rice (7 — blast, BLB, sheath blight, brown spot, sheath rot, leaf scald, narrow brown leaf spot, tungro), solanaceae (bacterial wilt, TYLCV, late blight, early blight, damping-off, TSWV), cucurbits (powdery mildew, downy mildew), fruit crops (anthracnose, SLCMV, Panama disease, black Sigatoka, mango malformation, citrus greening/HLB, PRSV, banana bunchy top, pineapple mealybug wilt), plantation (tea blister blight, tea grey blight, rubber white root disease, Weligama coconut leaf wilt, coffee leaf rust, cinnamon stripe canker, cinnamon leaf spot), brassica (clubroot), and rhizome crops (ginger soft rot).
  - **28 pests** — 6 rice (BPH, YSB, rice gall midge, rice paddy bug, rice leaf folder, rice hispa) + 22 others: Oriental fruit fly, fall armyworm, whitefly, red palm weevil, coconut mite, chilli thrips, papaya mealybug, pink hibiscus mealybug, aphid, leaf miner, tea red spider mite, tea shot-hole borer, tea tortrix, mango hopper, diamondback moth, brinjal fruit-and-shoot borer, coconut black-headed caterpillar, banana weevil, white grub, red pumpkin beetle, sugarcane internode borer, onion thrips.
  - **11 remedies** — chemical (Tebuconazole/Tricyclazole/Isoprothiolane for rice blast; Mancozeb broad-spectrum; Chlorothalonil), cultural (burnt paddy husk + certified seed; methyl eugenol fruit-fly trap; ferrugineol red-palm-weevil trap), biological (*Trichoderma asperellum* for bacterial wilt; neem/azadirachtin broad-spectrum biopesticide), resistant-variety (Lanka Sour × HORDI Tomato 03 grafting for bacterial wilt).
  - **Source register** (`corpus/seed/sources.json`) covering DOA, Ministry of Agriculture, DEA, Wikipedia, CBD-CHM, EDB, FAO GAEZ + 50+ peer-reviewed and institutional sources cited across provenance records.
  - **AEZ + soil-group narrative notes** for Wet / Intermediate / Dry zones and the five major Great Soil Groups (polygon vectorisation still deferred).
  - **Reference documents** on Sri Lanka's apiculture (4 bee species), invasive aquatic weeds (water hyacinth + Salvinia), and the broader agri digital-services ecosystem.
  - **Total: 201 schema-validated corpus records** covering virtually all principal crops and major disease/pest/remedy combinations of Sri Lankan agriculture.

### Fixed
- Pest schema: added `notes` (i18n string) to `favored_conditions` for parity with Disease schema.

### Planned for `v0.1.0`
- Expand seed set to the 30 priority crops and 60 priority diseases targeted by doc 08.
- Vectorize NRMC AEZ polygons and publish as canonical records.
- Agronomist review pass over the seed drafts.
- Nightly exporter that concatenates reviewed records from `corpus/seed/` into release JSONL/GeoJSON/Parquet snapshots under `corpus/releases/v0.1.0/`.

### Planned for `v0.1.0`
- First 30 priority crops drafted, reviewed, and published.
- 60 priority diseases with ≥ 3 reviewed images each.
- Cultivation steps for priority crops × 2 seasons.
- AEZ polygons with attributes.
- Nightly corpus exporter from canonical store to this directory.

## [0.0.0] — 2026-04-17

### Added
- Repository scaffolding: README, CONTRIBUTING, CODE_OF_CONDUCT, LICENSE, planning docs under `docs/`.
- This directory as the placeholder for versioned corpus releases.
