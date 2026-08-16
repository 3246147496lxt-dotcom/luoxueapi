-- 202_chat_message_activities.sql
-- Durable OpenAI Responses reasoning-summary activity for first-party Web Chat.
-- Activity rows belong only to assistant messages and are deleted with them.

ALTER TABLE chat_request_attempts
    ADD COLUMN IF NOT EXISTS stop_requested_at TIMESTAMPTZ;

COMMENT ON COLUMN chat_request_attempts.stop_requested_at IS
    'Authenticated first-party Web Chat stop intent; wins over later disconnect finalization';

CREATE TABLE IF NOT EXISTS chat_message_activities (
    id BIGSERIAL PRIMARY KEY,
    message_id BIGINT NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    response_id VARCHAR(128) NOT NULL DEFAULT '',
    source VARCHAR(32) NOT NULL,
    activity_type VARCHAR(48) NOT NULL,
    item_id VARCHAR(128) NOT NULL DEFAULT '',
    output_index INTEGER NOT NULL,
    summary_index INTEGER NOT NULL,
    sort_order BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL,
    text TEXT NOT NULL DEFAULT '',
    sequence_start BIGINT NOT NULL DEFAULT 0,
    sequence_end BIGINT NOT NULL DEFAULT 0,
    reasoning_mode VARCHAR(32),
    reasoning_effort VARCHAR(32),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chat_message_activities_source_check
        CHECK (source = 'openai_responses'),
    CONSTRAINT chat_message_activities_type_check
        CHECK (activity_type = 'reasoning_summary'),
    CONSTRAINT chat_message_activities_status_check
        CHECK (status IN (
            'in_progress',
            'completed',
            'incomplete',
            'failed',
            'interrupted',
            'stopped',
            'disconnected'
        )),
    CONSTRAINT chat_message_activities_output_index_check CHECK (output_index >= 0),
    CONSTRAINT chat_message_activities_summary_index_check CHECK (summary_index >= 0),
    CONSTRAINT chat_message_activities_sort_order_check CHECK (sort_order > 0),
    CONSTRAINT chat_message_activities_sequence_check
        CHECK (sequence_start >= 0 AND sequence_end >= sequence_start),
    CONSTRAINT chat_message_activities_completed_at_check
        CHECK (completed_at IS NULL OR completed_at >= started_at),
    CONSTRAINT chat_message_activities_metadata_object_check
        CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT chat_message_activities_summary_part_unique
        UNIQUE (
            message_id,
            source,
            response_id,
            item_id,
            output_index,
            summary_index
        )
);

CREATE INDEX IF NOT EXISTS idx_chat_message_activities_message_order
    ON chat_message_activities (message_id, sort_order, id);

CREATE INDEX IF NOT EXISTS idx_chat_message_activities_response
    ON chat_message_activities (response_id, item_id)
    WHERE response_id <> '';

CREATE OR REPLACE FUNCTION enforce_chat_message_activity_assistant()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM chat_messages
        WHERE id = NEW.message_id
          AND role = 'assistant'
    ) THEN
        RAISE EXCEPTION 'chat message activity requires an assistant message'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_chat_message_activities_assistant
    ON chat_message_activities;

CREATE TRIGGER trg_chat_message_activities_assistant
BEFORE INSERT OR UPDATE OF message_id
ON chat_message_activities
FOR EACH ROW
EXECUTE FUNCTION enforce_chat_message_activity_assistant();

COMMENT ON TABLE chat_message_activities IS
    'Snapshot-upserted OpenAI Responses reasoning summaries attached to Web Chat assistant messages';
