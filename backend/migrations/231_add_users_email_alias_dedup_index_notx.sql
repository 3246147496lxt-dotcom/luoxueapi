-- Registration alias dedup probes users by the dot-stripped email form.
-- Keep this index creation non-transactional so it does not block writes.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email_dot_stripped
    ON users ((REPLACE(LOWER(TRIM(email)), '.', '')) text_pattern_ops)
    WHERE deleted_at IS NULL;

-- Read-only audit (run manually before/after rollout, this migration does not
-- delete or rewrite user data). Keep the earliest account for each inbox and
-- review later rows before any business-approved remediation:
-- SELECT MIN(id) AS first_user_id, ARRAY_AGG(id ORDER BY id) AS user_ids,
--        REPLACE(LOWER(TRIM(email)), '.', '') AS probe_identity
-- FROM users WHERE deleted_at IS NULL
-- GROUP BY REPLACE(LOWER(TRIM(email)), '.', '') HAVING COUNT(*) > 1
