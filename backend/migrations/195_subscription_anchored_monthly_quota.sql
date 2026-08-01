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

ALTER TABLE billing_usage_entries
    ADD COLUMN IF NOT EXISTS subscription_amount DECIMAL(20, 10);

ALTER TABLE billing_usage_entries
    DROP CONSTRAINT IF EXISTS billing_usage_entries_subscription_amount_check;

ALTER TABLE billing_usage_entries
    ADD CONSTRAINT billing_usage_entries_subscription_amount_check
    CHECK (subscription_amount IS NULL OR subscription_amount >= 0) NOT VALID;

ALTER TABLE billing_usage_entries
    VALIDATE CONSTRAINT billing_usage_entries_subscription_amount_check;

-- Abort publication when current-window ledger evidence cannot be
-- reconstructed unambiguously. An explicit subscription_amount may replace a
-- missing usage log, but it cannot make contradictory receipt metadata,
-- multiple matching logs, or negative membership usage safe to publish.
DO $$
DECLARE
    invalid_receipt_count BIGINT;
    invalid_legacy_log_count BIGINT;
BEGIN
    WITH anchored AS (
        SELECT
            us.id,
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
        FROM user_subscriptions us
    )
    SELECT COUNT(*)
    INTO invalid_receipt_count
    FROM billing_usage_entries bue
    JOIN anchored a
        ON a.id = bue.subscription_id
    LEFT JOIN LATERAL (
        SELECT
            COUNT(*) AS candidate_count,
            COUNT(*) FILTER (
                WHERE ul.billing_type <> 1
            ) AS wrong_billing_type_count,
            COUNT(*) FILTER (
                WHERE ul.actual_cost < 0
            ) AS negative_actual_cost_count
        FROM usage_logs ul
        WHERE ul.subscription_id = bue.subscription_id
          AND (
              ul.id = bue.usage_log_id
              OR (
                  bue.request_id IS NOT NULL
                  AND ul.request_id = bue.request_id
                  AND ul.api_key_id = bue.api_key_id
              )
          )
    ) candidate_scan ON TRUE
    LEFT JOIN LATERAL (
        SELECT ul.actual_cost
        FROM usage_logs ul
        WHERE ul.subscription_id = bue.subscription_id
          AND ul.billing_type = 1
          AND (
              ul.id = bue.usage_log_id
              OR (
                  bue.request_id IS NOT NULL
                  AND ul.request_id = bue.request_id
                  AND ul.api_key_id = bue.api_key_id
              )
          )
        ORDER BY
            CASE WHEN ul.id = bue.usage_log_id THEN 0 ELSE 1 END,
            ul.id
        LIMIT 1
    ) matched_log ON TRUE
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
      AND (
          bue.billing_type <> 1
          OR candidate_scan.candidate_count > 1
          OR candidate_scan.wrong_billing_type_count > 0
          OR candidate_scan.negative_actual_cost_count > 0
          OR (
              bue.subscription_amount IS NULL
              AND matched_log.actual_cost IS NULL
          )
      );

    -- A subscription-billed usage log without a receipt is the compatibility
    -- path for pre-ledger writers. It is safe to replay only when the amount
    -- could have been produced by the positive-only quota increment path.
    WITH anchored AS (
        SELECT
            us.id,
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
        FROM user_subscriptions us
    )
    SELECT COUNT(*)
    INTO invalid_legacy_log_count
    FROM usage_logs ul
    JOIN anchored a
        ON a.id = ul.subscription_id
    WHERE ul.billing_type IN (1)
      AND ul.actual_cost < 0
      AND ul.created_at >= LEAST(
          a.weekly_period_start,
          a.monthly_period_start
      )
      AND ul.created_at < GREATEST(
          a.weekly_period_start + 604800 * INTERVAL '1 second',
          a.monthly_period_start + 2592000 * INTERVAL '1 second'
      )
      AND NOT EXISTS (
          SELECT 1
          FROM billing_usage_entries bue
          WHERE bue.applied
            AND bue.status = 'subscription'
            AND bue.subscription_id = ul.subscription_id
            AND (
                bue.usage_log_id = ul.id
                OR (
                    bue.request_id IS NOT NULL
                    AND ul.request_id = bue.request_id
                    AND bue.api_key_id = ul.api_key_id
                )
            )
      );

    IF invalid_receipt_count > 0 OR invalid_legacy_log_count > 0 THEN
        RAISE EXCEPTION
            'cannot rebuild anchored subscription quota: % invalid current-window receipt(s), % invalid unmatched subscription usage log(s)',
            invalid_receipt_count,
            invalid_legacy_log_count
            USING HINT =
                'Drain billing traffic, reconcile missing, ambiguous, mismatched, or negative ledger evidence, then retry migration 195.';
    END IF;
END $$;

WITH anchored AS (
    SELECT
        us.id,
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
    FROM user_subscriptions us
),
receipt_usage AS (
    SELECT
        bue.subscription_id AS id,
        bue.created_at AS occurred_at,
        COALESCE(bue.subscription_amount, matched_log.actual_cost)::DECIMAL(20, 10) AS amount
    FROM billing_usage_entries bue
    JOIN anchored a
        ON a.id = bue.subscription_id
    LEFT JOIN LATERAL (
        SELECT ul.actual_cost
        FROM usage_logs ul
        WHERE ul.subscription_id = bue.subscription_id
          AND ul.billing_type = 1
          AND (
              ul.id = bue.usage_log_id
              OR (
                  bue.request_id IS NOT NULL
                  AND ul.request_id = bue.request_id
                  AND ul.api_key_id = bue.api_key_id
              )
          )
        ORDER BY
            CASE WHEN ul.id = bue.usage_log_id THEN 0 ELSE 1 END,
            ul.id
        LIMIT 1
    ) matched_log ON TRUE
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
),
legacy_log_usage AS (
    SELECT
        ul.subscription_id AS id,
        ul.created_at AS occurred_at,
        ul.actual_cost::DECIMAL(20, 10) AS amount
    FROM usage_logs ul
    JOIN anchored a
        ON a.id = ul.subscription_id
    WHERE ul.billing_type = 1
      AND ul.created_at >= LEAST(
          a.weekly_period_start,
          a.monthly_period_start
      )
      AND ul.created_at < GREATEST(
          a.weekly_period_start + 604800 * INTERVAL '1 second',
          a.monthly_period_start + 2592000 * INTERVAL '1 second'
      )
      AND NOT EXISTS (
          SELECT 1
          FROM billing_usage_entries bue
          WHERE bue.applied
            AND bue.status = 'subscription'
            AND bue.subscription_id = ul.subscription_id
            AND (
                bue.usage_log_id = ul.id
                OR (
                    bue.request_id IS NOT NULL
                    AND ul.request_id = bue.request_id
                    AND bue.api_key_id = ul.api_key_id
                )
            )
      )
),
accounted_usage AS (
    SELECT id, occurred_at, amount
    FROM receipt_usage

    UNION ALL

    SELECT id, occurred_at, amount
    FROM legacy_log_usage
),
aggregated_usage AS (
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
    FROM anchored a
    LEFT JOIN accounted_usage au ON au.id = a.id
    GROUP BY a.id
)
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
FROM anchored a
JOIN aggregated_usage au ON au.id = a.id
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
