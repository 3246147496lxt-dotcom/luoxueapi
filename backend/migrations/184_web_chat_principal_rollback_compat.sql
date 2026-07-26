-- Web-chat billing principals must remain invisible to binaries that predate
-- the purpose column. Those binaries universally exclude soft-deleted API keys
-- from management and authentication, while usage_logs can still reference the
-- retained row by ID.

-- Keep one active principal for each user/group pair before installing the
-- broader unique index. Prefer the currently visible row when upgrading from
-- migration 183, then the oldest durable identity.
WITH ranked_active_principals AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY user_id, group_id
            ORDER BY (deleted_at IS NULL) DESC, created_at ASC, id ASC
        ) AS principal_rank
    FROM api_keys
    WHERE purpose = 'web_chat'
      AND status = 'active'
)
UPDATE api_keys AS candidate
SET status = 'disabled',
    updated_at = CURRENT_TIMESTAMP
FROM ranked_active_principals AS ranked
WHERE candidate.id = ranked.id
  AND ranked.principal_rank > 1;

-- Hide both active and disabled historical web-chat rows from old binaries.
UPDATE api_keys
SET deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE purpose = 'web_chat'
  AND deleted_at IS NULL;

ALTER TABLE api_keys
    DROP CONSTRAINT IF EXISTS api_keys_web_chat_hidden_check;

-- Enforce rollback compatibility at the database boundary, not only in the
-- current repository implementation.
ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_web_chat_hidden_check
    CHECK (purpose <> 'web_chat' OR deleted_at IS NOT NULL);
