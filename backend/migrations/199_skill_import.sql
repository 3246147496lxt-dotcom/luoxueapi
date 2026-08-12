-- Durable, resumable Skill catalog imports. Upstream discovery and artifact
-- validation run outside the database; every state transition and publication
-- decision is persisted here for audit and crash recovery.

ALTER TABLE skills
    ADD COLUMN IF NOT EXISTS origin_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS catalog_source_priority INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS catalog_source_rank INTEGER NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'skills_origin_url_check'
    ) THEN
        ALTER TABLE skills
            ADD CONSTRAINT skills_origin_url_check CHECK (
                origin_url = ''
                OR (
                    char_length(origin_url) <= 2048
                    AND origin_url ~ '^https://[^[:space:]]+$'
                )
            );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'skills_catalog_source_priority_check'
    ) THEN
        ALTER TABLE skills
            ADD CONSTRAINT skills_catalog_source_priority_check
            CHECK (catalog_source_priority BETWEEN 0 AND 1000000);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'skills_catalog_source_rank_check'
    ) THEN
        ALTER TABLE skills
            ADD CONSTRAINT skills_catalog_source_rank_check
            CHECK (catalog_source_rank IS NULL OR catalog_source_rank > 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_skills_catalog_public_order
    ON skills (
        status,
        featured DESC,
        catalog_source_priority ASC,
        catalog_source_rank ASC NULLS LAST,
        sort_order ASC,
        id ASC
    );

COMMENT ON COLUMN skills.origin_url IS
    'Canonical upstream catalog/detail URL, independent of the downloadable GitHub source_url';
COMMENT ON COLUMN skills.catalog_source_priority IS
    'Stable cross-catalog ordering bucket; lower values are displayed first';
COMMENT ON COLUMN skills.catalog_source_rank IS
    'Last accepted rank inside the upstream catalog; NULL for manually curated Skills';

CREATE TABLE IF NOT EXISTS skill_import_sources (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    adapter VARCHAR(64) NOT NULL,
    namespace VARCHAR(160) NOT NULL,
    base_url TEXT NOT NULL,
    source_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    catalog_priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT skill_import_sources_adapter_namespace_key UNIQUE (adapter, namespace),
    CONSTRAINT skill_import_sources_name_check CHECK (btrim(name) <> ''),
    CONSTRAINT skill_import_sources_adapter_check CHECK (
        adapter ~ '^[a-z0-9]+([_-][a-z0-9]+)*$'
    ),
    CONSTRAINT skill_import_sources_namespace_check CHECK (
        namespace ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{0,159}$'
    ),
    CONSTRAINT skill_import_sources_base_url_check CHECK (
        base_url = ''
        OR (
            char_length(base_url) <= 2048
            AND base_url ~ '^https://[^[:space:]]+$'
        )
    ),
    CONSTRAINT skill_import_sources_config_object_check CHECK (
        jsonb_typeof(source_config) = 'object'
    ),
    CONSTRAINT skill_import_sources_catalog_priority_check CHECK (
        catalog_priority BETWEEN 0 AND 1000000
    )
);

CREATE TABLE IF NOT EXISTS skill_import_schedules (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES skill_import_sources(id) ON DELETE RESTRICT,
    name VARCHAR(120) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cron_expression VARCHAR(120) NOT NULL,
    timezone VARCHAR(80) NOT NULL DEFAULT 'UTC',
    selection JSONB NOT NULL DEFAULT '{}'::jsonb,
    run_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    publish_policy VARCHAR(24) NOT NULL DEFAULT 'review',
    metadata_policy VARCHAR(24) NOT NULL DEFAULT 'refresh',
    next_run_at TIMESTAMPTZ NULL,
    last_run_at TIMESTAMPTZ NULL,
    last_run_id BIGINT NULL,
    lease_owner VARCHAR(160) NULL,
    lease_expires_at TIMESTAMPTZ NULL,
    created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT skill_import_schedules_source_name_key UNIQUE (source_id, name),
    CONSTRAINT skill_import_schedules_id_source_key UNIQUE (id, source_id),
    CONSTRAINT skill_import_schedules_name_check CHECK (btrim(name) <> ''),
    CONSTRAINT skill_import_schedules_cron_check CHECK (btrim(cron_expression) <> ''),
    CONSTRAINT skill_import_schedules_timezone_check CHECK (btrim(timezone) <> ''),
    CONSTRAINT skill_import_schedules_selection_object_check CHECK (
        jsonb_typeof(selection) = 'object'
    ),
    CONSTRAINT skill_import_schedules_run_config_object_check CHECK (
        jsonb_typeof(run_config) = 'object'
    ),
    CONSTRAINT skill_import_schedules_publish_policy_check CHECK (
        publish_policy IN ('review', 'auto_publish')
    ),
    CONSTRAINT skill_import_schedules_metadata_policy_check CHECK (
        metadata_policy IN ('create_only', 'refresh')
    ),
    CONSTRAINT skill_import_schedules_lease_pair_check CHECK (
        (lease_owner IS NULL AND lease_expires_at IS NULL)
        OR (lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS skill_import_runs (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES skill_import_sources(id) ON DELETE RESTRICT,
    schedule_id BIGINT NULL,
    parent_run_id BIGINT NULL REFERENCES skill_import_runs(id) ON DELETE RESTRICT,
    trigger_type VARCHAR(20) NOT NULL,
    mode VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    request_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    snapshot_sha256 CHAR(64) NOT NULL DEFAULT '',
    idempotency_key_hash CHAR(64) NOT NULL DEFAULT '',
    scheduled_for TIMESTAMPTZ NULL,
    requested_count INTEGER NOT NULL DEFAULT 0,
    discovered_count INTEGER NOT NULL DEFAULT 0,
    prepared_count INTEGER NOT NULL DEFAULT 0,
    created_count INTEGER NOT NULL DEFAULT 0,
    updated_count INTEGER NOT NULL DEFAULT 0,
    unchanged_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    blocked_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    published_count INTEGER NOT NULL DEFAULT 0,
    cancel_requested_at TIMESTAMPTZ NULL,
    cancel_requested_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NULL,
    lease_owner VARCHAR(160) NULL,
    lease_expires_at TIMESTAMPTZ NULL,
    heartbeat_at TIMESTAMPTZ NULL,
    last_error_code VARCHAR(100) NOT NULL DEFAULT '',
    last_error_message TEXT NOT NULL DEFAULT '',
    created_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT skill_import_runs_id_source_key UNIQUE (id, source_id),
    CONSTRAINT skill_import_runs_schedule_source_fk FOREIGN KEY (schedule_id, source_id)
        REFERENCES skill_import_schedules(id, source_id) ON DELETE RESTRICT,
    CONSTRAINT skill_import_runs_trigger_type_check CHECK (
        trigger_type IN ('manual', 'scheduled', 'retry', 'bootstrap')
    ),
    CONSTRAINT skill_import_runs_mode_check CHECK (
        mode IN ('review', 'auto_publish', 'dry_run')
    ),
    CONSTRAINT skill_import_runs_status_check CHECK (
        status IN (
            'queued', 'discovering', 'preparing', 'waiting_retry', 'ready',
            'awaiting_review', 'publishing', 'succeeded',
            'partial_succeeded', 'failed', 'cancelled'
        )
    ),
    CONSTRAINT skill_import_runs_request_config_object_check CHECK (
        jsonb_typeof(request_config) = 'object'
    ),
    CONSTRAINT skill_import_runs_snapshot_object_check CHECK (
        jsonb_typeof(snapshot) = 'object'
    ),
    CONSTRAINT skill_import_runs_snapshot_sha256_check CHECK (
        snapshot_sha256 = '' OR snapshot_sha256 ~ '^[a-f0-9]{64}$'
    ),
    CONSTRAINT skill_import_runs_idempotency_hash_check CHECK (
        idempotency_key_hash = '' OR idempotency_key_hash ~ '^[a-f0-9]{64}$'
    ),
    CONSTRAINT skill_import_runs_counts_check CHECK (
        requested_count >= 0 AND discovered_count >= 0
        AND prepared_count >= 0 AND created_count >= 0
        AND updated_count >= 0 AND unchanged_count >= 0
        AND skipped_count >= 0 AND blocked_count >= 0
        AND failed_count >= 0 AND published_count >= 0
    ),
    CONSTRAINT skill_import_runs_attempt_count_check CHECK (attempt_count >= 0),
    CONSTRAINT skill_import_runs_cancel_pair_check CHECK (
        (cancel_requested_at IS NULL AND cancel_requested_by IS NULL)
        OR cancel_requested_at IS NOT NULL
    ),
    CONSTRAINT skill_import_runs_lease_pair_check CHECK (
        (lease_owner IS NULL AND lease_expires_at IS NULL)
        OR (lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
    )
);

ALTER TABLE skill_import_schedules
    DROP CONSTRAINT IF EXISTS skill_import_schedules_last_run_fk;
ALTER TABLE skill_import_schedules
    ADD CONSTRAINT skill_import_schedules_last_run_fk
    FOREIGN KEY (last_run_id, source_id)
    REFERENCES skill_import_runs(id, source_id)
    DEFERRABLE INITIALLY IMMEDIATE;

CREATE TABLE IF NOT EXISTS skill_import_run_items (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL,
    source_id BIGINT NOT NULL REFERENCES skill_import_sources(id) ON DELETE RESTRICT,
    namespace VARCHAR(160) NOT NULL,
    external_id VARCHAR(512) NOT NULL,
    rank INTEGER NULL,
    market_slug VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    stage_action VARCHAR(24) NOT NULL DEFAULT '',
    upstream_name VARCHAR(240) NOT NULL DEFAULT '',
    origin_url TEXT NOT NULL,
    source_revision VARCHAR(240) NOT NULL DEFAULT '',
    source_content_sha256 CHAR(64) NOT NULL DEFAULT '',
    package_sha256 CHAR(64) NOT NULL DEFAULT '',
    staged_artifact JSONB NOT NULL DEFAULT '{}'::jsonb,
    staged_package_data BYTEA NULL,
    desired_skill JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    validation_report JSONB NOT NULL DEFAULT '{"valid":false,"errors":[],"warnings":[]}'::jsonb,
    provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
    license_unverified BOOLEAN NOT NULL DEFAULT FALSE,
    excluded_files JSONB NOT NULL DEFAULT '[]'::jsonb,
    warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
    skill_id BIGINT NULL REFERENCES skills(id) ON DELETE RESTRICT,
    version_id BIGINT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NULL,
    lease_owner VARCHAR(160) NULL,
    lease_expires_at TIMESTAMPTZ NULL,
    heartbeat_at TIMESTAMPTZ NULL,
    error_code VARCHAR(100) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT skill_import_run_items_run_source_fk FOREIGN KEY (run_id, source_id)
        REFERENCES skill_import_runs(id, source_id) ON DELETE RESTRICT,
    CONSTRAINT skill_import_run_items_stable_key UNIQUE (
        run_id, source_id, namespace, external_id
    ),
    CONSTRAINT skill_import_run_items_id_run_key UNIQUE (id, run_id),
    CONSTRAINT skill_import_run_items_version_skill_fk FOREIGN KEY (version_id, skill_id)
        REFERENCES skill_versions(id, skill_id)
        DEFERRABLE INITIALLY IMMEDIATE,
    CONSTRAINT skill_import_run_items_namespace_check CHECK (
        namespace ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{0,159}$'
    ),
    CONSTRAINT skill_import_run_items_external_id_check CHECK (
        btrim(external_id) <> '' AND char_length(external_id) <= 512
    ),
    CONSTRAINT skill_import_run_items_rank_check CHECK (rank IS NULL OR rank > 0),
    CONSTRAINT skill_import_run_items_slug_check CHECK (
        market_slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'
    ),
    CONSTRAINT skill_import_run_items_status_check CHECK (
        status IN (
            'queued', 'processing', 'ready', 'unchanged', 'blocked',
            'failed', 'published', 'skipped', 'cancelled'
        )
    ),
    CONSTRAINT skill_import_run_items_stage_action_check CHECK (
        stage_action IN ('', 'create', 'new_version', 'unchanged')
    ),
    CONSTRAINT skill_import_run_items_origin_url_check CHECK (
        origin_url = ''
        OR (
            char_length(origin_url) <= 2048
            AND origin_url ~ '^https://[^[:space:]]+$'
        )
    ),
    CONSTRAINT skill_import_run_items_source_sha256_check CHECK (
        source_content_sha256 = '' OR source_content_sha256 ~ '^[a-f0-9]{64}$'
    ),
    CONSTRAINT skill_import_run_items_package_sha256_check CHECK (
        package_sha256 = '' OR package_sha256 ~ '^[a-f0-9]{64}$'
    ),
    CONSTRAINT skill_import_run_items_staged_artifact_object_check CHECK (
        jsonb_typeof(staged_artifact) = 'object'
    ),
    CONSTRAINT skill_import_run_items_staged_artifact_size_check CHECK (
        octet_length(staged_artifact::text) <= 6291456
    ),
    CONSTRAINT skill_import_run_items_staged_package_size_check CHECK (
        staged_package_data IS NULL
        OR octet_length(staged_package_data) BETWEEN 1 AND 5242880
    ),
    CONSTRAINT skill_import_run_items_staged_payload_check CHECK (
        (stage_action = '' AND staged_artifact = '{}'::jsonb AND staged_package_data IS NULL)
        OR (
            stage_action = 'unchanged'
            AND staged_artifact <> '{}'::jsonb
            AND staged_package_data IS NULL
        )
        OR (
            stage_action IN ('create', 'new_version')
            AND staged_artifact <> '{}'::jsonb
            AND (
                staged_package_data IS NOT NULL
                OR status IN ('published', 'cancelled')
            )
        )
    ),
    CONSTRAINT skill_import_run_items_desired_skill_object_check CHECK (
        jsonb_typeof(desired_skill) = 'object'
    ),
    CONSTRAINT skill_import_run_items_source_payload_object_check CHECK (
        jsonb_typeof(source_payload) = 'object'
    ),
    CONSTRAINT skill_import_run_items_validation_object_check CHECK (
        jsonb_typeof(validation_report) = 'object'
    ),
    CONSTRAINT skill_import_run_items_provenance_object_check CHECK (
        jsonb_typeof(provenance) = 'object'
    ),
    CONSTRAINT skill_import_run_items_excluded_files_array_check CHECK (
        jsonb_typeof(excluded_files) = 'array'
    ),
    CONSTRAINT skill_import_run_items_warnings_array_check CHECK (
        jsonb_typeof(warnings) = 'array'
    ),
    CONSTRAINT skill_import_run_items_attempt_count_check CHECK (attempt_count >= 0),
    CONSTRAINT skill_import_run_items_lease_pair_check CHECK (
        (lease_owner IS NULL AND lease_expires_at IS NULL)
        OR (lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
    ),
    CONSTRAINT skill_import_run_items_artifact_pair_check CHECK (
        (skill_id IS NULL AND version_id IS NULL)
        OR (skill_id IS NOT NULL AND version_id IS NOT NULL)
    )
);

CREATE TABLE IF NOT EXISTS skill_origins (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES skill_import_sources(id) ON DELETE RESTRICT,
    namespace VARCHAR(160) NOT NULL,
    external_id VARCHAR(512) NOT NULL,
    skill_id BIGINT NOT NULL REFERENCES skills(id) ON DELETE RESTRICT,
    market_slug VARCHAR(64) NOT NULL,
    origin_url TEXT NOT NULL,
    source_revision VARCHAR(240) NOT NULL DEFAULT '',
    source_content_sha256 CHAR(64) NOT NULL DEFAULT '',
    first_seen_run_id BIGINT NOT NULL,
    last_seen_run_id BIGINT NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_rank INTEGER NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT skill_origins_stable_key UNIQUE (source_id, namespace, external_id),
    CONSTRAINT skill_origins_skill_source_key UNIQUE (skill_id, source_id),
    CONSTRAINT skill_origins_first_run_source_fk FOREIGN KEY (first_seen_run_id, source_id)
        REFERENCES skill_import_runs(id, source_id) ON DELETE RESTRICT,
    CONSTRAINT skill_origins_last_run_source_fk FOREIGN KEY (last_seen_run_id, source_id)
        REFERENCES skill_import_runs(id, source_id) ON DELETE RESTRICT,
    CONSTRAINT skill_origins_namespace_check CHECK (
        namespace ~ '^[A-Za-z0-9][A-Za-z0-9._:/-]{0,159}$'
    ),
    CONSTRAINT skill_origins_external_id_check CHECK (
        btrim(external_id) <> '' AND char_length(external_id) <= 512
    ),
    CONSTRAINT skill_origins_slug_check CHECK (
        market_slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'
    ),
    CONSTRAINT skill_origins_origin_url_check CHECK (
        origin_url = ''
        OR (
            char_length(origin_url) <= 2048
            AND origin_url ~ '^https://[^[:space:]]+$'
        )
    ),
    CONSTRAINT skill_origins_source_sha256_check CHECK (
        source_content_sha256 = '' OR source_content_sha256 ~ '^[a-f0-9]{64}$'
    ),
    CONSTRAINT skill_origins_last_rank_check CHECK (last_rank IS NULL OR last_rank > 0)
);

CREATE TABLE IF NOT EXISTS skill_version_origins (
    id BIGSERIAL PRIMARY KEY,
    version_id BIGINT NOT NULL REFERENCES skill_versions(id) ON DELETE RESTRICT,
    origin_id BIGINT NOT NULL REFERENCES skill_origins(id) ON DELETE RESTRICT,
    run_item_id BIGINT NOT NULL REFERENCES skill_import_run_items(id) ON DELETE RESTRICT,
    source_revision VARCHAR(240) NOT NULL DEFAULT '',
    source_content_sha256 CHAR(64) NOT NULL DEFAULT '',
    transformed BOOLEAN NOT NULL DEFAULT FALSE,
    provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT skill_version_origins_run_item_version_key UNIQUE (run_item_id, version_id),
    CONSTRAINT skill_version_origins_source_sha256_check CHECK (
        source_content_sha256 = '' OR source_content_sha256 ~ '^[a-f0-9]{64}$'
    ),
    CONSTRAINT skill_version_origins_provenance_object_check CHECK (
        jsonb_typeof(provenance) = 'object'
    )
);

CREATE TABLE IF NOT EXISTS skill_import_events (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES skill_import_runs(id) ON DELETE RESTRICT,
    run_item_id BIGINT NULL,
    level VARCHAR(12) NOT NULL DEFAULT 'info',
    event_type VARCHAR(100) NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT skill_import_events_level_check CHECK (
        level IN ('debug', 'info', 'warn', 'error')
    ),
    CONSTRAINT skill_import_events_item_run_fk FOREIGN KEY (run_item_id, run_id)
        REFERENCES skill_import_run_items(id, run_id) ON DELETE RESTRICT,
    CONSTRAINT skill_import_events_type_check CHECK (btrim(event_type) <> ''),
    CONSTRAINT skill_import_events_payload_object_check CHECK (
        jsonb_typeof(payload) = 'object'
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_skill_import_runs_schedule_slot
    ON skill_import_runs (schedule_id, scheduled_for)
    WHERE schedule_id IS NOT NULL AND scheduled_for IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_skill_import_runs_idempotency
    ON skill_import_runs (source_id, idempotency_key_hash)
    WHERE idempotency_key_hash <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_skill_import_runs_active_source
    ON skill_import_runs (source_id)
    WHERE status IN (
        'queued', 'discovering', 'preparing', 'waiting_retry',
        'ready', 'awaiting_review', 'publishing'
    );

CREATE INDEX IF NOT EXISTS idx_skill_import_schedules_due_claim
    ON skill_import_schedules (next_run_at ASC, lease_expires_at ASC NULLS FIRST, id ASC)
    WHERE enabled = TRUE AND next_run_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_skill_import_runs_worker_claim
    ON skill_import_runs (next_attempt_at ASC NULLS FIRST, lease_expires_at ASC NULLS FIRST, created_at ASC, id ASC)
    WHERE status IN ('queued', 'discovering', 'preparing', 'waiting_retry');

CREATE INDEX IF NOT EXISTS idx_skill_import_runs_source_history
    ON skill_import_runs (source_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_skill_import_run_items_worker_claim
    ON skill_import_run_items (run_id, next_attempt_at ASC NULLS FIRST, lease_expires_at ASC NULLS FIRST, rank ASC NULLS LAST, id ASC)
    WHERE status IN ('queued', 'processing');

CREATE INDEX IF NOT EXISTS idx_skill_import_run_items_run_status
    ON skill_import_run_items (run_id, status, rank ASC NULLS LAST, id ASC);

-- Deliberately non-unique: the repository serializes candidate slug selection
-- and combines these staged reservations with the owning run's active status.
-- Terminal/cancelled runs therefore stop reserving without deleting audit rows.
CREATE INDEX IF NOT EXISTS idx_skill_import_run_items_staged_slug_reservation
    ON skill_import_run_items (market_slug, run_id)
    WHERE status IN ('ready', 'unchanged')
      AND stage_action IN ('create', 'new_version');

CREATE INDEX IF NOT EXISTS idx_skill_origins_skill
    ON skill_origins (skill_id, active, id);

CREATE INDEX IF NOT EXISTS idx_skill_version_origins_version_origin
    ON skill_version_origins (version_id, origin_id, id);

CREATE INDEX IF NOT EXISTS idx_skill_import_events_run_stream
    ON skill_import_events (run_id, id ASC);

COMMENT ON TABLE skill_import_sources IS
    'Soft-disabled adapter configurations; rows are retained so historical runs remain explainable';
COMMENT ON TABLE skill_import_schedules IS
    'Recurring import definitions with database-backed leases for multi-instance scheduling';
COMMENT ON TABLE skill_import_runs IS
    'Immutable-input, resumable import executions and denormalized progress counters';
COMMENT ON TABLE skill_import_run_items IS
    'Per-upstream-Skill workflow state keyed by source_id + namespace + external_id';
COMMENT ON TABLE skill_origins IS
    'Durable mapping from upstream stable identity to one local Skill';
COMMENT ON TABLE skill_version_origins IS
    'Append-only provenance linking an immutable Skill version to its upstream item';
COMMENT ON TABLE skill_import_events IS
    'Append-only operational and audit event stream for an import run';

COMMENT ON COLUMN skill_import_sources.source_config IS
    'Adapter-specific non-secret configuration; credentials are referenced indirectly, never embedded';
COMMENT ON COLUMN skill_import_runs.snapshot IS
    'Bounded discovery snapshot or manifest sufficient to reproduce selection decisions';
COMMENT ON COLUMN skill_import_run_items.desired_skill IS
    'Validated staged public metadata applied only inside atomic eligible publication';
COMMENT ON COLUMN skill_import_run_items.validation_report IS
    'Artifact validation evidence captured before staging or publication';
COMMENT ON COLUMN skill_import_run_items.stage_action IS
    'Explicit two-phase publication classification; empty until an item is staged';
COMMENT ON COLUMN skill_import_run_items.staged_artifact IS
    'Bounded prepared artifact metadata retained for audit and atomic publication';
COMMENT ON COLUMN skill_import_run_items.staged_package_data IS
    'Bounded prepared ZIP bytes cleared after publication or immediate cancellation';

-- Safe bootstrap defaults: deployment creates the connector definition but
-- never starts unattended collection until an administrator enables the
-- schedule. Both inserts are idempotent across repaired/replayed migrations.
INSERT INTO skill_import_sources (
    name,
    adapter,
    namespace,
    base_url,
    source_config,
    catalog_priority,
    enabled
)
VALUES (
    'skills.sh All Time',
    'skills_sh',
    'skills.sh',
    'https://skills.sh',
	'{"view":"all-time","acquisition_order":["github","skills_sh_snapshot"],"download_quota_per_hour":60,"allowed_source_hosts":["open.feishu.cn","uizze.com","agent.qq.com","cli.sentry.dev"],"source_base_urls":{"open.feishu.cn":"https://open.feishu.cn","uizze.com":"https://uizze.com","agent.qq.com":"https://agent.qq.com","sentry/dev":"https://cli.sentry.dev"}}'::jsonb,
    100,
    TRUE
)
ON CONFLICT (adapter, namespace) DO NOTHING;

INSERT INTO skill_import_sources (
    name,
    adapter,
    namespace,
    base_url,
    source_config,
    catalog_priority,
    enabled
)
VALUES (
    'Manual Manifest Upload',
    'manifest',
    'manual-upload',
    '',
    '{"upload_only":true}'::jsonb,
    200,
    TRUE
)
ON CONFLICT (adapter, namespace) DO NOTHING;

INSERT INTO skill_import_schedules (
    source_id,
    name,
    enabled,
    cron_expression,
    timezone,
    selection,
    run_config,
    publish_policy,
    metadata_policy,
    next_run_at
)
SELECT
    id,
    'Daily Top 500',
    FALSE,
    '0 3 * * *',
    'Asia/Shanghai',
    '{"start_rank":1,"limit":500}'::jsonb,
    '{"safe_gate":true,"auto_publish_gate":{"require_all_valid":false,"allow_license_unverified":true}}'::jsonb,
    'auto_publish',
    'refresh',
    NULL
FROM skill_import_sources
WHERE adapter = 'skills_sh' AND namespace = 'skills.sh'
ON CONFLICT (source_id, name) DO NOTHING;

-- One durable bootstrap run lets the worker attach explicit provenance from
-- the already-published top-500 ZIPs before normal refreshes begin. The fixed
-- idempotency hash and NOT EXISTS guard make this safe on repaired databases.
INSERT INTO skill_import_runs (
    source_id,
    trigger_type,
    mode,
    status,
    request_config,
    snapshot,
    idempotency_key_hash,
    requested_count
)
SELECT
    id,
    'bootstrap',
    'dry_run',
    'queued',
    '{"bootstrap_existing":true}'::jsonb,
    '{}'::jsonb,
    'b00757a9e5d127fe24a83c84f5a48a663f18895d28b42b10cf0b769891c5f241',
    0
FROM skill_import_sources source
WHERE source.adapter = 'skills_sh'
  AND source.namespace = 'skills.sh'
  AND NOT EXISTS (
      SELECT 1 FROM skill_import_runs run
      WHERE run.source_id = source.id
        AND run.trigger_type = 'bootstrap'
  )
ON CONFLICT DO NOTHING;

CREATE OR REPLACE FUNCTION prevent_skill_import_history_delete()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION '% is retained for Skill import audit history and cannot be deleted', TG_TABLE_NAME;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_skill_import_sources_no_delete ON skill_import_sources;
CREATE TRIGGER trg_skill_import_sources_no_delete
BEFORE DELETE ON skill_import_sources
FOR EACH ROW EXECUTE FUNCTION prevent_skill_import_history_delete();

DROP TRIGGER IF EXISTS trg_skill_import_schedules_no_delete ON skill_import_schedules;
CREATE TRIGGER trg_skill_import_schedules_no_delete
BEFORE DELETE ON skill_import_schedules
FOR EACH ROW EXECUTE FUNCTION prevent_skill_import_history_delete();

DROP TRIGGER IF EXISTS trg_skill_import_runs_no_delete ON skill_import_runs;
CREATE TRIGGER trg_skill_import_runs_no_delete
BEFORE DELETE ON skill_import_runs
FOR EACH ROW EXECUTE FUNCTION prevent_skill_import_history_delete();

DROP TRIGGER IF EXISTS trg_skill_import_run_items_no_delete ON skill_import_run_items;
CREATE TRIGGER trg_skill_import_run_items_no_delete
BEFORE DELETE ON skill_import_run_items
FOR EACH ROW EXECUTE FUNCTION prevent_skill_import_history_delete();

DROP TRIGGER IF EXISTS trg_skill_origins_no_delete ON skill_origins;
CREATE TRIGGER trg_skill_origins_no_delete
BEFORE DELETE ON skill_origins
FOR EACH ROW EXECUTE FUNCTION prevent_skill_import_history_delete();

DROP TRIGGER IF EXISTS trg_skill_version_origins_no_delete ON skill_version_origins;
CREATE TRIGGER trg_skill_version_origins_no_delete
BEFORE DELETE ON skill_version_origins
FOR EACH ROW EXECUTE FUNCTION prevent_skill_import_history_delete();

CREATE OR REPLACE FUNCTION enforce_skill_import_append_only_row()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION '% is append-only and cannot be updated', TG_TABLE_NAME;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_skill_version_origins_no_update ON skill_version_origins;
CREATE TRIGGER trg_skill_version_origins_no_update
BEFORE UPDATE ON skill_version_origins
FOR EACH ROW EXECUTE FUNCTION enforce_skill_import_append_only_row();

DROP TRIGGER IF EXISTS trg_skill_import_events_no_update ON skill_import_events;
CREATE TRIGGER trg_skill_import_events_no_update
BEFORE UPDATE ON skill_import_events
FOR EACH ROW EXECUTE FUNCTION enforce_skill_import_append_only_row();
