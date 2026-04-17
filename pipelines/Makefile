.PHONY: install fmt lint test crawl-doa extract-doa clean

install:
	uv sync

fmt:
	uv run ruff format .

lint:
	uv run ruff check .

test:
	uv run pytest -q

crawl-doa:
	uv run goyama crawl doa --limit 10

extract-doa:
	uv run goyama extract doa --limit 10

clean:
	rm -rf data/raw data/staging data/cache
