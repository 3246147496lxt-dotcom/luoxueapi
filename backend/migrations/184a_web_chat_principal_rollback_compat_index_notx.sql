-- Tombstoned principals still need one active identity per user/group pair.
-- Drop first so a prior interrupted CREATE CONCURRENTLY cannot leave an
-- invalid same-name index that IF NOT EXISTS would otherwise skip on retry.
DROP INDEX CONCURRENTLY IF EXISTS idx_api_keys_web_chat_principal_user_group;

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_api_keys_web_chat_principal_user_group
    ON api_keys (user_id, group_id)
    WHERE purpose = 'web_chat'
      AND status = 'active';

DROP INDEX CONCURRENTLY IF EXISTS idx_api_keys_web_chat_active_user_group;
