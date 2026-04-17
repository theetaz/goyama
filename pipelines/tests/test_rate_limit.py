from __future__ import annotations

import asyncio
import time

import pytest

from core.rate_limit import HostRateLimiter


@pytest.mark.asyncio
async def test_rate_limit_enforces_min_interval() -> None:
    limiter = HostRateLimiter(default_rate_per_sec=5.0)  # 200ms between calls
    url = "https://example.com/a"

    start = time.monotonic()
    await limiter.acquire(url)
    await limiter.acquire(url)
    await limiter.acquire(url)
    elapsed = time.monotonic() - start
    # 3 calls at 5/s => at least 2 * 0.2 = 0.4s spacing.
    assert elapsed >= 0.35, elapsed


@pytest.mark.asyncio
async def test_rate_limit_is_per_host() -> None:
    limiter = HostRateLimiter(default_rate_per_sec=2.0)

    start = time.monotonic()
    await asyncio.gather(
        limiter.acquire("https://a.example/"),
        limiter.acquire("https://b.example/"),
    )
    elapsed = time.monotonic() - start
    assert elapsed < 0.1, elapsed
