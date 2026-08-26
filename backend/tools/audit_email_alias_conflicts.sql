-- Read-only preflight for registration alias collisions.
-- Review results before any remediation; do not delete automatically.
-- Keep the earliest non-deleted account (MIN(id)) as the canonical owner and
-- manually decide what to do with later rows after checking billing/audit data.
SELECT
    MIN(id) AS canonical_user_id,
    ARRAY_AGG(id ORDER BY id) AS conflicting_user_ids,
    ARRAY_AGG(email ORDER BY id) AS conflicting_emails,
    REPLACE(LOWER(TRIM(email)), '.', '') AS dot_stripped_identity,
    COUNT(*) AS account_count
FROM users
WHERE deleted_at IS NULL
GROUP BY REPLACE(LOWER(TRIM(email)), '.', '')
HAVING COUNT(*) > 1
ORDER BY MIN(id);
