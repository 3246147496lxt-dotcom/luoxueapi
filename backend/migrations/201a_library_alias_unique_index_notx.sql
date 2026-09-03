-- Build the reusable-alias uniqueness guard without blocking ordinary writes
-- to the existing chat_attachments table for the duration of an index scan.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_chat_attachments_library_file_unique
    ON chat_attachments (library_file_id)
    WHERE library_file_id IS NOT NULL;
