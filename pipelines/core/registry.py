from __future__ import annotations

from importlib import import_module
from pathlib import Path
from typing import Any

import yaml

SOURCES_DIR = Path(__file__).resolve().parent.parent / "sources"


def list_sources() -> list[str]:
    return sorted(p.name for p in SOURCES_DIR.iterdir() if p.is_dir() and (p / "config.yaml").exists())


def load_source_config(source_id: str) -> dict[str, Any]:
    path = SOURCES_DIR / source_id / "config.yaml"
    if not path.exists():
        raise FileNotFoundError(f"no config for source '{source_id}': {path}")
    return yaml.safe_load(path.read_text())


def _import(source_id: str, module: str, attr: str) -> Any:
    mod = import_module(f"sources.{source_id}.{module}")
    return getattr(mod, attr)


def make_crawler(source_id: str, fetcher, store):
    from .crawler import CrawlConfig
    cfg = load_source_config(source_id)
    klass = _import(source_id, "crawler", cfg["crawler_class"])
    crawl_cfg = CrawlConfig(
        source_id=source_id,
        seed_urls=cfg["seed_urls"],
        rate_per_sec=float(cfg.get("rate_per_sec", 1.0)),
        max_pages=cfg.get("max_pages"),
        allow_patterns=cfg.get("allow_patterns"),
        deny_patterns=cfg.get("deny_patterns"),
    )
    return klass(crawl_cfg, fetcher, store)


def make_extractor(source_id: str):
    cfg = load_source_config(source_id)
    klass = _import(source_id, "extractor", cfg["extractor_class"])
    return klass()
