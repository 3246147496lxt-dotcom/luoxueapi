-- Curated Codex skill marketplace. Small, validated ZIP artifacts are kept in
-- PostgreSQL so the MVP has no external object-storage dependency.
CREATE TABLE IF NOT EXISTS skills (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(64) NOT NULL UNIQUE,
    display_name VARCHAR(120) NOT NULL,
    summary VARCHAR(280) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    category VARCHAR(80) NOT NULL DEFAULT '',
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    icon VARCHAR(160) NOT NULL DEFAULT '',
    example_prompts JSONB NOT NULL DEFAULT '[]'::jsonb,
    risk_notes TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    featured BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    current_version_id BIGINT NULL,
    published_at TIMESTAMPTZ NULL,
    archived_at TIMESTAMPTZ NULL,
    created_by BIGINT NULL,
    updated_by BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT skills_slug_check
        CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    CONSTRAINT skills_status_check
        CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT skills_published_current_version_check
        CHECK (status <> 'published' OR current_version_id IS NOT NULL),
    CONSTRAINT skills_tags_array_check CHECK (jsonb_typeof(tags) = 'array'),
    CONSTRAINT skills_example_prompts_array_check CHECK (jsonb_typeof(example_prompts) = 'array')
);

CREATE TABLE IF NOT EXISTS skill_versions (
    id BIGSERIAL PRIMARY KEY,
    skill_id BIGINT NOT NULL REFERENCES skills(id) ON DELETE RESTRICT,
    version VARCHAR(64) NOT NULL,
    changelog TEXT NOT NULL DEFAULT '',
    manifest_name VARCHAR(64) NOT NULL,
    manifest_description VARCHAR(1024) NOT NULL,
    skill_md TEXT NOT NULL,
    package_data BYTEA NOT NULL,
    sha256 CHAR(64) NOT NULL,
    byte_size BIGINT NOT NULL,
    unpacked_size BIGINT NOT NULL,
    file_count INTEGER NOT NULL,
    file_manifest JSONB NOT NULL,
    validation_report JSONB NOT NULL,
    download_count BIGINT NOT NULL DEFAULT 0,
    released_at TIMESTAMPTZ NULL,
    released_by BIGINT NULL,
    yanked_at TIMESTAMPTZ NULL,
    yanked_by BIGINT NULL,
    created_by BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT skill_versions_skill_version_key UNIQUE (skill_id, version),
    CONSTRAINT skill_versions_skill_sha256_key UNIQUE (skill_id, sha256),
    CONSTRAINT skill_versions_id_skill_key UNIQUE (id, skill_id),
    CONSTRAINT skill_versions_version_check CHECK (
        version ~ '^[0-9]+\.[0-9]+\.[0-9]+[-+0-9A-Za-z.]*$'
    ),
    CONSTRAINT skill_versions_sha256_check CHECK (sha256 ~ '^[a-f0-9]{64}$'),
    CONSTRAINT skill_versions_byte_size_check
        CHECK (byte_size > 0 AND byte_size <= 5242880 AND octet_length(package_data) = byte_size),
    CONSTRAINT skill_versions_unpacked_size_check
        CHECK (unpacked_size > 0 AND unpacked_size <= 5242880),
    CONSTRAINT skill_versions_file_count_check
        CHECK (file_count > 0 AND file_count <= 100),
    CONSTRAINT skill_versions_file_manifest_array_check
        CHECK (jsonb_typeof(file_manifest) = 'array'),
    CONSTRAINT skill_versions_validation_report_object_check
        CHECK (jsonb_typeof(validation_report) = 'object'),
    CONSTRAINT skill_versions_download_count_check CHECK (download_count >= 0)
);

-- Keep the public metadata contract true even when a draft update races with
-- publication. Service validation gives useful errors; this storage invariant
-- is the final line of defense for every writer.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'skills_published_metadata_check'
    ) THEN
        ALTER TABLE skills
            ADD CONSTRAINT skills_published_metadata_check CHECK (
                status <> 'published'
                OR (
                    btrim(display_name) <> ''
                    AND btrim(summary) <> ''
                    AND btrim(description) <> ''
                    AND btrim(category) <> ''
                    AND btrim(risk_notes) <> ''
                    AND jsonb_array_length(example_prompts) > 0
                )
            );
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'skills_current_version_same_skill_fk'
    ) THEN
        ALTER TABLE skills
            ADD CONSTRAINT skills_current_version_same_skill_fk
            FOREIGN KEY (current_version_id, id)
            REFERENCES skill_versions (id, skill_id)
            DEFERRABLE INITIALLY IMMEDIATE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_skills_public_order
    ON skills (status, featured DESC, sort_order ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_skills_category_public_order
    ON skills (category, status, featured DESC, sort_order ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_skill_versions_public_history
    ON skill_versions (skill_id, released_at DESC, id DESC)
    WHERE released_at IS NOT NULL AND yanked_at IS NULL;

-- Artifact bytes and validation evidence are immutable. Release/yank markers
-- and counters are the only fields intentionally changed after insertion.
CREATE OR REPLACE FUNCTION enforce_skill_version_immutable_content()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.skill_id IS DISTINCT FROM OLD.skill_id
       OR NEW.version IS DISTINCT FROM OLD.version
       OR NEW.changelog IS DISTINCT FROM OLD.changelog
       OR NEW.manifest_name IS DISTINCT FROM OLD.manifest_name
       OR NEW.manifest_description IS DISTINCT FROM OLD.manifest_description
       OR NEW.skill_md IS DISTINCT FROM OLD.skill_md
       OR NEW.package_data IS DISTINCT FROM OLD.package_data
       OR NEW.sha256 IS DISTINCT FROM OLD.sha256
       OR NEW.byte_size IS DISTINCT FROM OLD.byte_size
       OR NEW.unpacked_size IS DISTINCT FROM OLD.unpacked_size
       OR NEW.file_count IS DISTINCT FROM OLD.file_count
       OR NEW.file_manifest IS DISTINCT FROM OLD.file_manifest
       OR NEW.validation_report IS DISTINCT FROM OLD.validation_report
       OR NEW.created_by IS DISTINCT FROM OLD.created_by
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'skill version content is immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_skill_versions_immutable_content ON skill_versions;
CREATE TRIGGER trg_skill_versions_immutable_content
BEFORE UPDATE ON skill_versions
FOR EACH ROW EXECUTE FUNCTION enforce_skill_version_immutable_content();

CREATE OR REPLACE FUNCTION prevent_skill_version_delete()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'skill versions are append-only and cannot be deleted';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_skill_versions_no_delete ON skill_versions;
CREATE TRIGGER trg_skill_versions_no_delete
BEFORE DELETE ON skill_versions
FOR EACH ROW EXECUTE FUNCTION prevent_skill_version_delete();
