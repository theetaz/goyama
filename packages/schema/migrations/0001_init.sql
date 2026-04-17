-- 0001_init.sql
-- Initial schema for the CropDoc canonical store.
-- Targets Postgres 16 with extensions: postgis, pgvector, age (Apache AGE).

BEGIN;

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pgcrypto;    -- gen_random_uuid
CREATE EXTENSION IF NOT EXISTS pg_trgm;     -- fuzzy alias matching
CREATE EXTENSION IF NOT EXISTS vector;      -- pgvector
-- Apache AGE is loaded per session via LOAD 'age'; graph creation is in 0002.

-- ─── enums ─────────────────────────────────────────────────────────────────
CREATE TYPE record_status AS ENUM ('draft', 'in_review', 'published', 'deprecated', 'rejected');
CREATE TYPE season         AS ENUM ('maha', 'yala', 'perennial', 'year_round');
CREATE TYPE zone_group     AS ENUM ('wet', 'intermediate', 'dry');
CREATE TYPE elevation_class AS ENUM ('low_country', 'mid_country', 'up_country');
CREATE TYPE suitability_class AS ENUM ('S1', 'S2', 'S3', 'N');
CREATE TYPE crop_category  AS ENUM ('field_crop', 'vegetable', 'fruit', 'spice', 'plantation', 'forage', 'ornamental', 'medicinal');
CREATE TYPE life_cycle     AS ENUM ('annual', 'biennial', 'perennial');
CREATE TYPE effort_level   AS ENUM ('low', 'medium', 'high');
CREATE TYPE water_req      AS ENUM ('low', 'medium', 'high');
CREATE TYPE translation_status AS ENUM ('missing', 'machine_draft', 'human_reviewed');
CREATE TYPE remedy_type    AS ENUM ('cultural', 'biological', 'chemical', 'resistant_variety', 'mechanical', 'integrated');
CREATE TYPE media_type     AS ENUM ('image', 'video', 'pdf', 'audio', 'transcript');
CREATE TYPE media_hosting  AS ENUM ('own', 'external_link');

