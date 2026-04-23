from __future__ import annotations

from datetime import UTC, date, datetime
from unittest.mock import patch

import pytest

from core.crawler import CrawlConfig
from core.fetch import FetchResult
from sources.market_prices.crawler import HartiDailyPricesCrawler
from sources.market_prices.normalizer import (
    COMMODITY_TO_CROP_SLUG,
    HartiBulletinNormalizer,
    NormalizedRow,
)


def _html_result(url: str, body: bytes) -> FetchResult:
    return FetchResult(
        url=url,
        status=200,
        content=body,
        headers={"content-type": "text/html; charset=utf-8"},
        fetched_at=datetime.now(UTC),
        sha256="x" * 64,
        content_type="text/html; charset=utf-8",
    )


async def _collect(maybe_iter):
    if hasattr(maybe_iter, "__await__"):
        return await maybe_iter  # type: ignore[misc]
    return maybe_iter


@pytest.mark.asyncio
async def test_discover_enqueues_dated_pdf_and_pagination_but_rejects_off_site() -> None:
    html = b"""
    <html><body>
      <a href="/images/download/marketing/daily_food_commodities_bulletin/english/2026/2026-04-15.pdf">15 Apr 2026</a>
      <a href="/images/download/marketing/daily_food_commodities_bulletin/english/2026/2026-04-16.pdf">16 Apr 2026</a>
      <a href="/daily-price.php?month=2026-03">March 2026</a>
      <a href="/images/download/marketing/daily_food_commodities_bulletin/sinhala/2026/2026-04-15.pdf">Sinhala</a>
      <a href="/images/logo.png">Logo</a>
      <a href="https://www.example.com/some.pdf">Offsite</a>
      <a href="#top">anchor</a>
    </body></html>
    """
    cfg = CrawlConfig(source_id="market_prices", seed_urls=["https://www.harti.gov.lk/daily-price.php"])
    crawler = HartiDailyPricesCrawler(cfg, fetcher=None, store=None)  # type: ignore[arg-type]
    found = list(
        await _collect(
            crawler.discover(_html_result("https://www.harti.gov.lk/daily-price.php", html))
        )
    )

    assert any(u.endswith("/2026-04-15.pdf") and "english" in u for u in found)
    assert any(u.endswith("/2026-04-16.pdf") for u in found)
    assert any("daily-price.php?month=2026-03" in u for u in found)
    assert not any("sinhala" in u.lower() for u in found), "non-English bulletins must be skipped"
    assert not any("example.com" in u for u in found)
    assert not any("logo.png" in u for u in found)


@pytest.mark.asyncio
async def test_discover_ignores_pdf_pages_entirely() -> None:
    cfg = CrawlConfig(source_id="market_prices", seed_urls=["https://www.harti.gov.lk/daily-price.php"])
    crawler = HartiDailyPricesCrawler(cfg, fetcher=None, store=None)  # type: ignore[arg-type]
    # A terminal PDF shouldn't be walked for further links — discover returns [].
    pdf_result = FetchResult(
        url="https://www.harti.gov.lk/images/download/daily/2026-04-15.pdf",
        status=200,
        content=b"%PDF-1.4...",
        headers={"content-type": "application/pdf"},
        fetched_at=datetime.now(UTC),
        sha256="y" * 64,
        content_type="application/pdf",
    )
    out = list(await _collect(crawler.discover(pdf_result)))
    assert out == []


# ---------------------------------------------------------------------------
# Normaliser tests — exercise the regex + market-section logic against a
# hand-authored text block that matches what pdfminer produces for a HARTI
# bulletin. Real-PDF tests live in a separate fixture-backed suite once a
# licence-clean sample PDF is committed.
# ---------------------------------------------------------------------------
_FAKE_HARTI_TEXT = """\
Hector Kobbekaduwa Agrarian Research and Training Institute
Daily Food Commodities Bulletin — 15 April 2026

Dambulla Dedicated Economic Centre
Commodity            Grade     Min     Max     Avg
Brinjals             long      180     240     210
Tomato               A         140     200     170
Carrot               -         200     260     230
Chilli (Green)                 500     600     550

Meegoda Dedicated Economic Centre
Commodity            Grade     Min     Max     Avg
Cabbage              -         130     170     150
Beetroot                       190     250     220
"""


