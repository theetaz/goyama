#!/usr/bin/env python3
"""Validate every JSON record under corpus/seed/ against its canonical schema.

Standalone script — depends only on `jsonschema`, so CI can install it without
pulling in the full pipelines toolchain. Mirrors the validation logic in
`pipelines/core/extractor.py` so a record that passes here passes there.

Exits non-zero on the first failure batch it finds, after reporting every
error to stdout.
"""
from __future__ import annotations

import json
import sys
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
    "cultivation_steps": "cultivation-step.json",
    "diseases": "disease.json",
    "media": "media.json",
    "pests": "pest.json",
    "remedies": "remedy.json",
}


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


def main() -> int:
    if not SEED_DIR.exists():
        print(f"seed directory not found: {SEED_DIR}", file=sys.stderr)
        return 1

    registry = build_registry()
    total = 0
    failures: list[tuple[Path, list[str]]] = []

    for subdir, schema_name in DIR_TO_SCHEMA.items():
        target = SEED_DIR / subdir
        if not target.exists():
            continue
        schema_path = SCHEMA_DIR / schema_name
        if not schema_path.exists():
            print(f"schema missing for {subdir}: {schema_path}", file=sys.stderr)
            return 1
        for record_path in sorted(target.glob("*.json")):
            total += 1
            try:
                record = json.loads(record_path.read_text())
            except json.JSONDecodeError as exc:
                failures.append((record_path, [f"invalid JSON: {exc}"]))
                continue
            errors = validate(schema_path, record, registry)
            if errors:
                failures.append((record_path, errors))

    if failures:
        for path, errors in failures:
            rel = path.relative_to(ROOT)
            print(f"\n✗ {rel}")
            for err in errors:
                print(f"    {err}")
        print(f"\n{len(failures)} invalid / {total} total records")
        return 1

    print(f"✓ {total} records valid")
    return 0


if __name__ == "__main__":
    sys.exit(main())
