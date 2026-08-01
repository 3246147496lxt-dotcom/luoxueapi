package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	openaipkg "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

const desktopDeviceLimit = 5

type desktopRepository struct {
	db *sql.DB
}

func NewDesktopRepository(db *sql.DB) desktop.Repository {
	return &desktopRepository{db: db}
}

func NewDesktopCleanupRepository(db *sql.DB) desktop.CleanupRepository {
	return &desktopRepository{db: db}
}

func (r *desktopRepository) ReserveDevice(ctx context.Context, userID int64, pairing desktop.Pairing) (*desktop.Device, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin desktop device reservation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", desktopUserLockID(userID)); err != nil {
		return nil, fmt.Errorf("lock desktop device reservations: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM desktop_devices
		WHERE user_id = $1 AND status = 'pending' AND pairing_expires_at <= CURRENT_TIMESTAMP`, userID); err != nil {
		return nil, fmt.Errorf("remove expired desktop device reservations: %w", err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM desktop_devices
		WHERE user_id = $1
		  AND (status = 'active' OR (status = 'pending' AND pairing_expires_at > CURRENT_TIMESTAMP))`, userID).Scan(&count); err != nil {
		return nil, fmt.Errorf("count desktop devices: %w", err)
	}
	if count >= desktopDeviceLimit {
		return nil, desktop.ErrDeviceLimit
	}

	publicID := uuid.NewString()
	releaseChannel := pairing.ReleaseChannel
	if releaseChannel != desktop.ReleaseChannelInternal {
		releaseChannel = desktop.ReleaseChannelStable
	}
	row := tx.QueryRowContext(ctx, `
		INSERT INTO desktop_devices (
			user_id, public_id, installation_id_hash, name, platform, architecture,
			os_version, app_version, status, token_version, pairing_expires_at, release_channel,
			approved_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, 'macos', $5, $6, $7, 'pending', 1, $8, $9,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, approved_at`,
		userID, publicID, pairing.InstallationHash, pairing.DeviceName, pairing.Architecture,
		pairing.OSVersion, pairing.AppVersion, pairing.ExpiresAt, releaseChannel,
	)
	device := &desktop.Device{
		PublicID: publicID, UserID: userID, Name: pairing.DeviceName, Platform: "macos",
		Architecture: pairing.Architecture, OSVersion: pairing.OSVersion, AppVersion: pairing.AppVersion,
		Status: desktop.DeviceStatusPending, TokenVersion: 1, ReleaseChannel: releaseChannel,
	}
	if err := row.Scan(&device.ID, &device.ApprovedAt); err != nil {
		return nil, fmt.Errorf("insert desktop device reservation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit desktop device reservation: %w", err)
	}
	return device, nil
}

func (r *desktopRepository) DeletePendingDevice(ctx context.Context, deviceID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM desktop_devices WHERE id = $1 AND status = 'pending'", deviceID)
	return err
}

