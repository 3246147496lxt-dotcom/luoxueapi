-- 186_chat_request_attempts.sql
-- Claim first-party Web Chat attempts before forwarding upstream.
--
-- The browser-provided attempt_id is scoped to the authenticated user and is
-- never used as the billing request id. client_request_id remains
-- server-generated and is the public receipt correlation id.

CREATE TABLE IF NOT EXISTS chat_request_attempts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attempt_id VARCHAR(64) NOT NULL,
    client_request_id VARCHAR(64) NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'accepted',
    http_status INTEGER,
    failure_code VARCHAR(64),
    failure_reason VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_chat_request_attempts_status
        CHECK (status IN ('accepted', 'processing', 'failed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_request_attempts_user_attempt_unique
    ON chat_request_attempts (user_id, attempt_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_request_attempts_client_request_unique
    ON chat_request_attempts (client_request_id);

CREATE INDEX IF NOT EXISTS idx_chat_request_attempts_user_created
    ON chat_request_attempts (user_id, created_at DESC);
