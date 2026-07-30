-- Add a bounded, hash-only recovery record for quota-viewer refresh rotation.
-- The client supplies the successor token and can safely retry the exact tuple
-- after a committed response is lost. No plaintext refresh token is persisted.
ALTER TABLE quota_viewer_device_sessions
    ADD COLUMN IF NOT EXISTS rotation_id UUID,
    ADD COLUMN IF NOT EXISTS replacement_token_hash VARCHAR(64),
    ADD COLUMN IF NOT EXISTS recovery_expires_at TIMESTAMPTZ;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'quota_viewer_device_sessions_recovery_tuple_check'
          AND conrelid = 'quota_viewer_device_sessions'::regclass
    ) THEN
        ALTER TABLE quota_viewer_device_sessions
            ADD CONSTRAINT quota_viewer_device_sessions_recovery_tuple_check
            CHECK (
                (
                    rotation_id IS NULL
                    AND replacement_token_hash IS NULL
                    AND recovery_expires_at IS NULL
                )
                OR
                (
                    rotation_id IS NOT NULL
                    AND replacement_token_hash IS NOT NULL
                    AND recovery_expires_at IS NOT NULL
                    AND replacement_token_hash ~ '^[0-9a-f]{64}$'
                    AND consumed_at IS NOT NULL
                    AND recovery_expires_at > consumed_at
                    AND status IN ('consumed', 'revoked')
                )
            ) NOT VALID;
    END IF;
END
$$;

ALTER TABLE quota_viewer_device_sessions
    VALIDATE CONSTRAINT quota_viewer_device_sessions_recovery_tuple_check;

CREATE UNIQUE INDEX IF NOT EXISTS idx_quota_viewer_device_sessions_device_rotation
    ON quota_viewer_device_sessions(device_id, rotation_id)
    WHERE rotation_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_quota_viewer_device_sessions_recovery_expires
    ON quota_viewer_device_sessions(recovery_expires_at)
    WHERE recovery_expires_at IS NOT NULL;