func (r *desktopRepository) ActivateDevice(ctx context.Context, deviceID int64, familyID, refreshTokenHash string, expiresAt time.Time) (*desktop.Device, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin desktop activation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	device, err := queryDesktopDevice(ctx, tx, `
		SELECT d.id, d.public_id::text, d.user_id, d.name, d.platform, d.architecture,
		       d.os_version, d.app_version, d.status, d.token_version,
		       d.approved_at, d.activated_at, d.last_seen_at
		FROM desktop_devices d
		JOIN users u ON u.id = d.user_id AND u.deleted_at IS NULL AND u.status = 'active'
		WHERE d.id = $1 AND d.status = 'pending' AND d.pairing_expires_at > CURRENT_TIMESTAMP
		FOR UPDATE OF d`, deviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, desktop.ErrDeviceUnauthorized
		}
		return nil, err
	}
	now := time.Now()
	if _, err := tx.ExecContext(ctx, `
		UPDATE desktop_devices
		SET status = 'active', last_seen_at = $2, updated_at = $2
		WHERE id = $1`, deviceID, now); err != nil {
		return nil, fmt.Errorf("activate desktop device: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO desktop_device_sessions (
			device_id, family_id, refresh_token_hash, status, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, 'active', $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		deviceID, familyID, refreshTokenHash, expiresAt); err != nil {
		return nil, fmt.Errorf("create desktop refresh session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, desktop.ErrActivationOutcomeUnknown.WithCause(err)
	}
	device.Status = desktop.DeviceStatusActive
	device.ActivatedAt = nil
	device.LastSeenAt = &now
	return device, nil
}

func (r *desktopRepository) RotateSession(ctx context.Context, refreshTokenHash, replacementHash string, replacementExpiresAt time.Time) (*desktop.Device, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin desktop refresh rotation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sessionID int64
	var familyID string
	var sessionStatus string
	var sessionExpiresAt time.Time
	device, err := queryDesktopDeviceWithSession(ctx, tx, refreshTokenHash, &sessionID, &familyID, &sessionStatus, &sessionExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, desktop.ErrRefreshToken
		}
		return nil, err
	}
	if sessionStatus != "active" {
		_, _ = tx.ExecContext(ctx, `
			UPDATE desktop_device_sessions
			SET status = 'revoked', revoked_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE family_id = $1 AND status = 'active'`, familyID)
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit desktop refresh replay revocation: %w", err)
		}
		return nil, desktop.ErrRefreshReplay
	}
	if !sessionExpiresAt.After(time.Now()) || device.Status != desktop.DeviceStatusActive {
		_, _ = tx.ExecContext(ctx, `
			UPDATE desktop_device_sessions
			SET status = 'revoked', revoked_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE family_id = $1 AND status = 'active'`, familyID)
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit expired desktop refresh revocation: %w", err)
		}
		return nil, desktop.ErrRefreshToken
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE desktop_device_sessions
		SET status = 'consumed', consumed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND status = 'active'`, sessionID); err != nil {
		return nil, fmt.Errorf("consume desktop refresh token: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO desktop_device_sessions (
			device_id, family_id, refresh_token_hash, status, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, 'active', $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		device.ID, familyID, replacementHash, replacementExpiresAt); err != nil {
		return nil, fmt.Errorf("insert replacement desktop refresh token: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE desktop_devices SET last_seen_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = $1`, device.ID); err != nil {
		return nil, fmt.Errorf("touch desktop device: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit desktop refresh rotation: %w", err)
	}
	return device, nil
}

func (r *desktopRepository) GetAuthorizedDevice(ctx context.Context, deviceID, userID, tokenVersion int64) (*desktop.Device, error) {
	device, err := queryDesktopDevice(ctx, r.db, `
		SELECT d.id, d.public_id::text, d.user_id, d.name, d.platform, d.architecture,
		       d.os_version, d.app_version, d.status, d.token_version,
		       d.approved_at, d.activated_at, d.last_seen_at
		FROM desktop_devices d
		JOIN users u ON u.id = d.user_id AND u.deleted_at IS NULL AND u.status = 'active'
		WHERE d.id = $1 AND d.user_id = $2 AND d.token_version = $3 AND d.status = 'active'`,
		deviceID, userID, tokenVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, desktop.ErrDeviceUnauthorized
	}
	return device, err
}

func (r *desktopRepository) ListDevices(ctx context.Context, userID int64) ([]desktop.Device, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.public_id::text, d.user_id, d.name, d.platform, d.architecture,
		       d.os_version, d.app_version, d.status, d.token_version,
		       d.approved_at, d.activated_at, d.last_seen_at
		FROM desktop_devices d
		WHERE d.user_id = $1
		ORDER BY COALESCE(d.last_seen_at, d.approved_at, d.created_at) DESC, d.id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list desktop devices: %w", err)
	}
	defer func() { _ = rows.Close() }()

	devices := make([]desktop.Device, 0)
	for rows.Next() {
		device := new(desktop.Device)
		if err := scanDesktopDevice(rows, device); err != nil {
			return nil, fmt.Errorf("scan desktop device: %w", err)
		}
		devices = append(devices, *device)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate desktop devices: %w", err)
	}
	return devices, nil
}

