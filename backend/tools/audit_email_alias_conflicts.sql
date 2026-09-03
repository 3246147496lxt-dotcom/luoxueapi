-- Read-only preflight for registration alias collisions.
-- Review results before any remediation; do not delete automatically.
-- Keep the earliest non-deleted account (MIN(id)) as the canonical owner and
-- manually decide what to do with later rows after checking billing/audit data.
--
-- The identity expression mirrors NormalizeEmailForAliasDedup:
--   * strip a plus suffix for every provider;
--   * ignore dots for Gmail/Googlemail and fold Googlemail to Gmail;
--   * ignore a trailing root dot on the domain.
WITH email_parts AS (
    SELECT
        id,
        email,
        split_part(LOWER(TRIM(email)), '@', 1) AS local_part,
        RTRIM(split_part(LOWER(TRIM(email)), '@', 2), '.') AS domain_part
    FROM users
    WHERE deleted_at IS NULL
), inbox_identities AS (
    SELECT
        id,
        email,
        CASE
            WHEN domain_part IN ('gmail.com', 'googlemail.com') THEN
                REPLACE(
                    CASE
                        WHEN POSITION('+' IN local_part) > 1 THEN split_part(local_part, '+', 1)
                        ELSE local_part
                    END,
                    '.', ''
                ) || '@gmail.com'
            ELSE
                (CASE
                    WHEN POSITION('+' IN local_part) > 1 THEN split_part(local_part, '+', 1)
                    ELSE local_part
                END) || '@' || domain_part
        END AS inbox_identity
    FROM email_parts
)
SELECT
    MIN(id) AS canonical_user_id,
    ARRAY_AGG(id ORDER BY id) AS conflicting_user_ids,
    ARRAY_AGG(email ORDER BY id) AS conflicting_emails,
    inbox_identity,
    COUNT(*) AS account_count
FROM inbox_identities
GROUP BY inbox_identity
HAVING COUNT(*) > 1
ORDER BY MIN(id);
