-- 0006_remedy_reviewer_columns.sql
-- Record-level review audit for remedy — matching the shape of 0003
-- (cultivation_step), 0004 (disease), 0005 (pest). Remedies are the
-- highest-stakes entity per CLAUDE.md #5 (chemical dosages, PHI, WHO
-- hazard class), so the review gate matters most here.

BEGIN;

ALTER TABLE remedy
    ADD COLUMN IF NOT EXISTS reviewed_by   text,
    ADD COLUMN IF NOT EXISTS reviewed_at   timestamptz,
    ADD COLUMN IF NOT EXISTS review_notes  text;

CREATE INDEX IF NOT EXISTS remedy_status_idx
    ON remedy (status)
    WHERE status IN ('draft', 'in_review');

COMMIT;
