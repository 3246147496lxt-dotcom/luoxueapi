-- 189_web_chat_attempt_lease.sql
-- Bound first-party Web Chat processing attempts so a crashed handler cannot
-- leave a conversation head permanently streaming.

ALTER TABLE chat_request_attempts
    ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ;

COMMENT ON COLUMN chat_request_attempts.lease_expires_at IS
    'Renewable processing lease for server-owned Web Chat attempts; terminal attempts clear it';
