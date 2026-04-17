from __future__ import annotations

import os
from dataclasses import dataclass, field
from pathlib import Path

from dotenv import load_dotenv

load_dotenv()


@dataclass(frozen=True)
class Settings:
    raw_dir: Path = field(default_factory=lambda: Path(os.getenv("GOYAMA_RAW_DIR", "./data/raw")))
    staging_dir: Path = field(default_factory=lambda: Path(os.getenv("GOYAMA_STAGING_DIR", "./data/staging")))
    cache_dir: Path = field(default_factory=lambda: Path(os.getenv("GOYAMA_CACHE_DIR", "./data/cache")))
    user_agent: str = field(default_factory=lambda: os.getenv("GOYAMA_USER_AGENT", "GoyamaBot/0.0.1 (+https://goyama.lk/bot)"))
    http_timeout_sec: float = field(default_factory=lambda: float(os.getenv("GOYAMA_HTTP_TIMEOUT_SEC", "30")))
    llm_provider: str = field(default_factory=lambda: os.getenv("LLM_PROVIDER", "none"))
    llm_model: str = field(default_factory=lambda: os.getenv("LLM_MODEL", "claude-sonnet-4-6"))
    anthropic_api_key: str | None = field(default_factory=lambda: os.getenv("ANTHROPIC_API_KEY"))
    database_url: str | None = field(default_factory=lambda: os.getenv("DATABASE_URL"))
    log_level: str = field(default_factory=lambda: os.getenv("LOG_LEVEL", "INFO"))

    def ensure_dirs(self) -> None:
        for d in (self.raw_dir, self.staging_dir, self.cache_dir):
            d.mkdir(parents=True, exist_ok=True)


settings = Settings()
