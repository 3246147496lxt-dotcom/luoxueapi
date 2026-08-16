-- Validate separately from the transaction that adds these NOT VALID
-- constraints. This avoids retaining 201's stronger ALTER TABLE lock while
-- PostgreSQL scans the existing chat_attachments rows.
ALTER TABLE chat_attachments
    VALIDATE CONSTRAINT chat_attachments_library_file_fkey;

ALTER TABLE chat_attachments
    VALIDATE CONSTRAINT chat_attachments_library_alias_conversation_check;

ALTER TABLE chat_attachments
    VALIDATE CONSTRAINT chat_attachments_library_alias_storage_check;
