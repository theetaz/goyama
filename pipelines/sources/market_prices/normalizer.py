"""Parse a HARTI Daily Food Commodities Bulletin PDF into canonical CSV rows.

The downstream contract is the CSV shape documented in this source's
``README.md`` — any tool that can produce that shape (this normaliser, a
hand-edited spreadsheet, a future LLM-assisted extractor) can feed the
``marketload`` Go command directly.

The PDF layout is a tabular wholesale-price table, typically one section
per DEC (Dambulla, Meegoda, Welisara, Keppetipola, Narahenpita) and a
retail-prices section for Pettah. We extract text with ``pdfminer.six``
because table-layout grids in HARTI PDFs are honoured by its default text
ordering, then shape rows with regex.

The PDF format has shifted on HARTI in the past (column reordering, header
language, unit changes). Treat the regexes below as best-effort against
the 2024–2026 layout and tune when the format drifts — don't silently
accept partially-parsed rows.
"""
from __future__ import annotations

import io
import re
from collections.abc import Iterable
from dataclasses import asdict, dataclass, field
from datetime import date

from pdfminer.high_level import extract_text

from core.log import get_logger

log = get_logger(__name__)

# ---------------------------------------------------------------------------
# Commodity → canonical crop slug mapping.
#
# The bulletin's labels are preserved verbatim on the CSV's `commodity_label`
# column for provenance. The `crop_slug` column — when resolvable — lets
# downstream joins land on canonical crops without re-normalising on the
# read side.
# ---------------------------------------------------------------------------
COMMODITY_TO_CROP_SLUG: dict[str, str] = {
    # Solanaceae
    "brinjals": "brinjal",
    "brinjal": "brinjal",
    "tomato": "tomato",
    "capsicum": "capsicum",
    "chilli (green)": "chilli",
    "green chilli": "chilli",
    # Cucurbits
    "bitter gourd": "bitter-gourd",
    "snake gourd": "snake-gourd",
    "pumpkin": "pumpkin",
    "cucumber": "cucumber",
    "luffa": "luffa",
    # Alliums
    "onion (red)": "onion",
    "red onion": "onion",
    "onion (big)": "onion",
    "big onion": "onion",
    "garlic": "garlic",
    "leeks": "leeks",
    # Brassicas + roots + leafy
    "cabbage": "cabbage",
    "cauliflower": "cauliflower",
    "carrot": "carrot",
    "beetroot": "beet-root",
    "beet root": "beet-root",
    "radish": "radish",
    "knol khol": "knol-khol",
    "okra": "okra",
    "ladies' fingers": "okra",
    "ladies fingers": "okra",
}

# HARTI bulletin tables render one observation per row, typically:
#
#   <Commodity>  <Grade>  <Low>  <High>  <Avg>
#
# with prices in LKR/kg. Numeric cells are fully-formed decimals (e.g.
# "210.00") or bare integers. Grade is an optional free-text token that may
# be absent; when absent the table shows an em-dash or blank. This regex
# tolerates both and lets the normaliser accept rows with any of the three
# price fields populated.
_ROW_RE = re.compile(
    r"""
    ^\s*
    (?P<label>[A-Za-z][A-Za-z\s\(\)\'\.\-/]+?)   # commodity label (non-greedy)
    (?:\s{2,}|\t)+
    (?:                                         # optional grade cell
        (?P<grade>[A-Za-z][A-Za-z\-]{0,20})     #   a single word-ish token
        | (?P<dash>[-\u2013\u2014])             #   or a dash meaning "no grade"
    )?
    (?:\s{2,}|\t)*
    (?P<min>\d{1,5}(?:\.\d+)?)?
    (?:\s{2,}|\t)+
    (?P<max>\d{1,5}(?:\.\d+)?)?
    (?:\s{2,}|\t)+
    (?P<avg>\d{1,5}(?:\.\d+)?)
    \s*$
    """,
    re.VERBOSE | re.MULTILINE,
)

# Section headers in the bulletin. The order in the PDF is stable at the
# time of writing; we read the headers to tag the `market_code` per row.
_MARKET_HEADERS: tuple[tuple[str, str], ...] = (
    ("dambulla", "dambulla-dec"),
    ("meegoda", "meegoda-dec"),
    ("welisara", "welisara-dec"),
    ("keppetipola", "keppetipola-dec"),
    ("narahenpita", "narahenpita-dec"),
    ("pettah", "pettah-retail"),
)


