-- 198_web_chat_attachments.sql
-- Private, user-owned Web Chat attachments. Binary payloads are kept outside
-- PostgreSQL; this schema stores only metadata, integrity digests and bounded
-- extracted text used to assemble model context.

CREATE TABLE IF NOT EXISTS chat_attachments (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(80) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id BIGINT,
    original_name VARCHAR(255) NOT NULL,
    kind VARCHAR(16) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    byte_size BIGINT NOT NULL,
    stored_size BIGINT NOT NULL DEFAULT 0,
    sha256 VARCHAR(64) NOT NULL,
    storage_key TEXT,
    extracted_text TEXT,
    page_count INTEGER,
    width INTEGER,
    height INTEGER,
    status VARCHAR(16) NOT NULL DEFAULT 'ready',
    expires_at TIMESTAMPTZ NOT NULL,
    cleaned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chat_attachments_conversation_user_fkey
        FOREIGN KEY (conversation_id, user_id)
        REFERENCES chat_conversations(id, user_id)
        ON DELETE CASCADE,
    CONSTRAINT chat_attachments_user_public_unique UNIQUE (user_id, public_id),
    CONSTRAINT chat_attachments_kind_check CHECK (kind IN ('image', 'pdf', 'docx')),
    CONSTRAINT chat_attachments_status_check CHECK (status IN ('pending', 'ready', 'expired', 'deleted')),
    CONSTRAINT chat_attachments_byte_size_check CHECK (byte_size > 0),
    CONSTRAINT chat_attachments_stored_size_check CHECK (stored_size >= 0),
    CONSTRAINT chat_attachments_page_count_check CHECK (page_count IS NULL OR page_count > 0),
    CONSTRAINT chat_attachments_width_check CHECK (width IS NULL OR width > 0),
    CONSTRAINT chat_attachments_height_check CHECK (height IS NULL OR height > 0)
);

CREATE TABLE IF NOT EXISTS chat_message_attachments (
    message_id BIGINT NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    attachment_id BIGINT NOT NULL REFERENCES chat_attachments(id) ON DELETE CASCADE,
    position SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, attachment_id),
    CONSTRAINT chat_message_attachments_message_position_unique UNIQUE (message_id, position),
    CONSTRAINT chat_message_attachments_position_check CHECK (position BETWEEN 1 AND 4)
);

CREATE INDEX IF NOT EXISTS idx_chat_attachments_expiry_cleanup
    ON chat_attachments (expires_at, id)
    WHERE status = 'ready';

CREATE INDEX IF NOT EXISTS idx_chat_attachments_pending_blob_cleanup
    ON chat_attachments (updated_at, id)
    WHERE status IN ('pending', 'expired', 'deleted') AND storage_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chat_attachments_conversation
    ON chat_attachments (user_id, conversation_id, id)
    WHERE conversation_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_chat_message_attachments_attachment
    ON chat_message_attachments (attachment_id);

COMMENT ON TABLE chat_attachments IS
    'Private Web Chat attachment metadata; binary payloads live in private attachment storage';
