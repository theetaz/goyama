# corpus

The Goyama open knowledge corpus.

This directory will host versioned snapshots of the curated canonical data — crops, varieties, diseases, pests, remedies, cultivation steps, agro-ecological zones — as machine-readable files with full provenance.

## Release format

Each tagged release will include:

```
corpus/releases/v0.1.0/
├── manifest.json         # release metadata: version, counts, checksums, licences
├── crops.jsonl
├── crop_varieties.jsonl
├── diseases.jsonl
├── pests.jsonl
├── symptoms.jsonl
├── remedies.jsonl
├── cultivation_steps.jsonl
├── media.jsonl           # references only; we do not redistribute third-party media
├── aez.geojson
└── translations.jsonl
```

Formats:

- **JSONL** — one canonical record per line, conforming to the JSON Schemas in `packages/schema/schemas/`.
- **GeoJSON** — for geospatial layers (AEZ polygons, etc.).
- **Parquet** — also published for large tables once volume justifies (built from JSONL).

## Licences

- Content records: **CC-BY-SA 4.0**
- Geodata: **ODbL**
- Images we produce: **CC-BY 4.0**
- Third-party media: linked out, not redistributed; original licences preserved in the `licence` field.

## Governance

- Release cadence: monthly after `v0.1.0`.
- Schema changes: deprecations are kept for two releases before removal.
- Corrections: open an issue or PR; agronomist-reviewed changes enter the next release.
- See [CHANGELOG.md](./CHANGELOG.md).

## Current status

**`v0.0.0` — skeleton.** No records published yet. The pipelines in `pipelines/` are being scaffolded; first published records will land in `v0.1.0`.
