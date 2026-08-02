-- 195: restore an authoritative starts_at-anchored 30x24h subscription
-- monthly quota alongside the anchored 7x24h weekly quota.
--
-- Migration 192 deliberately stopped incrementing the legacy monthly counter.
-- Consequently no persisted monthly value can be carried forward safely, even
-- when its old marker happens to match. Rebuild the current monthly half-open
-- period from the durable billing ledger, with unmatched legacy usage_logs as a
-- compatibility fallback. Preserve a weekly counter whose marker is already
-- the exact current anchor; only rebuild weekly rows with a stale/off-anchor
-- marker so an already-authoritative DB counter is never replaced by a
-- best-effort projection.
--
-- Keep all period arithmetic in seconds. PostgreSQL treats INTERVAL 'N days'
-- as calendar days for timestamptz values, which is not a fixed N*24h duration
-- in DST-observing database time zones.
--
-- Deployment compatibility:
--   * Current unified billing updates user_subscriptions and inserts its
--     billing_usage_entries receipt in the same transaction, in that order.
--     This table lock waits for an in-flight transaction to finish and blocks
--     a new subscription UPDATE until this migration commits.
--   * Drain old/degraded writers before applying this migration. A legacy
--     writer that commits a subscription UPDATE before writing its usage_log
--     has no atomic receipt and cannot be made race-free by this lock alone.
--   * A receipt is the authoritative event and timestamp. Where its matching
--     usage_log exists, actual_cost is the exact amount added to subscription
--     counters. New writers persist that amount in subscription_amount in the
--     same transaction. A historical receipt with neither value is not safely
--     reconstructable: abort instead of presenting a false zero or gross-cost
--     estimate. subscription_amount stays nullable for migration-first rollout.

LOCK TABLE user_subscriptions IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE groups IN SHARE MODE;

ALTER TABLE billing_usage_entries
    ADD COLUMN IF NOT EXISTS subscription_amount DECIMAL(20, 10);

ALTER TABLE billing_usage_entries
    DROP CONSTRAINT IF EXISTS billing_usage_entries_subscription_amount_check;

ALTER TABLE billing_usage_entries
    ADD CONSTRAINT billing_usage_entries_subscription_amount_check
    CHECK (
        subscription_amount IS NULL
        OR (
            subscription_amount >= 0
            AND subscription_amount <> 'NaN'::numeric
            AND subscription_amount <> 'Infinity'::numeric
            AND subscription_amount <> '-Infinity'::numeric
        )
    ) NOT VALID;

ALTER TABLE billing_usage_entries
    VALIDATE CONSTRAINT billing_usage_entries_subscription_amount_check;

ALTER TABLE user_subscriptions
    DROP CONSTRAINT IF EXISTS user_subscriptions_usage_amounts_check;

ALTER TABLE user_subscriptions
    ADD CONSTRAINT user_subscriptions_usage_amounts_check
    CHECK (
        weekly_usage_usd >= 0
        AND weekly_usage_usd <> 'NaN'::numeric
        AND weekly_usage_usd <> 'Infinity'::numeric
        AND weekly_usage_usd <> '-Infinity'::numeric
        AND monthly_usage_usd >= 0
        AND monthly_usage_usd <> 'NaN'::numeric
        AND monthly_usage_usd <> 'Infinity'::numeric
        AND monthly_usage_usd <> '-Infinity'::numeric
    ) NOT VALID;

ALTER TABLE user_subscriptions
    VALIDATE CONSTRAINT user_subscriptions_usage_amounts_check;

-- Standard groups historically accept negative values as an input shorthand
-- for clearing a limit. Keep that compatibility while preventing subscription
-- plans from persisting non-finite or negative weekly/monthly limits.
ALTER TABLE groups
    DROP CONSTRAINT IF EXISTS groups_subscription_quota_limits_check;

ALTER TABLE groups
    ADD CONSTRAINT groups_subscription_quota_limits_check
    CHECK (
        subscription_type <> 'subscription'
        OR (
            (
                weekly_limit_usd IS NULL
                OR (
                    weekly_limit_usd >= 0
                    AND weekly_limit_usd <> 'NaN'::numeric
                    AND weekly_limit_usd <> 'Infinity'::numeric
                    AND weekly_limit_usd <> '-Infinity'::numeric
                )
            )
            AND (
                monthly_limit_usd IS NULL
                OR (
                    monthly_limit_usd >= 0
                    AND monthly_limit_usd <> 'NaN'::numeric
                    AND monthly_limit_usd <> 'Infinity'::numeric
                    AND monthly_limit_usd <> '-Infinity'::numeric
                )
            )
        )
    ) NOT VALID;

