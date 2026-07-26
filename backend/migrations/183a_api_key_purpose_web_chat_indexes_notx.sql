CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_purpose
    ON api_keys (purpose);

-- A disabled or soft-deleted internal principal may be replaced, while each
-- active user/group pair has exactly one durable usage/charge foreign key.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_web_chat_active_user_group
    ON api_keys (user_id, group_id)
    WHERE purpose = 'web_chat'
      AND status = 'active'
      AND deleted_at IS NULL;
