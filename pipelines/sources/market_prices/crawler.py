"""Crawler for HARTI's Daily Food Commodities Bulletin index.

The HARTI ``daily-price.php`` page is an HTML listing that links to PDF
bulletins by date. Each bulletin covers wholesale prices for every DEC
(Dambulla, Meegoda, Welisara, Keppetipola, Narahenpita) plus Pettah retail.
Our job here is scoped tightly:

1. Fetch the index page(s).
2. Yield the bulletin PDF URLs so the Crawler base class stores them under
   ``data/raw/market_prices/``.

The actual CSV shaping happens in :mod:`normalizer` and is driven by the
``goyama market-prices normalize`` CLI command, not by the generic
``goyama extract`` pipeline — market prices are CSV-bound, not JSON-bound.
"""
from __future__ import annotations

import re
from collections.abc import Iterable
from urllib.parse import urljoin, urlsplit

from selectolax.parser import HTMLParser

from core.crawler import Crawler
from core.fetch import FetchResult
from core.log import get_logger

log = get_logger(__name__)

# Dated PDF filenames on HARTI look like ``2026-04-15.pdf`` or variants with
# underscores / explicit language suffixes. Stay flexible but anchored on a
# date so we never enqueue a non-bulletin PDF (annual report, brochure, etc.).
_BULLETIN_DATE_RE = re.compile(r"/(?P<date>\d{4}[-_]?\d{2}[-_]?\d{2})[^/]*\.pdf$", re.IGNORECASE)

# HARTI historically publishes Sinhala / Tamil cuts alongside English. Drop
# them for now; we'll add a language-aware pass once the English parser is
# stable and the review queue is caught up.
_NON_ENGLISH_HINTS = ("sinhala", "tamil", "_si", "_ta", "/si/", "/ta/")


class HartiDailyPricesCrawler(Crawler):
    """Discovers dated bulletin PDFs from HARTI's Daily Food Commodities index."""

    async def discover(self, page: FetchResult) -> Iterable[str]:
        if not (page.content_type or "").startswith("text/html"):
            # Bulletin PDFs are terminal nodes — don't try to walk them.
            return []

        try:
            tree = HTMLParser(page.content.decode(errors="replace"))
        except Exception as e:  # noqa: BLE001
            log.warning("html parse failed", url=page.url, error=str(e))
            return []

        base = page.url
        out: list[str] = []
        for a in tree.css("a[href]"):
            href = (a.attributes.get("href") or "").strip()
            if not href or href.startswith(("#", "mailto:", "tel:", "javascript:")):
                continue

            absolute = urljoin(base, href).split("#", 1)[0]
            if urlsplit(absolute).netloc.lower() not in {"harti.gov.lk", "www.harti.gov.lk"}:
                continue
            low = absolute.lower()

            # Pagination / date-filter variants of the index itself — keep
            # following so we discover bulletins past the first page.
            if "daily-price" in low and low.endswith((".php", "/")) or "daily-price.php?" in low:
                out.append(absolute)
                continue

            if any(h in low for h in _NON_ENGLISH_HINTS):
                continue
            if _BULLETIN_DATE_RE.search(absolute):
                out.append(absolute)

        if out:
            log.info(
                "harti bulletins discovered",
                source=page.url,
                count=len(out),
                sample=out[:3],
            )
        return out
