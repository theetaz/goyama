# corpus/seed

**Draft records, awaiting agronomist review.**

These records were extracted from public, authoritative Sri Lankan sources (primarily the Department of Agriculture and its research institutes) and written into the canonical schema with per-field provenance citing the source URL and a verbatim quote. They are **not yet published** — every record carries `"status": "draft"` and must be reviewed before it is promoted into a tagged corpus release.

## Layout

```
seed/
├── crops/            — one JSON file per crop, filename is the slug
├── crop_varieties/   — rice varieties released by RRDI, one per file
├── diseases/         — plant diseases
├── pests/            — insect / arthropod pests
└── aez/              — agro-ecological zone notes (geometry deferred)
```

## Review workflow

1. An agronomist opens a PR-sized batch (≤ 10 files) from this directory.
2. For each file, verify every field against its cited source; correct or annotate as needed.
3. Flip `status` to `published` when satisfied, add `reviewed_by` + `reviewed_at` to each provenance entry, and move the file to `corpus/releases/<version>/`.
4. The nightly exporter concatenates published files into the release JSONL snapshots.

## Provenance convention

Every numeric field and every scientific name carries an entry in `field_provenance` with at minimum `source_id`, `source_url`, `fetched_at`, and `quote`. Unit conversions (e.g. `t/ha` → `kg/acre`) preserve the original quoted string and note the conversion in `review_notes`.

## Unit conversion notes

- 1 hectare = 2.47105 acres → `X t/ha × 404.686 = X kg/acre`.
- 1 t/ha = 404.686 kg/acre.

## What is NOT yet here

- **AEZ polygons** — vectorization of Natural Resources Management Centre maps is a dedicated workstream. The `aez/` directory currently contains narrative notes only.
- **Chemical dosages** for non-rice crops — require explicit agronomist sign-off before publication.
- **Images** — dedicated capture rounds are scheduled per doc 06.
- **Sinhala / Tamil parity** — most records have English only; local names are included only when directly sourced.
