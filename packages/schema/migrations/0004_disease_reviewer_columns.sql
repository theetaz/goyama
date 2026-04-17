-- 0004_disease_reviewer_columns.sql
-- Adds record-level review audit to disease, mirroring the columns that
-- landed on cultivation_step in 0003. Identical shape so a future
-- "review any canonical entity" generalisation can query across tables
-- uniformly.

BEGIN;

ALTER TABLE disease
    ADD COLUMN IF NOT EXISTS reviewed_by   text,
    ADD COLUMN IF NOT EXISTS reviewed_at   timestamptz,
    ADD COLUMN IF NOT EXISTS review_notes  text;

CREATE INDEX IF NOT EXISTS disease_status_idx
    ON disease (status)
    WHERE status IN ('draft', 'in_review');

COMMIT;
