-- 188_chat_admin_content_access_logs.sql
-- Immutable, append-only audit snapshots for administrator access to Web Chat
-- message bodies. Identity columns intentionally have no foreign keys so user,
-- administrator, or conversation deletion cannot erase the access trail.

CREATE TABLE IF NOT EXISTS chat_admin_content_access_logs (
    id BIGSERIAL PRIMARY KEY,
    admin_id BIGINT NOT NULL,
    target_user_id BIGINT NOT NULL,
    conversation_id BIGINT NOT NULL,
    conversation_public_id VARCHAR(80) NOT NULL,
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    request_id VARCHAR(64) NOT NULL DEFAULT '',
    client_ip VARCHAR(64) NOT NULL DEFAULT '',
    user_agent VARCHAR(512) NOT NULL DEFAULT '',
    before_position BIGINT,
    page_limit INTEGER NOT NULL,
    CONSTRAINT chk_chat_admin_content_access_page_limit
        CHECK (page_limit BETWEEN 1 AND 100)
);

CREATE INDEX IF NOT EXISTS idx_chat_admin_content_access_admin_viewed
    ON chat_admin_content_access_logs (admin_id, viewed_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_chat_admin_content_access_target_viewed
    ON chat_admin_content_access_logs (target_user_id, viewed_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_chat_admin_content_access_conversation_viewed
    ON chat_admin_content_access_logs (conversation_id, viewed_at DESC, id DESC);

COMMENT ON TABLE chat_admin_content_access_logs IS
    'Append-only audit trail for administrator reads of Web Chat message bodies';
