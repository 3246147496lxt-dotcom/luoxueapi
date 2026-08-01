WITH
params AS (
    SELECT
        :'preflight_mode'::text AS preflight_mode,
        NULLIF(:'legacy_writers_stopped_at', '')::timestamptz
            AS legacy_writers_stopped_at,
        GREATEST(:'term_guard_hours'::integer, 1) AS term_guard_hours,
        :'acknowledge_ambiguous_term_events'::boolean
            AS acknowledge_ambiguous_term_events,
        :'expected_migration_checksum'::text AS expected_migration_checksum
),
schema_state AS (
    SELECT
        EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'billing_usage_entries'
              AND column_name = 'subscription_amount'
        ) AS has_subscription_amount,
        COALESCE((
            SELECT sm.checksum
            FROM schema_migrations sm
            WHERE sm.filename = '195_subscription_anchored_monthly_quota.sql'
        ), '') AS recorded_checksum,
        COALESCE((
            SELECT BOOL_AND(c.convalidated)
            FROM pg_constraint c
            WHERE c.conname IN (
                'billing_usage_entries_subscription_amount_check',
                'user_subscriptions_weekly_window_anchored_check',
                'user_subscriptions_monthly_window_anchored_check'
            )
              AND c.conrelid IN (
                  to_regclass('public.billing_usage_entries'),
                  to_regclass('public.user_subscriptions')
              )
        ), FALSE) AS quota_constraints_validated,
        (
            SELECT COUNT(*)
            FROM pg_constraint c
            WHERE c.conname IN (
                'billing_usage_entries_subscription_amount_check',
                'user_subscriptions_weekly_window_anchored_check',
                'user_subscriptions_monthly_window_anchored_check'
            )
              AND c.conrelid IN (
                  to_regclass('public.billing_usage_entries'),
                  to_regclass('public.user_subscriptions')
              )
        ) AS quota_constraint_count
),
anchored AS MATERIALIZED (
    SELECT
        us.id,
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
    FROM user_subscriptions us
),
relevant_receipts AS MATERIALIZED (
    SELECT
        bue.id,
        bue.usage_log_id,
        bue.subscription_id,
        bue.api_key_id,
        bue.request_id,
        bue.billing_type,
        bue.created_at,
        NULLIF(to_jsonb(bue) ->> 'subscription_amount', '')::numeric
            AS subscription_amount,
        a.starts_at,
        a.created_at AS subscription_created_at,
        a.weekly_period_start,
        a.monthly_period_start
    FROM billing_usage_entries bue
    JOIN anchored a
        ON a.id = bue.subscription_id
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
receipt_classified AS MATERIALIZED (
    SELECT
        rr.*,
        COALESCE(matches.all_candidate_count, 0) AS all_candidate_count,
        COALESCE(matches.valid_candidate_count, 0) AS valid_candidate_count,
        COALESCE(matches.invalid_candidate_count, 0) AS invalid_candidate_count,
        COALESCE(matches.negative_candidate_count, 0) AS negative_candidate_count,
        matches.matched_actual_cost,
        matches.matched_created_at
    FROM relevant_receipts rr
    LEFT JOIN LATERAL (
        SELECT
            COUNT(*) AS all_candidate_count,
            COUNT(*) FILTER (
                WHERE ul.billing_type = 1
            ) AS valid_candidate_count,
            COUNT(*) FILTER (
                WHERE ul.billing_type <> 1
            ) AS invalid_candidate_count,
            COUNT(*) FILTER (
                WHERE ul.billing_type = 1
                  AND ul.actual_cost < 0
            ) AS negative_candidate_count,
            (
                ARRAY_AGG(
                    ul.actual_cost
                    ORDER BY
                        CASE WHEN ul.id = rr.usage_log_id THEN 0 ELSE 1 END,
                        ul.id
                ) FILTER (WHERE ul.billing_type = 1)
            )[1] AS matched_actual_cost,
            (
                ARRAY_AGG(
                    ul.created_at
                    ORDER BY
                        CASE WHEN ul.id = rr.usage_log_id THEN 0 ELSE 1 END,
                        ul.id
                ) FILTER (WHERE ul.billing_type = 1)
            )[1] AS matched_created_at
        FROM usage_logs ul
        WHERE ul.subscription_id = rr.subscription_id
          AND (
              ul.id = rr.usage_log_id
              OR (
                  rr.request_id IS NOT NULL
                  AND ul.request_id = rr.request_id
                  AND ul.api_key_id = rr.api_key_id
              )
          )
    ) matches ON TRUE
),
unreceipted_logs AS MATERIALIZED (
    SELECT
        ul.id,
        ul.subscription_id,
        ul.billing_type,
        ul.actual_cost,
        ul.created_at,
        a.starts_at,
        a.subscription_created_at,
        a.weekly_period_start,
        a.monthly_period_start
    FROM usage_logs ul
    JOIN (
        SELECT
            id,
            starts_at,
            created_at AS subscription_created_at,
            weekly_period_start,
            monthly_period_start
        FROM anchored
    ) a ON a.id = ul.subscription_id
    WHERE ul.created_at >= LEAST(
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
receipt_usage AS MATERIALIZED (
    SELECT
        rc.subscription_id AS id,
        rc.created_at AS occurred_at,
        COALESCE(rc.subscription_amount, rc.matched_actual_cost)::numeric
            AS amount
    FROM receipt_classified rc
    WHERE rc.billing_type = 1
      AND rc.valid_candidate_count <= 1
      AND rc.invalid_candidate_count = 0
      AND rc.negative_candidate_count = 0
      AND COALESCE(rc.subscription_amount, rc.matched_actual_cost) >= 0
),
legacy_usage AS MATERIALIZED (
    SELECT
        ul.subscription_id AS id,
        ul.created_at AS occurred_at,
        ul.actual_cost::numeric AS amount
    FROM unreceipted_logs ul
    WHERE ul.billing_type = 1
      AND ul.actual_cost >= 0
),
accounted_usage AS MATERIALIZED (
    SELECT id, occurred_at, amount
    FROM receipt_usage

    UNION ALL

    SELECT id, occurred_at, amount
    FROM legacy_usage
),
aggregated_usage AS MATERIALIZED (
    SELECT
        a.id,
        COALESCE(
            SUM(au.amount) FILTER (
                WHERE au.occurred_at >= a.weekly_period_start
                  AND au.occurred_at < a.weekly_period_start
                      + 604800 * INTERVAL '1 second'
            ),
            0
        )::numeric AS weekly_used,
        COALESCE(
            SUM(au.amount) FILTER (
                WHERE au.occurred_at >= a.monthly_period_start
                  AND au.occurred_at < a.monthly_period_start
                      + 2592000 * INTERVAL '1 second'
            ),
            0
        )::numeric AS monthly_used
    FROM anchored a
    LEFT JOIN accounted_usage au ON au.id = a.id
    GROUP BY a.id
),
term_risk AS (
    SELECT
        COUNT(*) FILTER (
            WHERE rc.matched_created_at < rc.starts_at
        ) AS definite_old_term_receipts,
        COUNT(*) FILTER (
            WHERE rc.matched_created_at IS NOT NULL
              AND rc.matched_created_at >= rc.starts_at
        ) AS ambiguous_receipts,
        (
            SELECT COUNT(*)
            FROM unreceipted_logs ul
            CROSS JOIN params p2
            WHERE p2.legacy_writers_stopped_at IS NOT NULL
              AND ul.billing_type = 1
              AND ul.starts_at > ul.subscription_created_at + INTERVAL '1 second'
              AND ul.created_at >= ul.starts_at
              AND ul.created_at < LEAST(
                  ul.starts_at + p2.term_guard_hours * INTERVAL '1 hour',
                  p2.legacy_writers_stopped_at
              )
        ) AS ambiguous_legacy_logs
    FROM receipt_classified rc
    CROSS JOIN params p
    WHERE p.legacy_writers_stopped_at IS NOT NULL
      AND rc.starts_at > rc.subscription_created_at + INTERVAL '1 second'
      AND rc.created_at >= rc.starts_at
      AND rc.created_at < LEAST(
          rc.starts_at + p.term_guard_hours * INTERVAL '1 hour',
          p.legacy_writers_stopped_at
      )
),
target_lock_state AS (
    SELECT
        COUNT(DISTINCT a.pid) FILTER (
            WHERE l.granted
              AND l.mode IN (
                  'RowExclusiveLock',
                  'ShareUpdateExclusiveLock',
                  'ShareLock',
                  'ShareRowExclusiveLock',
                  'ExclusiveLock',
                  'AccessExclusiveLock'
              )
        ) AS writer_sessions,
        COUNT(*) FILTER (WHERE NOT l.granted) AS waiting_locks
    FROM pg_locks l
    JOIN pg_class c ON c.oid = l.relation
    JOIN pg_stat_activity a ON a.pid = l.pid
    WHERE (
        c.relname IN ('user_subscriptions', 'billing_usage_entries', 'usage_logs')
        OR c.relname LIKE 'usage_logs\_%' ESCAPE '\'
    )
      AND a.pid <> pg_backend_pid()
),
transaction_state AS (
    SELECT
        COUNT(*) FILTER (
            WHERE state = 'idle in transaction'
        ) AS idle_in_transaction_sessions
    FROM pg_stat_activity
    WHERE datname = current_database()
      AND pid <> pg_backend_pid()
),
prepared_transaction_state AS (
    SELECT COUNT(*)::bigint AS prepared_transactions
    FROM pg_prepared_xacts
    WHERE database = current_database()
),
session_observer_state AS (
    SELECT
        COALESCE((
            SELECT r.rolsuper
            FROM pg_roles r
            WHERE r.rolname = CURRENT_USER
        ), FALSE)
        OR pg_has_role(CURRENT_USER, 'pg_read_all_stats', 'MEMBER')
            AS can_observe_all_sessions
),
summary AS (
    SELECT
        (SELECT COUNT(*) FROM anchored) AS subscriptions_scanned,
        (SELECT COUNT(*) FROM relevant_receipts) AS current_window_receipts,
        (SELECT COUNT(*) FROM receipt_classified
         WHERE subscription_amount IS NULL
           AND matched_actual_cost IS NULL) AS unresolved_receipts,
        (SELECT COUNT(*) FROM receipt_classified
         WHERE billing_type <> 1) AS receipt_billing_type_mismatches,
        (SELECT COUNT(*) FROM receipt_classified
         WHERE valid_candidate_count > 1
            OR invalid_candidate_count > 0) AS ambiguous_or_mixed_receipt_matches,
        (SELECT COUNT(*) FROM receipt_classified
         WHERE subscription_amount < 0
            OR negative_candidate_count > 0) AS negative_receipt_amounts,
        (SELECT COUNT(*) FROM unreceipted_logs
         WHERE billing_type = 1) AS legacy_subscription_logs,
        (SELECT COUNT(*) FROM unreceipted_logs
         WHERE billing_type <> 1) AS excluded_balance_logs,
        (SELECT COUNT(*) FROM unreceipted_logs
         WHERE billing_type = 1
           AND actual_cost < 0) AS negative_legacy_amounts,
        (SELECT COUNT(*)
         FROM anchored a
         JOIN aggregated_usage au ON au.id = a.id
         WHERE a.weekly_window_start = a.weekly_period_start
           AND ABS(a.weekly_usage_usd - au.weekly_used) > 0.00000001
        ) AS authoritative_weekly_ledger_mismatches,
        (SELECT COUNT(*)
         FROM anchored a
         WHERE a.weekly_window_start IS DISTINCT FROM a.weekly_period_start
            OR a.monthly_window_start IS DISTINCT FROM a.monthly_period_start
        ) AS windows_requiring_reanchor,
        (SELECT COUNT(*)
         FROM anchored a
         JOIN groups g ON g.id = a.group_id
         WHERE a.deleted_at IS NULL
           AND a.status = 'active'
           AND a.expires_at > CURRENT_TIMESTAMP
           AND (
               g.deleted_at IS NOT NULL
               OR g.subscription_type <> 'subscription'
               OR g.weekly_limit_usd IS NULL
               OR g.weekly_limit_usd < 0
               OR g.monthly_limit_usd < 0
           )
        ) AS invalid_active_plan_configuration,
        (SELECT COUNT(*)
         FROM anchored a
         WHERE a.deleted_at IS NULL
           AND (
               a.expires_at <= a.starts_at
               OR (a.status = 'active' AND a.starts_at > CURRENT_TIMESTAMP)
           )
        ) AS invalid_subscription_times,
        tr.definite_old_term_receipts,
        tr.ambiguous_receipts + tr.ambiguous_legacy_logs
            AS ambiguous_reopened_term_events,
        tls.writer_sessions,
        tls.waiting_locks,
        ts.idle_in_transaction_sessions,
        pts.prepared_transactions,
        sos.can_observe_all_sessions
    FROM term_risk tr
    CROSS JOIN target_lock_state tls
    CROSS JOIN transaction_state ts
    CROSS JOIN prepared_transaction_state pts
    CROSS JOIN session_observer_state sos
),
checks AS (
    SELECT
        'schema_migration_state'::text AS check_name,
        CASE
            WHEN ss.recorded_checksum <> ''
             AND ss.recorded_checksum <> p.expected_migration_checksum
                THEN 'BLOCK'
            WHEN ss.recorded_checksum = '' AND ss.has_subscription_amount
                THEN 'BLOCK'
            WHEN ss.recorded_checksum <> '' AND NOT ss.has_subscription_amount
                THEN 'BLOCK'
            WHEN ss.recorded_checksum <> ''
             AND (ss.quota_constraint_count <> 3 OR NOT ss.quota_constraints_validated)
                THEN 'BLOCK'
            ELSE 'PASS'
        END AS status,
        CASE
            WHEN ss.recorded_checksum = '' THEN 0
            ELSE 1
        END::bigint AS affected_rows,
        CASE
            WHEN ss.recorded_checksum = ''
                THEN 'migration not yet applied; pre-migration schema is consistent'
            WHEN ss.recorded_checksum = p.expected_migration_checksum
                THEN 'migration already applied with matching checksum and validated constraints'
            ELSE 'migration schema/checksum state is inconsistent'
        END AS detail
    FROM schema_state ss
    CROSS JOIN params p

    UNION ALL

    SELECT
        'receipt_reconstructability',
        CASE WHEN s.unresolved_receipts = 0 THEN 'PASS' ELSE 'BLOCK' END,
        s.unresolved_receipts,
        'subscription receipts must have an exact subscription_amount or one membership-billed usage log'
    FROM summary s

    UNION ALL

    SELECT
        'receipt_billing_identity',
        CASE
            WHEN s.receipt_billing_type_mismatches = 0
             AND s.ambiguous_or_mixed_receipt_matches = 0
                THEN 'PASS'
            ELSE 'BLOCK'
        END,
        s.receipt_billing_type_mismatches
            + s.ambiguous_or_mixed_receipt_matches,
        'receipt billing_type and matched usage-log identity must be unambiguous membership billing'
    FROM summary s

    UNION ALL

    SELECT
        'nonnegative_usage_amounts',
        CASE
            WHEN s.negative_receipt_amounts = 0
             AND s.negative_legacy_amounts = 0
                THEN 'PASS'
            ELSE 'BLOCK'
        END,
        s.negative_receipt_amounts + s.negative_legacy_amounts,
        'quota reconstruction never accepts negative usage amounts'
    FROM summary s

    UNION ALL

    SELECT
        'authoritative_weekly_ledger_consistency',
        CASE
            WHEN s.authoritative_weekly_ledger_mismatches = 0
                THEN 'PASS'
            ELSE 'BLOCK'
        END,
        s.authoritative_weekly_ledger_mismatches,
        'current authoritative weekly counters must match drained durable billing evidence before monthly rebuild'
    FROM summary s

    UNION ALL

    SELECT
        'active_plan_configuration',
        CASE
            WHEN s.invalid_active_plan_configuration = 0
                THEN 'PASS'
            ELSE 'BLOCK'
        END,
        s.invalid_active_plan_configuration,
        'active memberships require a subscription group, weekly limit, and nonnegative explicit monthly limit'
    FROM summary s

    UNION ALL

    SELECT
        'subscription_time_invariants',
        CASE WHEN s.invalid_subscription_times = 0 THEN 'PASS' ELSE 'BLOCK' END,
        s.invalid_subscription_times,
        'active subscription time ranges must be internally consistent'
    FROM summary s

    UNION ALL

    SELECT
        'definite_old_term_events',
        CASE WHEN s.definite_old_term_receipts = 0 THEN 'PASS' ELSE 'BLOCK' END,
        s.definite_old_term_receipts,
        CASE
            WHEN p.legacy_writers_stopped_at IS NULL
                THEN 'not evaluated until legacy_writers_stopped_at is supplied'
            ELSE 'a matched usage log predates the reopened starts_at while its receipt falls in the new term'
        END
    FROM summary s
    CROSS JOIN params p

    UNION ALL

    SELECT
        'ambiguous_reopened_term_events',
        CASE
            WHEN p.legacy_writers_stopped_at IS NULL THEN 'WARN'
            WHEN s.ambiguous_reopened_term_events = 0 THEN 'PASS'
            WHEN p.acknowledge_ambiguous_term_events THEN 'WARN'
            ELSE 'BLOCK'
        END,
        s.ambiguous_reopened_term_events,
        CASE
            WHEN p.legacy_writers_stopped_at IS NULL
                THEN 'supply the exact old-writer stop time during the maintenance gate'
            WHEN s.ambiguous_reopened_term_events > 0
             AND p.acknowledge_ambiguous_term_events
                THEN 'operator acknowledged a separately reconciled aggregate risk set'
            ELSE 'events near a reopened starts_at require separate authorized reconciliation'
        END
    FROM summary s
    CROSS JOIN params p

    UNION ALL

    SELECT
        'database_writer_locks',
        CASE
            WHEN s.writer_sessions = 0
             AND s.waiting_locks = 0
             AND s.idle_in_transaction_sessions = 0
                THEN 'PASS'
            WHEN p.preflight_mode = 'maintenance' THEN 'BLOCK'
            ELSE 'WARN'
        END,
        s.writer_sessions + s.waiting_locks + s.idle_in_transaction_sessions,
        'maintenance requires zero target-table writers, waiting locks, and idle-in-transaction sessions'
    FROM summary s
    CROSS JOIN params p

    UNION ALL

    SELECT
        'database_prepared_transactions',
        CASE
            WHEN s.prepared_transactions = 0 THEN 'PASS'
            WHEN p.preflight_mode = 'maintenance' THEN 'BLOCK'
            ELSE 'WARN'
        END,
        s.prepared_transactions,
        'maintenance requires zero prepared transactions in the target database because their locks have no pg_stat_activity pid'
    FROM summary s
    CROSS JOIN params p

    UNION ALL

    SELECT
        'database_writer_visibility',
        CASE
            WHEN s.can_observe_all_sessions THEN 'PASS'
            WHEN p.preflight_mode = 'maintenance' THEN 'BLOCK'
            ELSE 'WARN'
        END,
        CASE WHEN s.can_observe_all_sessions THEN 0 ELSE 1 END,
        'maintenance requires superuser or pg_read_all_stats membership so writer and transaction checks cannot silently miss other roles'
    FROM summary s
    CROSS JOIN params p

    UNION ALL

    SELECT
        'legacy_subscription_logs',
        'INFO',
        s.legacy_subscription_logs,
        'unreceipted membership-billed logs included once in the compatibility reconstruction'
    FROM summary s

    UNION ALL

    SELECT
        'excluded_balance_logs',
        'INFO',
        s.excluded_balance_logs,
        'unreceipted non-membership logs carrying subscription_id are excluded from quota reconstruction'
    FROM summary s

    UNION ALL

    SELECT
        'windows_requiring_reanchor',
        'INFO',
        s.windows_requiring_reanchor,
        'subscription rows whose persisted window marker will be corrected by migration 195'
    FROM summary s
),
check_totals AS (
    SELECT
        COUNT(*) FILTER (WHERE status = 'BLOCK') AS blocker_count,
        COUNT(*) FILTER (WHERE status = 'WARN') AS warning_count
    FROM checks
),
relation_metrics AS (
    SELECT COALESCE(
        JSONB_AGG(
            JSONB_BUILD_OBJECT(
                'relation', c.relname,
                'estimated_rows', COALESCE(s.n_live_tup, 0)::bigint::text,
                'total_bytes', pg_total_relation_size(c.oid)::text,
                'dead_rows', COALESCE(s.n_dead_tup, 0)::bigint::text
            )
            ORDER BY c.relname
        ),
        '[]'::jsonb
    ) AS metrics
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    LEFT JOIN pg_stat_user_tables s ON s.relid = c.oid
    WHERE n.nspname = 'public'
      AND c.relname IN (
          'user_subscriptions',
          'billing_usage_entries',
          'usage_logs'
      )
),
result AS (
    SELECT JSONB_BUILD_OBJECT(
        'contract', 'migration-195-preflight/v1',
        'migration', '195_subscription_anchored_monthly_quota.sql',
		'migration_sha256', p.expected_migration_checksum,
        'status', CASE
            WHEN ct.blocker_count > 0 THEN 'blocked'
            WHEN ct.warning_count > 0 THEN 'warn'
            ELSE 'pass'
        END,
        'safe_to_apply', FALSE,
        'already_applied', (
            ss.recorded_checksum = p.expected_migration_checksum
            AND ss.recorded_checksum <> ''
        ),
        'schema_ok', NOT EXISTS (
            SELECT 1
            FROM checks
            WHERE check_name = 'schema_migration_state'
              AND status = 'BLOCK'
        ),
        'read_only_verified', (
            current_setting('transaction_read_only') = 'on'
        ),
        'checked_at', TO_CHAR(
            CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'counts', JSONB_BUILD_OBJECT(
            'subscriptions_scanned', s.subscriptions_scanned::text,
            'current_window_receipts', s.current_window_receipts::text,
            'unresolved_receipts', s.unresolved_receipts::text,
            'receipt_billing_type_mismatches', s.receipt_billing_type_mismatches::text,
            'ambiguous_or_mixed_receipt_matches', s.ambiguous_or_mixed_receipt_matches::text,
            'negative_usage_amounts', (
                s.negative_receipt_amounts + s.negative_legacy_amounts
            )::text,
            'legacy_subscription_logs', s.legacy_subscription_logs::text,
            'excluded_balance_logs', s.excluded_balance_logs::text,
            'authoritative_weekly_ledger_mismatches',
                s.authoritative_weekly_ledger_mismatches::text,
            'windows_requiring_reanchor', s.windows_requiring_reanchor::text,
            'invalid_active_plan_configuration', s.invalid_active_plan_configuration::text,
            'invalid_subscription_times', s.invalid_subscription_times::text,
            'definite_old_term_receipts', s.definite_old_term_receipts::text,
            'ambiguous_reopened_term_events', s.ambiguous_reopened_term_events::text,
            'database_writer_locks', (
                s.writer_sessions + s.waiting_locks
                    + s.idle_in_transaction_sessions
            )::text,
            'database_prepared_transactions', s.prepared_transactions::text,
            'database_writer_visibility',
                CASE WHEN s.can_observe_all_sessions THEN '1' ELSE '0' END
        ),
        'checks', COALESCE((
            SELECT JSONB_AGG(
                JSONB_BUILD_OBJECT(
                    'name', check_name,
                    'status', status,
                    'affected_rows', affected_rows::text,
                    'detail', detail
                )
                ORDER BY
                    CASE status
                        WHEN 'BLOCK' THEN 0
                        WHEN 'WARN' THEN 1
                        WHEN 'PASS' THEN 2
                        ELSE 3
                    END,
                    check_name
            )
            FROM checks
        ), '[]'::jsonb),
        'blockers', COALESCE((
            SELECT JSONB_AGG(check_name ORDER BY check_name)
            FROM checks
            WHERE status = 'BLOCK'
        ), '[]'::jsonb),
        'warnings', COALESCE((
            SELECT JSONB_AGG(check_name ORDER BY check_name)
            FROM checks
            WHERE status = 'WARN'
        ), '[]'::jsonb),
        'relation_metrics', rm.metrics
    ) AS payload
    FROM summary s
    CROSS JOIN params p
    CROSS JOIN schema_state ss
    CROSS JOIN check_totals ct
    CROSS JOIN relation_metrics rm
)
SELECT payload::text
FROM result;
