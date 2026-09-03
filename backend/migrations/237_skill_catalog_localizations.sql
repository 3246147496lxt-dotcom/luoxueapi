-- Product-owned localized copy for the public Skill catalog.
--
-- The upstream metadata in skills remains the source of truth for imports and
-- administrator editing.  Localized catalog copy is keyed by the immutable
-- marketplace slug so importer refreshes cannot overwrite reviewed text and a
-- fresh deployment can load translations before the matching Skill is added.
CREATE TABLE IF NOT EXISTS skill_catalog_localizations (
    slug VARCHAR(64) NOT NULL,
    locale VARCHAR(16) NOT NULL,
    display_name VARCHAR(120) NOT NULL,
    summary VARCHAR(280) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (slug, locale),
    CONSTRAINT skill_catalog_localizations_slug_check
        CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT skill_catalog_localizations_locale_check
        CHECK (locale ~ '^[a-z]{2}(-[A-Z]{2})?$'),
    CONSTRAINT skill_catalog_localizations_copy_check
        CHECK (
            btrim(display_name) <> ''
            AND btrim(summary) <> ''
            AND btrim(description) <> ''
        )
);

CREATE INDEX IF NOT EXISTS idx_skill_catalog_localizations_locale_slug
    ON skill_catalog_localizations (locale, slug);
