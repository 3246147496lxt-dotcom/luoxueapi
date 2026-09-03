-- 201_library_files.sql
-- Durable, user-owned file library metadata. File bodies remain in private
-- object storage; PostgreSQL stores only metadata and integrity information.

CREATE TABLE IF NOT EXISTS library_files (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(80) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    original_name VARCHAR(255) NOT NULL,
    storage_kind VARCHAR(16) NOT NULL,
    category VARCHAR(16) NOT NULL,
    file_type VARCHAR(24) NOT NULL,
    source VARCHAR(16) NOT NULL DEFAULT 'uploaded',
    -- Stable producer identity for generated files.  It deliberately survives
    -- a user soft-delete so a retried producer cannot resurrect that file.
    source_key VARCHAR(512),
    mime_type VARCHAR(100) NOT NULL,
    extension VARCHAR(16) NOT NULL,
    byte_size BIGINT NOT NULL,
    stored_size BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL,
    storage_key TEXT,
    page_count INTEGER,
    width INTEGER,
    height INTEGER,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    last_used_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    cleanup_claimed_at TIMESTAMPTZ,
    cleaned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT library_files_public_unique UNIQUE (public_id),
    -- PostgreSQL requires an exact unique key for the composite ownership FK
    -- from chat_attachments. Keeping user_id in that FK prevents an alias from
    -- ever resolving a library row owned by another user.
    CONSTRAINT library_files_id_user_unique UNIQUE (id, user_id),
    CONSTRAINT library_files_storage_kind_check CHECK (
        storage_kind IN ('jpeg', 'png', 'webp', 'pdf', 'docx', 'xlsx', 'pptx', 'txt', 'md', 'csv', 'json')
    ),
    CONSTRAINT library_files_category_check CHECK (category IN ('image', 'file')),
    CONSTRAINT library_files_file_type_check CHECK (
        file_type IN ('image', 'pdf', 'document', 'spreadsheet', 'presentation', 'other')
    ),
    CONSTRAINT library_files_source_check CHECK (source IN ('uploaded', 'generated')),
    CONSTRAINT library_files_source_key_check CHECK (source_key IS NULL OR source = 'generated'),
    CONSTRAINT library_files_status_check CHECK (status IN ('pending', 'ready', 'deleted')),
    CONSTRAINT library_files_byte_size_check CHECK (byte_size > 0),
    CONSTRAINT library_files_stored_size_check CHECK (stored_size > 0),
    CONSTRAINT library_files_sha256_check CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    -- Public/API metadata stores canonical extensions without a leading dot
    -- (for example "pdf"), matching LibraryService and the frontend contract.
    CONSTRAINT library_files_extension_check CHECK (extension ~ '^[a-z0-9]{1,10}$'),
    CONSTRAINT library_files_page_count_check CHECK (page_count IS NULL OR page_count > 0),
    CONSTRAINT library_files_width_check CHECK (width IS NULL OR width > 0),
    CONSTRAINT library_files_height_check CHECK (height IS NULL OR height > 0),
    CONSTRAINT library_files_deleted_state_check CHECK (
        (status = 'deleted' AND deleted_at IS NOT NULL) OR
        (status <> 'deleted' AND deleted_at IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_library_files_source_key_unique
    ON library_files (source_key)
    WHERE source_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_library_files_user_status_updated
    ON library_files (user_id, status, updated_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_library_files_user_category_updated
    ON library_files (user_id, category, updated_at DESC, id DESC)
    WHERE status = 'ready';

CREATE INDEX IF NOT EXISTS idx_library_files_user_type_updated
    ON library_files (user_id, file_type, updated_at DESC, id DESC)
    WHERE status = 'ready';

CREATE INDEX IF NOT EXISTS idx_library_files_user_source_updated
    ON library_files (user_id, source, updated_at DESC, id DESC)
    WHERE status = 'ready';

CREATE INDEX IF NOT EXISTS idx_library_files_user_name
    ON library_files (user_id, LOWER(original_name), id)
    WHERE status = 'ready';

CREATE INDEX IF NOT EXISTS idx_library_files_user_size
    ON library_files (user_id, byte_size, id)
    WHERE status = 'ready';

CREATE INDEX IF NOT EXISTS idx_library_files_usage
    ON library_files (user_id) INCLUDE (stored_size)
    WHERE status IN ('pending', 'ready');

CREATE INDEX IF NOT EXISTS idx_library_files_cleanup
    ON library_files (deleted_at, id)
    WHERE status = 'deleted' AND storage_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_library_files_pending_cleanup
    ON library_files (created_at, id)
    WHERE status = 'pending' AND storage_key IS NOT NULL;

-- Ingestion is intentionally downstream of billing settlement.  This small
-- state row gives completed batch jobs bounded, observable best-effort retries
-- without making library failures part of the financial transaction.
CREATE TABLE IF NOT EXISTS batch_image_library_ingests (
    job_id VARCHAR(64) PRIMARY KEY REFERENCES batch_image_jobs(batch_id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    imported_count INTEGER NOT NULL DEFAULT 0,
    suppressed_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    last_error_code VARCHAR(128),
    last_error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT batch_image_library_ingests_status_check CHECK (
        status IN ('pending', 'retrying', 'completed', 'completed_with_errors')
    ),
    CONSTRAINT batch_image_library_ingests_attempts_check CHECK (attempts >= 0),
    CONSTRAINT batch_image_library_ingests_counts_check CHECK (
        imported_count >= 0 AND suppressed_count >= 0 AND failed_count >= 0
    )
);

CREATE INDEX IF NOT EXISTS idx_batch_image_library_ingests_retrying
    ON batch_image_library_ingests (updated_at, job_id)
    WHERE status IN ('pending', 'retrying');

-- A library file gets one reusable Chat attachment alias. Put alterations of
-- this existing, write-heavy table last so its ACCESS EXCLUSIVE lock is held
-- only for these metadata changes and the final transaction commit.
ALTER TABLE chat_attachments
    ADD COLUMN IF NOT EXISTS library_file_id BIGINT;

ALTER TABLE chat_attachments
    DROP CONSTRAINT IF EXISTS chat_attachments_library_file_fkey;

ALTER TABLE chat_attachments
    ADD CONSTRAINT chat_attachments_library_file_fkey
        FOREIGN KEY (library_file_id, user_id)
        REFERENCES library_files(id, user_id)
        ON DELETE SET NULL (library_file_id)
        NOT VALID;

ALTER TABLE chat_attachments
    DROP CONSTRAINT IF EXISTS chat_attachments_library_alias_conversation_check;

ALTER TABLE chat_attachments
    ADD CONSTRAINT chat_attachments_library_alias_conversation_check CHECK (
        library_file_id IS NULL OR conversation_id IS NULL
    ) NOT VALID;

ALTER TABLE chat_attachments
    DROP CONSTRAINT IF EXISTS chat_attachments_library_alias_storage_check;

ALTER TABLE chat_attachments
    ADD CONSTRAINT chat_attachments_library_alias_storage_check CHECK (
        library_file_id IS NULL OR storage_key IS NULL
    ) NOT VALID;

COMMENT ON TABLE library_files IS
    'Private durable file-library metadata; file bodies live in private object storage';
COMMENT ON COLUMN chat_attachments.library_file_id IS
    'Optional reusable alias target; non-NULL aliases must keep conversation_id NULL';
COMMENT ON COLUMN library_files.source_key IS
    'Stable generated-file producer identity; retained after user soft-delete to suppress resurrection';
COMMENT ON TABLE batch_image_library_ingests IS
    'Bounded best-effort ingestion state for durable batch-image results entering the file library';
