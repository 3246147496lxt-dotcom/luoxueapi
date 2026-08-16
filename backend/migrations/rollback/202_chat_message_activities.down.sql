-- Manual rollback companion for 202_chat_message_activities.sql.
-- The rollback directory is intentionally excluded from the embedded forward
-- migration set. Execute this only from a reviewed maintenance runbook.

DROP TABLE IF EXISTS chat_message_activities;
DROP FUNCTION IF EXISTS enforce_chat_message_activity_assistant();
ALTER TABLE chat_request_attempts
    DROP COLUMN IF EXISTS stop_requested_at;
