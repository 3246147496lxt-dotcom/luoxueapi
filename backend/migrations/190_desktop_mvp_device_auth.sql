CREATE TABLE IF NOT EXISTS desktop_devices (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_id UUID NOT NULL UNIQUE,
    installation_id_hash VARCHAR(64) NOT NULL,
    name VARCHAR(100) NOT NULL,
    platform VARCHAR(20) NOT NULL DEFAULT 'macos',
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
    CONSTRAINT desktop_devices_status_check CHECK (status IN ('pending', 'active', 'revoked')),
    CONSTRAINT desktop_devices_platform_check CHECK (platform = 'macos')
);

CREATE INDEX IF NOT EXISTS idx_desktop_devices_user_id
    ON desktop_devices(user_id);
CREATE INDEX IF NOT EXISTS idx_desktop_devices_user_status
    ON desktop_devices(user_id, status);
CREATE INDEX IF NOT EXISTS idx_desktop_devices_installation_hash
    ON desktop_devices(installation_id_hash);
CREATE INDEX IF NOT EXISTS idx_desktop_devices_pairing_expires_at
    ON desktop_devices(pairing_expires_at);

CREATE TABLE IF NOT EXISTS desktop_device_sessions (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES desktop_devices(id) ON DELETE CASCADE,
    family_id UUID NOT NULL,
    refresh_token_hash VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT desktop_device_sessions_status_check CHECK (status IN ('active', 'consumed', 'revoked'))
);

CREATE INDEX IF NOT EXISTS idx_desktop_device_sessions_device_status
    ON desktop_device_sessions(device_id, status);
CREATE INDEX IF NOT EXISTS idx_desktop_device_sessions_family
    ON desktop_device_sessions(family_id);
CREATE INDEX IF NOT EXISTS idx_desktop_device_sessions_expires_at
    ON desktop_device_sessions(expires_at);

ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS managed_device_id BIGINT REFERENCES desktop_devices(id) ON DELETE RESTRICT;

ALTER TABLE api_keys
    DROP CONSTRAINT IF EXISTS api_keys_purpose_check;
ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_purpose_check
    CHECK (purpose IN ('user', 'web_chat', 'desktop'));

ALTER TABLE api_keys
    DROP CONSTRAINT IF EXISTS api_keys_desktop_managed_fields_check;
ALTER TABLE api_keys
    ADD CONSTRAINT api_keys_desktop_managed_fields_check
    CHECK (
        (purpose = 'desktop' AND managed_device_id IS NOT NULL AND group_id IS NOT NULL)
        OR (purpose <> 'desktop' AND managed_device_id IS NULL)
    );

CREATE INDEX IF NOT EXISTS idx_api_keys_managed_device_id
    ON api_keys(managed_device_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_desktop_device_group
    ON api_keys(managed_device_id, group_id)
    WHERE purpose = 'desktop' AND deleted_at IS NULL;
