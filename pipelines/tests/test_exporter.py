from __future__ import annotations

import json
from pathlib import Path

from core.exporter import export_corpus


def _write(p: Path, obj: dict) -> None:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(obj))


def test_export_includes_drafts_when_flag_set(tmp_path: Path) -> None:
    seed = tmp_path / "seed"
    releases = tmp_path / "releases"

    _write(seed / "crops" / "rice.json", {"slug": "rice", "status": "draft", "names": {"en": "Rice"}})
    _write(seed / "crops" / "wheat.json", {"slug": "wheat", "status": "published", "names": {"en": "Wheat"}})

    manifest = export_corpus(seed, releases, "v0.0.1-test", include_draft=True)

    crops_file = releases / "v0.0.1-test" / "crops.jsonl"
    lines = crops_file.read_text().strip().splitlines()
    assert len(lines) == 2
    assert manifest["totals"]["records_in_bundle"] == 2
    assert manifest["totals"]["published_in_seed"] == 1
    assert manifest["totals"]["drafts_in_seed"] == 1
    assert manifest["include_draft"] is True
    assert len(manifest["bundles"][0]["sha256"]) == 64


def test_export_publishes_only_published_by_default(tmp_path: Path) -> None:
    seed = tmp_path / "seed"
    releases = tmp_path / "releases"

    _write(seed / "crops" / "rice.json", {"slug": "rice", "status": "draft", "names": {"en": "Rice"}})
    _write(seed / "crops" / "wheat.json", {"slug": "wheat", "status": "published", "names": {"en": "Wheat"}})

    manifest = export_corpus(seed, releases, "v0.1.0-test", include_draft=False)

    crops_file = releases / "v0.1.0-test" / "crops.jsonl"
    records = [json.loads(ln) for ln in crops_file.read_text().strip().splitlines()]
    assert len(records) == 1
    assert records[0]["slug"] == "wheat"
    assert manifest["include_draft"] is False


def test_manifest_records_licence_and_timestamp(tmp_path: Path) -> None:
    seed = tmp_path / "seed"
    _write(seed / "crops" / "rice.json", {"slug": "rice", "status": "published", "names": {"en": "Rice"}})

    manifest = export_corpus(seed, tmp_path / "releases", "v0.0.1-test", include_draft=False)

    assert manifest["licence"]["content"] == "CC-BY-SA-4.0"
    assert manifest["licence"]["geodata"] == "ODbL-1.0"
    assert manifest["licence"]["code"] == "MIT"
    assert "generated_at" in manifest
