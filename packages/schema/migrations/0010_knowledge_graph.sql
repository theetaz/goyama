-- 0010_knowledge_graph.sql
-- Turns the corpus from "rows of facts" into a knowledge graph:
--
-- 1. Every canonical record carries an `authority_level` so farmer-facing
--    surfaces can render "DOA-validated" differently from "Tamil Nadu
--    practice, not locally validated yet". This is the honesty gate that
--    lets us ingest cross-regional signal without pretending it's law.
--
-- 2. `cultivation_plan` + its children (activity / pest_risk / economics)
--    model the DOA per-crop-per-AEZ-per-season calendar as a single
--    versionable aggregate. One row per source document; the timeline
--    is a child table keyed on week_idx.
--
-- 3. `knowledge_source` + `knowledge_chunk` capture unstructured signal
--    (video transcripts, article paragraphs, agronomist field notes)
--    with pgvector embeddings so retrieval can mix structured joins and
--    semantic similarity. `entity_refs` is the bridge to the canonical
--    rows above.

BEGIN;

-- ─── authority ─────────────────────────────────────────────────────────────
CREATE TYPE authority_level AS ENUM (
    'doa_official',         -- Sri Lanka DOA, NRMC, DAPH, research institutes
    'peer_reviewed',        -- journal article / university monograph
    'regional_authority',   -- Tamil Nadu Agri Univ, ICAR, FAO, CGIAR
    'practitioner_report',  -- YouTube agronomist, farmer blog, field notes
    'inferred_by_analogy',  -- cross-applied from another region; ALWAYS labelled
    'agent_synthesis'       -- Claude's own multi-source synthesis; triage only
);

ALTER TABLE crop     ADD COLUMN IF NOT EXISTS authority authority_level NOT NULL DEFAULT 'doa_official';
ALTER TABLE disease  ADD COLUMN IF NOT EXISTS authority authority_level NOT NULL DEFAULT 'doa_official';
ALTER TABLE pest     ADD COLUMN IF NOT EXISTS authority authority_level NOT NULL DEFAULT 'doa_official';
ALTER TABLE remedy   ADD COLUMN IF NOT EXISTS authority authority_level NOT NULL DEFAULT 'doa_official';

CREATE INDEX IF NOT EXISTS crop_authority_idx    ON crop    (authority);
CREATE INDEX IF NOT EXISTS disease_authority_idx ON disease (authority);
CREATE INDEX IF NOT EXISTS pest_authority_idx    ON pest    (authority);
CREATE INDEX IF NOT EXISTS remedy_authority_idx  ON remedy  (authority);

-- ─── cultivation plan aggregate ────────────────────────────────────────────
CREATE TYPE activity_type AS ENUM (
    'land_prep', 'basal_fertilizer', 'seed_sowing', 'transplanting',
    'herbicide_pre', 'herbicide_post',
    'top_dressing', 'irrigation', 'weed_control',
    'pest_monitoring', 'pollination_support',
    'seed_harvest', 'harvest', 'post_harvest'
);

CREATE TYPE weather_expectation AS ENUM ('dry', 'mixed', 'rainy');

CREATE TYPE risk_level AS ENUM ('low', 'moderate', 'high');

CREATE TABLE cultivation_plan (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                  text NOT NULL,
    version               integer NOT NULL,
    status                record_status NOT NULL DEFAULT 'draft',
    authority             authority_level NOT NULL DEFAULT 'doa_official',
    crop_slug             text NOT NULL,
    variety_slug          text,                 -- nullable: plan may apply to all varieties
    season                season NOT NULL,
    aez_codes             text[] NOT NULL,      -- plan applies to this set of AEZs
    title                 jsonb NOT NULL DEFAULT '{}',   -- {en, si, ta}
    summary               jsonb NOT NULL DEFAULT '{}',
    start_month           integer CHECK (start_month BETWEEN 1 AND 12),
    duration_weeks        integer,
    expected_yield_kg_per_acre_min real,
    expected_yield_kg_per_acre_max real,
    source_document_url   text,
    source_document_title text,
    field_provenance      jsonb NOT NULL DEFAULT '{}',
    reviewed_by           text,
    reviewed_at           timestamptz,
    review_notes          text,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (slug, version)
);
CREATE INDEX cultivation_plan_crop_season_idx ON cultivation_plan (crop_slug, season);
CREATE INDEX cultivation_plan_aez_idx ON cultivation_plan USING gin (aez_codes);
CREATE INDEX cultivation_plan_published_idx ON cultivation_plan (crop_slug) WHERE status = 'published';

CREATE TABLE cultivation_activity (
    plan_slug     text NOT NULL,
    plan_version  integer NOT NULL,
    week_idx      integer NOT NULL,           -- 1..duration_weeks
    order_in_week integer NOT NULL DEFAULT 0,
    activity      activity_type NOT NULL,
    dap_min       integer,
    dap_max       integer,
    title         jsonb NOT NULL DEFAULT '{}',
    body          jsonb NOT NULL DEFAULT '{}',
    inputs        jsonb NOT NULL DEFAULT '[]',    -- [{type, name, amount, unit, per_unit_area}]
    weather_hint  weather_expectation,
    media_slugs   text[],
    PRIMARY KEY (plan_slug, plan_version, week_idx, order_in_week, activity),
    FOREIGN KEY (plan_slug, plan_version)
        REFERENCES cultivation_plan (slug, version) ON DELETE CASCADE
);
CREATE INDEX cultivation_activity_week_idx ON cultivation_activity (plan_slug, plan_version, week_idx);

