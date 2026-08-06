-- Simplify curated Skill publication metadata and add a durable GitHub source
-- snapshot. GitHub is refreshed asynchronously; public reads never depend on
-- GitHub availability.
ALTER TABLE skills
    ADD COLUMN IF NOT EXISTS source_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_repository VARCHAR(140) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS repository_stars BIGINT NULL,
    ADD COLUMN IF NOT EXISTS repository_stars_fetched_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS repository_stars_refresh_after TIMESTAMPTZ NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'skills_source_pair_check'
    ) THEN
        ALTER TABLE skills
            ADD CONSTRAINT skills_source_pair_check CHECK (
                (
                    source_url = ''
                    AND source_repository = ''
                    AND repository_stars IS NULL
                    AND repository_stars_fetched_at IS NULL
                    AND repository_stars_refresh_after IS NULL
                )
                OR (
                    source_url <> ''
                    AND source_repository <> ''
                    AND (
                        (repository_stars IS NULL AND repository_stars_fetched_at IS NULL)
                        OR (repository_stars IS NOT NULL AND repository_stars_fetched_at IS NOT NULL)
                    )
                )
            );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'skills_source_url_check'
    ) THEN
        ALTER TABLE skills
            ADD CONSTRAINT skills_source_url_check CHECK (
                source_url = ''
                OR (
                    char_length(source_url) <= 2048
                    AND source_url ~ '^https://github[.]com/[A-Za-z0-9]([A-Za-z0-9-]{0,37}[A-Za-z0-9])?/[A-Za-z0-9][A-Za-z0-9._-]{0,99}(/tree/[^/?#[:space:]]+(/[^/?#[:space:]]+)*|/blob/[^/?#[:space:]]+/[^/?#[:space:]]+(/[^/?#[:space:]]+)*)?$'
                )
            );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'skills_source_repository_check'
    ) THEN
        ALTER TABLE skills
            ADD CONSTRAINT skills_source_repository_check CHECK (
                source_repository = ''
                OR source_repository ~ '^[A-Za-z0-9]([A-Za-z0-9-]{0,37}[A-Za-z0-9])?/[A-Za-z0-9][A-Za-z0-9._-]{0,99}$'
            );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'skills_repository_stars_check'
    ) THEN
        ALTER TABLE skills
            ADD CONSTRAINT skills_repository_stars_check CHECK (
                repository_stars IS NULL OR repository_stars >= 0
            );
    END IF;
END $$;

DROP INDEX IF EXISTS idx_skills_repository_stars_refresh;
CREATE INDEX idx_skills_repository_stars_refresh
    ON skills (repository_stars_refresh_after ASC NULLS FIRST, id ASC)
    WHERE source_url <> '' AND source_repository <> '';

-- The public detail page no longer requires hand-authored risk copy or example
-- prompts. Relax the published-row constraint before updating existing data,
-- otherwise clearing risk_notes on an already-published curated Skill would
-- violate the constraint installed by migration 196.
ALTER TABLE skills DROP CONSTRAINT IF EXISTS skills_published_metadata_check;
ALTER TABLE skills
    ADD CONSTRAINT skills_published_metadata_check CHECK (
        status <> 'published'
        OR (
            btrim(display_name) <> ''
            AND btrim(summary) <> ''
            AND btrim(description) <> ''
            AND btrim(category) <> ''
        )
    );

-- This curated Skill keeps its original upstream identity. A NULL snapshot is
-- intentional until the background refresher completes its first successful
-- GitHub request.
UPDATE skills
SET display_name = 'frontend-design',
    source_url = 'https://github.com/anthropics/skills/tree/main/skills/frontend-design',
    source_repository = 'anthropics/skills',
    repository_stars = NULL,
    repository_stars_fetched_at = NULL,
    repository_stars_refresh_after = NOW(),
    risk_notes = '',
    updated_at = NOW()
WHERE slug = 'frontend-design';
