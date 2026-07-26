-- Promote the legacy reconciliation table into the durable, append-only
-- receipt source for usage charges. Usage logs remain immutable and may be
-- written after the charge transaction, so usage_log_id is deliberately
-- nullable and can be resolved by (request_id, api_key_id) at read time.

ALTER TABLE billing_usage_entries
    ALTER COLUMN usage_log_id DROP NOT NULL;

ALTER TABLE billing_usage_entries
    DROP CONSTRAINT IF EXISTS billing_usage_entries_usage_log_id_fkey;

ALTER TABLE billing_usage_entries
    ADD CONSTRAINT billing_usage_entries_usage_log_id_fkey
    FOREIGN KEY (usage_log_id) REFERENCES usage_logs(id) ON DELETE SET NULL;

ALTER TABLE billing_usage_entries
    ADD COLUMN IF NOT EXISTS request_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS request_fingerprint VARCHAR(64),
    ADD COLUMN IF NOT EXISTS source VARCHAR(20) NOT NULL DEFAULT 'api',
    ADD COLUMN IF NOT EXISTS account_id BIGINT,
    ADD COLUMN IF NOT EXISTS model VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS requested_model VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS input_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS output_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS gross_amount DECIMAL(20, 10) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS charged_amount DECIMAL(20, 10) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS balance_before DECIMAL(20, 10),
    ADD COLUMN IF NOT EXISTS balance_after DECIMAL(20, 10),
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'not_charged',
    ADD COLUMN IF NOT EXISTS overdraft BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE billing_usage_entries
    DROP CONSTRAINT IF EXISTS billing_usage_entries_account_id_fkey;

ALTER TABLE billing_usage_entries
    ADD CONSTRAINT billing_usage_entries_account_id_fkey
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL;

-- Preserve any historical rows that may have been written against migration
-- 027, even though the table was not part of the active runtime write path.
UPDATE billing_usage_entries
SET gross_amount = GREATEST(gross_amount, ABS(delta_usd)),
    charged_amount = CASE
        WHEN billing_type = 0 THEN GREATEST(charged_amount, ABS(delta_usd))
        ELSE 0
    END,
    status = CASE
        WHEN billing_type = 1 THEN 'subscription'
        WHEN ABS(delta_usd) > 0 THEN 'charged'
        ELSE 'not_charged'
    END;

ALTER TABLE billing_usage_entries
    DROP CONSTRAINT IF EXISTS billing_usage_entries_source_check,
    DROP CONSTRAINT IF EXISTS billing_usage_entries_status_check,
    DROP CONSTRAINT IF EXISTS billing_usage_entries_token_counts_check,
    DROP CONSTRAINT IF EXISTS billing_usage_entries_amounts_check,
    DROP CONSTRAINT IF EXISTS billing_usage_entries_balance_pair_check,
    DROP CONSTRAINT IF EXISTS billing_usage_entries_overdraft_check;

ALTER TABLE billing_usage_entries
    ADD CONSTRAINT billing_usage_entries_source_check
        CHECK (source IN ('api', 'web_chat')),
    ADD CONSTRAINT billing_usage_entries_status_check
        CHECK (status IN ('charged', 'not_charged', 'subscription')),
    ADD CONSTRAINT billing_usage_entries_token_counts_check
        CHECK (
            input_tokens >= 0
            AND output_tokens >= 0
            AND cache_creation_tokens >= 0
            AND cache_read_tokens >= 0
        ),
    ADD CONSTRAINT billing_usage_entries_amounts_check
        CHECK (gross_amount >= 0 AND charged_amount >= 0),
    ADD CONSTRAINT billing_usage_entries_balance_pair_check
        CHECK ((balance_before IS NULL) = (balance_after IS NULL)),
    ADD CONSTRAINT billing_usage_entries_overdraft_check
        CHECK (NOT overdraft OR status = 'charged');

CREATE UNIQUE INDEX IF NOT EXISTS billing_usage_entries_request_api_key_unique
    ON billing_usage_entries (request_id, api_key_id)
    WHERE request_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_billing_usage_entries_request_id
    ON billing_usage_entries (request_id)
    WHERE request_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_billing_usage_entries_admin_filters
    ON billing_usage_entries (source, status, created_at DESC, id DESC);

COMMENT ON TABLE billing_usage_entries IS
    'Append-only usage charge receipts; usage details are snapshots and usage_log_id may be linked asynchronously';