CREATE TABLE cultivation_pest_risk (
    plan_slug     text NOT NULL,
    plan_version  integer NOT NULL,
    week_idx      integer NOT NULL,
    disease_slug  text,
    pest_slug     text,
    risk          risk_level NOT NULL,
    recommended_remedy_slugs text[],
    notes         jsonb NOT NULL DEFAULT '{}',
    CHECK (disease_slug IS NOT NULL OR pest_slug IS NOT NULL),
    -- Treat NULL slugs as empty strings for the PK so Postgres's default
    -- "NULLs are distinct" behaviour doesn't let the same row repeat.
    PRIMARY KEY (plan_slug, plan_version, week_idx,
                 COALESCE(disease_slug, ''), COALESCE(pest_slug, '')),
    FOREIGN KEY (plan_slug, plan_version)
        REFERENCES cultivation_plan (slug, version) ON DELETE CASCADE
);
CREATE INDEX cultivation_pest_risk_week_idx ON cultivation_pest_risk (plan_slug, plan_version, week_idx);

CREATE TABLE cultivation_economics (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_slug       text NOT NULL,
    plan_version    integer NOT NULL,
    reference_year  integer NOT NULL,
    unit_area       text NOT NULL DEFAULT 'acre',
    currency        text NOT NULL DEFAULT 'LKR',
    cost_lines      jsonb NOT NULL DEFAULT '[]',     -- [{category, label_i18n, amount, notes}]
    total_cost_without_family_labour numeric(12,2),
    total_cost_with_family_labour    numeric(12,2),
    yield_kg         real,
    unit_price       numeric(12,2),
    gross_revenue    numeric(12,2),
    net_revenue_without_family_labour numeric(12,2),
    net_revenue_with_family_labour    numeric(12,2),
    field_provenance jsonb NOT NULL DEFAULT '{}',
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (plan_slug, plan_version, reference_year, unit_area),
    FOREIGN KEY (plan_slug, plan_version)
        REFERENCES cultivation_plan (slug, version) ON DELETE CASCADE
);

-- ─── knowledge graph: unstructured sources + chunks ────────────────────────
CREATE TYPE knowledge_medium AS ENUM (
    'web_article', 'pdf', 'video_transcript', 'audio_transcript',
    'social_post', 'field_note', 'research_paper', 'book', 'image_caption'
);

CREATE TABLE knowledge_source (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug           text NOT NULL UNIQUE,     -- e.g. 'yt-tnau-onion-thrips-2023'
    display_name   text NOT NULL,            -- 'TNAU — Onion Thrips IPM (YouTube, 2023)'
    medium         knowledge_medium NOT NULL,
    publisher      text,                     -- 'Tamil Nadu Agricultural University'
    authority      authority_level NOT NULL DEFAULT 'regional_authority',
    url            text,                     -- canonical upstream URL
    language       text,                     -- 'en' | 'si' | 'ta' | 'hi' | ...
    licence        text,                     -- upstream's licence; NULL if unclear
    published_at   date,
    fetched_at     timestamptz NOT NULL DEFAULT now(),
    notes          text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX knowledge_source_authority_idx ON knowledge_source (authority);
CREATE INDEX knowledge_source_medium_idx ON knowledge_source (medium);

CREATE TABLE knowledge_chunk (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug              text NOT NULL UNIQUE,
    source_slug       text NOT NULL REFERENCES knowledge_source(slug) ON UPDATE CASCADE,
    chunk_idx         integer NOT NULL DEFAULT 0,    -- order within the source
    language          text NOT NULL DEFAULT 'en',
    title             text,
    body              text NOT NULL,
    body_translated   jsonb NOT NULL DEFAULT '{}',   -- {en, si, ta} when we have translations
    entity_refs       jsonb NOT NULL DEFAULT '[]',   -- [{type: 'crop', slug: 'red-onion'}, ...]
    authority         authority_level NOT NULL,      -- may be stricter than source's default
    applies_to_aez_codes text[],                     -- scope; empty = general
    applies_to_countries text[] NOT NULL DEFAULT '{LK}',
    topic_tags        text[],                        -- ['pest_control', 'thrips', 'ipm']
    confidence        real CHECK (confidence BETWEEN 0 AND 1),
    quote             text,                          -- verbatim quote from source when body is paraphrased
    content_embedding vector(1024),                  -- populated by a separate embedding worker
    status            record_status NOT NULL DEFAULT 'draft',
    reviewed_by       text,
    reviewed_at       timestamptz,
    review_notes      text,
    field_provenance  jsonb NOT NULL DEFAULT '{}',
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX knowledge_chunk_source_idx ON knowledge_chunk (source_slug, chunk_idx);
CREATE INDEX knowledge_chunk_entity_refs_gin ON knowledge_chunk USING gin (entity_refs);
CREATE INDEX knowledge_chunk_topic_tags_gin ON knowledge_chunk USING gin (topic_tags);
CREATE INDEX knowledge_chunk_aez_gin ON knowledge_chunk USING gin (applies_to_aez_codes);
CREATE INDEX knowledge_chunk_published_idx ON knowledge_chunk (source_slug)
    WHERE status = 'published';
CREATE INDEX knowledge_chunk_embedding_idx ON knowledge_chunk
    USING hnsw (content_embedding vector_cosine_ops);

COMMIT;
