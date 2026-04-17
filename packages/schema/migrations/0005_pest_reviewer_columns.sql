-- 0005_pest_reviewer_columns.sql
-- Record-level review audit for pest, identical in shape to the
-- cultivation_step (0003) and disease (0004) migrations. Keeping the
-- columns uniform across canonical tables lets a future generalised
-- review surface query them without table-specific special-casing.

BEGIN;

ALTER TABLE pest
    ADD COLUMN IF NOT EXISTS reviewed_by   text,
    ADD COLUMN IF NOT EXISTS reviewed_at   timestamptz,
    ADD COLUMN IF NOT EXISTS review_notes  text;

CREATE INDEX IF NOT EXISTS pest_status_idx
    ON pest (status)
    WHERE status IN ('draft', 'in_review');

COMMIT;
