# Local-dev Postgres for Goyama.
#
# Bundles the three extensions every canonical table depends on:
#   - PostGIS      (geography / geometry on the aez and media tables)
#   - pgvector     (content_embedding columns on crop / disease / pest)
#   - pg_trgm      (fuzzy alias + scientific-name search — ships with Postgres)
#
# Apache AGE (graph projection for 0002_graph.sql) is deliberately *not*
# included yet — we're not running graph queries until the relational layer
# is solid. Swap the base image for apache/age when that work lands.
#
# Base image: imresamu/postgis publishes multi-arch (amd64 + arm64) builds,
# which the upstream postgis/postgis image does not. Matches upstream layout
# and version numbers.
FROM imresamu/postgis:16-3.5

RUN apt-get update \
 && apt-get install -y --no-install-recommends postgresql-16-pgvector \
 && rm -rf /var/lib/apt/lists/*
