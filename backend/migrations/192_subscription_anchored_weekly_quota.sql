-- 192: make the subscription quota a single starts_at-anchored 7x24h window.
--
-- Compatibility policy:
--   * legacy daily/monthly configuration and counters are retained for rollback,
--     but application code no longer enforces or increments them;
--   * a weekly counter is retained only when its old marker already identifies
--     the exact current anchored period;
--   * otherwise the current half-open period is rebuilt from usage_logs so a
--     counter from an older window cannot leak forward and block indefinitely.

WITH anchored AS (
    SELECT
        us.id,
        us.starts_at
            + FLOOR(
                GREATEST(
                    0,
                    EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - us.starts_at))
                ) / 604800
            ) * INTERVAL '7 days' AS period_start
    FROM user_subscriptions us
),
logged_usage AS (
    SELECT
        a.id,
        COALESCE(SUM(ul.actual_cost), 0)::DECIMAL(20, 10) AS used
    FROM anchored a
    LEFT JOIN usage_logs ul
        ON ul.subscription_id = a.id
       AND ul.created_at >= a.period_start
       AND ul.created_at < a.period_start + INTERVAL '7 days'
    GROUP BY a.id
)
UPDATE user_subscriptions us
SET weekly_window_start = a.period_start,
    weekly_usage_usd = CASE
        WHEN us.weekly_window_start = a.period_start
            THEN us.weekly_usage_usd
        ELSE lu.used
    END,
    updated_at = NOW()
FROM anchored a
JOIN logged_usage lu ON lu.id = a.id
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
