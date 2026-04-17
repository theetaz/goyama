from __future__ import annotations

import asyncio
import time
from collections import defaultdict
from urllib.parse import urlsplit


class HostRateLimiter:
    """Simple per-host token bucket. Defaults to ≤ 1 req/sec/host."""

    def __init__(self, default_rate_per_sec: float = 1.0) -> None:
        self._default = default_rate_per_sec
        self._rates: dict[str, float] = {}
        self._next_allowed: dict[str, float] = defaultdict(float)
        self._locks: dict[str, asyncio.Lock] = defaultdict(asyncio.Lock)

    def set_rate(self, host: str, per_sec: float) -> None:
        self._rates[host] = per_sec

    def _host(self, url: str) -> str:
        return urlsplit(url).netloc.lower()

    async def acquire(self, url: str) -> None:
        host = self._host(url)
        rate = self._rates.get(host, self._default)
        min_interval = 1.0 / rate if rate > 0 else 0.0
        async with self._locks[host]:
            now = time.monotonic()
            wait = self._next_allowed[host] - now
            if wait > 0:
                await asyncio.sleep(wait)
                now = time.monotonic()
            self._next_allowed[host] = now + min_interval
