from __future__ import annotations

import json
from abc import ABC, abstractmethod
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import jsonschema
from functools import lru_cache
from referencing import Registry, Resource
from referencing.jsonschema import DRAFT202012

from .config import settings
from .log import get_logger
from .provenance import Provenance

log = get_logger(__name__)


SCHEMA_DIR = Path(__file__).resolve().parents[2] / "packages" / "schema" / "schemas"


def load_schema(name: str) -> dict[str, Any]:
    path = SCHEMA_DIR / name
    if not path.exists():
        raise FileNotFoundError(f"schema not found: {path}")
    return json.loads(path.read_text())


@lru_cache(maxsize=1)
def _registry() -> Registry:
    registry: Registry = Registry()
    for path in SCHEMA_DIR.glob("*.json"):
        schema = json.loads(path.read_text())
        resource = Resource(contents=schema, specification=DRAFT202012)
        sid = schema.get("$id")
        if sid:
            registry = registry.with_resource(uri=sid, resource=resource)
        registry = registry.with_resource(uri=path.name, resource=resource)
    return registry


def validate_record(schema_name: str, record: dict[str, Any]) -> list[str]:
    """Return a list of validation error messages (empty = valid)."""
    schema = load_schema(schema_name)
    validator = jsonschema.Draft202012Validator(schema, registry=_registry())
    return [f"{list(e.absolute_path)}: {e.message}" for e in validator.iter_errors(record)]


@dataclass
class ExtractionResult:
    entity_type: str                          # e.g. "crop", "disease"
    slug: str
    version: int
    record: dict[str, Any]                    # conforms to schema
    provenance_by_field: dict[str, Provenance]


class Extractor(ABC):
    """Produces one or more draft records from a single raw artifact.

    Implementations may use LLM-assisted parsing; they MUST attach provenance to
    every numeric or substantive field and must not invent values without a
    source quote.
    """

    extractor_version: str = "0.0.1"

    @abstractmethod
    async def extract(self, raw_body: bytes, source_url: str, fetched_at: Any) -> list[ExtractionResult]:
        ...


class LLMClient:
    """Thin wrapper around the Anthropic SDK for structured extraction. No-op when LLM_PROVIDER=none."""

    def __init__(self) -> None:
        self._provider = settings.llm_provider
        self._model = settings.llm_model
        self._client: Any = None
        if self._provider == "anthropic":
            if not settings.anthropic_api_key:
                log.warning("ANTHROPIC_API_KEY not set; LLM extraction disabled")
                self._provider = "none"
            else:
                from anthropic import Anthropic  # type: ignore[import-not-found]
                self._client = Anthropic(api_key=settings.anthropic_api_key)

    @property
    def enabled(self) -> bool:
        return self._provider != "none"

    @property
    def model_id(self) -> str:
        return self._model if self.enabled else "none"

    def extract_json(self, *, system: str, user: str, max_tokens: int = 4096) -> dict[str, Any] | None:
        """Call the model with a JSON-only instruction. Returns parsed JSON or None."""
        if not self.enabled:
            return None
        msg = self._client.messages.create(
            model=self._model,
            max_tokens=max_tokens,
            system=system,
            messages=[{"role": "user", "content": user}],
        )
        text_parts = [b.text for b in msg.content if getattr(b, "type", "") == "text"]
        text = "\n".join(text_parts).strip()
        if text.startswith("```"):
            text = text.split("```", 2)[1]
            if text.startswith("json"):
                text = text[4:].lstrip()
        try:
            return json.loads(text)
        except json.JSONDecodeError as e:
            log.error("llm returned non-json", error=str(e), head=text[:400])
            return None
