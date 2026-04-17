# Goyama — repo-wide Makefile.
# Targets are thin wrappers over docker compose, psql, and the scripts/ helpers.

DB_HOST ?= localhost
DB_PORT ?= 54320
DB_URL  ?= postgres://goyama:goyama@$(DB_HOST):$(DB_PORT)/goyama?sslmode=disable
COMPOSE = docker compose

.PHONY: db-up db-down db-reset db-migrate db-seed db-psql db-logs help

help:
	@echo "Goyama make targets:"
	@echo "  db-up       Start the local Postgres (detached)"
	@echo "  db-down     Stop the local Postgres (keeps data)"
	@echo "  db-reset    Stop and wipe the database volume"
	@echo "  db-migrate  Apply SQL migrations from packages/schema/migrations/"
	@echo "  db-seed     Load crops from corpus/seed/ into the local DB"
	@echo "  db-psql     Open a psql shell against the local DB"
	@echo "  db-logs     Tail the DB container logs"

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

db-psql:
	PGPASSWORD=goyama psql -h $(DB_HOST) -p $(DB_PORT) -U goyama -d goyama

db-logs:
	$(COMPOSE) logs -f db
