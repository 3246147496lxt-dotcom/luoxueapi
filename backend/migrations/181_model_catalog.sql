-- Public model marketplace: curated metadata points at a public standard group.
-- Pricing is intentionally resolved from live channel/group configuration and is
-- never duplicated here.
CREATE TABLE IF NOT EXISTS model_catalog_models (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(160) NOT NULL UNIQUE,
    model VARCHAR(255) NOT NULL,
    platform VARCHAR(64) NOT NULL,
    metadata_model_id VARCHAR(255) NOT NULL DEFAULT '',
    display_name_zh VARCHAR(160) NOT NULL DEFAULT '',
    display_name_en VARCHAR(160) NOT NULL DEFAULT '',
    summary_zh TEXT NOT NULL DEFAULT '',
    summary_en TEXT NOT NULL DEFAULT '',
    provider VARCHAR(80) NOT NULL DEFAULT '',
    logo_key VARCHAR(80) NOT NULL DEFAULT '',
    category VARCHAR(80) NOT NULL DEFAULT '',
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    context_window BIGINT NULL,
    max_output_tokens BIGINT NULL,
    public_group_id BIGINT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    featured BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    published_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT model_catalog_models_status_check
        CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT model_catalog_models_platform_model_key UNIQUE (platform, model),
    CONSTRAINT model_catalog_models_tags_array_check CHECK (jsonb_typeof(tags) = 'array'),
    CONSTRAINT model_catalog_models_capabilities_array_check CHECK (jsonb_typeof(capabilities) = 'array'),
    CONSTRAINT model_catalog_models_context_window_check CHECK (context_window IS NULL OR context_window > 0),
    CONSTRAINT model_catalog_models_max_output_tokens_check CHECK (max_output_tokens IS NULL OR max_output_tokens > 0)
);

CREATE INDEX IF NOT EXISTS idx_model_catalog_models_public_order
    ON model_catalog_models (status, featured DESC, sort_order ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_model_catalog_models_public_group_id
    ON model_catalog_models (public_group_id);