ALTER TABLE groups
    VALIDATE CONSTRAINT groups_subscription_quota_limits_check;

-- Materialize the current subscription/ledger classification once for the
-- remainder of this transaction. The previous migration text repeated the
-- same correlated receipt/log matching four times; on a production-sized
-- ledger those repeated LATERAL probes could exceed the migrate-only deadline.
-- CURRENT_TIMESTAMP is transaction-stable, and ON COMMIT DROP keeps these
-- snapshots private to this one migration attempt (including rollback).
CREATE TEMP TABLE migration_195_anchored
ON COMMIT DROP
AS
SELECT
    us.id,
    us.user_id,
    us.group_id,
    us.status,
    us.created_at,
    us.starts_at,
    us.expires_at,
    us.weekly_window_start,
    us.monthly_window_start,
    us.weekly_usage_usd,
    us.monthly_usage_usd,
    us.deleted_at,
    us.starts_at
        + FLOOR(
            GREATEST(
                0,
                EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - us.starts_at))
            ) / 604800
        ) * 604800 * INTERVAL '1 second' AS weekly_period_start,
    us.starts_at
        + FLOOR(
            GREATEST(
                0,
                EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - us.starts_at))
            ) / 2592000
        ) * 2592000 * INTERVAL '1 second' AS monthly_period_start
FROM user_subscriptions us;

CREATE UNIQUE INDEX migration_195_anchored_id_idx
    ON migration_195_anchored (id);

CREATE TEMP TABLE migration_195_receipt_classification
ON COMMIT DROP
AS
WITH relevant_receipts AS MATERIALIZED (
    SELECT
        bue.id,
        bue.usage_log_id,
        bue.subscription_id,
        bue.user_id AS receipt_user_id,
        bue.api_key_id,
        receipt_key.user_id AS receipt_api_key_user_id,
        bue.request_id,
        bue.billing_type,
        bue.created_at AS receipt_created_at,
        bue.subscription_amount,
        a.starts_at,
        a.user_id AS subscription_user_id,
        a.weekly_period_start,
        a.monthly_period_start
    FROM billing_usage_entries bue
    JOIN migration_195_anchored a
        ON a.id = bue.subscription_id
    LEFT JOIN api_keys receipt_key
        ON receipt_key.id = bue.api_key_id
    WHERE bue.applied
      AND bue.status = 'subscription'
      AND bue.subscription_id IS NOT NULL
      AND bue.created_at >= LEAST(
          a.weekly_period_start,
          a.monthly_period_start
      )
      AND bue.created_at < GREATEST(
          a.weekly_period_start + 604800 * INTERVAL '1 second',
          a.monthly_period_start + 2592000 * INTERVAL '1 second'
      )
)
SELECT
    rr.*,
    COALESCE(candidate_scan.candidate_count, 0) AS candidate_count,
    COALESCE(candidate_scan.invalid_identity_count, 0)
        AS invalid_identity_count,
    COALESCE(candidate_scan.invalid_actual_cost_count, 0)
        AS invalid_actual_cost_count,
    matched_log.actual_cost AS matched_actual_cost,
    matched_log.created_at AS matched_created_at
FROM relevant_receipts rr
LEFT JOIN LATERAL (
    SELECT
        COUNT(*) AS candidate_count,
        COUNT(*) FILTER (
            WHERE ul.subscription_id IS DISTINCT FROM rr.subscription_id
               OR ul.billing_type <> 1
               OR ul.user_id IS DISTINCT FROM rr.subscription_user_id
               OR ul.user_id IS DISTINCT FROM rr.receipt_user_id
               OR ul.api_key_id IS DISTINCT FROM rr.api_key_id
               OR (
                   ul.id IS NOT DISTINCT FROM rr.usage_log_id
                   AND rr.request_id IS NOT NULL
                   AND ul.request_id IS NOT NULL
                   AND ul.request_id IS DISTINCT FROM rr.request_id
               )
        ) AS invalid_identity_count,
        COUNT(*) FILTER (
            WHERE ul.actual_cost < 0
               OR ul.actual_cost = 'NaN'::numeric
               OR ul.actual_cost = 'Infinity'::numeric
               OR ul.actual_cost = '-Infinity'::numeric
        ) AS invalid_actual_cost_count
    FROM usage_logs ul
    WHERE ul.id = rr.usage_log_id
       OR (
           rr.request_id IS NOT NULL
           AND ul.request_id = rr.request_id
           AND ul.api_key_id = rr.api_key_id
       )
) candidate_scan ON TRUE
LEFT JOIN LATERAL (
    SELECT
        ul.actual_cost,
        ul.created_at
    FROM usage_logs ul
    WHERE ul.subscription_id = rr.subscription_id
      AND ul.billing_type = 1
      AND ul.user_id = rr.subscription_user_id
      AND ul.user_id = rr.receipt_user_id
      AND ul.api_key_id = rr.api_key_id
      AND (
          ul.id IS DISTINCT FROM rr.usage_log_id
          OR rr.request_id IS NULL
          OR ul.request_id IS NULL
          OR ul.request_id = rr.request_id
      )
      AND (
          ul.id = rr.usage_log_id
          OR (
              rr.request_id IS NOT NULL
              AND ul.request_id = rr.request_id
              AND ul.api_key_id = rr.api_key_id
          )
      )
    ORDER BY
        CASE WHEN ul.id = rr.usage_log_id THEN 0 ELSE 1 END,
        ul.id
    LIMIT 1
) matched_log ON TRUE;

