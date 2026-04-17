from __future__ import annotations

import hashlib
from contextlib import asynccontextmanager
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import AsyncIterator

import httpx
from tenacity import AsyncRetrying, retry_if_exception_type, stop_after_attempt, wait_exponential

from .config import settings
from .log import get_logger
from .rate_limit import HostRateLimiter
from .robots import RobotsCache

log = get_logger(__name__)


@dataclass
class FetchResult:
    url: str
    status: int
    content: bytes
    headers: dict[str, str]
    fetched_at: datetime
    sha256: str
    content_type: str | None


class PoliteFetcher:
    """Polite HTTP client: UA identity, robots.txt, per-host rate limiting, retries with backoff."""

    def __init__(
        self,
        rate_limiter: HostRateLimiter | None = None,
        user_agent: str | None = None,
    ) -> None:
        self._ua = user_agent or settings.user_agent
        self._limiter = rate_limiter or HostRateLimiter()
        self._client: httpx.AsyncClient | None = None
        self._robots: RobotsCache | None = None

    async def __aenter__(self) -> "PoliteFetcher":
        self._client = httpx.AsyncClient(
            headers={"User-Agent": self._ua, "Accept-Language": "en, si, ta;q=0.8"},
            timeout=settings.http_timeout_sec,
            follow_redirects=True,
        )
        self._robots = RobotsCache(self._ua, self._client)
        return self

    async def __aexit__(self, *_exc: object) -> None:
        if self._client is not None:
            await self._client.aclose()

    def set_rate(self, host: str, per_sec: float) -> None:
        self._limiter.set_rate(host, per_sec)

    async def get(self, url: str, *, ignore_robots: bool = False) -> FetchResult | None:
        assert self._client is not None, "use PoliteFetcher as async context manager"
        assert self._robots is not None

        if not ignore_robots and not await self._robots.allowed(url):
            log.info("disallowed by robots.txt", url=url)
            return None

        await self._limiter.acquire(url)

        async for attempt in AsyncRetrying(
            retry=retry_if_exception_type((httpx.TransportError, httpx.TimeoutException)),
            stop=stop_after_attempt(3),
            wait=wait_exponential(multiplier=1, min=1, max=16),
            reraise=True,
        ):
            with attempt:
                resp = await self._client.get(url)

        content = resp.content
        sha = hashlib.sha256(content).hexdigest()
        return FetchResult(
            url=str(resp.url),
            status=resp.status_code,
            content=content,
            headers=dict(resp.headers),
            fetched_at=datetime.now(UTC),
            sha256=sha,
            content_type=resp.headers.get("content-type"),
        )


@asynccontextmanager
async def fetcher() -> AsyncIterator[PoliteFetcher]:
    async with PoliteFetcher() as f:
        yield f
