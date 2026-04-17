from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path

import pytest

from core.fetch import FetchResult
from sources.doa.crawler import DoaCrawler
from core.crawler import CrawlConfig

FIXTURES = Path(__file__).parent / "fixtures" / "doa"


def _result(url: str, body: bytes) -> FetchResult:
    return FetchResult(
        url=url,
        status=200,
        content=body,
        headers={"content-type": "text/html; charset=utf-8"},
        fetched_at=datetime.now(UTC),
        sha256="x" * 64,
        content_type="text/html; charset=utf-8",
    )


@pytest.mark.asyncio
async def test_discover_filters_offsite_links() -> None:
    html = b"""
    <html><body>
      <a href="/hordi/brinjal">brinjal</a>
      <a href="https://www.doa.gov.lk/rrdi/rice">rice</a>
      <a href="https://example.com/offsite">offsite</a>
      <a href="#anchor">anchor</a>
      <a href="mailto:a@b">mail</a>
    </body></html>
    """
    cfg = CrawlConfig(source_id="doa", seed_urls=["https://www.doa.gov.lk/"])
    crawler = DoaCrawler(cfg, fetcher=None, store=None)  # type: ignore[arg-type]
    found = list(await _collect(crawler.discover(_result("https://www.doa.gov.lk/", html))))
    assert "https://www.doa.gov.lk/hordi/brinjal" in found
    assert "https://www.doa.gov.lk/rrdi/rice" in found
    assert not any("example.com" in u for u in found)
    assert not any(u.endswith("#anchor") for u in found)


@pytest.mark.asyncio
async def test_discover_handles_brinjal_fixture() -> None:
    html = (FIXTURES / "brinjal.html").read_bytes()
    cfg = CrawlConfig(source_id="doa", seed_urls=["https://www.doa.gov.lk/"])
    crawler = DoaCrawler(cfg, fetcher=None, store=None)  # type: ignore[arg-type]
    links = list(await _collect(crawler.discover(_result("https://www.doa.gov.lk/hordi/brinjal", html))))
    # fixture has no outbound links
    assert links == []


async def _collect(maybe_iter):
    if hasattr(maybe_iter, "__await__"):
        return await maybe_iter  # type: ignore[misc]
    return maybe_iter
