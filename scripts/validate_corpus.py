#!/usr/bin/env python3
"""Validate every JSON record under corpus/seed/ against its canonical schema,
then apply a second layer of corpus-quality gates that catch problems JSON
Schema can't express (cross-field invariants, authority-appropriate content,
trilingual coverage).

Standalone script — depends only on `jsonschema` + `referencing`, so CI can
install it without pulling in the full pipelines toolchain. Mirrors the
validation logic in `pipelines/core/extractor.py` so a record that passes
here passes there.

Exit codes:
  0  — every record valid against schema + quality gates
  1  — schema validation failures (listed per record)
  2  — quality-gate failures (listed per record); schema was clean
"""
from __future__ import annotations

import json
import sys
from collections import Counter
from pathlib import Path

import jsonschema
from referencing import Registry, Resource
from referencing.jsonschema import DRAFT202012

ROOT = Path(__file__).resolve().parents[1]
SCHEMA_DIR = ROOT / "packages" / "schema" / "schemas"
SEED_DIR = ROOT / "corpus" / "seed"

# Seed-directory -> schema-file. Only directories that hold canonical entity
# records are validated; free-form references (markdown, link lists) are not.
DIR_TO_SCHEMA: dict[str, str] = {
    "aez": "aez.json",
    "crops": "crop.json",
    "crop_varieties": "crop-variety.json",
    "cultivation_plans": "cultivation-plan.json",
    "cultivation_steps": "cultivation-step.json",
    "diseases": "disease.json",
    "knowledge_chunks": "knowledge-chunk.json",
    "knowledge_sources": "knowledge-source.json",
    "media": "media.json",
    "pests": "pest.json",
    "remedies": "remedy.json",
}

# Authority bands that are allowed to drive farmer-facing recommendations.
# Lower-authority bands render as advisory in the UI and get stricter
# quality gates (caveat required, confidence ceiling, etc.).
AUTHORITATIVE = {"doa_official", "peer_reviewed"}


def build_registry() -> Registry:
    registry: Registry = Registry()
    for path in SCHEMA_DIR.glob("*.json"):
        schema = json.loads(path.read_text())
        resource = Resource(contents=schema, specification=DRAFT202012)
        sid = schema.get("$id")
        if sid:
            registry = registry.with_resource(uri=sid, resource=resource)
        registry = registry.with_resource(uri=path.name, resource=resource)
    return registry


def validate(schema_path: Path, record: dict, registry: Registry) -> list[str]:
    schema = json.loads(schema_path.read_text())
    validator = jsonschema.Draft202012Validator(schema, registry=registry)
    return [f"{list(e.absolute_path)}: {e.message}" for e in validator.iter_errors(record)]


# ─── quality gates ─────────────────────────────────────────────────────────
# Cross-field invariants that the schema can't enforce. Keep each gate to
# one concern so failure messages point at the actual problem.

def quality_gates_plan(record: dict) -> list[str]:
    """Cultivation-plan quality gates."""
    errors: list[str] = []

    activities = record.get("activities") or []
    if len(activities) == 0:
        errors.append("plan has zero activities — at least one is required")

    # Published + in_review plans must have pest_risks, because a farmer-
    # facing plan without a pest calendar is just a fertilizer schedule.
    # Drafts are exempt so ingestion can land incrementally.
    if record.get("status") in ("in_review", "published") and len(record.get("pest_risks") or []) == 0:
        errors.append("published/in_review plan has zero pest_risks")

    # Authority is required; schema already covers this but double-check
    # because record-level defaults can slip through.
    if not record.get("authority"):
        errors.append("authority is required")

    # If source_document_url is set, source_document_title should be too —
    # otherwise the farmer page renders an ugly "[unnamed source]".
    if record.get("source_document_url") and not record.get("source_document_title"):
        errors.append("source_document_url is set but source_document_title is empty")

    return errors


def quality_gates_chunk(record: dict) -> list[str]:
    """Knowledge-chunk quality gates."""
    errors: list[str] = []

    authority = record.get("authority")
    status = record.get("status")

    # Every chunk must reference a source_slug; the handler's source
    # attachment assumes it resolves.
    if not record.get("source_slug"):
        errors.append("source_slug is required")

    body = record.get("body", "")
    if len(body) < 50:
        errors.append(f"body too short ({len(body)} chars) — minimum 50")

    # Published high-authority chunks must carry a verbatim quote so the
    # reviewer / farmer can trace the claim to its source.
    if status == "published" and authority in AUTHORITATIVE:
        quote = record.get("quote", "").strip()
        if not quote:
            errors.append("published doa_official/peer_reviewed chunk must have a verbatim quote")

    # Analogy-mode chunks should document confidence honestly — cap at 0.85
    # so the UI never scores them as "near-certain" against validated
    # sources.
    if authority == "inferred_by_analogy":
        confidence = record.get("confidence")
        if confidence is None:
            errors.append("inferred_by_analogy chunk must declare confidence")
        elif confidence > 0.85:
            errors.append(f"inferred_by_analogy confidence {confidence} exceeds 0.85 ceiling")

        # The caveat is the whole point of labelling something inferred-by-
        # analogy. Schema can't see provenance shape variance, so check
        # here.
        prov = record.get("field_provenance") or {}
        if not prov.get("caveat"):
            errors.append("inferred_by_analogy chunk must include a caveat in field_provenance")

    return errors


