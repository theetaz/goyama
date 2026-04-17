# pipelines

Crawlers, extractors, and data jobs that populate the Goyama canonical store and the open corpus.

Python 3.12 + [uv](https://docs.astral.sh/uv/) for dependency management.

## Layout

```
pipelines/
├── core/          # shared framework: fetch, storage, crawlers, extractors, provenance
├── prompts/       # versioned LLM extraction prompts
├── sources/
│   └── <source>/  # one directory per source (doa, hordi, rrdi, meteo, youtube, ...)
│       ├── config.yaml
│       ├── crawler.py
│       ├── extractor.py
│       └── README.md
└── tests/
    └── fixtures/  # frozen raw samples per source
```

## Quick start

```bash
cd pipelines
uv sync
uv run goyama sources list
uv run goyama crawl doa --dry-run
uv run goyama extract doa --limit 5
```

## Ground rules (mirror CLAUDE.md)

- Respect `robots.txt`, declare User-Agent, rate-limit politely.
- Raw crawl output goes to `data/raw/` (gitignored).
- Every extracted field records `source_url`, `quote`, `extractor_version`, `model_id`, `confidence`.
- Extractor output lands in staging as draft records; human review gates publication.
- No LLM invention of numbers, dosages, or dates. If no citation, the field is `null`.

## Environment

Copy `.env.example` to `.env` and fill in. Minimally:

- `GOYAMA_RAW_DIR` — where to write raw artifacts (default `./data/raw`).
- `GOYAMA_STAGING_DIR` — draft records (default `./data/staging`).
- `ANTHROPIC_API_KEY` — for LLM-assisted extraction (or configure `LLM_PROVIDER=none` for dry-run).
- `DATABASE_URL` — Postgres connection string (required for `ingest` command).