CREATE INDEX migration_195_receipt_subscription_idx
    ON migration_195_receipt_classification (subscription_id);

CREATE TEMP TABLE migration_195_window_log_classification
ON COMMIT DROP
AS
WITH window_logs AS MATERIALIZED (
    SELECT
        ul.id,
        ul.subscription_id,
        ul.user_id AS log_user_id,
        ul.api_key_id,
        ul.request_id,
        ul.billing_type,
        ul.actual_cost,
        ul.created_at AS log_created_at,
        a.starts_at,
        a.user_id AS subscription_user_id,
        log_key.user_id AS log_api_key_user_id,
        a.weekly_period_start,
        a.monthly_period_start
    FROM usage_logs ul
    JOIN migration_195_anchored a
        ON a.id = ul.subscription_id
    LEFT JOIN api_keys log_key
        ON log_key.id = ul.api_key_id
    WHERE ul.billing_type = 1
      AND ul.created_at >= LEAST(
          a.weekly_period_start,
          a.monthly_period_start
      )
      AND ul.created_at < GREATEST(
          a.weekly_period_start + 604800 * INTERVAL '1 second',
          a.monthly_period_start + 2592000 * INTERVAL '1 second'
      )
)
SELECT
    wl.*,
    COALESCE(receipt_scan.all_receipt_count, 0) AS all_receipt_count,
    COALESCE(receipt_scan.valid_receipt_count, 0) AS valid_receipt_count
FROM window_logs wl
LEFT JOIN LATERAL (
    SELECT
        COUNT(*) AS all_receipt_count,
        COUNT(*) FILTER (
            WHERE bue.applied
              AND bue.status = 'subscription'
              AND bue.billing_type = 1
              AND bue.subscription_id = wl.subscription_id
              AND bue.user_id = wl.subscription_user_id
              AND bue.user_id = wl.log_user_id
              AND bue.api_key_id = wl.api_key_id
              AND wl.log_api_key_user_id = wl.subscription_user_id
              AND wl.log_api_key_user_id = wl.log_user_id
              AND (
                  bue.usage_log_id IS DISTINCT FROM wl.id
                  OR bue.request_id IS NULL
                  OR wl.request_id IS NULL
                  OR bue.request_id = wl.request_id
              )
        ) AS valid_receipt_count
    FROM billing_usage_entries bue
    WHERE bue.usage_log_id = wl.id
       OR (
           bue.request_id IS NOT NULL
           AND wl.request_id = bue.request_id
           AND bue.api_key_id = wl.api_key_id
       )
) receipt_scan ON TRUE;

CREATE INDEX migration_195_window_log_subscription_idx
    ON migration_195_window_log_classification (subscription_id);

ANALYZE migration_195_anchored;
ANALYZE migration_195_receipt_classification;
ANALYZE migration_195_window_log_classification;

