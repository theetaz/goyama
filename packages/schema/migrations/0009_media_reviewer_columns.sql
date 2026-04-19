-- 0009_media_reviewer_columns.sql
-- Brings the media table into line with disease/pest/remedy: adds the
-- reviewer audit trio + an expression index on related->>'entity_slug'
-- so the disease-images gallery is a single index hit.

BEGIN;

ALTER TABLE media
    ADD COLUMN IF NOT EXISTS reviewed_by  text,
    ADD COLUMN IF NOT EXISTS reviewed_at  timestamptz,
    ADD COLUMN IF NOT EXISTS review_notes text;

-- The existing GIN index on `related` covers @> containment queries;
-- this btree expression index is for the much hotter "list images for
-- one entity slug" path used by both the admin gallery and the public
-- /v1/diseases/{slug}/images endpoint.
CREATE INDEX IF NOT EXISTS media_related_entity_slug_idx
    ON media ((related->>'entity_type'), (related->>'entity_slug'));

COMMIT;
