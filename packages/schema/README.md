# @cropdoc/schema

Canonical schemas for the CropDoc knowledge corpus.

This package is the single source of truth for data shapes used across the crawler pipelines, the backend API, the mobile and web apps, and the published open corpus.

## What's here

- `schemas/` — JSON Schema (draft 2020-12) definitions for every canonical entity.
- `migrations/` — SQL migrations (Postgres + PostGIS + pgvector + Apache AGE) matching the schemas.

## Rules

- **JSON Schema is canonical.** SQL migrations must match. When fields change, update the JSON Schema first, then regenerate the migration.
- **All user-visible text fields are i18n keys**, stored in a separate `translations` table. The JSON Schema uses the `I18nString` type from `common.json`.
- **Every published record carries a `provenance` block** (see `provenance.json`).
- **Multilingual authoring** — English-first. Sinhala and Tamil parity is tracked per record.

## Entities (v0)

| Schema | Purpose |
|---|---|
| `common.json` | Shared types: slug, locale, unit, geometry, range |
| `provenance.json` | Source, quote, confidence, reviewer on every published field |
| `crop.json` | Canonical crop entity |
| `crop-variety.json` | Released cultivars (BG / BW / AT / etc.) |
| `aez.json` | Agro-ecological zone polygons + attributes |
| `disease.json` | Plant diseases |
| `pest.json` | Insect / arthropod / vertebrate pests |
| `symptom.json` | Observable signs of a disease or pest |
| `remedy.json` | Cultural / biological / chemical / resistant-variety remedies |
| `cultivation-step.json` | Step in a `(crop × AEZ × season)` cultivation procedure |
| `media.json` | Images, videos, PDFs, audio with licence metadata |

## Regenerating types

```bash
# TypeScript types for apps + API
pnpm --filter @cropdoc/schema build:ts

# Python Pydantic models for pipelines
cd pipelines && uv run datamodel-codegen \
  --input ../packages/schema/schemas \
  --output core/generated_models.py \
  --input-file-type jsonschema
```
