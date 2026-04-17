from __future__ import annotations

from abc import ABC, abstractmethod
from collections.abc import AsyncIterator, Iterable
from dataclasses import dataclass

from .fetch import FetchResult, PoliteFetcher
from .log import get_logger
from .storage import RawArtifact, RawStore

log = get_logger(__name__)


@dataclass
class CrawlConfig:
    source_id: str
    seed_urls: list[str]
    rate_per_sec: float = 1.0
    max_pages: int | None = None
    allow_patterns: list[str] | None = None    # regex list; if set, only matching URLs are enqueued
    deny_patterns: list[str] | None = None


class Crawler(ABC):
    """Base class for source crawlers.

    Subclasses implement `discover(page)` to yield further URLs from a fetched page
    and may override `should_follow(url)` for domain-specific filters.
    """

    config: CrawlConfig

    def __init__(self, config: CrawlConfig, fetcher: PoliteFetcher, store: RawStore) -> None:
        self.config = config
        self._fetcher = fetcher
        self._store = store
        self._seen: set[str] = set()

    @abstractmethod
    async def discover(self, page: FetchResult) -> Iterable[str]:
        """Yield further URLs to enqueue given a fetched page."""

    def should_follow(self, url: str) -> bool:
        import re
        if self.config.deny_patterns and any(re.search(p, url) for p in self.config.deny_patterns):
            return False
        if self.config.allow_patterns and not any(re.search(p, url) for p in self.config.allow_patterns):
            return False
        return True

    async def crawl(self, *, dry_run: bool = False) -> AsyncIterator[RawArtifact]:
        # Configure rate for this source's hosts, discovered from seeds.
        for seed in self.config.seed_urls:
            host = seed.split("/", 3)[2] if "://" in seed else seed
            self._fetcher.set_rate(host, self.config.rate_per_sec)

        queue: list[str] = [u for u in self.config.seed_urls if u not in self._seen]
        fetched_count = 0

        while queue:
            url = queue.pop(0)
            if url in self._seen:
                continue
            self._seen.add(url)
            if not self.should_follow(url):
                continue
            if self.config.max_pages is not None and fetched_count >= self.config.max_pages:
                log.info("max_pages reached", max=self.config.max_pages)
                break

            if dry_run:
                log.info("dry-run would fetch", url=url)
                continue

            result = await self._fetcher.get(url)
            if result is None:
                continue
            if result.status >= 400:
                log.warning("non-2xx", url=url, status=result.status)
                continue

            artifact = self._store.put(self.config.source_id, result)
            fetched_count += 1
            yield artifact

            for next_url in await _iter_to_list(self.discover(result)):
                if next_url not in self._seen:
                    queue.append(next_url)


async def _iter_to_list(i: Iterable[str]) -> list[str]:
    if hasattr(i, "__aiter__"):
        return [x async for x in i]  # type: ignore[misc]
    return list(i)
