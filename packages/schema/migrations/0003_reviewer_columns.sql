-- 0003_reviewer_columns.sql
-- Adds record-level review audit to cultivation_step so the admin review
-- queue can record who promoted a draft and when. Field-level provenance
-- already carries per-field `reviewed_by`; this is the gate at the *record*
-- level, matching the `status` lifecycle (draft → in_review → published).
--
-- Keep this migration narrow: the same treatment will roll out to crop /
-- disease / pest / remedy in a follow-up once the review UX is proven on
-- cultivation_step.

BEGIN;

ALTER TABLE cultivation_step
    ADD COLUMN IF NOT EXISTS reviewed_by   text,
    ADD COLUMN IF NOT EXISTS reviewed_at   timestamptz,
    ADD COLUMN IF NOT EXISTS review_notes  text;

-- Review-queue workloads filter by status across all steps; an index lets the
-- admin list page stay snappy as the queue grows.
CREATE INDEX IF NOT EXISTS cultivation_step_status_idx
    ON cultivation_step (status)
    WHERE status IN ('draft', 'in_review');

COMMIT;