func (r *desktopRepository) RenameDevice(ctx context.Context, userID int64, publicID, name string) (*desktop.Device, error) {
	device, err := queryDesktopDevice(ctx, r.db, `
		UPDATE desktop_devices
		SET name = $3, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND public_id = $2::uuid AND status IN ('pending', 'active')
		RETURNING id, public_id::text, user_id, name, platform, architecture,
		          os_version, app_version, status, token_version,
		          approved_at, activated_at, last_seen_at`, userID, publicID, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, desktop.ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("rename desktop device: %w", err)
	}
	return device, nil
}

func (r *desktopRepository) HeartbeatDevice(ctx context.Context, deviceID, userID, tokenVersion int64) (*desktop.Device, error) {
	device, err := queryDesktopDevice(ctx, r.db, `
		UPDATE desktop_devices
		SET last_seen_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2 AND token_version = $3 AND status = 'active'
		RETURNING id, public_id::text, user_id, name, platform, architecture,
		          os_version, app_version, status, token_version,
		          approved_at, activated_at, last_seen_at`, deviceID, userID, tokenVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, desktop.ErrDeviceUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("heartbeat desktop device: %w", err)
	}
	return device, nil
}

func (r *desktopRepository) MarkDeviceActivated(
	ctx context.Context,
	deviceID, userID, tokenVersion int64,
	activatedAt time.Time,
) (*desktop.Device, error) {
	device, err := queryDesktopDevice(ctx, r.db, `
		UPDATE desktop_devices
		SET activated_at = COALESCE(activated_at, $4),
		    last_seen_at = $4, updated_at = $4
		WHERE id = $1 AND user_id = $2 AND token_version = $3 AND status = 'active'
		RETURNING id, public_id::text, user_id, name, platform, architecture,
		          os_version, app_version, status, token_version,
		          approved_at, activated_at, last_seen_at`,
		deviceID, userID, tokenVersion, activatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, desktop.ErrDeviceUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("activate current desktop device: %w", err)
	}
	return device, nil
}

func (r *desktopRepository) GetReleaseChannel(ctx context.Context, deviceID, userID, tokenVersion int64) (string, error) {
	var channel string
	err := r.db.QueryRowContext(ctx, `
		SELECT d.release_channel
		FROM desktop_devices d
		JOIN users u ON u.id = d.user_id AND u.deleted_at IS NULL AND u.status = 'active'
		WHERE d.id = $1 AND d.user_id = $2 AND d.token_version = $3 AND d.status = 'active'`,
		deviceID, userID, tokenVersion).Scan(&channel)
	if errors.Is(err, sql.ErrNoRows) {
		return "", desktop.ErrDeviceUnauthorized
	}
	if err != nil {
		return "", fmt.Errorf("get desktop release channel: %w", err)
	}
	return channel, nil
}

