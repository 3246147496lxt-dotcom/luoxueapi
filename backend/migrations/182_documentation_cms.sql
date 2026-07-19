-- Dynamic documentation CMS. The singleton is lazily seeded by the service
-- with the bundled static tutorials so existing installations never start
-- with an empty public documentation site.
CREATE TABLE IF NOT EXISTS documentation_documents (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    draft_content JSONB NULL,
    draft_version BIGINT NOT NULL DEFAULT 0 CHECK (draft_version >= 0),
    draft_updated_at TIMESTAMPTZ NULL,
    draft_updated_by BIGINT NULL,
    published_content JSONB NULL,
    published_version BIGINT NOT NULL DEFAULT 0 CHECK (published_version >= 0),
    published_at TIMESTAMPTZ NULL,
    published_by BIGINT NULL,
    CHECK (draft_content IS NULL OR jsonb_typeof(draft_content) = 'object'),
    CHECK (published_content IS NULL OR jsonb_typeof(published_content) = 'object')
);

CREATE TABLE IF NOT EXISTS documentation_revisions (
    id BIGSERIAL PRIMARY KEY,
    version BIGINT NOT NULL UNIQUE CHECK (version > 0),
    content JSONB NOT NULL CHECK (jsonb_typeof(content) = 'object'),
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_by BIGINT NULL
);

CREATE INDEX IF NOT EXISTS idx_documentation_revisions_published_at
    ON documentation_revisions (published_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS documentation_assets (
    id CHAR(64) PRIMARY KEY CHECK (id ~ '^[a-f0-9]{64}$'),
    content_type VARCHAR(32) NOT NULL CHECK (content_type IN ('image/png', 'image/jpeg', 'image/gif', 'image/webp')),
    byte_size BIGINT NOT NULL CHECK (byte_size > 0 AND byte_size <= 5242880),
    width INTEGER NOT NULL CHECK (width > 0 AND width <= 10000),
    height INTEGER NOT NULL CHECK (height > 0 AND height <= 10000),
    data BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT NULL
);