def quality_gates_chunk_source_ref(
    chunk: dict, known_sources: set[str]
) -> list[str]:
    """The chunk's source_slug must resolve to a committed source."""
    src = chunk.get("source_slug")
    if src and src not in known_sources:
        return [f"source_slug '{src}' does not resolve to any knowledge_source fixture"]
    return []


QUALITY_GATES = {
    "cultivation_plans": quality_gates_plan,
    "knowledge_chunks": quality_gates_chunk,
}


def trilingual_report(records_by_dir: dict[str, list[tuple[Path, dict]]]) -> None:
    """Print a non-fatal trilingual-coverage summary so CI can advertise
    translation progress without failing the build on partial coverage."""
    for subdir in ("cultivation_plans", "knowledge_chunks"):
        records = records_by_dir.get(subdir, [])
        if not records:
            continue
        title_coverage: Counter = Counter()
        body_coverage: Counter = Counter()
        for _, rec in records:
            field = "title" if subdir == "cultivation_plans" else "body_translated"
            node = rec.get(field) or {}
            for locale in ("en", "si", "ta"):
                if node.get(locale):
                    body_coverage[locale] += 1
            # plans also track summary; roll in if present
            if subdir == "cultivation_plans":
                summary = rec.get("summary") or {}
                for locale in ("en", "si", "ta"):
                    if summary.get(locale):
                        title_coverage[locale] += 1
        total = len(records)
        if subdir == "cultivation_plans":
            print(
                f"\ni18n coverage — {subdir} titles/summaries ({total} plans):"
                f"  en={body_coverage['en']}  si={body_coverage['si']}  ta={body_coverage['ta']}"
            )
        else:
            print(
                f"i18n coverage — {subdir} body_translated ({total} chunks):"
                f"  en={body_coverage['en']}  si={body_coverage['si']}  ta={body_coverage['ta']}"
            )


def main() -> int:
    if not SEED_DIR.exists():
        print(f"seed directory not found: {SEED_DIR}", file=sys.stderr)
        return 1

    registry = build_registry()
    total = 0
    schema_failures: list[tuple[Path, list[str]]] = []
    quality_failures: list[tuple[Path, list[str]]] = []

    # Accumulate records so we can run cross-record gates (like "chunk.source_slug
    # must resolve to a source"). Also used for the trilingual report.
    records_by_dir: dict[str, list[tuple[Path, dict]]] = {}

    for subdir, schema_name in DIR_TO_SCHEMA.items():
        target = SEED_DIR / subdir
        if not target.exists():
            continue
        schema_path = SCHEMA_DIR / schema_name
        if not schema_path.exists():
            print(f"schema missing for {subdir}: {schema_path}", file=sys.stderr)
            return 1
        records_by_dir[subdir] = []
        for record_path in sorted(target.glob("*.json")):
            total += 1
            try:
                record = json.loads(record_path.read_text())
            except json.JSONDecodeError as exc:
                schema_failures.append((record_path, [f"invalid JSON: {exc}"]))
                continue
            errors = validate(schema_path, record, registry)
            if errors:
                schema_failures.append((record_path, errors))
                continue
            records_by_dir[subdir].append((record_path, record))

    # Schema errors are fatal; report and exit before running quality gates
    # on a corrupted set.
    if schema_failures:
        for path, errors in schema_failures:
            rel = path.relative_to(ROOT)
            print(f"\n✗ {rel}")
            for err in errors:
                print(f"    {err}")
        print(f"\n{len(schema_failures)} schema-invalid / {total} total records")
        return 1

    # Quality gates — cross-field invariants and cross-record references.
    known_sources = {rec.get("slug") for _, rec in records_by_dir.get("knowledge_sources", [])}

    for subdir, records in records_by_dir.items():
        gate = QUALITY_GATES.get(subdir)
        if not gate:
            continue
        for path, rec in records:
            errors = gate(rec)
            if subdir == "knowledge_chunks":
                errors = errors + quality_gates_chunk_source_ref(rec, known_sources)
            if errors:
                quality_failures.append((path, errors))

    if quality_failures:
        for path, errors in quality_failures:
            rel = path.relative_to(ROOT)
            print(f"\n⚠ {rel}")
            for err in errors:
                print(f"    {err}")
        print(f"\n{len(quality_failures)} quality-gate failures / {total} total records")
        return 2

    print(f"✓ {total} records valid (schema + quality gates)")
    trilingual_report(records_by_dir)
    return 0


if __name__ == "__main__":
    sys.exit(main())
