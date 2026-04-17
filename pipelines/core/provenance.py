from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime


@dataclass
class Provenance:
    source_id: str
    source_url: str
    fetched_at: datetime
    quote: str | None = None
    extractor_version: str | None = None
    model_id: str | None = None
    prompt_version: str | None = None
    confidence: float | None = None
    reviewed_by: str | None = None
    reviewed_at: datetime | None = None
    review_notes: str | None = None

    def to_dict(self) -> dict:
        out: dict = {
            "source_id": self.source_id,
            "source_url": self.source_url,
            "fetched_at": self.fetched_at.isoformat(),
        }
        for key in ("quote", "extractor_version", "model_id", "prompt_version", "confidence", "reviewed_by", "review_notes"):
            val = getattr(self, key)
            if val is not None:
                out[key] = val
        if self.reviewed_at is not None:
            out["reviewed_at"] = self.reviewed_at.isoformat()
        return out
