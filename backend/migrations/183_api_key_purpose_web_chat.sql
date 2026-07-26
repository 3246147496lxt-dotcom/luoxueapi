-- Distinguish user-managed API keys from internal web-chat billing principals.
-- Existing rows remain user-managed. Web-chat keys are intentionally hidden
-- from every user/admin API-key management query.
ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS purpose VARCHAR(20) NOT NULL DEFAULT 'user';

ALTER TABLE api_keys
    DROP CONSTRAINT IF EXISTS api_keys_purpose_check;

ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_purpose_check
    CHECK (purpose IN ('user', 'web_chat'));

ALTER TABLE api_keys
    DROP CONSTRAINT IF EXISTS api_keys_web_chat_group_check;

ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_web_chat_group_check
    CHECK (purpose <> 'web_chat' OR group_id IS NOT NULL);
