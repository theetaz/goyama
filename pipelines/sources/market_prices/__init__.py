"""HARTI Daily Food Commodities Bulletin — scraper + CSV normaliser.

The public entry points are:

- :class:`HartiDailyPricesCrawler` — discovers dated PDF bulletins from the
  HARTI ``daily-price.php`` index and downloads them into the raw store.
- :class:`HartiBulletinNormalizer` — parses a downloaded bulletin PDF into
  canonical CSV rows that :command:`marketload` can upsert.
- :data:`COMMODITY_TO_CROP_SLUG` — the bulletin-label to canonical-slug map
  documented in the module README.
"""
from .crawler import HartiDailyPricesCrawler
from .normalizer import HartiBulletinNormalizer, NormalizedRow, COMMODITY_TO_CROP_SLUG

__all__ = [
    "HartiDailyPricesCrawler",
    "HartiBulletinNormalizer",
    "NormalizedRow",
    "COMMODITY_TO_CROP_SLUG",
]
