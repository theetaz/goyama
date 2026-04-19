-- 0007_admin_geo_layers.sql
-- Adds the administrative-boundary layers required to satisfy the Phase 0
-- exit criterion: a (lat, lng) returns
-- {district, ds_division, aez, soil, elevation_class, rainfall_normal_mm}.
--
-- AEZ polygons already live in `aez` (0001). This migration adds the two
-- administrative layers needed to complete the lookup envelope.

BEGIN;

CREATE TABLE admin_district (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code          text NOT NULL UNIQUE,                  -- e.g. 'LK-1' (ISO 3166-2) or DOA district code
    name_en       text NOT NULL,
    name_si       text,
    name_ta       text,
    province_code text,
    province_name text,
    geom          geography(MultiPolygon, 4326) NOT NULL,
    field_provenance jsonb NOT NULL DEFAULT '{}',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admin_district_geom_gist ON admin_district USING gist (geom);
CREATE INDEX admin_district_province_idx ON admin_district (province_code);

CREATE TABLE admin_ds_division (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code           text NOT NULL UNIQUE,                 -- DSD code (e.g. '11-01')
    district_code  text NOT NULL REFERENCES admin_district(code) ON UPDATE CASCADE ON DELETE RESTRICT,
    name_en        text NOT NULL,
    name_si        text,
    name_ta        text,
    geom           geography(MultiPolygon, 4326) NOT NULL,
    field_provenance jsonb NOT NULL DEFAULT '{}',
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admin_ds_division_geom_gist ON admin_ds_division USING gist (geom);
CREATE INDEX admin_ds_division_district_idx ON admin_ds_division (district_code);

COMMIT;