func (r *desktopRepository) RevokeDevice(ctx context.Context, userID int64, publicID string) (*desktop.DeviceRevocation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin desktop device revocation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	device, err := queryDesktopDevice(ctx, tx, `
		SELECT id, public_id::text, user_id, name, platform, architecture,
		       os_version, app_version, status, token_version,
		       approved_at, activated_at, last_seen_at
		FROM desktop_devices
		WHERE user_id = $1 AND public_id = $2::uuid
		FOR UPDATE`, userID, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, desktop.ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock desktop device for revocation: %w", err)
	}

	if device.Status != desktop.DeviceStatusRevoked {
		if _, err := tx.ExecContext(ctx, `
			UPDATE desktop_devices
			SET status = 'revoked', token_version = token_version + 1,
			    revoked_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1`, device.ID); err != nil {
			return nil, fmt.Errorf("revoke desktop device: %w", err)
		}
		device.Status = desktop.DeviceStatusRevoked
		device.TokenVersion++
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE desktop_device_sessions
		SET status = 'revoked', revoked_at = COALESCE(revoked_at, CURRENT_TIMESTAMP),
		    updated_at = CURRENT_TIMESTAMP
		WHERE device_id = $1 AND status <> 'revoked'`, device.ID); err != nil {
		return nil, fmt.Errorf("revoke desktop device sessions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE api_keys
		SET status = 'disabled', updated_at = CURRENT_TIMESTAMP
		WHERE managed_device_id = $1 AND purpose = 'desktop'
		  AND deleted_at IS NULL AND status <> 'disabled'`, device.ID); err != nil {
		return nil, fmt.Errorf("disable desktop managed keys: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT key
		FROM api_keys
		WHERE managed_device_id = $1 AND purpose = 'desktop' AND deleted_at IS NULL`, device.ID)
	if err != nil {
		return nil, fmt.Errorf("list revoked desktop managed keys: %w", err)
	}
	managedKeys := make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan revoked desktop managed key: %w", err)
		}
		managedKeys = append(managedKeys, key)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close revoked desktop managed keys: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate revoked desktop managed keys: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit desktop device revocation: %w", err)
	}
	return &desktop.DeviceRevocation{Device: *device, ManagedKeys: managedKeys}, nil
}

func (r *desktopRepository) EnsureManagedKey(ctx context.Context, deviceID, userID, groupID int64, generatedKey string) (*desktop.ManagedKey, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin managed key ensure: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var authorized bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM desktop_devices d
			JOIN users u ON u.id = d.user_id AND u.deleted_at IS NULL AND u.status = 'active'
			WHERE d.id = $1 AND d.user_id = $2 AND d.status = 'active'
		)`, deviceID, userID).Scan(&authorized); err != nil {
		return nil, fmt.Errorf("authorize managed key device: %w", err)
	}
	if !authorized {
		return nil, desktop.ErrDeviceUnauthorized
	}

	managed := &desktop.ManagedKey{GroupID: groupID}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO api_keys (
			user_id, key, name, group_id, status, purpose, managed_device_id,
			created_at, updated_at
		) VALUES ($1, $2, 'Desktop managed key', $3, 'active', 'desktop', $4,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (managed_device_id, group_id)
			WHERE purpose = 'desktop' AND deleted_at IS NULL
		DO NOTHING
		RETURNING id, key`, userID, generatedKey, groupID, deviceID).Scan(&managed.ID, &managed.Key)
	switch {
	case err == nil:
		managed.Created = true
	case errors.Is(err, sql.ErrNoRows):
		err = tx.QueryRowContext(ctx, `
			SELECT id, key FROM api_keys
			WHERE managed_device_id = $1 AND group_id = $2
			  AND purpose = 'desktop' AND deleted_at IS NULL`, deviceID, groupID).Scan(&managed.ID, &managed.Key)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, desktop.ErrManagedKeyCollision
		}
		if err != nil {
			return nil, fmt.Errorf("read converged managed key: %w", err)
		}
	default:
		if isUniqueViolation(err) {
			return nil, desktop.ErrManagedKeyCollision
		}
		return nil, fmt.Errorf("insert managed key: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit managed key ensure: %w", err)
	}
	return managed, nil
}

func (r *desktopRepository) GetTodayUsage(
	ctx context.Context,
	deviceID, userID int64,
	startTime, endTime time.Time,
) (*desktop.TodayUsage, error) {
	usage := new(desktop.TodayUsage)
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(ul.id),
		       COALESCE(SUM(
		           ul.input_tokens + ul.output_tokens +
		           ul.cache_creation_tokens + ul.cache_read_tokens
		       ), 0)::bigint,
		       COALESCE(SUM(ul.actual_cost), 0)::double precision,
		       u.balance::double precision
		FROM users u
		JOIN desktop_devices d
		  ON d.user_id = u.id AND d.id = $1 AND d.status = 'active'
		LEFT JOIN api_keys k
		  ON k.managed_device_id = d.id
		 AND k.user_id = u.id
		 AND k.purpose = 'desktop'
		LEFT JOIN usage_logs ul
		  ON ul.api_key_id = k.id
		 AND ul.user_id = u.id
		 AND ul.created_at >= $3
		 AND ul.created_at < $4
		WHERE u.id = $2 AND u.deleted_at IS NULL AND u.status = 'active'
		GROUP BY u.balance`, deviceID, userID, startTime, endTime).
		Scan(&usage.Requests, &usage.Tokens, &usage.Cost, &usage.Balance)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, desktop.ErrDeviceUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("get desktop today usage: %w", err)
	}
	return usage, nil
}

