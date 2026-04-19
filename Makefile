# Goyama — repo-wide Makefile.
# Targets are thin wrappers over docker compose, psql, and the scripts/ helpers.

DB_HOST ?= localhost
DB_PORT ?= 54320
DB_URL  ?= postgres://goyama:goyama@$(DB_HOST):$(DB_PORT)/goyama?sslmode=disable
COMPOSE = docker compose

.PHONY: db-up db-down db-reset db-migrate db-seed db-psql db-logs db-load-geo-fixtures db-load-market-prices-fixtures help

help:
	@echo "Goyama make targets:"
	@echo "  db-up                              Start the local Postgres (detached)"
	@echo "  db-down                            Stop the local Postgres (keeps data)"
	@echo "  db-reset                           Stop and wipe the database volume"
	@echo "  db-migrate                         Apply SQL migrations from packages/schema/migrations/"
	@echo "  db-seed                            Load crops from corpus/seed/ into the local DB"
	@echo "  db-load-geo-fixtures               Load dev geo fixtures (districts + AEZ) for /v1/geo/lookup"
	@echo "  db-load-market-prices-fixtures     Load sample Dambulla DEC prices for /v1/market-prices"
	@echo "  db-psql                            Open a psql shell against the local DB"
	@echo "  db-logs                            Tail the DB container logs"

db-up:
	$(COMPOSE) up -d --build db
	@echo "waiting for DB to be healthy..."
	@until [ "$$($(COMPOSE) ps --format '{{.Health}}' db)" = "healthy" ]; do \
		$(COMPOSE) ps db >/dev/null; \
	done
	@echo "DB is ready on localhost:$(DB_PORT) (user=goyama db=goyama)"

db-down:
	$(COMPOSE) down

db-reset:
	$(COMPOSE) down -v

db-migrate:
	@for f in packages/schema/migrations/*.sql; do \
		case "$$f" in *0002_graph.sql) \
			echo "skip $$f (requires Apache AGE; not in the local image)"; \
			continue;; \
		esac; \
		echo "applying $$f"; \
		PGPASSWORD=goyama psql -h $(DB_HOST) -p $(DB_PORT) -U goyama -d goyama -v ON_ERROR_STOP=1 -f $$f; \
	done

db-seed:
	cd services/api && DATABASE_URL='$(DB_URL)' go run ./cmd/seed

# Load the simplified dev geo fixtures into Postgres so /v1/geo/lookup
# returns sane responses for points in Colombo, Kandy, Anuradhapura.
# Replace with real Sri Lanka layers (see pipelines/geo/README.md) before
# publishing anything geo-derived.
db-load-geo-fixtures:
	cd services/api && DATABASE_URL='$(DB_URL)' go run ./cmd/geoload \
		--layer=districts --file=../../pipelines/geo/fixtures/districts.geojson
	cd services/api && DATABASE_URL='$(DB_URL)' go run ./cmd/geoload \
		--layer=aez       --file=../../pipelines/geo/fixtures/aez.geojson

# Load the sample Dambulla DEC daily price CSV. Real source wiring lives
# in pipelines/sources/market_prices/ (see its README for the runbook).
db-load-market-prices-fixtures:
	cd services/api && DATABASE_URL='$(DB_URL)' go run ./cmd/marketload \
		--file=../../pipelines/sources/market_prices/fixtures/dambulla-2026-04-15.csv

db-psql:
	PGPASSWORD=goyama psql -h $(DB_HOST) -p $(DB_PORT) -U goyama -d goyama

db-logs:
	$(COMPOSE) logs -f db
