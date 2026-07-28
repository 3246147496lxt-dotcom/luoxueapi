ALTER TABLE desktop_devices
    ADD COLUMN IF NOT EXISTS release_channel VARCHAR(20) NOT NULL DEFAULT 'stable';

ALTER TABLE desktop_devices
    DROP CONSTRAINT IF EXISTS desktop_devices_release_channel_check;
ALTER TABLE desktop_devices
    ADD CONSTRAINT desktop_devices_release_channel_check
    CHECK (release_channel IN ('internal', 'stable'));

CREATE TABLE IF NOT EXISTS desktop_diagnostics (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES desktop_devices(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_id UUID NOT NULL UNIQUE,
    app_version VARCHAR(50) NOT NULL,
    platform VARCHAR(20) NOT NULL,
    architecture VARCHAR(20) NOT NULL,
    os_version VARCHAR(50) NOT NULL,
    gateway_status VARCHAR(20) NOT NULL,
    codex_config_status VARCHAR(20) NOT NULL,
    request_sample_count INTEGER NOT NULL DEFAULT 0,
    request_error_count INTEGER NOT NULL DEFAULT 0,
    encrypted_payload TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT desktop_diagnostics_platform_check CHECK (platform = 'macos')
);

CREATE INDEX IF NOT EXISTS idx_desktop_diagnostics_device_created
    ON desktop_diagnostics(device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_desktop_diagnostics_user_created
    ON desktop_diagnostics(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_desktop_diagnostics_expires_at
    ON desktop_diagnostics(expires_at);