-- Abort publication when current-window ledger evidence cannot be
-- reconstructed unambiguously. An explicit subscription_amount may replace a
-- missing usage log, but it cannot make contradictory receipt metadata,
-- cross-owner links, multiple matching logs, or invalid membership usage safe
-- to publish.
DO $$
DECLARE
    invalid_receipt_count BIGINT;
    invalid_subscription_log_count BIGINT;
    invalid_active_plan_count BIGINT;
    invalid_subscription_time_count BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO invalid_receipt_count
    FROM migration_195_receipt_classification rc
    WHERE rc.billing_type <> 1
       OR rc.receipt_user_id IS DISTINCT FROM rc.subscription_user_id
       OR rc.receipt_api_key_user_id
            IS DISTINCT FROM rc.subscription_user_id
       OR rc.receipt_api_key_user_id IS DISTINCT FROM rc.receipt_user_id
       OR rc.candidate_count > 1
       OR rc.invalid_identity_count > 0
       OR rc.invalid_actual_cost_count > 0
       OR rc.matched_created_at < rc.starts_at
       OR (
          rc.subscription_amount IS NULL
          AND rc.matched_actual_cost IS NULL
       );

    -- A subscription-billed usage log is a compatibility fallback only when
    -- it has no receipt at all. Any linked receipt must be exactly one applied
    -- membership receipt; otherwise source identity is contradictory and the
    -- migration must fail instead of replaying the log as legacy usage.
    SELECT COUNT(*)
    INTO invalid_subscription_log_count
    FROM migration_195_window_log_classification wl
    WHERE wl.billing_type = 1
      AND (
          wl.actual_cost < 0
          OR wl.actual_cost = 'NaN'::numeric
          OR wl.actual_cost = 'Infinity'::numeric
          OR wl.actual_cost = '-Infinity'::numeric
          OR wl.log_user_id IS DISTINCT FROM wl.subscription_user_id
          OR wl.log_api_key_user_id
                IS DISTINCT FROM wl.subscription_user_id
          OR wl.log_api_key_user_id IS DISTINCT FROM wl.log_user_id
          OR (
              wl.all_receipt_count > 0
              AND (
                  wl.all_receipt_count <> 1
                  OR wl.valid_receipt_count <> 1
              )
          )
      );

    SELECT COUNT(*)
    INTO invalid_active_plan_count
    FROM migration_195_anchored us
    LEFT JOIN groups g ON g.id = us.group_id
    WHERE us.deleted_at IS NULL
      AND us.status = 'active'
      AND us.expires_at > CURRENT_TIMESTAMP
      AND (
          g.id IS NULL
          OR g.deleted_at IS NOT NULL
          OR g.subscription_type <> 'subscription'
          OR g.weekly_limit_usd IS NULL
          OR g.weekly_limit_usd < 0
          OR g.weekly_limit_usd = 'NaN'::numeric
          OR g.weekly_limit_usd = 'Infinity'::numeric
          OR g.weekly_limit_usd = '-Infinity'::numeric
          OR g.monthly_limit_usd < 0
          OR g.monthly_limit_usd = 'NaN'::numeric
          OR g.monthly_limit_usd = 'Infinity'::numeric
          OR g.monthly_limit_usd = '-Infinity'::numeric
      );

    SELECT COUNT(*)
    INTO invalid_subscription_time_count
    FROM migration_195_anchored us
    WHERE us.deleted_at IS NULL
      AND (
          us.expires_at <= us.starts_at
          OR (us.status = 'active' AND us.starts_at > CURRENT_TIMESTAMP)
      );

    IF invalid_receipt_count > 0
       OR invalid_subscription_log_count > 0
       OR invalid_active_plan_count > 0
       OR invalid_subscription_time_count > 0 THEN
        RAISE EXCEPTION
            'cannot rebuild anchored subscription quota: % invalid current-window receipt(s), % invalid current-window subscription usage log(s), % invalid active plan(s), % invalid subscription time range(s)',
            invalid_receipt_count,
            invalid_subscription_log_count,
            invalid_active_plan_count,
            invalid_subscription_time_count
            USING HINT =
                'Drain billing traffic, reconcile missing, ambiguous, cross-owner, mismatched, negative, or non-finite ledger evidence, then retry migration 195.';
    END IF;

END $$;

