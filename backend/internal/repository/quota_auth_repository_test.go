package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/stretchr/testify/require"
)

const quotaRepositoryTestPublicID = "8a5b7f6b-6771-4d47-93e9-ea0ba338bcdf"

func TestQuotaAuthRepositoryRefreshReplayRevokesWholeDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewQuotaAuthRepository(db)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT s\.id, s\.family_id::text, s\.status, s\.expires_at,.*WHERE s\.refresh_token_hash = \$1.*FOR UPDATE OF s, d`).
		WithArgs("consumed-hash").
		WillReturnRows(quotaAuthSessionRows().AddRow(
			int64(9),
			"c03f972a-2639-421d-9c62-8aaf0e4faaf0",
			"consumed",
			now.Add(24*time.Hour),
			nil,
			nil,
			nil,
			int64(41),
			quotaRepositoryTestPublicID,
			int64(7),
			quotaauth.ClientID,
			quotaauth.ScopeRead,
			"Work laptop",
			"macos",
			"arm64",
			"15.0",
			"1.0.0",
			quotaauth.DeviceStatusActive,
			int64(1),
			now,
			now,
			now,
			nil,
		))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_devices.*status = 'revoked'.*WHERE id = \$1`).
		WithArgs(int64(41)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_device_sessions.*WHERE device_id = \$1 AND status <> 'revoked'`).
		WithArgs(int64(41)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	_, err = repo.RotateSession(context.Background(), quotaauth.RefreshRotation{
		PredecessorHash:   "consumed-hash",
		ReplacementHash:   "replacement-hash",
		ReplacementExpiry: now.Add(30 * 24 * time.Hour),
	})
	require.ErrorIs(t, err, quotaauth.ErrRefreshReplay)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuotaAuthRepositoryRefreshRotationStoresOnlyReplacementHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewQuotaAuthRepository(db)
	now := time.Now().UTC()
	replacementExpiresAt := now.Add(30 * 24 * time.Hour)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT s\.id, s\.family_id::text, s\.status, s\.expires_at,.*WHERE s\.refresh_token_hash = \$1.*FOR UPDATE OF s, d`).
		WithArgs("current-token-hash").
		WillReturnRows(quotaAuthSessionRows().AddRow(
			int64(9),
			"c03f972a-2639-421d-9c62-8aaf0e4faaf0",
			"active",
			now.Add(24*time.Hour),
			nil,
			nil,
			nil,
			int64(41),
			quotaRepositoryTestPublicID,
			int64(7),
			quotaauth.ClientID,
			quotaauth.ScopeRead,
			"Work laptop",
			"macos",
			"arm64",
			"15.0",
			"1.0.0",
			quotaauth.DeviceStatusActive,
			int64(1),
			now,
			now,
			now,
			nil,
		))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_device_sessions.*SET status = 'consumed'.*WHERE id = \$1 AND status = 'active'`).
		WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO quota_viewer_device_sessions.*VALUES \(\$1, \$2::uuid, \$3, 'active', \$4`).
		WithArgs(
			int64(41),
			"c03f972a-2639-421d-9c62-8aaf0e4faaf0",
			"replacement-token-hash",
			replacementExpiresAt,
		).
		WillReturnResult(sqlmock.NewResult(10, 1))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_devices.*SET last_seen_at = \$2.*WHERE id = \$1 AND status = 'active'`).
		WithArgs(int64(41), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	outcome, err := repo.RotateSession(context.Background(), quotaauth.RefreshRotation{
		PredecessorHash:   "current-token-hash",
		ReplacementHash:   "replacement-token-hash",
		ReplacementExpiry: replacementExpiresAt,
	})
	require.NoError(t, err)
	require.Equal(t, int64(7), outcome.Device.UserID)
	require.Equal(t, quotaauth.RefreshRotationCommitted, outcome.Result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuotaAuthRepositoryCandidateRotationRecoversAfterCommittedResponseIsLost(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewQuotaAuthRepository(db)
	now := time.Now().UTC()
	familyID := "c03f972a-2639-421d-9c62-8aaf0e4faaf0"
	rotationID := "af69f4b4-5824-4f48-9149-3e55f1b5ca2d"
	predecessorHash := "predecessor-hash"
	candidateHash := "candidate-hash"
	candidateExpiry := now.Add(30 * 24 * time.Hour)
	recoveryExpiry := now.Add(10 * time.Minute)
	rotation := quotaauth.RefreshRotation{
		PredecessorHash:   predecessorHash,
		ReplacementHash:   candidateHash,
		ReplacementExpiry: candidateExpiry,
		RotationID:        rotationID,
		RecoveryExpiry:    recoveryExpiry,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(quotaAuthSessionQueryPattern()).
		WithArgs(predecessorHash).
		WillReturnRows(quotaAuthSessionRow(
			now,
			familyID,
			"active",
			nil,
			nil,
			nil,
		))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_device_sessions.*SET status = 'consumed'.*rotation_id = \$2::uuid.*replacement_token_hash = \$3.*recovery_expires_at = \$4.*WHERE id = \$1 AND status = 'active'`).
		WithArgs(int64(9), rotationID, candidateHash, recoveryExpiry).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO quota_viewer_device_sessions.*VALUES \(\$1, \$2::uuid, \$3, 'active', \$4`).
		WithArgs(int64(41), familyID, candidateHash, candidateExpiry).
		WillReturnResult(sqlmock.NewResult(10, 1))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_devices.*SET last_seen_at = \$2.*WHERE id = \$1 AND status = 'active'`).
		WithArgs(int64(41), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	committed, err := repo.RotateSession(context.Background(), rotation)
	require.NoError(t, err)
	require.Equal(t, quotaauth.RefreshRotationCommitted, committed.Result)

	// Simulate a lost HTTP response: the client retries the exact predecessor,
	// rotation id and candidate tuple after the first transaction committed.
	mock.ExpectBegin()
	mock.ExpectQuery(quotaAuthSessionQueryPattern()).
		WithArgs(predecessorHash).
		WillReturnRows(quotaAuthSessionRow(
			now,
			familyID,
			"consumed",
			rotationID,
			candidateHash,
			recoveryExpiry,
		))
	mock.ExpectQuery(`(?s)SELECT device_id, family_id::text, status, expires_at.*WHERE refresh_token_hash = \$1.*FOR UPDATE`).
		WithArgs(candidateHash).
		WillReturnRows(sqlmock.NewRows([]string{
			"device_id",
			"family_id",
			"status",
			"expires_at",
		}).AddRow(
			int64(41),
			familyID,
			"active",
			candidateExpiry,
		))
	mock.ExpectRollback()

	recovered, err := repo.RotateSession(context.Background(), rotation)
	require.NoError(t, err)
	require.Equal(t, quotaauth.RefreshRotationRecovered, recovered.Result)
	require.Equal(t, int64(41), recovered.Device.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuotaAuthRepositoryCandidateRotationMismatchStillRevokesDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewQuotaAuthRepository(db)
	now := time.Now().UTC()
	familyID := "c03f972a-2639-421d-9c62-8aaf0e4faaf0"

	mock.ExpectBegin()
	mock.ExpectQuery(quotaAuthSessionQueryPattern()).
		WithArgs("predecessor-hash").
		WillReturnRows(quotaAuthSessionRow(
			now,
			familyID,
			"consumed",
			"af69f4b4-5824-4f48-9149-3e55f1b5ca2d",
			"candidate-hash",
			now.Add(10*time.Minute),
		))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_devices.*status = 'revoked'.*WHERE id = \$1`).
		WithArgs(int64(41)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_device_sessions.*WHERE device_id = \$1 AND status <> 'revoked'`).
		WithArgs(int64(41)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	_, err = repo.RotateSession(context.Background(), quotaauth.RefreshRotation{
		PredecessorHash:   "predecessor-hash",
		ReplacementHash:   "different-candidate-hash",
		ReplacementExpiry: now.Add(30 * 24 * time.Hour),
		RotationID:        "af69f4b4-5824-4f48-9149-3e55f1b5ca2d",
		RecoveryExpiry:    now.Add(10 * time.Minute),
	})
	require.ErrorIs(t, err, quotaauth.ErrRefreshReplay)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuotaAuthRepositoryExactRetryOfConsumedCandidateIsSupersededWithoutRevocation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewQuotaAuthRepository(db)
	now := time.Now().UTC()
	familyID := "c03f972a-2639-421d-9c62-8aaf0e4faaf0"
	rotationID := "af69f4b4-5824-4f48-9149-3e55f1b5ca2d"
	candidateHash := "candidate-hash"

	mock.ExpectBegin()
	mock.ExpectQuery(quotaAuthSessionQueryPattern()).
		WithArgs("predecessor-hash").
		WillReturnRows(quotaAuthSessionRow(
			now,
			familyID,
			"consumed",
			rotationID,
			candidateHash,
			now.Add(10*time.Minute),
		))
	mock.ExpectQuery(`(?s)SELECT device_id, family_id::text, status, expires_at.*WHERE refresh_token_hash = \$1.*FOR UPDATE`).
		WithArgs(candidateHash).
		WillReturnRows(sqlmock.NewRows([]string{
			"device_id",
			"family_id",
			"status",
			"expires_at",
		}).AddRow(
			int64(41),
			familyID,
			"consumed",
			now.Add(30*24*time.Hour),
		))
	mock.ExpectRollback()

	_, err = repo.RotateSession(context.Background(), quotaauth.RefreshRotation{
		PredecessorHash:   "predecessor-hash",
		ReplacementHash:   candidateHash,
		ReplacementExpiry: now.Add(30 * 24 * time.Hour),
		RotationID:        rotationID,
		RecoveryExpiry:    now.Add(10 * time.Minute),
	})
	require.ErrorIs(t, err, quotaauth.ErrRefreshRotationSuperseded)
	require.NoError(t, mock.ExpectationsWereMet(), "superseded recovery must not revoke the device")
}

func TestQuotaAuthRepositoryExpiredRecoveryDoesNotRevokeDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewQuotaAuthRepository(db)
	now := time.Now().UTC()
	rotationID := "af69f4b4-5824-4f48-9149-3e55f1b5ca2d"
	candidateHash := "candidate-hash"

	mock.ExpectBegin()
	mock.ExpectQuery(quotaAuthSessionQueryPattern()).
		WithArgs("predecessor-hash").
		WillReturnRows(quotaAuthSessionRow(
			now,
			"c03f972a-2639-421d-9c62-8aaf0e4faaf0",
			"consumed",
			rotationID,
			candidateHash,
			now.Add(-time.Minute),
		))
	mock.ExpectRollback()

	_, err = repo.RotateSession(context.Background(), quotaauth.RefreshRotation{
		PredecessorHash:   "predecessor-hash",
		ReplacementHash:   candidateHash,
		ReplacementExpiry: now.Add(30 * 24 * time.Hour),
		RotationID:        rotationID,
		RecoveryExpiry:    now.Add(10 * time.Minute),
	})
	require.ErrorIs(t, err, quotaauth.ErrRefreshRecoveryExpired)
	require.NoError(t, mock.ExpectationsWereMet(), "expired recovery must not revoke the device")
}

