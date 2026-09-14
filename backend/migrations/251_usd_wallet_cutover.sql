-- Convert the legacy wallet unit (points) to USD.
--
-- This migration is intentionally guarded by the migration runner's schema
-- origin.  A database created from the complete migration set already uses
-- USD and must not be divided again.  The production database was bootstrapped
-- before the migration runner recorded its origin, so it is the only target
-- that receives the conversion.

DO $usd_wallet_cutover$
DECLARE
    origin TEXT;
    marker TEXT;
    audit_row RECORD;
    payload JSONB;
    converted_payload JSONB;
    key_name TEXT;
    numeric_value NUMERIC;
BEGIN
    INSERT INTO settings (key, value, updated_at)
    VALUES ('billing_unit_migration_v1', 'pending', NOW())
    ON CONFLICT (key) DO NOTHING;

    SELECT value
    INTO marker
    FROM settings
    WHERE key = 'billing_unit_migration_v1';

    SELECT state_value
    INTO origin
    FROM schema_migration_runner_state
    WHERE state_key = 'schema_origin';

    IF marker IN ('usd_div10', 'skipped_fresh') THEN
        RETURN;
    END IF;

    IF origin IS DISTINCT FROM 'legacy' THEN
        UPDATE settings
        SET value = 'skipped_fresh', updated_at = NOW()
        WHERE key = 'billing_unit_migration_v1';
        RETURN;
    END IF;

    -- User wallet state and wallet-derived thresholds/defaults.
    UPDATE users
    SET balance = ROUND(balance / 10, 8),
        frozen_balance = ROUND(COALESCE(frozen_balance, 0) / 10, 8),
        total_recharged = ROUND(COALESCE(total_recharged, 0) / 10, 8),
        balance_notify_threshold = CASE
            WHEN balance_notify_threshold IS NULL THEN NULL
            ELSE ROUND(balance_notify_threshold / 10, 8)
        END,
        updated_at = NOW();

    UPDATE settings
    SET value = CASE
            WHEN value ~ '^-?[0-9]+([.][0-9]+)?$'
                THEN ROUND(value::NUMERIC / 10, 8)::TEXT
            ELSE value
        END,
        updated_at = NOW()
    WHERE key IN (
        'default_balance',
        'balance_low_notify_threshold',
        'auth_source_default_email_balance',
        'auth_source_default_linuxdo_balance',
        'auth_source_default_oidc_balance',
        'auth_source_default_wechat_balance',
        'auth_source_default_github_balance',
        'auth_source_default_google_balance',
        'auth_source_default_dingtalk_balance'
    );

    -- The payment gateway still receives CNY.  Only the credited wallet
    -- amount and its refund amount use the legacy points unit.
    UPDATE payment_orders
    SET amount = ROUND(amount / 10, 2),
        refund_amount = ROUND(refund_amount / 10, 2),
        updated_at = NOW()
    WHERE order_type = 'balance';

    UPDATE settings
    SET value = '1', updated_at = NOW()
    WHERE key = 'BALANCE_RECHARGE_MULTIPLIER';

    UPDATE redeem_codes
    SET value = ROUND(value / 10, 8)
    WHERE type IN ('balance', 'admin_balance');

    UPDATE promo_codes
    SET bonus_amount = ROUND(bonus_amount / 10, 8), updated_at = NOW();

    UPDATE promo_code_usages
    SET bonus_amount = ROUND(bonus_amount / 10, 8);

    -- Affiliate rebates earned from a balance recharge were stored in the
    -- legacy wallet unit.  A subscription rebate is already a USD quota
    -- amount (subscription plans are priced in USD), so keep those rows and
    -- their contribution to the aggregate affiliate balances unchanged.
    WITH subscription_parts AS (
        SELECT
            ual.user_id,
            COALESCE(SUM(ual.amount) FILTER (
                WHERE ual.action = 'accrue' AND ual.frozen_until IS NULL
            ), 0) AS available_amount,
            COALESCE(SUM(ual.amount) FILTER (
                WHERE ual.action = 'accrue' AND ual.frozen_until IS NOT NULL
            ), 0) AS frozen_amount,
            COALESCE(SUM(ual.amount) FILTER (
                WHERE ual.action = 'accrue'
            ), 0) AS history_amount
        FROM user_affiliate_ledger ual
        JOIN payment_orders po ON po.id = ual.source_order_id
        WHERE po.order_type = 'subscription'
        GROUP BY ual.user_id
    )
    UPDATE user_affiliates ua
    SET aff_quota = ROUND((COALESCE(ua.aff_quota, 0) - COALESCE(sp.available_amount, 0)) / 10, 8)
            + COALESCE(sp.available_amount, 0),
        aff_history_quota = ROUND((COALESCE(ua.aff_history_quota, 0) - COALESCE(sp.history_amount, 0)) / 10, 8)
            + COALESCE(sp.history_amount, 0),
        aff_frozen_quota = ROUND((COALESCE(ua.aff_frozen_quota, 0) - COALESCE(sp.frozen_amount, 0)) / 10, 8)
            + COALESCE(sp.frozen_amount, 0),
        updated_at = NOW()
    FROM subscription_parts sp
    WHERE ua.user_id = sp.user_id;

    UPDATE user_affiliates ua
    SET aff_quota = ROUND(COALESCE(ua.aff_quota, 0) / 10, 8),
        aff_history_quota = ROUND(COALESCE(ua.aff_history_quota, 0) / 10, 8),
        aff_frozen_quota = ROUND(COALESCE(ua.aff_frozen_quota, 0) / 10, 8),
        updated_at = NOW()
    WHERE NOT EXISTS (
        SELECT 1
        FROM user_affiliate_ledger ual
        JOIN payment_orders po ON po.id = ual.source_order_id
        WHERE ual.user_id = ua.user_id
          AND ual.action = 'accrue'
          AND po.order_type = 'subscription'
    );

    UPDATE user_affiliate_ledger AS ual
    SET amount = ROUND(ual.amount / 10, 8),
        balance_after = CASE
            WHEN ual.balance_after IS NULL THEN NULL
            ELSE ROUND(ual.balance_after / 10, 8)
        END,
        aff_quota_after = CASE
            WHEN ual.aff_quota_after IS NULL THEN NULL
            ELSE ROUND(ual.aff_quota_after / 10, 8)
        END,
        aff_frozen_quota_after = CASE
            WHEN ual.aff_frozen_quota_after IS NULL THEN NULL
            ELSE ROUND(ual.aff_frozen_quota_after / 10, 8)
        END,
        aff_history_quota_after = CASE
            WHEN ual.aff_history_quota_after IS NULL THEN NULL
            ELSE ROUND(ual.aff_history_quota_after / 10, 8)
        END,
        updated_at = NOW()
    WHERE ual.action <> 'accrue'
       OR ual.source_order_id IS NULL
       OR EXISTS (
            SELECT 1
            FROM payment_orders po
            WHERE po.id = ual.source_order_id
              AND po.order_type = 'balance'
       );

    -- Standard balance billing receipts and usage rows also carry the old
    -- wallet unit.  Subscription rows are already USD quota accounting and
    -- deliberately remain unchanged.
    UPDATE usage_logs
    SET input_cost = ROUND(input_cost / 10, 10),
        output_cost = ROUND(output_cost / 10, 10),
        cache_creation_cost = ROUND(cache_creation_cost / 10, 10),
        cache_read_cost = ROUND(cache_read_cost / 10, 10),
        image_input_cost = ROUND(image_input_cost / 10, 10),
        image_output_cost = ROUND(image_output_cost / 10, 10),
        total_cost = ROUND(total_cost / 10, 10),
        actual_cost = ROUND(actual_cost / 10, 10)
    WHERE billing_type = 0;

    -- Dashboard aggregates do not retain billing_type.  Rebuild their cost
    -- fields from the now-normalized usage rows so subscription USD usage is
    -- preserved and account/provider cost is not accidentally divided.  A
    -- bucket with no surviving usage rows is left alone so retention does not
    -- erase aggregate-only history.
    WITH hourly AS (
        SELECT
            date_trunc('hour', created_at AT TIME ZONE current_setting('TIMEZONE')) AT TIME ZONE current_setting('TIMEZONE') AS bucket_start,
            COALESCE(SUM(total_cost), 0) AS total_cost,
            COALESCE(SUM(actual_cost), 0) AS actual_cost,
            COALESCE(SUM(COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1)), 0) AS account_cost
        FROM usage_logs
        GROUP BY 1
    )
    UPDATE usage_dashboard_hourly d
    SET total_cost = ROUND(hourly.total_cost, 10),
        actual_cost = ROUND(hourly.actual_cost, 10),
        account_cost = ROUND(hourly.account_cost, 10),
        computed_at = NOW()
    FROM hourly
    WHERE d.bucket_start = hourly.bucket_start;

    WITH daily AS (
        SELECT
            (created_at AT TIME ZONE current_setting('TIMEZONE'))::date AS bucket_date,
            COALESCE(SUM(total_cost), 0) AS total_cost,
            COALESCE(SUM(actual_cost), 0) AS actual_cost,
            COALESCE(SUM(COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1)), 0) AS account_cost
        FROM usage_logs
        GROUP BY 1
    )
    UPDATE usage_dashboard_daily d
    SET total_cost = ROUND(daily.total_cost, 10),
        actual_cost = ROUND(daily.actual_cost, 10),
        account_cost = ROUND(daily.account_cost, 10),
        computed_at = NOW()
    FROM daily
    WHERE d.bucket_date = daily.bucket_date;

    UPDATE billing_usage_entries
    SET gross_amount = ROUND(gross_amount / 10, 10),
        charged_amount = ROUND(charged_amount / 10, 10),
        delta_usd = ROUND(delta_usd / 10, 10),
        balance_before = CASE
            WHEN balance_before IS NULL THEN NULL
            ELSE ROUND(balance_before / 10, 10)
        END,
        balance_after = CASE
            WHEN balance_after IS NULL THEN NULL
            ELSE ROUND(balance_after / 10, 10)
        END
    WHERE billing_type = 0;

    -- API-key quota limits and counters are wallet-denominated spending
    -- limits.  Keep their exhaustion boundary equivalent after the wallet
    -- conversion.  Provider/account quota snapshots use a separate USD unit
    -- and are intentionally not touched here.
    UPDATE api_keys
    SET quota = ROUND(quota / 10, 8),
        quota_used = ROUND(quota_used / 10, 8),
        rate_limit_5h = ROUND(rate_limit_5h / 10, 8),
        rate_limit_1d = ROUND(rate_limit_1d / 10, 8),
        rate_limit_7d = ROUND(rate_limit_7d / 10, 8),
        usage_5h = ROUND(usage_5h / 10, 8),
        usage_1d = ROUND(usage_1d / 10, 8),
        usage_7d = ROUND(usage_7d / 10, 8),
        updated_at = NOW();

    -- User platform quotas are standard-wallet spending windows.  Their
    -- *_usd names describe the post-cutover API contract, while legacy rows
    -- were accumulated from the old points-denominated actual_cost.
    UPDATE user_platform_quotas
    SET daily_limit_usd = CASE WHEN daily_limit_usd IS NULL THEN NULL ELSE ROUND(daily_limit_usd / 10, 10) END,
        weekly_limit_usd = CASE WHEN weekly_limit_usd IS NULL THEN NULL ELSE ROUND(weekly_limit_usd / 10, 10) END,
        monthly_limit_usd = CASE WHEN monthly_limit_usd IS NULL THEN NULL ELSE ROUND(monthly_limit_usd / 10, 10) END,
        daily_usage_usd = ROUND(daily_usage_usd / 10, 10),
        weekly_usage_usd = ROUND(weekly_usage_usd / 10, 10),
        monthly_usage_usd = ROUND(monthly_usage_usd / 10, 10),
        updated_at = NOW();

    -- Batch-image holds and snapshots are charged against the same user
    -- wallet.  Convert both pending and completed jobs, including per-item
    -- billing snapshots, so retries and historical views stay consistent.
    UPDATE batch_image_jobs
    SET estimated_cost = ROUND(estimated_cost / 10, 10),
        hold_amount = CASE WHEN hold_amount IS NULL THEN NULL ELSE ROUND(hold_amount / 10, 10) END,
        actual_cost = CASE WHEN actual_cost IS NULL THEN NULL ELSE ROUND(actual_cost / 10, 10) END,
        base_unit_price = ROUND(base_unit_price / 10, 10),
        billable_unit_price = ROUND(billable_unit_price / 10, 10),
        hold_unit_price = ROUND(hold_unit_price / 10, 10),
        updated_at = NOW();

    UPDATE batch_image_items
    SET billed_amount = CASE WHEN billed_amount IS NULL THEN NULL ELSE ROUND(billed_amount / 10, 10) END;

    -- Stored channel prices were seeded as official USD prices ×70.  The new
    -- runtime baseline is ×1, so normalize both flat and interval prices.
    UPDATE channel_model_pricing
    SET input_price = CASE WHEN input_price IS NULL THEN NULL ELSE ROUND(input_price / 70, 12) END,
        output_price = CASE WHEN output_price IS NULL THEN NULL ELSE ROUND(output_price / 70, 12) END,
        cache_write_price = CASE WHEN cache_write_price IS NULL THEN NULL ELSE ROUND(cache_write_price / 70, 12) END,
        cache_read_price = CASE WHEN cache_read_price IS NULL THEN NULL ELSE ROUND(cache_read_price / 70, 12) END,
        image_input_price = CASE WHEN image_input_price IS NULL THEN NULL ELSE ROUND(image_input_price / 70, 12) END,
        image_output_price = CASE WHEN image_output_price IS NULL THEN NULL ELSE ROUND(image_output_price / 70, 8) END,
        per_request_price = CASE WHEN per_request_price IS NULL THEN NULL ELSE ROUND(per_request_price / 70, 12) END,
        updated_at = NOW();

    UPDATE channel_pricing_intervals
    SET input_price = CASE WHEN input_price IS NULL THEN NULL ELSE ROUND(input_price / 70, 12) END,
        output_price = CASE WHEN output_price IS NULL THEN NULL ELSE ROUND(output_price / 70, 12) END,
        cache_write_price = CASE WHEN cache_write_price IS NULL THEN NULL ELSE ROUND(cache_write_price / 70, 12) END,
        cache_read_price = CASE WHEN cache_read_price IS NULL THEN NULL ELSE ROUND(cache_read_price / 70, 12) END,
        per_request_price = CASE WHEN per_request_price IS NULL THEN NULL ELSE ROUND(per_request_price / 70, 12) END,
        updated_at = NOW();

    -- Payment audit details are immutable snapshots shown in the admin order
    -- view.  Convert only wallet-denominated keys; payment gateway CNY keys
    -- such as payAmount, paymentAmount, paidAmount, expected, and paid remain
    -- untouched.
    FOR audit_row IN
        SELECT pal.id, pal.detail
        FROM payment_audit_logs pal
        JOIN payment_orders po ON po.id::TEXT = pal.order_id
        WHERE po.order_type = 'balance'
    LOOP
        BEGIN
            payload := audit_row.detail::JSONB;
        EXCEPTION WHEN OTHERS THEN
            CONTINUE;
        END;

        IF jsonb_typeof(payload) <> 'object' THEN
            CONTINUE;
        END IF;

        converted_payload := payload;
        FOREACH key_name IN ARRAY ARRAY[
            'creditedAmount',
            'baseAmount',
            'rebateAmount',
            'refundAmount',
            'balanceDeducted',
            'balanceBefore',
            'balanceAfter'
        ]::TEXT[]
        LOOP
            IF jsonb_typeof(converted_payload -> key_name) = 'number' THEN
                BEGIN
                    numeric_value := (converted_payload ->> key_name)::NUMERIC;
                EXCEPTION WHEN OTHERS THEN
                    numeric_value := NULL;
                END;
                IF numeric_value IS NOT NULL THEN
                    converted_payload := jsonb_set(
                        converted_payload,
                        ARRAY[key_name],
                        to_jsonb(ROUND(numeric_value / 10, 8)),
                        TRUE
                    );
                END IF;
            END IF;
        END LOOP;

        IF converted_payload <> payload THEN
            UPDATE payment_audit_logs
            SET detail = converted_payload::TEXT
            WHERE id = audit_row.id;
        END IF;
    END LOOP;

    UPDATE settings
    SET value = 'usd_div10', updated_at = NOW()
    WHERE key = 'billing_unit_migration_v1';
END
$usd_wallet_cutover$;
