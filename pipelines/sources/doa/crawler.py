from __future__ import annotations

from collections.abc import Iterable
from urllib.parse import urljoin, urlsplit

from selectolax.parser import HTMLParser

from core.crawler import Crawler
from core.fetch import FetchResult
from core.log import get_logger

log = get_logger(__name__)


class DoaCrawler(Crawler):
    """Crawls doa.gov.lk and its institute subdomains, following internal links."""

    async def discover(self, page: FetchResult) -> Iterable[str]:
        if not (page.content_type or "").startswith("text/html"):
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
            host = urlsplit(absolute).netloc.lower()
            if "doa.gov.lk" not in host:
                continue
            out.append(absolute)
        return out
