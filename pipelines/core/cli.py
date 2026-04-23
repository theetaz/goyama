from __future__ import annotations

import asyncio
import json
from pathlib import Path

import typer
from rich.console import Console
from rich.table import Table

from . import registry
from .config import settings
from .exporter import export_corpus
from .extractor import validate_record
from .fetch import PoliteFetcher
from .log import configure_logging, get_logger
from .storage import RawStore, StagingStore

app = typer.Typer(help="Goyama data pipelines", no_args_is_help=True)
sources_app = typer.Typer(help="Source registry operations")
app.add_typer(sources_app, name="sources")
market_prices_app = typer.Typer(help="HARTI bulletin normalisation (market-prices source)")
app.add_typer(market_prices_app, name="market-prices")
console = Console()
log = get_logger(__name__)


@app.callback()
def _bootstrap() -> None:
    configure_logging()
    settings.ensure_dirs()


@sources_app.command("list")
def sources_list() -> None:
    """List configured sources."""
    table = Table(title="Sources")
    table.add_column("id"); table.add_column("seeds"); table.add_column("rate/s")
    for sid in registry.list_sources():
        cfg = registry.load_source_config(sid)
        table.add_row(sid, str(len(cfg.get("seed_urls", []))), str(cfg.get("rate_per_sec", 1.0)))
    console.print(table)


@app.command()
def crawl(
    source_id: str,
    dry_run: bool = typer.Option(False, "--dry-run", help="List URLs without fetching."),
    limit: int | None = typer.Option(None, help="Override max_pages."),
) -> None:
    """Run a source's crawler, writing raw artifacts under data/raw/."""
    async def _run() -> None:
        store = RawStore()
        async with PoliteFetcher() as f:
            crawler = registry.make_crawler(source_id, f, store)
            if limit is not None:
                crawler.config.max_pages = limit
            count = 0
            async for _ in crawler.crawl(dry_run=dry_run):
                count += 1
            console.print(f"[green]crawl complete[/]: source={source_id} artifacts={count}")
    asyncio.run(_run())


@app.command()
def extract(
    source_id: str,
    limit: int = typer.Option(10, help="Max raw artifacts to process this run."),
    validate: bool = typer.Option(True, help="Validate against JSON Schema."),
) -> None:
    """Run a source's extractor over staged raw artifacts, writing draft records to staging."""
    async def _run() -> None:
        raw_dir = settings.raw_dir / source_id
        if not raw_dir.exists():
            console.print(f"[yellow]no raw artifacts for[/] {source_id} at {raw_dir}")
            raise typer.Exit(1)

        staging = StagingStore()
        extractor = registry.make_extractor(source_id)

        bins = sorted(raw_dir.rglob("*.bin"))[:limit]
        produced = 0
        for body_path in bins:
            meta_path = body_path.with_suffix(".json")
            if not meta_path.exists():
                continue
            meta = json.loads(meta_path.read_text())
            from datetime import datetime
            fetched_at = datetime.fromisoformat(meta["fetched_at"])
            body = body_path.read_bytes()
            results = await extractor.extract(body, meta["url"], fetched_at)
            for res in results:
                if validate:
                    schema_file = f"{res.entity_type}.json"
                    errors = validate_record(schema_file, res.record)
                    if errors:
                        log.warning("schema validation failed", slug=res.slug, errors=errors[:5])
                        continue
                staging.put(res.entity_type, res.slug, res.version, res.record)
                produced += 1
        console.print(f"[green]extract complete[/]: source={source_id} drafts={produced}")
    asyncio.run(_run())


@app.command()
def validate(entity_type: str, path: Path) -> None:
    """Validate a local JSON record against its schema."""
    record = json.loads(path.read_text())
    errors = validate_record(f"{entity_type}.json", record)
    if errors:
        console.print(f"[red]{len(errors)} error(s)[/]")
        for e in errors:
            console.print(f"  - {e}")
        raise typer.Exit(1)
    console.print("[green]ok[/]")


@market_prices_app.command("normalize")
def market_prices_normalize(
    pdf_path: Path = typer.Argument(..., help="Downloaded HARTI bulletin PDF (absolute or relative)."),
    observed_on: str = typer.Option(
        ...,
        "--observed-on",
        help="Bulletin date (YYYY-MM-DD) — used as the observation date on every row.",
    ),
    out: Path = typer.Option(
        None,
        "--out",
        help="CSV output path. Defaults to data/staging/market_prices/<observed_on>.csv.",
    ),
    source_url: str = typer.Option(
        "",
        "--source-url",
        help="Override source_url column (falls back to the bulletin's file URL if known).",
    ),
) -> None:
    """Parse a HARTI daily bulletin PDF into a CSV consumable by ``marketload``."""
    from datetime import date as _date

    # Lazy-import so the CLI stays usable when the market_prices package isn't
    # installed (e.g. during DOA-only operations). Raises a readable error if
    # the package is missing rather than a cryptic import trace.
    try:
        from sources.market_prices.normalizer import HartiBulletinNormalizer
    except ImportError as e:  # noqa: BLE001
        console.print(f"[red]market_prices package not importable:[/] {e}")
        raise typer.Exit(1) from e

    if not pdf_path.exists():
        console.print(f"[red]no such file:[/] {pdf_path}")
        raise typer.Exit(1)

    observed = _date.fromisoformat(observed_on)
    target = out or (settings.staging_dir / "market_prices" / f"{observed_on}.csv")
    target.parent.mkdir(parents=True, exist_ok=True)

    normaliser = HartiBulletinNormalizer(source_url=source_url)
    try:
        rows = normaliser.normalize(
            pdf_path.read_bytes(),
            observed_on=observed,
            source_url=source_url or None,
        )
    except ValueError as e:
        console.print(f"[red]normalize failed:[/] {e}")
        raise typer.Exit(2) from e

    target.write_text(HartiBulletinNormalizer.to_csv(rows))
    console.print(
        f"[green]wrote[/] {target} [dim]({len(rows)} rows, "
        f"{len({r.market_code for r in rows})} markets)[/]"
    )


@app.command()
def export(
    version: str = typer.Argument(..., help="Release version tag, e.g. v0.0.1-drafts or v0.1.0"),
    seed_root: Path = typer.Option(Path("../corpus/seed"), "--seed", help="Seed directory"),
    release_root: Path = typer.Option(Path("../corpus/releases"), "--releases", help="Release output root"),
    include_draft: bool = typer.Option(
        False, "--include-draft", help="Include status=draft records (for pre-release drafts bundles)."
    ),
) -> None:
    """Export corpus seed records into a tagged release bundle."""
    manifest = export_corpus(
        seed_root=seed_root.resolve(),
        release_root=release_root.resolve(),
        version=version,
        include_draft=include_draft,
    )
    table = Table(title=f"Corpus release {version}")
    table.add_column("bundle"); table.add_column("records"); table.add_column("sha256 (first 12)")
    for b in manifest["bundles"]:
        table.add_row(b["entity_type"], str(b["records"]), b["sha256"][:12])
    console.print(table)
    console.print(
        f"[green]records in bundle:[/] {manifest['totals']['records_in_bundle']}   "
        f"[dim](published in seed: {manifest['totals']['published_in_seed']}, "
        f"drafts: {manifest['totals']['drafts_in_seed']})[/]"
    )