def test_normalize_parses_rows_into_correct_markets() -> None:
    normaliser = HartiBulletinNormalizer(source_url="https://example/harti/2026-04-15.pdf")
    # Patch pdfminer to return the fake layout text so we can test the
    # structural logic without committing a real PDF.
    with patch("sources.market_prices.normalizer.extract_text", return_value=_FAKE_HARTI_TEXT):
        rows = normaliser.normalize(
            b"%PDF-1.4\n<stub>",
            observed_on=date(2026, 4, 15),
        )

    markets = {r.market_code for r in rows}
    assert markets == {"dambulla-dec", "meegoda-dec"}

    dambulla = [r for r in rows if r.market_code == "dambulla-dec"]
    commodities = {r.commodity_label for r in dambulla}
    assert commodities == {"Brinjals", "Tomato", "Carrot", "Chilli (Green)"}

    brinjals = next(r for r in dambulla if r.commodity_label == "Brinjals")
    assert brinjals.grade == "long"
    assert brinjals.price_lkr_per_kg_min == "180"
    assert brinjals.price_lkr_per_kg_max == "240"
    assert brinjals.price_lkr_per_kg_avg == "210"
    assert brinjals.crop_slug == "brinjal"
    assert brinjals.observed_on == "2026-04-15"
    assert brinjals.source_url == "https://example/harti/2026-04-15.pdf"

    carrot = next(r for r in dambulla if r.commodity_label == "Carrot")
    assert carrot.grade == "", "em/en dash grades should persist as empty"
    assert carrot.crop_slug == "carrot"

    # Grade is blank in the source for Chilli (Green) — the normaliser
    # should still emit a row with the price fields resolved.
    chilli = next(r for r in dambulla if r.commodity_label == "Chilli (Green)")
    assert chilli.price_lkr_per_kg_avg == "550"
    assert chilli.crop_slug == "chilli"


def test_normalize_raises_if_pdf_has_no_recognisable_rows() -> None:
    normaliser = HartiBulletinNormalizer()
    unrelated = "Welcome to HARTI\nAbout us\nContact page\n"
    with patch("sources.market_prices.normalizer.extract_text", return_value=unrelated):
        with pytest.raises(ValueError, match="no rows parsed"):
            normaliser.normalize(b"%PDF-1.4", observed_on=date(2026, 4, 15))


def test_to_csv_header_matches_marketload_contract() -> None:
    # If this test fails, the Go marketload contract has drifted from the
    # normaliser's output. Keep them in lockstep.
    row = NormalizedRow(
        observed_on="2026-04-15",
        market_code="dambulla-dec",
        commodity_label="Brinjals",
        grade="long",
        crop_slug="brinjal",
        price_lkr_per_kg_avg="210",
        source_url="https://x/",
    )
    csv_text = HartiBulletinNormalizer.to_csv([row])
    header = csv_text.splitlines()[0].split(",")
    expected = {
        "observed_on",
        "market_code",
        "commodity_label",
        "grade",
        "crop_slug",
        "price_lkr_per_kg_min",
        "price_lkr_per_kg_max",
        "price_lkr_per_kg_avg",
        "unit",
        "currency",
        "sample_size",
        "source_url",
    }
    assert set(header) == expected


def test_commodity_map_covers_fixture_commodities() -> None:
    # The committed Dambulla fixture's commodity labels must all be
    # resolvable — if someone adds a label to the fixture without updating
    # the map, the resulting CSV will have empty crop_slug for that row.
    labels = {
        "brinjals",
        "tomato",
        "carrot",
        "capsicum",
        "bitter gourd",
        "snake gourd",
        "okra",
        "cabbage",
        "cauliflower",
        "beetroot",
        "pumpkin",
        "onion (red)",
        "garlic",
        "chilli (green)",
    }
    unknown = labels - COMMODITY_TO_CROP_SLUG.keys()
    assert not unknown, f"COMMODITY_TO_CROP_SLUG missing: {sorted(unknown)}"
