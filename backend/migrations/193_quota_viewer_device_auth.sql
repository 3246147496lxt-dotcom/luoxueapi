CREATE TABLE IF NOT EXISTS quota_viewer_devices (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_id UUID NOT NULL UNIQUE,
    client_id VARCHAR(64) NOT NULL,
    scope VARCHAR(64) NOT NULL,
    installation_id_hash VARCHAR(64) NOT NULL,
    name VARCHAR(100) NOT NULL,
    platform VARCHAR(20) NOT NULL,
    architecture VARCHAR(20) NOT NULL,
    os_version VARCHAR(50) NOT NULL DEFAULT '',
    app_version VARCHAR(50) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    token_version BIGINT NOT NULL DEFAULT 1,
    pairing_expires_at TIMESTAMPTZ NOT NULL,
    approved_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT quota_viewer_devices_client_check
        CHECK (client_id = 'luoxue-quota-viewer'),
    CONSTRAINT quota_viewer_devices_scope_check
        CHECK (scope = 'quota:read'),
    CONSTRAINT quota_viewer_devices_status_check
        CHECK (status IN ('pending', 'active', 'revoked')),
    CONSTRAINT quota_viewer_devices_platform_check
        CHECK (platform IN ('macos', 'windows')),
    CONSTRAINT quota_viewer_devices_architecture_check
        CHECK (architecture IN ('arm64', 'x86_64')),
    CONSTRAINT quota_viewer_devices_token_version_check
        CHECK (token_version > 0),
    CONSTRAINT quota_viewer_devices_installation_hash_check
        CHECK (length(installation_id_hash) = 64)
);

CREATE INDEX IF NOT EXISTS idx_quota_viewer_devices_user_status
    ON quota_viewer_devices(user_id, status);
CREATE INDEX IF NOT EXISTS idx_quota_viewer_devices_installation_hash
    ON quota_viewer_devices(installation_id_hash);
CREATE INDEX IF NOT EXISTS idx_quota_viewer_devices_pairing_expires_at
    ON quota_viewer_devices(pairing_expires_at);

CREATE TABLE IF NOT EXISTS quota_viewer_device_sessions (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES quota_viewer_devices(id) ON DELETE CASCADE,
    family_id UUID NOT NULL,
    refresh_token_hash VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT quota_viewer_device_sessions_status_check
        CHECK (status IN ('active', 'consumed', 'revoked')),
    CONSTRAINT quota_viewer_device_sessions_hash_check
        CHECK (length(refresh_token_hash) = 64)
);

CREATE INDEX IF NOT EXISTS idx_quota_viewer_device_sessions_device_status
    ON quota_viewer_device_sessions(device_id, status);
CREATE INDEX IF NOT EXISTS idx_quota_viewer_device_sessions_family
    ON quota_viewer_device_sessions(family_id);
CREATE INDEX IF NOT EXISTS idx_quota_viewer_device_sessions_expires_at
    ON quota_viewer_device_sessions(expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_quota_viewer_device_sessions_one_active_device
    ON quota_viewer_device_sessions(device_id)
    WHERE status = 'active';
CREATE UNIQUE INDEX IF NOT EXISTS idx_quota_viewer_device_sessions_one_active_family
    ON quota_viewer_device_sessions(family_id)
    WHERE status = 'active';