-- ─── sources register ──────────────────────────────────────────────────────
CREATE TABLE source (
    id             text PRIMARY KEY,                 -- e.g. 'doa.gov.lk'
    display_name   text NOT NULL,
    homepage_url   text NOT NULL,
    licence        text,
    robots_checked_at timestamptz,
    rate_limit_per_sec real NOT NULL DEFAULT 1.0,
    notes          text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

-- ─── raw zone index (actual blobs live in object storage) ──────────────────
CREATE TABLE raw_artifact (
    sha256         bytea PRIMARY KEY,
    source_id      text NOT NULL REFERENCES source(id),
    url            text NOT NULL,
    fetched_at     timestamptz NOT NULL,
    content_type   text,
    content_length bigint,
    storage_url    text NOT NULL,                    -- s3://... or file://...
    http_status    integer,
    headers        jsonb,
    lang_detected  text,
    CONSTRAINT raw_artifact_url_fetched UNIQUE (source_id, url, fetched_at)
);
CREATE INDEX raw_artifact_source_fetched_idx ON raw_artifact (source_id, fetched_at DESC);

-- ─── core entities ─────────────────────────────────────────────────────────
-- All canonical tables are append-only by (slug, version). The latest published
-- row per slug is the current truth; history is preserved.

CREATE TABLE crop (
    id                     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                   text NOT NULL,
    version                integer NOT NULL,
    status                 record_status NOT NULL DEFAULT 'draft',
    scientific_name        text NOT NULL,
    family                 text,
    category               crop_category NOT NULL,
    life_cycle             life_cycle NOT NULL,
    growth_habit           text,
    default_season         season,
    duration_days_min      integer,
    duration_days_max      integer,
    elevation_m_min        real,
    elevation_m_max        real,
    rainfall_mm_min        real,
    rainfall_mm_max        real,
    temperature_c_min      real,
    temperature_c_max      real,
    soil_ph_min            real,
    soil_ph_max            real,
    preferred_soil_groups  text[],
    water_requirement      water_req,
    effort_level           effort_level,
    spacing_within_row_cm  real,
    spacing_between_row_cm real,
    seed_rate_kg_per_acre  real,
    yield_kg_per_acre_min  real,
    yield_kg_per_acre_max  real,
    companions             text[],
    rotates_with           text[],
    attrs                  jsonb NOT NULL DEFAULT '{}',
    content_embedding      vector(1024),
    field_provenance       jsonb NOT NULL DEFAULT '{}',
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    UNIQUE (slug, version)
);
CREATE INDEX crop_latest_published_idx ON crop (slug) WHERE status = 'published';
CREATE INDEX crop_category_idx         ON crop (category);
CREATE INDEX crop_scientific_trgm_idx  ON crop USING gin (scientific_name gin_trgm_ops);
CREATE INDEX crop_embedding_idx        ON crop USING hnsw (content_embedding vector_cosine_ops);

CREATE TABLE crop_variety (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                text NOT NULL,
    version             integer NOT NULL,
    status              record_status NOT NULL DEFAULT 'draft',
    crop_slug           text NOT NULL,
    name                text NOT NULL,
    released_by         text,
    release_year        integer,
    duration_days       integer,
    yield_kg_per_acre_min real,
    yield_kg_per_acre_max real,
    recommended_aez_codes text[],
    traits              jsonb NOT NULL DEFAULT '{}',
    field_provenance    jsonb NOT NULL DEFAULT '{}',
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (slug, version)
);
CREATE INDEX crop_variety_crop_idx ON crop_variety (crop_slug);

CREATE TABLE aez (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code              text NOT NULL,
    version           integer NOT NULL,
    status            record_status NOT NULL DEFAULT 'draft',
    zone_group        zone_group NOT NULL,
    elevation_class   elevation_class NOT NULL,
    avg_rainfall_mm   real,
    avg_temperature_c real,
    dominant_soil_groups text[],
    geom              geography(MultiPolygon, 4326) NOT NULL,
    field_provenance  jsonb NOT NULL DEFAULT '{}',
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (code, version)
);
CREATE INDEX aez_geom_gist ON aez USING gist (geom);
CREATE INDEX aez_zone_group_idx ON aez (zone_group);

CREATE TABLE disease (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                 text NOT NULL,
    version              integer NOT NULL,
    status               record_status NOT NULL DEFAULT 'draft',
    scientific_name      text,
    causal_organism      text NOT NULL,
    causal_species       text,
    affected_crop_slugs  text[] NOT NULL,
    affected_parts       text[],
    transmission         text[],
    favored_conditions   jsonb NOT NULL DEFAULT '{}',
    severity             text,
    confused_with        text[],
    attrs                jsonb NOT NULL DEFAULT '{}',
    content_embedding    vector(1024),
    field_provenance     jsonb NOT NULL DEFAULT '{}',
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    UNIQUE (slug, version)
);
CREATE INDEX disease_affected_crops_idx ON disease USING gin (affected_crop_slugs);
CREATE INDEX disease_embedding_idx      ON disease USING hnsw (content_embedding vector_cosine_ops);

CREATE TABLE pest (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                 text NOT NULL,
    version              integer NOT NULL,
    status               record_status NOT NULL DEFAULT 'draft',
    scientific_name      text,
    kingdom              text NOT NULL,
    affected_crop_slugs  text[] NOT NULL,
    life_stages          text[],
    feeding_type         text[],
    favored_conditions   jsonb NOT NULL DEFAULT '{}',
    attrs                jsonb NOT NULL DEFAULT '{}',
    content_embedding    vector(1024),
    field_provenance     jsonb NOT NULL DEFAULT '{}',
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    UNIQUE (slug, version)
);
CREATE INDEX pest_affected_crops_idx ON pest USING gin (affected_crop_slugs);

CREATE TABLE symptom (
    id                        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                      text NOT NULL,
    version                   integer NOT NULL,
    status                    record_status NOT NULL DEFAULT 'draft',
    disease_slug              text,
    pest_slug                 text,
    stage                     text NOT NULL,
    affected_part             text NOT NULL,
    image_slugs               text[],
    confused_with_symptom_slugs text[],
    field_provenance          jsonb NOT NULL DEFAULT '{}',
    created_at                timestamptz NOT NULL DEFAULT now(),
    updated_at                timestamptz NOT NULL DEFAULT now(),
    UNIQUE (slug, version),
    CHECK (disease_slug IS NOT NULL OR pest_slug IS NOT NULL)
);

CREATE TABLE remedy (
    id                        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                      text NOT NULL,
    version                   integer NOT NULL,
    status                    record_status NOT NULL DEFAULT 'draft',
    type                      remedy_type NOT NULL,
    target_disease_slugs      text[],
    target_pest_slugs         text[],
    applicable_crop_slugs     text[],
    active_ingredient         text,
    concentration             text,
    formulation               text,
    doa_registration_no       text,
    dosage                    text,
    dosage_per_acre_min       real,
    dosage_per_acre_max       real,
    application_method        text,
    frequency                 text,
    pre_harvest_interval_days integer,
    re_entry_interval_hours   integer,
    who_hazard_class          text,
    effectiveness             text,
    cost_tier                 text,
    organic_compatible        boolean,
    attrs                     jsonb NOT NULL DEFAULT '{}',
    field_provenance          jsonb NOT NULL DEFAULT '{}',
    created_at                timestamptz NOT NULL DEFAULT now(),
    updated_at                timestamptz NOT NULL DEFAULT now(),
    UNIQUE (slug, version),
    CHECK (target_disease_slugs IS NOT NULL OR target_pest_slugs IS NOT NULL),
    CHECK (type <> 'chemical' OR active_ingredient IS NOT NULL),
    CHECK (type <> 'chemical' OR pre_harvest_interval_days IS NOT NULL)
);
CREATE INDEX remedy_diseases_idx ON remedy USING gin (target_disease_slugs);

CREATE TABLE cultivation_step (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                  text NOT NULL,
    version               integer NOT NULL,
    status                record_status NOT NULL DEFAULT 'draft',
    crop_slug             text NOT NULL,
    variety_slug          text,
    aez_code              text,
    season                season,
    stage                 text NOT NULL,
    order_idx             integer NOT NULL,
    dap_min               integer,
    dap_max               integer,
    inputs                jsonb NOT NULL DEFAULT '[]',
    media_slugs           text[],
    field_provenance      jsonb NOT NULL DEFAULT '{}',
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (slug, version)
);
CREATE INDEX cultivation_step_crop_season_idx ON cultivation_step (crop_slug, season, order_idx);

CREATE TABLE media (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            text NOT NULL,
    version         integer NOT NULL,
    status          record_status NOT NULL DEFAULT 'draft',
    type            media_type NOT NULL,
    hosting         media_hosting NOT NULL,
    url             text,
    external_url    text,
    credit          text,
    licence         text NOT NULL,
    captured_at     timestamptz,
    captured_at_geom geography(Point, 4326),
    related         jsonb NOT NULL DEFAULT '{}',
    language        text,
    duration_seconds real,
    tags            text[],
    field_provenance jsonb NOT NULL DEFAULT '{}',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (slug, version),
    CHECK ((hosting = 'own'          AND url IS NOT NULL)
        OR (hosting = 'external_link' AND external_url IS NOT NULL))
);
CREATE INDEX media_related_idx ON media USING gin (related);

-- ─── suitability edge ──────────────────────────────────────────────────────
CREATE TABLE crop_suitability (
    crop_slug        text NOT NULL,
    aez_code         text NOT NULL,
    suitability      suitability_class NOT NULL,
    model_version    text,
    notes            text,
    reviewed_by      text,
    reviewed_at      timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (crop_slug, aez_code)
);

-- ─── translations ──────────────────────────────────────────────────────────
CREATE TABLE translation (
    entity_type  text   NOT NULL,   -- 'crop' | 'disease' | ...
    entity_slug  text   NOT NULL,
    field        text   NOT NULL,   -- e.g. 'names', 'description', 'title'
    locale       text   NOT NULL,   -- 'en' | 'si' | 'ta'
    value        text   NOT NULL,
    status       translation_status NOT NULL DEFAULT 'machine_draft',
    reviewed_by  text,
    reviewed_at  timestamptz,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (entity_type, entity_slug, field, locale)
);
CREATE INDEX translation_entity_idx ON translation (entity_type, entity_slug);
CREATE INDEX translation_search_trgm_idx ON translation USING gin (value gin_trgm_ops);

-- ─── alias resolution ──────────────────────────────────────────────────────
CREATE TABLE entity_alias (
    id            bigserial PRIMARY KEY,
    entity_type   text NOT NULL,
    entity_slug   text NOT NULL,
    alias         text NOT NULL,
    locale        text,
    source_id     text REFERENCES source(id),
    confidence    real NOT NULL DEFAULT 1.0,
    reviewed      boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (entity_type, entity_slug, alias, locale)
);
CREATE INDEX entity_alias_trgm_idx ON entity_alias USING gin (alias gin_trgm_ops);

COMMIT;
