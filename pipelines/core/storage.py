from __future__ import annotations

import json
from dataclasses import asdict, dataclass, is_dataclass
from datetime import datetime
from pathlib import Path
from typing import Any

from .config import settings
from .fetch import FetchResult
from .log import get_logger

log = get_logger(__name__)


def _default(obj: Any) -> Any:
    if isinstance(obj, datetime):
        return obj.isoformat()
    if is_dataclass(obj) and not isinstance(obj, type):
        return asdict(obj)
    if isinstance(obj, Path):
        return str(obj)
    if isinstance(obj, bytes):
        return obj.decode("utf-8", "replace")
    raise TypeError(f"Not serializable: {type(obj).__name__}")


@dataclass
class RawArtifact:
    source_id: str
    url: str
    sha256: str
    content_type: str | None
    fetched_at: datetime
    body_path: Path
    meta_path: Path


class RawStore:
    """Append-only store of raw fetched bytes, keyed by sha256.

    Layout:
        data/raw/<source_id>/<sha256[:2]>/<sha256>.bin
        data/raw/<source_id>/<sha256[:2]>/<sha256>.json   (meta)
    """

    def __init__(self, root: Path | None = None) -> None:
        self.root = root or settings.raw_dir
        self.root.mkdir(parents=True, exist_ok=True)

    def _paths(self, source_id: str, sha: str) -> tuple[Path, Path]:
        d = self.root / source_id / sha[:2]
        d.mkdir(parents=True, exist_ok=True)
        return d / f"{sha}.bin", d / f"{sha}.json"

    def put(self, source_id: str, result: FetchResult) -> RawArtifact:
        body, meta = self._paths(source_id, result.sha256)
        if not body.exists():
            body.write_bytes(result.content)
        meta_payload = {
            "source_id": source_id,
            "url": result.url,
            "status": result.status,
            "sha256": result.sha256,
            "content_type": result.content_type,
            "content_length": len(result.content),
            "fetched_at": result.fetched_at.isoformat(),
            "headers": result.headers,
        }
        meta.write_text(json.dumps(meta_payload, ensure_ascii=False, indent=2))
        log.info("raw stored", source=source_id, url=result.url, sha=result.sha256[:12])
        return RawArtifact(
            source_id=source_id,
            url=result.url,
            sha256=result.sha256,
            content_type=result.content_type,
            fetched_at=result.fetched_at,
            body_path=body,
            meta_path=meta,
        )


class StagingStore:
    """Draft records awaiting agronomist review. One JSON file per (entity_type, slug, version)."""

    def __init__(self, root: Path | None = None) -> None:
        self.root = root or settings.staging_dir
        self.root.mkdir(parents=True, exist_ok=True)

    def put(self, entity_type: str, slug: str, version: int, record: dict[str, Any]) -> Path:
        d = self.root / entity_type
        d.mkdir(parents=True, exist_ok=True)
        path = d / f"{slug}.v{version}.json"
        path.write_text(json.dumps(record, ensure_ascii=False, indent=2, default=_default))
        log.info("staged", entity=entity_type, slug=slug, version=version)
        return path
