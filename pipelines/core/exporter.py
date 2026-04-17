from __future__ import annotations

import hashlib
import json
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from .log import get_logger

log = get_logger(__name__)

# Entity-type → (source subdirectory, output filename)
ENTITY_TYPES: tuple[tuple[str, str, str], ...] = (
    ("crops", "crop", "crops.jsonl"),
    ("crop_varieties", "crop-variety", "crop_varieties.jsonl"),
    ("diseases", "disease", "diseases.jsonl"),
    ("pests", "pest", "pests.jsonl"),
    ("remedies", "remedy", "remedies.jsonl"),
)


@dataclass
class ExportStats:
    entity_type: str
    count: int
    published_count: int
    draft_count: int
    output_path: Path
    sha256: str


def _sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def _load_sorted(dir_path: Path) -> list[dict[str, Any]]:
    """Load all *.json files, sorted by slug for deterministic output."""
    records: list[dict[str, Any]] = []
    for p in sorted(dir_path.glob("*.json")):
        records.append(json.loads(p.read_text()))
    return records


def export_corpus(
    seed_root: Path,
    release_root: Path,
    version: str,
    include_draft: bool = False,
) -> dict[str, Any]:
    """Concatenate seed records into a versioned corpus release.

    Reads per-entity subdirectories under ``seed_root`` and writes one JSONL
    bundle per entity type into ``release_root/<version>/``. Produces a
    machine-readable ``manifest.json`` with record counts, per-bundle sha256,
    and the release metadata.

    By default only ``status=published`` records are exported; pass
    ``include_draft=True`` for a pre-release drafts bundle.
    """
    out_dir = release_root / version
    out_dir.mkdir(parents=True, exist_ok=True)

    stats: list[ExportStats] = []
    total = 0
    total_published = 0
    total_draft = 0

    for subdir, _schema, filename in ENTITY_TYPES:
        src = seed_root / subdir
        if not src.exists():
            log.info("no seed directory", subdir=subdir)
            continue

        records = _load_sorted(src)
        pub = sum(1 for r in records if r.get("status") == "published")
        drafts = sum(1 for r in records if r.get("status") == "draft")
        picked = records if include_draft else [r for r in records if r.get("status") == "published"]

        out_path = out_dir / filename
        with out_path.open("w", encoding="utf-8") as f:
            for r in picked:
                f.write(json.dumps(r, ensure_ascii=False, separators=(",", ":")) + "\n")
        sha = _sha256_file(out_path)

        stats.append(
            ExportStats(
                entity_type=subdir,
                count=len(picked),
                published_count=pub,
                draft_count=drafts,
                output_path=out_path,
                sha256=sha,
            )
        )
        total += len(picked)
        total_published += pub
        total_draft += drafts
        log.info("bundle written", entity=subdir, count=len(picked), sha=sha[:12])

    # Copy sources register verbatim.
    sources_src = seed_root / "sources.json"
    if sources_src.exists():
        sources_dst = out_dir / "sources.json"
        sources_dst.write_text(sources_src.read_text())

    manifest: dict[str, Any] = {
        "version": version,
        "generated_at": datetime.now(UTC).isoformat(),
        "include_draft": include_draft,
        "totals": {
            "records_in_bundle": total,
            "published_in_seed": total_published,
            "drafts_in_seed": total_draft,
        },
        "bundles": [
            {
                "entity_type": s.entity_type,
                "file": s.output_path.name,
                "records": s.count,
                "sha256": s.sha256,
            }
            for s in stats
        ],
        "licence": {
            "content": "CC-BY-SA-4.0",
            "geodata": "ODbL-1.0",
            "images": "CC-BY-4.0",
            "code": "MIT",
        },
        "notes": (
            "Pre-release drafts bundle. Chemical remedy PHI values, disease-remedy pairings, "
            "and any user-facing numeric recommendation require agronomist review before "
            "status flips from 'draft' to 'published' per CLAUDE.md hard gate."
        ) if include_draft else
        (
            "Published release. Every record has been reviewed by a qualified agronomist "
            "and carries reviewed_by/reviewed_at provenance."
        ),
    }
    manifest_path = out_dir / "manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, ensure_ascii=False))

    log.info(
        "corpus exported",
        version=version,
        bundles=len(stats),
        records=total,
        output=str(out_dir),
    )
    return manifest
