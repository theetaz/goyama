# Goyama corpus release — v0.0.1-drafts

**Pre-release drafts bundle — NOT FOR END-USER CONSUMPTION.**

## What this release is

A machine-readable snapshot of the entire `corpus/seed/` authoring tree as of the export timestamp. Every record in this bundle is `status="draft"` and has **not yet been reviewed by an agronomist**. The purpose of this bundle is:

1. To make the drafts easy for agronomist reviewers to consume in one place (JSONL, sortable, diffable).
2. To act as a **fixed input** for the downstream review workflow — so reviewers and researchers compare against a stable snapshot rather than a moving target.
3. To serve as a reproducibility checkpoint for the first public open-corpus release `v0.1.0` (which will contain only reviewed, published records).

## Contents

| Bundle | Records | Description |
|---|---|---|
| `crops.jsonl` | 98 | Crops across field, vegetable, fruit, spice, plantation, medicinal, ornamental categories |
| `crop_varieties.jsonl` | 30 | Rice (modern + heirloom), banana, brinjal, tomato, king coconut, tea clone sets |
| `diseases.jsonl` | 34 | Full Sri Lankan disease coverage including SL-specific Weligama leaf wilt and SLCMV |
| `pests.jsonl` | 28 | Rice, vegetable, fruit, and plantation-crop pests with Sri Lanka-specific IPM |
| `remedies.jsonl` | 11 | Chemical, cultural, biological, and resistant-variety protocols |
| `sources.json` | — | Source register (DOA, Ministry of Agriculture, DEA, etc.) |
| `manifest.json` | — | Machine-readable manifest with sha256 checksums per bundle |

Total: **201 draft records**.

## Licences

- Content (JSONL records): **CC-BY-SA 4.0**
- Geodata (when polygons land): **ODbL v1.0**
- Images we produce: **CC-BY 4.0**
- Pipeline code: **MIT**
- Third-party source quotations preserved as provenance; we redistribute our structured extractions, not the original copyrighted text.

## Review workflow

For each record in the bundle an agronomist should:

1. **Verify** every numeric field against the cited `source_url` + `quote`.
2. **Correct** any errors (especially scientific-name spellings, unit conversions, and chemical PHI/dosage values — these are the hard-gate fields).
3. **Annotate** with `reviewed_by` + `reviewed_at` timestamps in each `field_provenance` entry.
4. **Flip** the record's `status` from `"draft"` to `"published"`.
5. Move the updated record back into `corpus/seed/<type>/<slug>.json`.

When a critical mass is published, run `goyama export v0.1.0` (without `--include-draft`) to produce the first publishable release.

## Reproduce this bundle

```bash
cd pipelines
uv sync --extra test
uv run goyama export v0.0.1-drafts --include-draft
```

The SHA-256 checksums in `manifest.json` are deterministic given the same input seed records (sorted by slug, compact JSON).

## Known limitations

- **No agronomist review** has been performed on any record in this bundle. Chemical dosages, PHI values, and disease-remedy pairings must not be relayed to end users as-is.
- **AEZ polygon geometry** is deferred — narrative notes exist, polygons pending NRMC vectorisation.
- **Sinhala / Tamil translations** are partial and machine-drafted where present; human review required.
- **Images / symptom photos** are out of scope for this bundle — field-capture programme to follow.

## Contact

See [CONTRIBUTING.md](../../../CONTRIBUTING.md) at the repo root for how to submit corrections.
