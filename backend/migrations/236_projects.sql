-- 236_projects.sql
-- User-owned ChatGPT-style projects and their reusable library sources.
CREATE TABLE IF NOT EXISTS chat_projects (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(80) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    icon VARCHAR(32) NOT NULL DEFAULT 'folder',
    color VARCHAR(32) NOT NULL DEFAULT 'gray',
    instructions TEXT NOT NULL DEFAULT '',
    memory_mode VARCHAR(32) NOT NULL DEFAULT 'default',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chat_projects_id_user_unique UNIQUE (id, user_id),
    CONSTRAINT chat_projects_name_check CHECK (char_length(btrim(name)) BETWEEN 1 AND 200),
    CONSTRAINT chat_projects_memory_mode_check CHECK (memory_mode IN ('default', 'project_only'))
);
CREATE INDEX IF NOT EXISTS idx_chat_projects_user_updated ON chat_projects(user_id, updated_at DESC, id DESC) WHERE deleted_at IS NULL;

ALTER TABLE chat_conversations ADD COLUMN IF NOT EXISTS project_id BIGINT;
ALTER TABLE chat_conversations DROP CONSTRAINT IF EXISTS chat_conversations_project_fkey;
ALTER TABLE chat_conversations ADD CONSTRAINT chat_conversations_project_fkey FOREIGN KEY (project_id) REFERENCES chat_projects(id) ON DELETE SET NULL NOT VALID;
CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_project_updated ON chat_conversations(user_id, project_id, updated_at DESC, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS chat_project_files (
    project_id BIGINT NOT NULL REFERENCES chat_projects(id) ON DELETE CASCADE,
    library_file_id BIGINT NOT NULL REFERENCES library_files(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_id, library_file_id),
    CONSTRAINT chat_project_files_user_project_fkey FOREIGN KEY (project_id, user_id) REFERENCES chat_projects(id, user_id) NOT VALID,
    CONSTRAINT chat_project_files_user_library_fkey FOREIGN KEY (library_file_id, user_id) REFERENCES library_files(id, user_id) NOT VALID
);
CREATE INDEX IF NOT EXISTS idx_chat_project_files_user ON chat_project_files(user_id, project_id, created_at DESC);
