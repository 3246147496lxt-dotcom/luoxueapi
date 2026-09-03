-- Candidate-only PostgreSQL 18 contract.
--
-- The caller must set candidate.review_run_id on this database session. This
-- script is only valid on an isolated restored database copy. It snapshots one
-- awaiting-review run and the durable origins reached through that run before
-- temporarily releasing the source's active-run slot.

BEGIN;

DO $validation$
DECLARE
    review_run_id BIGINT := NULLIF(
        current_setting('candidate.review_run_id', TRUE),
        ''
    )::BIGINT;
BEGIN
    IF review_run_id IS NULL OR review_run_id <= 0 THEN
        RAISE EXCEPTION 'candidate.review_run_id must be a positive bigint';
    END IF;
    IF to_regnamespace('candidate_validation') IS NOT NULL THEN
        RAISE EXCEPTION 'candidate_validation schema already exists';
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM skill_import_runs AS review_run
        WHERE review_run.id = review_run_id
          AND review_run.status = 'awaiting_review'
          AND review_run.lease_owner IS NULL
          AND review_run.lease_expires_at IS NULL
          AND review_run.next_attempt_at IS NULL
    ) THEN
        RAISE EXCEPTION 'review run is not an idle awaiting_review run';
    END IF;
END
$validation$;

CREATE SCHEMA candidate_validation;

CREATE TABLE candidate_validation.review_run_snapshot AS
SELECT preserved_run.*
FROM skill_import_runs AS preserved_run
WHERE preserved_run.id = current_setting('candidate.review_run_id')::BIGINT;

CREATE TABLE candidate_validation.review_origin_snapshot AS
SELECT preserved_origin.*
FROM skill_origins AS preserved_origin
WHERE EXISTS (
    SELECT 1
    FROM skill_import_run_items AS review_item
    WHERE review_item.run_id = current_setting('candidate.review_run_id')::BIGINT
      AND review_item.source_id = preserved_origin.source_id
      AND review_item.namespace = preserved_origin.namespace
      AND review_item.external_id = preserved_origin.external_id
      AND review_item.skill_id = preserved_origin.skill_id
);

DO $validation$
DECLARE
    affected INTEGER;
BEGIN
    IF (SELECT COUNT(*) FROM candidate_validation.review_run_snapshot) <> 1 THEN
        RAISE EXCEPTION 'expected exactly one saved review run';
    END IF;

    UPDATE skill_import_runs AS current_run
    SET status = 'cancelled'
    FROM candidate_validation.review_run_snapshot AS saved_run
    WHERE current_run.id = saved_run.id
      AND current_run.status = 'awaiting_review'
      AND saved_run.status = 'awaiting_review'
      AND (to_jsonb(current_run) - 'status') =
          (to_jsonb(saved_run) - 'status');

    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> 1 THEN
        RAISE EXCEPTION 'expected one review run suspension, updated %', affected;
    END IF;
END
$validation$;

COMMIT;