func TestQuotaAuthRepositoryCommitFailureHasRecoverableOutcomeUnknownError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewQuotaAuthRepository(db)
	now := time.Now().UTC()
	familyID := "c03f972a-2639-421d-9c62-8aaf0e4faaf0"
	rotationID := "af69f4b4-5824-4f48-9149-3e55f1b5ca2d"

	mock.ExpectBegin()
	mock.ExpectQuery(quotaAuthSessionQueryPattern()).
		WithArgs("predecessor-hash").
		WillReturnRows(quotaAuthSessionRow(now, familyID, "active", nil, nil, nil))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_device_sessions.*rotation_id = \$2::uuid.*WHERE id = \$1 AND status = 'active'`).
		WithArgs(int64(9), rotationID, "candidate-hash", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)INSERT INTO quota_viewer_device_sessions.*VALUES \(\$1, \$2::uuid, \$3, 'active', \$4`).
		WithArgs(int64(41), familyID, "candidate-hash", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(10, 1))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_devices.*SET last_seen_at = \$2.*WHERE id = \$1 AND status = 'active'`).
		WithArgs(int64(41), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(context.DeadlineExceeded)

	_, err = repo.RotateSession(context.Background(), quotaauth.RefreshRotation{
		PredecessorHash:   "predecessor-hash",
		ReplacementHash:   "candidate-hash",
		ReplacementExpiry: now.Add(30 * 24 * time.Hour),
		RotationID:        rotationID,
		RecoveryExpiry:    now.Add(10 * time.Minute),
	})
	require.ErrorIs(t, err, quotaauth.ErrRefreshOutcomeUnknown)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuotaAuthRepositoryDeviceRevocationIsOwnerScoped(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewQuotaAuthRepository(db)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT d\.id, d\.public_id::text, d\.user_id, d\.client_id, d\.scope,.*WHERE d\.user_id = \$1 AND d\.public_id = \$2::uuid AND d\.client_id = \$3.*FOR UPDATE`).
		WithArgs(int64(7), quotaRepositoryTestPublicID, quotaauth.ClientID).
		WillReturnRows(quotaAuthDeviceRows().AddRow(
			int64(41),
			quotaRepositoryTestPublicID,
			int64(7),
			quotaauth.ClientID,
			quotaauth.ScopeRead,
			"Work laptop",
			"macos",
			"arm64",
			"15.0",
			"1.0.0",
			quotaauth.DeviceStatusActive,
			int64(1),
			now,
			now,
			now,
			nil,
		))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_devices.*status = 'revoked'.*WHERE id = \$1`).
		WithArgs(int64(41)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`(?s)UPDATE quota_viewer_device_sessions.*WHERE device_id = \$1 AND status <> 'revoked'`).
		WithArgs(int64(41)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	device, err := repo.RevokeDevice(context.Background(), 7, quotaRepositoryTestPublicID)
	require.NoError(t, err)
	require.Equal(t, quotaauth.DeviceStatusRevoked, device.Status)
	require.Equal(t, int64(2), device.TokenVersion)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuotaAuthRepositoryAuthorizationRequiresActiveUserDeviceClientAndScope(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewQuotaAuthRepository(db)

	pattern := `(?s)SELECT d\.id, d\.public_id::text, d\.user_id, d\.client_id, d\.scope,.*JOIN users u ON u\.id = d\.user_id AND u\.deleted_at IS NULL AND u\.status = 'active'.*WHERE d\.id = \$1.*d\.token_version = \$2.*d\.client_id = \$3.*d\.scope = \$4.*d\.status = 'active'`
	mock.ExpectQuery(pattern).
		WithArgs(int64(41), int64(1), quotaauth.ClientID, quotaauth.ScopeRead).
		WillReturnError(sqlmock.ErrCancelled)

	_, err = repo.GetAuthorizedDevice(context.Background(), 41, 1)
	require.Error(t, err)
	require.NotErrorIs(t, err, quotaauth.ErrDeviceUnauthorized)
	require.NoError(t, mock.ExpectationsWereMet())
}

func quotaAuthDeviceRows() *sqlmock.Rows {
	return sqlmock.NewRows(quotaAuthDeviceColumns())
}

func quotaAuthDeviceColumns() []string {
	return []string{
		"id",
		"public_id",
		"user_id",
		"client_id",
		"scope",
		"name",
		"platform",
		"architecture",
		"os_version",
		"app_version",
		"status",
		"token_version",
		"approved_at",
		"activated_at",
		"last_seen_at",
		"revoked_at",
	}
}

func quotaAuthSessionRows() *sqlmock.Rows {
	return sqlmock.NewRows(append(
		[]string{
			"session_id",
			"family_id",
			"session_status",
			"session_expires_at",
			"rotation_id",
			"replacement_token_hash",
			"recovery_expires_at",
		},
		quotaAuthDeviceColumns()...,
	))
}

func quotaAuthSessionQueryPattern() string {
	return `(?s)SELECT s\.id, s\.family_id::text, s\.status, s\.expires_at,.*WHERE s\.refresh_token_hash = \$1.*FOR UPDATE OF s, d`
}

func quotaAuthSessionRow(
	now time.Time,
	familyID, status string,
	rotationID, replacementHash, recoveryExpiresAt any,
) *sqlmock.Rows {
	return quotaAuthSessionRows().AddRow(
		int64(9),
		familyID,
		status,
		now.Add(24*time.Hour),
		rotationID,
		replacementHash,
		recoveryExpiresAt,
		int64(41),
		quotaRepositoryTestPublicID,
		int64(7),
		quotaauth.ClientID,
		quotaauth.ScopeRead,
		"Work laptop",
		"macos",
		"arm64",
		"15.0",
		"1.0.0",
		quotaauth.DeviceStatusActive,
		int64(1),
		now,
		now,
		now,
		nil,
	)
}
