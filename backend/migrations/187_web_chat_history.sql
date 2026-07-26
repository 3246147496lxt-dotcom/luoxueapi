-- 187_web_chat_history.sql
-- Durable, user-owned Web Chat history and monotonic multi-device sync.
--
-- Conversation/message content is deliberately isolated from usage and billing
-- records. Deleting a conversation removes its message bodies but does not
-- delete or mutate chat attempts, usage logs, or billing receipts.

CREATE TABLE IF NOT EXISTS chat_history_sync_states (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chat_history_sync_states_version_check CHECK (version >= 0)
);

CREATE TABLE IF NOT EXISTS chat_conversations (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(80) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    model VARCHAR(128) NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1,
    version BIGINT NOT NULL DEFAULT 0,
    head_message_id BIGINT,
    message_count INTEGER NOT NULL DEFAULT 0,
    create_hash VARCHAR(64) NOT NULL,
    imported_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chat_conversations_revision_check CHECK (revision > 0),
    CONSTRAINT chat_conversations_version_check CHECK (version >= 0),
    CONSTRAINT chat_conversations_message_count_check CHECK (message_count >= 0),
    CONSTRAINT chat_conversations_user_public_unique UNIQUE (user_id, public_id),
    CONSTRAINT chat_conversations_id_user_unique UNIQUE (id, user_id)
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(80) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id BIGINT NOT NULL,
    position BIGINT NOT NULL,
    role VARCHAR(16) NOT NULL,
    content TEXT NOT NULL,
    delivery_status VARCHAR(20) NOT NULL,
    requested_model VARCHAR(128),
    finish_reason VARCHAR(64),
    error_code VARCHAR(64),
    error_message VARCHAR(500),
    excluded_from_context BOOLEAN NOT NULL DEFAULT FALSE,
    superseded_by_message_id BIGINT,
    checkpoint_seq BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    terminal_at TIMESTAMPTZ,
    CONSTRAINT chat_messages_conversation_user_fkey
        FOREIGN KEY (conversation_id, user_id)
        REFERENCES chat_conversations(id, user_id)
        ON DELETE CASCADE,
    CONSTRAINT chat_messages_role_check
        CHECK (role IN ('user', 'assistant')),
    CONSTRAINT chat_messages_delivery_status_check
        CHECK (delivery_status IN (
            'pending',
            'streaming',
            'completed',
            'partial',
            'stopped',
            'interrupted',
            'error'
        )),
    CONSTRAINT chat_messages_position_check CHECK (position > 0),
    CONSTRAINT chat_messages_checkpoint_seq_check CHECK (checkpoint_seq >= 0),
    CONSTRAINT chat_messages_conversation_public_unique
        UNIQUE (conversation_id, public_id),
    CONSTRAINT chat_messages_conversation_position_unique
        UNIQUE (conversation_id, position),
    CONSTRAINT chat_messages_id_conversation_unique
        UNIQUE (id, conversation_id)
);

ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS excluded_from_context BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS superseded_by_message_id BIGINT;

ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS checkpoint_seq BIGINT NOT NULL DEFAULT 0;

ALTER TABLE chat_messages
    DROP CONSTRAINT IF EXISTS chat_messages_checkpoint_seq_check;

ALTER TABLE chat_messages
    ADD CONSTRAINT chat_messages_checkpoint_seq_check
    CHECK (checkpoint_seq >= 0);

ALTER TABLE chat_conversations
    DROP CONSTRAINT IF EXISTS chat_conversations_head_message_id_fkey;

ALTER TABLE chat_conversations
    ADD CONSTRAINT chat_conversations_head_message_id_fkey
    FOREIGN KEY (head_message_id)
    REFERENCES chat_messages(id)
    ON DELETE SET NULL;

ALTER TABLE chat_messages
    DROP CONSTRAINT IF EXISTS chat_messages_superseded_by_message_id_fkey;

ALTER TABLE chat_messages
    ADD CONSTRAINT chat_messages_superseded_by_message_id_fkey
    FOREIGN KEY (superseded_by_message_id)
    REFERENCES chat_messages(id)
    ON DELETE SET NULL;

ALTER TABLE chat_request_attempts
    ADD COLUMN IF NOT EXISTS assistant_message_id BIGINT;

ALTER TABLE chat_request_attempts
    ADD COLUMN IF NOT EXISTS terminal_at TIMESTAMPTZ;

ALTER TABLE chat_request_attempts
    ADD COLUMN IF NOT EXISTS conversation_public_id VARCHAR(80);

ALTER TABLE chat_request_attempts
    ADD COLUMN IF NOT EXISTS assistant_message_public_id VARCHAR(80);

UPDATE chat_request_attempts attempt
SET
    conversation_public_id = conversation.public_id,
    assistant_message_public_id = message.public_id
FROM chat_messages message
JOIN chat_conversations conversation
  ON conversation.id = message.conversation_id
WHERE attempt.assistant_message_id = message.id
  AND (
    attempt.conversation_public_id IS NULL
    OR attempt.assistant_message_public_id IS NULL
  );

ALTER TABLE chat_request_attempts
    DROP CONSTRAINT IF EXISTS chat_request_attempts_assistant_message_id_fkey;

ALTER TABLE chat_request_attempts
    ADD CONSTRAINT chat_request_attempts_assistant_message_id_fkey
    FOREIGN KEY (assistant_message_id)
    REFERENCES chat_messages(id)
    ON DELETE SET NULL;

ALTER TABLE chat_request_attempts
    DROP CONSTRAINT IF EXISTS chk_chat_request_attempts_status;

ALTER TABLE chat_request_attempts
    ADD CONSTRAINT chk_chat_request_attempts_status
    CHECK (status IN (
        'accepted',
        'processing',
        'completed',
        'interrupted',
        'failed'
    ));

CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_request_attempts_assistant_message_unique
    ON chat_request_attempts (assistant_message_id)
    WHERE assistant_message_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS chat_history_changes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    version BIGINT NOT NULL,
    change_type VARCHAR(20) NOT NULL,
    conversation_id BIGINT REFERENCES chat_conversations(id) ON DELETE SET NULL,
    conversation_public_id VARCHAR(80) NOT NULL,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chat_history_changes_version_check CHECK (version > 0),
    CONSTRAINT chat_history_changes_type_check
        CHECK (change_type IN ('upsert', 'deleted')),
    CONSTRAINT chat_history_changes_deleted_check
        CHECK (
            (change_type = 'deleted' AND deleted_at IS NOT NULL)
            OR (change_type = 'upsert' AND deleted_at IS NULL)
        ),
    CONSTRAINT chat_history_changes_user_version_unique UNIQUE (user_id, version)
);

CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_active_updated
    ON chat_conversations (user_id, updated_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_position
    ON chat_messages (conversation_id, position DESC);

CREATE INDEX IF NOT EXISTS idx_chat_messages_superseded_by
    ON chat_messages (superseded_by_message_id)
    WHERE superseded_by_message_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chat_history_changes_user_version
    ON chat_history_changes (user_id, version);

CREATE INDEX IF NOT EXISTS idx_chat_history_changes_user_conversation
    ON chat_history_changes (user_id, conversation_public_id, version DESC);

COMMENT ON TABLE chat_history_changes IS
    'Metadata-only Web Chat sync feed; message bodies are never copied into this table';
