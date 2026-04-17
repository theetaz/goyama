from __future__ import annotations

import asyncio
from urllib.parse import urljoin, urlsplit
from urllib.robotparser import RobotFileParser

import httpx

from .log import get_logger

log = get_logger(__name__)


class RobotsCache:
    """Fetches and caches robots.txt per host. Denies by default on fetch failure for safety."""

    def __init__(self, user_agent: str, client: httpx.AsyncClient) -> None:
        self._ua = user_agent
        self._client = client
        self._cache: dict[str, RobotFileParser] = {}
        self._locks: dict[str, asyncio.Lock] = {}

    def _host_key(self, url: str) -> tuple[str, str]:
        parts = urlsplit(url)
        return f"{parts.scheme}://{parts.netloc}", parts.netloc

    async def _load(self, base: str) -> RobotFileParser:
        rp = RobotFileParser()
        robots_url = urljoin(base + "/", "robots.txt")
        try:
            resp = await self._client.get(robots_url, timeout=10.0)
            if resp.status_code == 200:
                rp.parse(resp.text.splitlines())
            elif resp.status_code in (401, 403):
                # Treat as "disallow all" per RFC draft on robots exclusion.
                rp.parse(["User-agent: *", "Disallow: /"])
            else:
                # 404 or other: allow everything (common convention).
                rp.parse(["User-agent: *", "Allow: /"])
        except Exception as e:  # noqa: BLE001
            log.warning("robots.txt fetch failed, denying by default", base=base, error=str(e))
            rp.parse(["User-agent: *", "Disallow: /"])
        return rp

    async def allowed(self, url: str) -> bool:
        base, host = self._host_key(url)
        if host not in self._cache:
            if host not in self._locks:
                self._locks[host] = asyncio.Lock()
            async with self._locks[host]:
                if host not in self._cache:
                    self._cache[host] = await self._load(base)
        return self._cache[host].can_fetch(self._ua, url)
