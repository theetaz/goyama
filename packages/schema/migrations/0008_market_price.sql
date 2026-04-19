-- 0008_market_price.sql
-- Daily wholesale price observations from Sri Lanka's Dedicated Economic
-- Centres (DECs) and HARTI weekly bulletins. Each row is one
-- (market, commodity, date) observation; the source job inserts a fresh
-- record per day and never edits historical rows.
--
-- The first source wired in is Dambulla DEC (the country's largest
-- wholesale market). Additional DECs (Welisara, Meegoda, Keppetipola,
-- Narahenpita, Thambuttegama, Veyangoda, Embilipitiya) come online by
-- adding rows to `market` and pointing more importer runs at them.

BEGIN;

CREATE TABLE market (
    code         text PRIMARY KEY,            -- e.g. 'dambulla-dec'
    display_name text NOT NULL,
    kind         text NOT NULL,               -- 'dec' | 'harti' | 'pola' | 'wholesale' | 'retail'
    district_code text,                       -- references admin_district.code once geo data is loaded
    operator     text,
    homepage_url text,
    notes        text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE market_price (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    market_code     text NOT NULL REFERENCES market(code) ON UPDATE CASCADE,
    crop_slug       text,                    -- nullable: source commodity may not yet map to a canonical crop
    commodity_label text NOT NULL,           -- raw commodity string from the source (e.g. "Brinjals (Long)")
    grade           text,                    -- e.g. 'A', 'B', 'long', 'small'
    observed_on     date NOT NULL,
    price_lkr_per_kg_min numeric(10,2),
    price_lkr_per_kg_max numeric(10,2),
    price_lkr_per_kg_avg numeric(10,2),
    unit            text NOT NULL DEFAULT 'kg',
    currency        text NOT NULL DEFAULT 'LKR',
    sample_size     integer,
    source_url      text,
    raw_artifact_sha256 bytea,               -- references raw_artifact(sha256) when scraped
    field_provenance jsonb NOT NULL DEFAULT '{}',
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (market_code, commodity_label, grade, observed_on)
);
CREATE INDEX market_price_market_date_idx ON market_price (market_code, observed_on DESC);
CREATE INDEX market_price_crop_date_idx   ON market_price (crop_slug, observed_on DESC) WHERE crop_slug IS NOT NULL;
CREATE INDEX market_price_observed_idx    ON market_price (observed_on DESC);

-- Seed the markets we have an importer for. Adding a row here is enough
-- to unblock importing prices for that market.
INSERT INTO market (code, display_name, kind, district_code, operator, homepage_url, notes)
VALUES
    ('dambulla-dec', 'Dambulla Dedicated Economic Centre', 'dec', 'LK-23',
     'Sri Lanka Department of Agrarian Development',
     'https://www.harti.gov.lk/',
     'Largest wholesale market for vegetables in Sri Lanka. Daily prices via HARTI bulletin.')
ON CONFLICT (code) DO NOTHING;

COMMIT;
