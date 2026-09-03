-- Candidate-only PostgreSQL 18 contract.
--
-- The caller must set candidate.review_run_id, candidate.bootstrap_run_a_id,
-- and candidate.bootstrap_run_b_id on this database session. The source of the
-- restored status is deliberately named saved_run: skill_import_runs also has
-- a JSONB column named snapshot, so using snapshot as a relation alias makes
-- references such as snapshot.status ambiguous on PostgreSQL 18.

BEGIN;

DO $validation$
DECLARE
    review_run_id BIGINT := NULLIF(
        current_setting('candidate.review_run_id', TRUE),
        ''
    )::BIGINT;
    bootstrap_run_a_id BIGINT := NULLIF(
        current_setting('candidate.bootstrap_run_a_id', TRUE),
        ''
    )::BIGINT;
    bootstrap_run_b_id BIGINT := NULLIF(
        current_setting('candidate.bootstrap_run_b_id', TRUE),
        ''
    )::BIGINT;
    affected INTEGER;
BEGIN
    IF review_run_id IS NULL OR review_run_id <= 0
       OR bootstrap_run_a_id IS NULL OR bootstrap_run_a_id <= 0
       OR bootstrap_run_b_id IS NULL OR bootstrap_run_b_id <= 0
       OR bootstrap_run_a_id = bootstrap_run_b_id THEN
        RAISE EXCEPTION 'candidate review/bootstrap run identifiers are invalid';
    END IF;
    IF to_regclass('candidate_validation.review_run_snapshot') IS NULL
       OR to_regclass('candidate_validation.review_origin_snapshot') IS NULL THEN
        RAISE EXCEPTION 'candidate review validation snapshot is missing';
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM skill_import_runs AS current_run
        JOIN candidate_validation.review_run_snapshot AS saved_run
          ON saved_run.id = current_run.id
        WHERE current_run.id = review_run_id
          AND current_run.status = 'cancelled'
          AND saved_run.status = 'awaiting_review'
          AND (to_jsonb(current_run) - 'status') =
              (to_jsonb(saved_run) - 'status')
    ) THEN
        RAISE EXCEPTION 'preserved review run changed beyond temporary status';
    END IF;
    IF (
        SELECT COUNT(*)
        FROM skill_import_runs AS bootstrap_run
        WHERE bootstrap_run.id IN (bootstrap_run_a_id, bootstrap_run_b_id)
          AND bootstrap_run.status = 'succeeded'
          AND bootstrap_run.lease_owner IS NULL
          AND bootstrap_run.lease_expires_at IS NULL
          AND bootstrap_run.next_attempt_at IS NULL
    ) <> 2 THEN
        RAISE EXCEPTION 'bootstrap validation runs are not both terminal-success';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM skill_import_runs AS active_run
        WHERE active_run.source_id = (
            SELECT saved_run.source_id
            FROM candidate_validation.review_run_snapshot AS saved_run
        )
          AND active_run.id <> review_run_id
          AND active_run.status IN (
              'queued', 'discovering', 'preparing', 'waiting_retry',
              'ready', 'awaiting_review', 'publishing'
          )
    ) THEN
        RAISE EXCEPTION 'validation source active-run slot is not free';
    END IF;
    IF EXISTS (
        (
            SELECT to_jsonb(saved_origin) - ARRAY[
                'source_revision', 'source_content_sha256', 'last_seen_run_id',
                'last_seen_at', 'last_rank', 'active', 'updated_at'
            ]::TEXT[]
            FROM candidate_validation.review_origin_snapshot AS saved_origin
            EXCEPT
            SELECT to_jsonb(current_origin) - ARRAY[
                'source_revision', 'source_content_sha256', 'last_seen_run_id',
                'last_seen_at', 'last_rank', 'active', 'updated_at'
            ]::TEXT[]
            FROM skill_origins AS current_origin
            WHERE EXISTS (
                SELECT 1
                FROM candidate_validation.review_origin_snapshot AS saved_origin
                WHERE saved_origin.id = current_origin.id
            )
        )
        UNION ALL
        (
            SELECT to_jsonb(current_origin) - ARRAY[
                'source_revision', 'source_content_sha256', 'last_seen_run_id',
                'last_seen_at', 'last_rank', 'active', 'updated_at'
            ]::TEXT[]
            FROM skill_origins AS current_origin
            WHERE EXISTS (
                SELECT 1
                FROM candidate_validation.review_origin_snapshot AS saved_origin
                WHERE saved_origin.id = current_origin.id
            )
            EXCEPT
            SELECT to_jsonb(saved_origin) - ARRAY[
                'source_revision', 'source_content_sha256', 'last_seen_run_id',
                'last_seen_at', 'last_rank', 'active', 'updated_at'
            ]::TEXT[]
            FROM candidate_validation.review_origin_snapshot AS saved_origin
        )
    ) THEN
        RAISE EXCEPTION 'preserved review origins changed during bootstrap validation';
    END IF;

    UPDATE skill_import_runs AS current_run
    SET status = saved_run.status
    FROM candidate_validation.review_run_snapshot AS saved_run
    WHERE current_run.id = saved_run.id
      AND current_run.id = review_run_id
      AND current_run.status = 'cancelled'
      AND saved_run.status = 'awaiting_review';

    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> 1 THEN
        RAISE EXCEPTION 'expected one preserved review restore, updated %', affected;
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM skill_import_runs AS current_run
        JOIN candidate_validation.review_run_snapshot AS saved_run
          ON saved_run.id = current_run.id
        WHERE current_run.id = review_run_id
          AND to_jsonb(current_run) = to_jsonb(saved_run)
    ) THEN
        RAISE EXCEPTION 'preserved review row did not restore exactly';
    END IF;
END
$validation$;

DROP TABLE candidate_validation.review_origin_snapshot;
DROP TABLE candidate_validation.review_run_snapshot;
DROP SCHEMA candidate_validation;

COMMIT;