-- Derive the authoritative weekly/monthly totals once from the validated
-- classifications. Both the weekly mismatch gate and the final UPDATE reuse
-- this snapshot instead of rescanning the durable ledger.
CREATE TEMP TABLE migration_195_aggregated_usage
ON COMMIT DROP
AS
WITH receipt_usage AS MATERIALIZED (
    SELECT
        rc.subscription_id AS id,
        rc.receipt_created_at AS occurred_at,
        COALESCE(
            rc.subscription_amount,
            rc.matched_actual_cost
        )::DECIMAL(20, 10) AS amount
    FROM migration_195_receipt_classification rc
    WHERE rc.billing_type = 1
      AND rc.receipt_user_id = rc.subscription_user_id
      AND rc.receipt_api_key_user_id = rc.subscription_user_id
      AND rc.receipt_api_key_user_id = rc.receipt_user_id
),
legacy_log_usage AS MATERIALIZED (
    SELECT
        wl.subscription_id AS id,
        wl.log_created_at AS occurred_at,
        wl.actual_cost::DECIMAL(20, 10) AS amount
    FROM migration_195_window_log_classification wl
    WHERE wl.billing_type = 1
      AND wl.log_user_id = wl.subscription_user_id
      AND wl.log_api_key_user_id = wl.subscription_user_id
      AND wl.log_api_key_user_id = wl.log_user_id
      AND wl.all_receipt_count = 0
),
accounted_usage AS MATERIALIZED (
    SELECT id, occurred_at, amount
    FROM receipt_usage

    UNION ALL

    SELECT id, occurred_at, amount
    FROM legacy_log_usage
)
SELECT
    a.id,
    COALESCE(
        SUM(au.amount) FILTER (
            WHERE au.occurred_at >= a.weekly_period_start
              AND au.occurred_at < a.weekly_period_start
                  + 604800 * INTERVAL '1 second'
        ),
        0
    )::DECIMAL(20, 10) AS weekly_used,
    COALESCE(
        SUM(au.amount) FILTER (
            WHERE au.occurred_at >= a.monthly_period_start
              AND au.occurred_at < a.monthly_period_start
                  + 2592000 * INTERVAL '1 second'
        ),
        0
    )::DECIMAL(20, 10) AS monthly_used
FROM migration_195_anchored a
LEFT JOIN accounted_usage au ON au.id = a.id
GROUP BY a.id;

CREATE UNIQUE INDEX migration_195_aggregated_usage_id_idx
    ON migration_195_aggregated_usage (id);

ANALYZE migration_195_aggregated_usage;

DO $$
DECLARE
    authoritative_weekly_ledger_mismatch_count BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO authoritative_weekly_ledger_mismatch_count
    FROM migration_195_anchored a
    JOIN migration_195_aggregated_usage au ON au.id = a.id
    WHERE a.weekly_window_start = a.weekly_period_start
      AND ABS(a.weekly_usage_usd - au.weekly_used) > 0.00000001;

    IF authoritative_weekly_ledger_mismatch_count > 0 THEN
        RAISE EXCEPTION
            'cannot rebuild anchored subscription quota: % authoritative weekly counter(s) do not match durable billing evidence',
            authoritative_weekly_ledger_mismatch_count
            USING HINT =
                'Reconcile authoritative weekly counters against receipt and legacy-log evidence before retrying migration 195.';
    END IF;
END $$;

UPDATE user_subscriptions us
SET weekly_window_start = a.weekly_period_start,
    weekly_usage_usd = CASE
        WHEN us.weekly_window_start = a.weekly_period_start
            THEN us.weekly_usage_usd
        ELSE au.weekly_used
    END,
    monthly_window_start = a.monthly_period_start,
    monthly_usage_usd = au.monthly_used,
    updated_at = NOW()
FROM migration_195_anchored a
JOIN migration_195_aggregated_usage au ON au.id = a.id
WHERE us.id = a.id;

ALTER TABLE user_subscriptions
    DROP CONSTRAINT IF EXISTS user_subscriptions_weekly_window_anchored_check;

ALTER TABLE user_subscriptions
    ADD CONSTRAINT user_subscriptions_weekly_window_anchored_check
    CHECK (
        weekly_window_start IS NULL
        OR (
            weekly_window_start >= starts_at
            AND MOD(
                EXTRACT(EPOCH FROM (weekly_window_start - starts_at))::NUMERIC,
                604800
            ) = 0
        )
    ) NOT VALID;

ALTER TABLE user_subscriptions
    VALIDATE CONSTRAINT user_subscriptions_weekly_window_anchored_check;

ALTER TABLE user_subscriptions
    DROP CONSTRAINT IF EXISTS user_subscriptions_monthly_window_anchored_check;

ALTER TABLE user_subscriptions
    ADD CONSTRAINT user_subscriptions_monthly_window_anchored_check
    CHECK (
        monthly_window_start IS NULL
        OR (
            monthly_window_start >= starts_at
            AND MOD(
                EXTRACT(EPOCH FROM (monthly_window_start - starts_at))::NUMERIC,
                2592000
            ) = 0
        )
    ) NOT VALID;

ALTER TABLE user_subscriptions
    VALIDATE CONSTRAINT user_subscriptions_monthly_window_anchored_check;