@dataclass
class NormalizedRow:
    """One row in the CSV that :command:`marketload` reads.

    Field names match the CSV header — :func:`HartiBulletinNormalizer.to_csv`
    writes them in that order, and the loader is tolerant of column
    ordering so field reorderings here don't require loader changes.
    """

    observed_on: str                  # ISO date
    market_code: str                  # e.g. "dambulla-dec"
    commodity_label: str              # raw bulletin label (for provenance)
    grade: str = ""
    crop_slug: str = ""
    price_lkr_per_kg_min: str = ""    # str so empty cells round-trip
    price_lkr_per_kg_max: str = ""
    price_lkr_per_kg_avg: str = ""
    unit: str = "kg"
    currency: str = "LKR"
    sample_size: str = ""
    source_url: str = ""
    field_provenance: dict = field(default_factory=dict)  # reserved for future

    def to_csv_dict(self) -> dict[str, str]:
        d = asdict(self)
        d.pop("field_provenance", None)
        return {k: "" if v is None else str(v) for k, v in d.items()}


class HartiBulletinNormalizer:
    """Produce :class:`NormalizedRow` objects from a HARTI bulletin PDF."""

    def __init__(self, *, source_url: str = "") -> None:
        self._source_url = source_url

    # ------------------------------------------------------------------
    # Main entry point
    # ------------------------------------------------------------------
    def normalize(
        self,
        pdf_bytes: bytes,
        *,
        observed_on: date,
        source_url: str | None = None,
    ) -> list[NormalizedRow]:
        """Parse a bulletin PDF. Raises :class:`ValueError` on empty output."""
        text = extract_text(io.BytesIO(pdf_bytes))
        if not text.strip():
            raise ValueError("pdfminer returned empty text; corrupt or image-only PDF?")

        rows = list(self._iter_rows(text, observed_on, source_url or self._source_url))
        if not rows:
            # Parsing produced nothing — a strong signal that the bulletin
            # layout changed. Refuse to silently emit an empty CSV.
            raise ValueError(
                "no rows parsed from bulletin — HARTI layout may have changed; "
                "dump the PDF text and update regexes in normalizer.py"
            )
        log.info(
            "harti bulletin normalised",
            rows=len(rows),
            observed_on=observed_on.isoformat(),
        )
        return rows

    # ------------------------------------------------------------------
    # Internals
    # ------------------------------------------------------------------
    def _iter_rows(
        self,
        text: str,
        observed_on: date,
        source_url: str,
    ) -> Iterable[NormalizedRow]:
        current_market: str | None = None
        for raw_line in text.splitlines():
            line = raw_line.rstrip()
            if not line:
                continue

            header_hit = self._match_market_header(line)
            if header_hit is not None:
                current_market = header_hit
                continue
            if current_market is None:
                # Skip preamble (title, date line, column legend) until the
                # first market header fires.
                continue

            match = _ROW_RE.match(line)
            if not match:
                continue
            yield self._row_from_match(
                match,
                market_code=current_market,
                observed_on=observed_on,
                source_url=source_url,
            )

    @staticmethod
    def _match_market_header(line: str) -> str | None:
        low = line.lower()
        for needle, code in _MARKET_HEADERS:
            if needle in low:
                return code
        return None

    @staticmethod
    def _row_from_match(
        match: re.Match[str],
        *,
        market_code: str,
        observed_on: date,
        source_url: str,
    ) -> NormalizedRow:
        label = match.group("label").strip()
        # grade is either a literal word token or a dash placeholder; the
        # dash case persists as an empty string so downstream queries can
        # filter on truthiness.
        grade = (match.group("grade") or "").strip()
        crop_slug = COMMODITY_TO_CROP_SLUG.get(label.lower(), "")
        return NormalizedRow(
            observed_on=observed_on.isoformat(),
            market_code=market_code,
            commodity_label=label,
            grade=grade,
            crop_slug=crop_slug,
            price_lkr_per_kg_min=match.group("min") or "",
            price_lkr_per_kg_max=match.group("max") or "",
            price_lkr_per_kg_avg=match.group("avg"),
            source_url=source_url,
        )

    # ------------------------------------------------------------------
    # CSV serialisation
    # ------------------------------------------------------------------
    @staticmethod
    def to_csv(rows: list[NormalizedRow]) -> str:
        import csv

        if not rows:
            return ""
        buf = io.StringIO()
        fieldnames = list(rows[0].to_csv_dict().keys())
        writer = csv.DictWriter(buf, fieldnames=fieldnames)
        writer.writeheader()
        for r in rows:
            writer.writerow(r.to_csv_dict())
        return buf.getvalue()
