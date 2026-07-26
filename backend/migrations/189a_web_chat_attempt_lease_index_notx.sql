CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_chat_request_attempts_processing_lease
    ON chat_request_attempts (lease_expires_at ASC NULLS FIRST, id ASC)
    WHERE status = 'processing'
      AND conversation_public_id IS NOT NULL
      AND assistant_message_public_id IS NOT NULL;
