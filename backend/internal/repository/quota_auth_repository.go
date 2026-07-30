package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/google/uuid"
)

const quotaAuthDeviceLimit = 10

type quotaAuthRepository struct {
	db *sql.DB
}

func NewQuotaAuthRepository(db *sql.DB) quotaauth.Repository {
	return &quotaAuthRepository{db: db}
}

func (r *quotaAuthRepository) ReserveDevice(
	ctx context.Context,
	userID int64,
	pairing quotaauth.Pairing,
) (*quotaauth.Device, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("quota viewer repository is unavailable")
	}
	if userID <= 0 || pairing.ClientID != quotaauth.ClientID || pairing.Scope != quotaauth.ScopeRead {
		return nil, quotaauth.ErrInvalidAuthorizationRequest
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin quota viewer device reservation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", quotaAuthUserLockID(userID)); err != nil {
		return nil, fmt.Errorf("lock quota viewer device reservations: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM quota_viewer_devices
		WHERE user_id = $1 AND status = 'pending' AND pairing_expires_at <= CURRENT_TIMESTAMP`, userID); err != nil {
		return nil, fmt.Errorf("remove expired quota viewer device reservations: %w", err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM quota_viewer_devices
		WHERE user_id = $1
		  AND client_id = $2
		  AND (status = 'active' OR (status = 'pending' AND pairing_expires_at > CURRENT_TIMESTAMP))`,
		userID, quotaauth.ClientID,
	).Scan(&count); err != nil {
		return nil, fmt.Errorf("count quota viewer devices: %w", err)
	}
	if count >= quotaAuthDeviceLimit {
		return nil, quotaauth.ErrDeviceLimit
	}

	publicID := uuid.NewString()
	device := &quotaauth.Device{
		PublicID:     publicID,
		UserID:       userID,
		ClientID:     quotaauth.ClientID,
		Scope:        quotaauth.ScopeRead,
		Name:         pairing.DeviceName,
		Platform:     pairing.Platform,
		Architecture: pairing.Architecture,
		OSVersion:    pairing.OSVersion,
		AppVersion:   pairing.AppVersion,
		Status:       quotaauth.DeviceStatusPending,
		TokenVersion: 1,
	}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO quota_viewer_devices (
			user_id, public_id, client_id, scope, installation_id_hash, name,
			platform, architecture, os_version, app_version, status, token_version,
			pairing_expires_at, approved_at, created_at, updated_at
		) VALUES (
			$1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', 1,
			$11, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
		RETURNING id, approved_at`,
		userID,
		publicID,
		quotaauth.ClientID,
		quotaauth.ScopeRead,
		pairing.InstallationHash,
		pairing.DeviceName,
		pairing.Platform,
		pairing.Architecture,
		pairing.OSVersion,
		pairing.AppVersion,
		pairing.ExpiresAt,
	).Scan(&device.ID, &device.ApprovedAt)
	if err != nil {
		return nil, fmt.Errorf("insert quota viewer device reservation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit quota viewer device reservation: %w", err)
	}
	return device, nil
}

func (r *quotaAuthRepository) DeletePendingDevice(ctx context.Context, deviceID int64) error {
	if r == nil || r.db == nil {
		return errors.New("quota viewer repository is unavailable")
	}
	_, err := r.db.ExecContext(
		ctx,
		"DELETE FROM quota_viewer_devices WHERE id = $1 AND status = 'pending'",
		deviceID,
	)
	return err
}

func (r *quotaAuthRepository) ActivateDevice(
	ctx context.Context,
	deviceID int64,
	familyID, refreshTokenHash string,
	expiresAt time.Time,
) (*quotaauth.Device, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("quota viewer repository is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin quota viewer activation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	device, err := queryQuotaAuthDevice(ctx, tx, `
		SELECT d.id, d.public_id::text, d.user_id, d.client_id, d.scope,
		       d.name, d.platform, d.architecture, d.os_version, d.app_version,
		       d.status, d.token_version, d.approved_at, d.activated_at,
		       d.last_seen_at, d.revoked_at
		FROM quota_viewer_devices d
		JOIN users u ON u.id = d.user_id AND u.deleted_at IS NULL AND u.status = 'active'
		WHERE d.id = $1
		  AND d.client_id = $2
		  AND d.scope = $3
		  AND d.status = 'pending'
		  AND d.pairing_expires_at > CURRENT_TIMESTAMP
		FOR UPDATE OF d`,
		deviceID, quotaauth.ClientID, quotaauth.ScopeRead,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, quotaauth.ErrDeviceUnauthorized
		}
		return nil, fmt.Errorf("lock quota viewer device activation: %w", err)
	}
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		UPDATE quota_viewer_devices
		SET status = 'active', activated_at = $2, last_seen_at = $2, updated_at = $2
		WHERE id = $1`, deviceID, now); err != nil {
		return nil, fmt.Errorf("activate quota viewer device: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO quota_viewer_device_sessions (
			device_id, family_id, refresh_token_hash, status, expires_at,
			created_at, updated_at
		) VALUES ($1, $2::uuid, $3, 'active', $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		deviceID, familyID, refreshTokenHash, expiresAt,
	); err != nil {
		return nil, fmt.Errorf("create quota viewer refresh session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, quotaauth.ErrActivationOutcomeUnknown.WithCause(err)
	}
	device.Status = quotaauth.DeviceStatusActive
	device.ActivatedAt = &now
	device.LastSeenAt = &now
	return device, nil
}

func (r *quotaAuthRepository) RotateSession(
	ctx context.Context,
	rotation quotaauth.RefreshRotation,
) (*quotaauth.RefreshRotationOutcome, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("quota viewer repository is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin quota viewer refresh rotation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var (
		sessionID               int64
		familyID                string
		sessionStatus           string
		sessionExpiresAt        time.Time
		storedRotationID        sql.NullString
		storedReplacementHash   sql.NullString
		storedRecoveryExpiresAt sql.NullTime
	)
	device, err := queryQuotaAuthDeviceWithSession(
		ctx,
		tx,
		rotation.PredecessorHash,
		&sessionID,
		&familyID,
		&sessionStatus,
		&sessionExpiresAt,
		&storedRotationID,
		&storedReplacementHash,
		&storedRecoveryExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, quotaauth.ErrRefreshToken
		}
		return nil, fmt.Errorf("read quota viewer refresh session: %w", err)
	}
	if sessionStatus != "active" {
		if rotation.RotationID != "" &&
			storedRotationID.Valid &&
			storedReplacementHash.Valid &&
			storedRecoveryExpiresAt.Valid &&
			storedRotationID.String == rotation.RotationID &&
			storedReplacementHash.String == rotation.ReplacementHash {
			return recoverQuotaAuthRotation(
				ctx,
				tx,
				device,
				familyID,
				sessionStatus,
				storedReplacementHash.String,
				storedRecoveryExpiresAt.Time,
			)
		}
		if err := revokeQuotaAuthDeviceTx(ctx, tx, device.ID); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, quotaauth.ErrRefreshOutcomeUnknown.WithCause(err)
		}
		return nil, quotaauth.ErrRefreshReplay
	}
	now := time.Now().UTC()
	if !sessionExpiresAt.After(now) ||
		device.Status != quotaauth.DeviceStatusActive ||
		device.ClientID != quotaauth.ClientID ||
		device.Scope != quotaauth.ScopeRead {
		if err := revokeQuotaAuthDeviceTx(ctx, tx, device.ID); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, quotaauth.ErrRefreshOutcomeUnknown.WithCause(err)
		}
		return nil, quotaauth.ErrRefreshToken
	}
	var result sql.Result
	if rotation.RotationID == "" {
		result, err = tx.ExecContext(ctx, `
			UPDATE quota_viewer_device_sessions
			SET status = 'consumed', consumed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND status = 'active'`, sessionID)
	} else {
		result, err = tx.ExecContext(ctx, `
			UPDATE quota_viewer_device_sessions
			SET status = 'consumed',
			    consumed_at = CURRENT_TIMESTAMP,
			    rotation_id = $2::uuid,
			    replacement_token_hash = $3,
			    recovery_expires_at = $4,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND status = 'active'`,
			sessionID,
			rotation.RotationID,
			rotation.ReplacementHash,
			rotation.RecoveryExpiry,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("consume quota viewer refresh token: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return nil, fmt.Errorf("count consumed quota viewer refresh token: %w", err)
		}
		if err := revokeQuotaAuthDeviceTx(ctx, tx, device.ID); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, quotaauth.ErrRefreshOutcomeUnknown.WithCause(err)
		}
		return nil, quotaauth.ErrRefreshReplay
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO quota_viewer_device_sessions (
			device_id, family_id, refresh_token_hash, status, expires_at,
			created_at, updated_at
		) VALUES ($1, $2::uuid, $3, 'active', $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		device.ID, familyID, rotation.ReplacementHash, rotation.ReplacementExpiry,
	); err != nil {
		return nil, fmt.Errorf("insert replacement quota viewer refresh token: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE quota_viewer_devices
		SET last_seen_at = $2, updated_at = $2
		WHERE id = $1 AND status = 'active'`, device.ID, now); err != nil {
		return nil, fmt.Errorf("touch quota viewer device: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, quotaauth.ErrRefreshOutcomeUnknown.WithCause(err)
	}
	device.LastSeenAt = &now
	return &quotaauth.RefreshRotationOutcome{
		Device: device,
		Result: quotaauth.RefreshRotationCommitted,
	}, nil
}

func recoverQuotaAuthRotation(
	ctx context.Context,
	tx *sql.Tx,
	device *quotaauth.Device,
	familyID, predecessorStatus, replacementHash string,
	recoveryExpiresAt time.Time,
) (*quotaauth.RefreshRotationOutcome, error) {
	if device == nil ||
		device.Status != quotaauth.DeviceStatusActive ||
		device.ClientID != quotaauth.ClientID ||
		device.Scope != quotaauth.ScopeRead ||
		predecessorStatus == "revoked" {
		return nil, quotaauth.ErrDeviceUnauthorized
	}
	now := time.Now().UTC()
	if !recoveryExpiresAt.After(now) {
		return nil, quotaauth.ErrRefreshRecoveryExpired
	}

	var (
		candidateDeviceID  int64
		candidateFamilyID  string
		candidateStatus    string
		candidateExpiresAt time.Time
	)
	err := tx.QueryRowContext(ctx, `
		SELECT device_id, family_id::text, status, expires_at
		FROM quota_viewer_device_sessions
		WHERE refresh_token_hash = $1
		FOR UPDATE`,
		replacementHash,
	).Scan(
		&candidateDeviceID,
		&candidateFamilyID,
		&candidateStatus,
		&candidateExpiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, quotaauth.ErrDeviceUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("read quota viewer recovery candidate: %w", err)
	}
	if candidateDeviceID != device.ID || candidateFamilyID != familyID {
		return nil, quotaauth.ErrDeviceUnauthorized
	}
	switch candidateStatus {
	case "consumed":
		return nil, quotaauth.ErrRefreshRotationSuperseded
	case "active":
		if !candidateExpiresAt.After(now) {
			return nil, quotaauth.ErrRefreshToken
		}
		return &quotaauth.RefreshRotationOutcome{
			Device: device,
			Result: quotaauth.RefreshRotationRecovered,
		}, nil
	default:
		return nil, quotaauth.ErrDeviceUnauthorized
	}
}

func (r *quotaAuthRepository) GetAuthorizedDevice(
	ctx context.Context,
	deviceID, tokenVersion int64,
) (*quotaauth.Device, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("quota viewer repository is unavailable")
	}
	device, err := queryQuotaAuthDevice(ctx, r.db, `
		SELECT d.id, d.public_id::text, d.user_id, d.client_id, d.scope,
		       d.name, d.platform, d.architecture, d.os_version, d.app_version,
		       d.status, d.token_version, d.approved_at, d.activated_at,
		       d.last_seen_at, d.revoked_at
		FROM quota_viewer_devices d
		JOIN users u ON u.id = d.user_id AND u.deleted_at IS NULL AND u.status = 'active'
		WHERE d.id = $1
		  AND d.token_version = $2
		  AND d.client_id = $3
		  AND d.scope = $4
		  AND d.status = 'active'`,
		deviceID, tokenVersion, quotaauth.ClientID, quotaauth.ScopeRead,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, quotaauth.ErrDeviceUnauthorized
	}
	if err != nil {
		return nil, fmt.Errorf("authorize quota viewer device: %w", err)
	}
	return device, nil
}

func (r *quotaAuthRepository) ListDevices(ctx context.Context, userID int64) ([]quotaauth.Device, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("quota viewer repository is unavailable")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.public_id::text, d.user_id, d.client_id, d.scope,
		       d.name, d.platform, d.architecture, d.os_version, d.app_version,
		       d.status, d.token_version, d.approved_at, d.activated_at,
		       d.last_seen_at, d.revoked_at
		FROM quota_viewer_devices d
		WHERE d.user_id = $1 AND d.client_id = $2
		ORDER BY COALESCE(d.last_seen_at, d.approved_at, d.created_at) DESC, d.id DESC`,
		userID, quotaauth.ClientID,
	)
	if err != nil {
		return nil, fmt.Errorf("list quota viewer devices: %w", err)
	}
	defer rows.Close()

	devices := make([]quotaauth.Device, 0)
	for rows.Next() {
		device := new(quotaauth.Device)
		if err := scanQuotaAuthDevice(rows, device); err != nil {
			return nil, fmt.Errorf("scan quota viewer device: %w", err)
		}
		devices = append(devices, *device)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quota viewer devices: %w", err)
	}
	return devices, nil
}

func (r *quotaAuthRepository) RevokeDevice(
	ctx context.Context,
	userID int64,
	publicID string,
) (*quotaauth.Device, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("quota viewer repository is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin quota viewer device revocation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	device, err := queryQuotaAuthDevice(ctx, tx, `
		SELECT d.id, d.public_id::text, d.user_id, d.client_id, d.scope,
		       d.name, d.platform, d.architecture, d.os_version, d.app_version,
		       d.status, d.token_version, d.approved_at, d.activated_at,
		       d.last_seen_at, d.revoked_at
		FROM quota_viewer_devices d
		WHERE d.user_id = $1 AND d.public_id = $2::uuid AND d.client_id = $3
		FOR UPDATE`,
		userID, publicID, quotaauth.ClientID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, quotaauth.ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock quota viewer device for revocation: %w", err)
	}
	if err := revokeQuotaAuthDeviceTx(ctx, tx, device.ID); err != nil {
		return nil, err
	}
	if device.Status != quotaauth.DeviceStatusRevoked {
		now := time.Now().UTC()
		device.Status = quotaauth.DeviceStatusRevoked
		device.TokenVersion++
		device.RevokedAt = &now
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit quota viewer device revocation: %w", err)
	}
	return device, nil
}

func revokeQuotaAuthDeviceTx(ctx context.Context, tx *sql.Tx, deviceID int64) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE quota_viewer_devices
		SET status = 'revoked',
		    token_version = CASE WHEN status = 'revoked' THEN token_version ELSE token_version + 1 END,
		    revoked_at = COALESCE(revoked_at, CURRENT_TIMESTAMP),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, deviceID); err != nil {
		return fmt.Errorf("revoke quota viewer device: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE quota_viewer_device_sessions
		SET status = 'revoked',
		    revoked_at = COALESCE(revoked_at, CURRENT_TIMESTAMP),
		    updated_at = CURRENT_TIMESTAMP
		WHERE device_id = $1 AND status <> 'revoked'`, deviceID); err != nil {
		return fmt.Errorf("revoke quota viewer device sessions: %w", err)
	}
	return nil
}

type quotaAuthQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type quotaAuthScanner interface {
	Scan(...any) error
}

func queryQuotaAuthDevice(
	ctx context.Context,
	q quotaAuthQueryer,
	query string,
	args ...any,
) (*quotaauth.Device, error) {
	device := new(quotaauth.Device)
	err := scanQuotaAuthDevice(q.QueryRowContext(ctx, query, args...), device)
	return device, err
}

func scanQuotaAuthDevice(scanner quotaAuthScanner, device *quotaauth.Device) error {
	return scanner.Scan(
		&device.ID,
		&device.PublicID,
		&device.UserID,
		&device.ClientID,
		&device.Scope,
		&device.Name,
		&device.Platform,
		&device.Architecture,
		&device.OSVersion,
		&device.AppVersion,
		&device.Status,
		&device.TokenVersion,
		&device.ApprovedAt,
		&device.ActivatedAt,
		&device.LastSeenAt,
		&device.RevokedAt,
	)
}

func queryQuotaAuthDeviceWithSession(
	ctx context.Context,
	tx *sql.Tx,
	refreshTokenHash string,
	sessionID *int64,
	familyID, sessionStatus *string,
	sessionExpiresAt *time.Time,
	rotationID, replacementTokenHash *sql.NullString,
	recoveryExpiresAt *sql.NullTime,
) (*quotaauth.Device, error) {
	device := new(quotaauth.Device)
	err := tx.QueryRowContext(ctx, `
		SELECT s.id, s.family_id::text, s.status, s.expires_at,
		       s.rotation_id::text, s.replacement_token_hash, s.recovery_expires_at,
		       d.id, d.public_id::text, d.user_id, d.client_id, d.scope,
		       d.name, d.platform, d.architecture, d.os_version, d.app_version,
		       d.status, d.token_version, d.approved_at, d.activated_at,
		       d.last_seen_at, d.revoked_at
		FROM quota_viewer_device_sessions s
		JOIN quota_viewer_devices d ON d.id = s.device_id
		JOIN users u ON u.id = d.user_id AND u.deleted_at IS NULL AND u.status = 'active'
		WHERE s.refresh_token_hash = $1
		FOR UPDATE OF s, d`,
		refreshTokenHash,
	).Scan(
		sessionID,
		familyID,
		sessionStatus,
		sessionExpiresAt,
		rotationID,
		replacementTokenHash,
		recoveryExpiresAt,
		&device.ID,
		&device.PublicID,
		&device.UserID,
		&device.ClientID,
		&device.Scope,
		&device.Name,
		&device.Platform,
		&device.Architecture,
		&device.OSVersion,
		&device.AppVersion,
		&device.Status,
		&device.TokenVersion,
		&device.ApprovedAt,
		&device.ActivatedAt,
		&device.LastSeenAt,
		&device.RevokedAt,
	)
	return device, err
}

func quotaAuthUserLockID(userID int64) int64 {
	return int64(uint64(userID) ^ uint64(0x51554f5441560000))
}
