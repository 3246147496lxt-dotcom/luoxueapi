-- Content fingerprint for candidate-only bootstrap/stars validation.
-- The three removed columns are the complete repository-stars runtime cache
-- projection. Every authored/business field, including updated_at, remains in
-- the digest and therefore must remain unchanged.

SELECT encode(
    sha256(
        convert_to(
            COALESCE(
                string_agg(
                    (
                        to_jsonb(skill_row) - ARRAY[
                            'repository_stars',
                            'repository_stars_fetched_at',
                            'repository_stars_refresh_after'
                        ]::TEXT[]
                    )::TEXT,
                    E'\n' ORDER BY skill_row.id
                ),
                ''
            ),
            'UTF8'
        )
    ),
    'hex'
) AS skill_content_sha256
FROM skills AS skill_row;