func (r *desktopRepository) CreateDiagnostic(ctx context.Context, record desktop.DiagnosticRecord) (*desktop.DiagnosticCreated, error) {
	created := new(desktop.DiagnosticCreated)
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO desktop_diagnostics (
			device_id, user_id, public_id, app_version, platform, architecture,
			os_version, gateway_status, codex_config_status,
			request_sample_count, request_error_count, encrypted_payload,
			expires_at, created_at, updated_at
		)
		SELECT d.id, d.user_id, $4::uuid, $5, $6, $7, $8, $9, $10,
		       $11, $12, $13, $14, $15, $15
		FROM desktop_devices d
		JOIN users u ON u.id = d.user_id AND u.deleted_at IS NULL AND u.status = 'active'
		WHERE d.id = $1 AND d.user_id = $2 AND d.token_version = $3 AND d.status = 'active'
		RETURNING public_id::text, created_at, expires_at`,
		record.DeviceInternalID, record.UserID, record.DeviceTokenVersion,
		record.ID, record.AppVersion, record.Platform, record.Architecture,
		record.OSVersion, record.GatewayStatus, record.CodexConfigStatus,
		record.RequestSampleCount, record.RequestErrorCount, record.EncryptedPayload,
		record.ExpiresAt, record.CreatedAt).
		Scan(&created.ID, &created.CreatedAt, &created.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, desktop.ErrDeviceUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("create desktop diagnostic: %w", err)
	}
	return created, nil
}

func (r *desktopRepository) ListDiagnostics(ctx context.Context, page, pageSize int) (*desktop.DiagnosticPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM desktop_diagnostics
		WHERE expires_at > CURRENT_TIMESTAMP`).Scan(&total); err != nil {
		return nil, fmt.Errorf("count desktop diagnostics: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT x.public_id::text, d.public_id::text, x.user_id,
		       x.app_version, x.platform, x.architecture, x.os_version,
		       x.gateway_status, x.codex_config_status,
		       x.request_sample_count, x.request_error_count,
		       x.created_at, x.expires_at
		FROM desktop_diagnostics x
		JOIN desktop_devices d ON d.id = x.device_id
		WHERE x.expires_at > CURRENT_TIMESTAMP
		ORDER BY x.created_at DESC, x.id DESC
		LIMIT $1 OFFSET $2`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, fmt.Errorf("list desktop diagnostics: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]desktop.DiagnosticMetadata, 0)
	for rows.Next() {
		var item desktop.DiagnosticMetadata
		if err := rows.Scan(
			&item.ID, &item.DeviceID, &item.UserID,
			&item.AppVersion, &item.Platform, &item.Architecture, &item.OSVersion,
			&item.GatewayStatus, &item.CodexConfigStatus,
			&item.RequestSampleCount, &item.RequestErrorCount,
			&item.CreatedAt, &item.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan desktop diagnostic: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate desktop diagnostics: %w", err)
	}
	return &desktop.DiagnosticPage{Items: items, Total: total}, nil
}

func (r *desktopRepository) GetDiagnostic(ctx context.Context, publicID string) (*desktop.DiagnosticRecord, error) {
	record := new(desktop.DiagnosticRecord)
	err := r.db.QueryRowContext(ctx, `
		SELECT x.public_id::text, d.public_id::text, x.user_id,
		       x.app_version, x.platform, x.architecture, x.os_version,
		       x.gateway_status, x.codex_config_status,
		       x.request_sample_count, x.request_error_count,
		       x.created_at, x.expires_at, x.device_id, x.encrypted_payload
		FROM desktop_diagnostics x
		JOIN desktop_devices d ON d.id = x.device_id
		WHERE x.public_id = $1::uuid
		  AND x.expires_at > CURRENT_TIMESTAMP`, publicID).
		Scan(
			&record.ID, &record.DeviceID, &record.UserID,
			&record.AppVersion, &record.Platform, &record.Architecture, &record.OSVersion,
			&record.GatewayStatus, &record.CodexConfigStatus,
			&record.RequestSampleCount, &record.RequestErrorCount,
			&record.CreatedAt, &record.ExpiresAt,
			&record.DeviceInternalID, &record.EncryptedPayload,
		)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, desktop.ErrDiagnosticNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get desktop diagnostic: %w", err)
	}
	return record, nil
}

func (r *desktopRepository) CleanupExpired(
	ctx context.Context,
	now, sessionRetentionCutoff time.Time,
	limit int,
) (*desktop.CleanupResult, error) {
	if limit <= 0 {
		return &desktop.CleanupResult{}, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin desktop cleanup: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	pendingResult, err := tx.ExecContext(ctx, `
		WITH candidates AS (
			SELECT id
			FROM desktop_devices
			WHERE status = 'pending' AND pairing_expires_at <= $1
			ORDER BY pairing_expires_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		DELETE FROM desktop_devices d
		USING candidates c
		WHERE d.id = c.id`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("delete expired pending desktop devices: %w", err)
	}
	pendingDeleted, err := pendingResult.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("count expired pending desktop devices: %w", err)
	}

	sessionResult, err := tx.ExecContext(ctx, `
		WITH candidates AS (
			SELECT id
			FROM desktop_device_sessions
			WHERE expires_at <= $1
			   OR (status = 'revoked' AND COALESCE(revoked_at, updated_at) <= $1)
			ORDER BY expires_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		DELETE FROM desktop_device_sessions s
		USING candidates c
		WHERE s.id = c.id`, sessionRetentionCutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("delete retained desktop sessions: %w", err)
	}
	sessionsDeleted, err := sessionResult.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("count retained desktop sessions: %w", err)
	}
	diagnosticResult, err := tx.ExecContext(ctx, `
		WITH candidates AS (
			SELECT id
			FROM desktop_diagnostics
			WHERE expires_at <= $1
			ORDER BY expires_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		DELETE FROM desktop_diagnostics x
		USING candidates c
		WHERE x.id = c.id`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("delete expired desktop diagnostics: %w", err)
	}
	diagnosticsDeleted, err := diagnosticResult.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("count expired desktop diagnostics: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit desktop cleanup: %w", err)
	}
	return &desktop.CleanupResult{
		PendingDevicesDeleted: pendingDeleted,
		SessionsDeleted:       sessionsDeleted,
		DiagnosticsDeleted:    diagnosticsDeleted,
	}, nil
}

type desktopGroupAccess interface {
	GetAvailableGroups(ctx context.Context, userID int64) ([]service.Group, error)
	GetUserGroupRates(ctx context.Context, userID int64) (map[int64]float64, error)
}

type desktopRouteCatalog struct {
	apiKeys desktopGroupAccess
}

func NewDesktopRouteCatalog(apiKeys *service.APIKeyService) desktop.RouteCatalog {
	return &desktopRouteCatalog{apiKeys: apiKeys}
}

func (c *desktopRouteCatalog) ListRoutes(ctx context.Context, userID int64) ([]desktop.Route, error) {
	groups, err := c.apiKeys.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	userRates, err := c.apiKeys.GetUserGroupRates(ctx, userID)
	if err != nil {
		return nil, err
	}
	routes := make([]desktop.Route, 0, len(groups))
	for i := range groups {
		group := &groups[i]
		if group.Platform != "openai" || !group.IsActive() {
			continue
		}
		models := []string{}
		if len(group.ModelsListConfig.Models) > 0 {
			models = desktopCodexModels(group.ModelsListConfig.Models)
		} else if !group.ModelsListConfig.Enabled {
			models = desktopDefaultOpenAIModels()
		}
		rateMultiplier := group.RateMultiplier
		if userRate, ok := userRates[group.ID]; ok {
			rateMultiplier = userRate
		}
		routes = append(routes, desktop.Route{
			GroupID: group.ID, Name: group.Name, RateMultiplier: rateMultiplier, Models: models,
		})
	}
	return routes, nil
}

func desktopDefaultOpenAIModels() []string {
	return desktopCodexModels(openaipkg.DefaultModelIDs())
}

func desktopCodexModels(candidates []string) []string {
	models := make([]string, 0, len(candidates))
	for _, model := range candidates {
		if strings.HasPrefix(model, "gpt-") && !strings.HasPrefix(model, "gpt-image-") {
			models = append(models, model)
		}
	}
	return models
}

type desktopKeyInvalidator struct {
	apiKeys *service.APIKeyService
}

func NewDesktopKeyInvalidator(apiKeys *service.APIKeyService) desktop.KeyInvalidator {
	return &desktopKeyInvalidator{apiKeys: apiKeys}
}

func (i *desktopKeyInvalidator) InvalidateManagedKey(ctx context.Context, key string) error {
	return i.apiKeys.InvalidateAuthCacheByKeyChecked(ctx, key)
}

type desktopQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type desktopScanner interface {
	Scan(...any) error
}

func queryDesktopDevice(ctx context.Context, q desktopQueryer, query string, args ...any) (*desktop.Device, error) {
	device := new(desktop.Device)
	err := scanDesktopDevice(q.QueryRowContext(ctx, query, args...), device)
	return device, err
}

func scanDesktopDevice(scanner desktopScanner, device *desktop.Device) error {
	return scanner.Scan(
		&device.ID, &device.PublicID, &device.UserID, &device.Name, &device.Platform, &device.Architecture,
		&device.OSVersion, &device.AppVersion, &device.Status, &device.TokenVersion,
		&device.ApprovedAt, &device.ActivatedAt, &device.LastSeenAt,
	)
}

func queryDesktopDeviceWithSession(
	ctx context.Context,
	tx *sql.Tx,
	refreshTokenHash string,
	sessionID *int64,
	familyID, sessionStatus *string,
	sessionExpiresAt *time.Time,
) (*desktop.Device, error) {
	device := new(desktop.Device)
	err := tx.QueryRowContext(ctx, `
		SELECT s.id, s.family_id::text, s.status, s.expires_at,
		       d.id, d.public_id::text, d.user_id, d.name, d.platform, d.architecture,
		       d.os_version, d.app_version, d.status, d.token_version,
		       d.approved_at, d.activated_at, d.last_seen_at
		FROM desktop_device_sessions s
		JOIN desktop_devices d ON d.id = s.device_id
		JOIN users u ON u.id = d.user_id AND u.deleted_at IS NULL AND u.status = 'active'
		WHERE s.refresh_token_hash = $1
		FOR UPDATE OF s, d`, refreshTokenHash).Scan(
		sessionID, familyID, sessionStatus, sessionExpiresAt,
		&device.ID, &device.PublicID, &device.UserID, &device.Name, &device.Platform, &device.Architecture,
		&device.OSVersion, &device.AppVersion, &device.Status, &device.TokenVersion,
		&device.ApprovedAt, &device.ActivatedAt, &device.LastSeenAt,
	)
	return device, err
}

func desktopUserLockID(userID int64) int64 {
	return int64(uint64(userID) ^ uint64(0x44534b544f500000))
}
