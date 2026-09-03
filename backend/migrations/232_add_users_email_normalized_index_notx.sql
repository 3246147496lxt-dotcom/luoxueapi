-- The alias guard also matches non-Gmail provider addresses after trimming
-- whitespace/case and trailing DNS root dots.  Keep that expression indexed so
-- public registration and send-code checks cannot turn into unbounded scans as
-- the users table grows.  This migration is intentionally non-transactional:
-- PostgreSQL requires CREATE INDEX CONCURRENTLY outside a transaction.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email_normalized
    ON users ((RTRIM(LOWER(TRIM(email)), '.')) text_pattern_ops)
    WHERE deleted_at IS NULL;
