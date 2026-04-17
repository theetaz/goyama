from __future__ import annotations

from datetime import datetime
from pathlib import Path
from typing import Any

from selectolax.parser import HTMLParser

from core import __version__ as core_version
from core.extractor import Extractor, ExtractionResult, LLMClient
from core.log import get_logger
from core.provenance import Provenance

log = get_logger(__name__)

PROMPT_PATH = Path(__file__).resolve().parents[2] / "prompts" / "crop_extraction_v1.md"
PROMPT_VERSION = "crop_extraction_v1"


def _page_text(html_bytes: bytes) -> tuple[str, str | None]:
    """Return (plain_text, detected_lang_or_none) from an HTML byte string."""
    try:
        tree = HTMLParser(html_bytes.decode(errors="replace"))
    except Exception:  # noqa: BLE001
        return "", None
    for sel in ("script", "style", "nav", "header", "footer", "noscript"):
        for node in tree.css(sel):
            node.decompose()
    text = tree.body.text(separator="\n", strip=True) if tree.body else tree.text(separator="\n", strip=True)
    lang = None
    if tree.tags("html") and tree.tags("html")[0].attributes.get("lang"):
        lang = tree.tags("html")[0].attributes.get("lang")
    return text, lang


def _load_prompt() -> tuple[str, str]:
    """Return (system_prompt, user_template) parsed from the prompt .md file."""
    body = PROMPT_PATH.read_text()
    # Minimal parse: everything after '## System' up to '## User message template' is system;
    # everything after '## User message template' is the user template.
    try:
        _, rest = body.split("## System", 1)
        system, rest = rest.split("## User message template", 1)
    except ValueError as e:
        raise RuntimeError(f"prompt file malformed: {PROMPT_PATH}") from e
    user_template = rest.strip()
    # Strip triple-backtick fences around the user template if present.
    if user_template.startswith("```"):
        user_template = user_template.split("```", 2)[1]
        if user_template.startswith(("text\n", "md\n")):
            user_template = user_template.split("\n", 1)[1]
    return system.strip(), user_template.strip()


class DoaCropExtractor(Extractor):
    extractor_version: str = core_version

    def __init__(self) -> None:
        self._llm = LLMClient()
        self._system, self._user_template = _load_prompt()

    async def extract(
        self,
        raw_body: bytes,
        source_url: str,
        fetched_at: datetime,
    ) -> list[ExtractionResult]:
        text, _lang = _page_text(raw_body)
        if len(text) < 400:
            return []

        if not self._llm.enabled:
            log.info("LLM disabled; producing text-only preview", url=source_url, chars=len(text))
            return []

        user = self._user_template.replace("{url}", source_url).replace(
            "{text}", text[:12000]
        )
        payload = self._llm.extract_json(system=self._system, user=user)
        if payload is None:
            return []
        if payload.get("__skip__"):
            log.info("page skipped", url=source_url, reason=payload.get("reason"))
            return []

        # Attach provenance + finalize required fields.
        prov_by_field: dict[str, Provenance] = {}
        field_prov: dict[str, Any] = payload.get("field_provenance") or {}
        for field, entry in list(field_prov.items()):
            prov = Provenance(
                source_id="doa",
                source_url=source_url,
                fetched_at=fetched_at,
                quote=entry.get("quote") if isinstance(entry, dict) else None,
                extractor_version=self.extractor_version,
                model_id=self._llm.model_id,
                prompt_version=PROMPT_VERSION,
                confidence=entry.get("confidence") if isinstance(entry, dict) else None,
            )
            prov_by_field[field] = prov
            field_prov[field] = prov.to_dict()
        payload["field_provenance"] = field_prov
        payload.setdefault("version", 1)
        payload.setdefault("status", "draft")

        slug = payload.get("slug")
        if not slug:
            log.warning("extractor produced record without slug", url=source_url)
            return []

        return [
            ExtractionResult(
                entity_type="crop",
                slug=slug,
                version=int(payload["version"]),
                record=payload,
                provenance_by_field=prov_by_field,
            )
        ]
