package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

const repositoryTestDevicePublicID = "c7d06b34-7c6a-4a59-ad48-648ade0525e1"

func TestDesktopRepositoryRejectsSixthDeviceUnderUserLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_xact_lock($1)")).
		WithArgs(desktopUserLockID(7)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM desktop_devices").WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectRollback()

	_, err = repo.ReserveDevice(context.Background(), 7, desktop.Pairing{ExpiresAt: time.Now().Add(time.Minute)})
	require.ErrorIs(t, err, desktop.ErrDeviceLimit)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryRejectsManagedKeyForDifferentDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT EXISTS").WithArgs(int64(42), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	_, err = repo.EnsureManagedKey(context.Background(), 42, 7, 9, "sk-secret")
	require.ErrorIs(t, err, desktop.ErrDeviceUnauthorized)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryRenameRejectsDifferentOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}

	mock.ExpectQuery("UPDATE desktop_devices").
		WithArgs(int64(8), repositoryTestDevicePublicID, "Other Mac").
		WillReturnRows(desktopDeviceRows())

	_, err = repo.RenameDevice(context.Background(), 8, repositoryTestDevicePublicID, "Other Mac")
	require.ErrorIs(t, err, desktop.ErrDeviceNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryHeartbeatRequiresCurrentTokenVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}

	mock.ExpectQuery("UPDATE desktop_devices").
		WithArgs(int64(41), int64(7), int64(2)).
		WillReturnRows(desktopDeviceRows())

	_, err = repo.HeartbeatDevice(context.Background(), 41, 7, 2)
	require.ErrorIs(t, err, desktop.ErrDeviceUnauthorized)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryMarksFirstRealActivationIdempotently(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}
	firstAt := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery("UPDATE desktop_devices").
		WithArgs(int64(41), int64(7), int64(1), firstAt).
		WillReturnRows(desktopDeviceRows().AddRow(
			int64(41), repositoryTestDevicePublicID, int64(7), "Mac", "macos", "arm64",
			"15.0", "0.1.0", "active", int64(1), firstAt.Add(-time.Hour), firstAt, firstAt,
		))

	device, err := repo.MarkDeviceActivated(context.Background(), 41, 7, 1, firstAt)
	require.NoError(t, err)
	require.Equal(t, firstAt, *device.ActivatedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryReleaseChannelRequiresCurrentActiveUserAndDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}
	query := `(?s)SELECT d\.release_channel.*JOIN users u.*u\.deleted_at IS NULL.*u\.status = 'active'.*d\.token_version = \$3.*d\.status = 'active'`

	mock.ExpectQuery(query).WithArgs(int64(41), int64(7), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"release_channel"}).AddRow(desktop.ReleaseChannelStable))
	channel, err := repo.GetReleaseChannel(context.Background(), 41, 7, 1)
	require.NoError(t, err)
	require.Equal(t, desktop.ReleaseChannelStable, channel)

	mock.ExpectQuery(query).WithArgs(int64(41), int64(7), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"release_channel"}))
	_, err = repo.GetReleaseChannel(context.Background(), 41, 7, 2)
	require.ErrorIs(t, err, desktop.ErrDeviceUnauthorized)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryRevokesDeviceSessionsAndManagedKeysAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, public_id::text").
		WithArgs(int64(7), repositoryTestDevicePublicID).
		WillReturnRows(desktopDeviceRows().AddRow(
			int64(41), repositoryTestDevicePublicID, int64(7), "Mac", "macos", "arm64",
			"15.0", "1.0.0", "active", int64(1), time.Now(), time.Now(), time.Now(),
		))
	mock.ExpectExec("UPDATE desktop_devices").WithArgs(int64(41)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE desktop_device_sessions").WithArgs(int64(41)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("UPDATE api_keys").WithArgs(int64(41)).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery("SELECT key").WithArgs(int64(41)).
		WillReturnRows(sqlmock.NewRows([]string{"key"}).AddRow("sk-one").AddRow("sk-two"))
	mock.ExpectCommit()

	revocation, err := repo.RevokeDevice(context.Background(), 7, repositoryTestDevicePublicID)
	require.NoError(t, err)
	require.Equal(t, desktop.DeviceStatusRevoked, revocation.Device.Status)
	require.Equal(t, int64(2), revocation.Device.TokenVersion)
	require.Equal(t, []string{"sk-one", "sk-two"}, revocation.ManagedKeys)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryRevocationRollsBackEveryChangeOnKeyFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, public_id::text").
		WithArgs(int64(7), repositoryTestDevicePublicID).
		WillReturnRows(desktopDeviceRows().AddRow(
			int64(41), repositoryTestDevicePublicID, int64(7), "Mac", "macos", "arm64",
			"15.0", "1.0.0", "active", int64(1), nil, nil, nil,
		))
	mock.ExpectExec("UPDATE desktop_devices").WithArgs(int64(41)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE desktop_device_sessions").WithArgs(int64(41)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE api_keys").WithArgs(int64(41)).WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()

	_, err = repo.RevokeDevice(context.Background(), 7, repositoryTestDevicePublicID)
	require.ErrorContains(t, err, "disable desktop managed keys")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryTodayUsageUsesManagedDeviceFinalBilling(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	end := start.AddDate(0, 0, 1)

	query := `(?s)` +
		regexp.QuoteMeta("SUM(") + `.*` +
		regexp.QuoteMeta("ul.input_tokens + ul.output_tokens") + `.*` +
		regexp.QuoteMeta("ul.cache_creation_tokens + ul.cache_read_tokens") + `.*` +
		regexp.QuoteMeta("SUM(ul.actual_cost)") + `.*` +
		regexp.QuoteMeta("k.managed_device_id = d.id") + `.*` +
		regexp.QuoteMeta("k.user_id = u.id") + `.*` +
		regexp.QuoteMeta("k.purpose = 'desktop'") + `.*` +
		regexp.QuoteMeta("ul.user_id = u.id") + `.*` +
		regexp.QuoteMeta("ul.created_at >= $3") + `.*` +
		regexp.QuoteMeta("ul.created_at < $4")
	mock.ExpectQuery(query).
		WithArgs(int64(41), int64(7), start, end).
		WillReturnRows(sqlmock.NewRows([]string{"requests", "tokens", "cost", "balance"}).
			AddRow(int64(4), int64(1500), 1.25, 9.75))

	usage, err := repo.GetTodayUsage(context.Background(), 41, 7, start, end)
	require.NoError(t, err)
	require.Equal(t, &desktop.TodayUsage{Requests: 4, Tokens: 1500, Cost: 1.25, Balance: 9.75}, usage)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryTodayUsageRejectsUnauthorizedDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}

	mock.ExpectQuery("SELECT COUNT").
		WithArgs(int64(41), int64(8), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"requests", "tokens", "cost", "balance"}))

	_, err = repo.GetTodayUsage(context.Background(), 41, 8, time.Now(), time.Now().Add(time.Hour))
	require.ErrorIs(t, err, desktop.ErrDeviceUnauthorized)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryCreatesDiagnosticOnlyForCurrentActiveDevice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	record := desktop.DiagnosticRecord{
		DiagnosticMetadata: desktop.DiagnosticMetadata{
			ID: "b9719d18-ddac-4f29-bc71-7fe44c233b67", UserID: 7,
			AppVersion: "0.1.0", Platform: "macos", Architecture: "arm64", OSVersion: "15.0",
			GatewayStatus: "running", CodexConfigStatus: "managed",
			RequestSampleCount: 3, RequestErrorCount: 1,
			CreatedAt: now, ExpiresAt: now.Add(7 * 24 * time.Hour),
		},
		DeviceInternalID: 41, DeviceTokenVersion: 2, EncryptedPayload: "ciphertext",
	}
	createQuery := `(?s)INSERT INTO desktop_diagnostics.*JOIN users u.*u\.deleted_at IS NULL.*u\.status = 'active'.*d\.token_version = \$3.*d\.status = 'active'`

	mock.ExpectQuery(createQuery).
		WithArgs(
			int64(41), int64(7), int64(2), record.ID, "0.1.0", "macos", "arm64", "15.0",
			"running", "managed", 3, 1, "ciphertext", record.ExpiresAt, now,
		).
		WillReturnRows(sqlmock.NewRows([]string{"public_id", "created_at", "expires_at"}).
			AddRow(record.ID, now, record.ExpiresAt))

	created, err := repo.CreateDiagnostic(context.Background(), record)
	require.NoError(t, err)
	require.Equal(t, record.ID, created.ID)

	record.DeviceTokenVersion = 3
	mock.ExpectQuery(createQuery).
		WithArgs(
			int64(41), int64(7), int64(3), record.ID, "0.1.0", "macos", "arm64", "15.0",
			"running", "managed", 3, 1, "ciphertext", record.ExpiresAt, now,
		).
		WillReturnRows(sqlmock.NewRows([]string{"public_id", "created_at", "expires_at"}))

	_, err = repo.CreateDiagnostic(context.Background(), record)
	require.ErrorIs(t, err, desktop.ErrDeviceUnauthorized)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryListsDiagnosticMetadataWithoutCiphertextAndHidesExpiredRecords(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	diagnosticID := "b9719d18-ddac-4f29-bc71-7fe44c233b67"

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM desktop_diagnostics.*WHERE expires_at > CURRENT_TIMESTAMP`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`(?s)FROM desktop_diagnostics x.*WHERE x.expires_at > CURRENT_TIMESTAMP.*ORDER BY`).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"public_id", "device_id", "user_id", "app_version", "platform", "architecture", "os_version",
			"gateway_status", "codex_config_status", "request_sample_count", "request_error_count", "created_at", "expires_at",
		}).AddRow(
			diagnosticID, repositoryTestDevicePublicID, int64(7), "0.1.0", "macos", "arm64", "15.0",
			"running", "managed", 3, 1, now, now.Add(7*24*time.Hour),
		))

	page, err := repo.ListDiagnostics(context.Background(), 1, 20)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, diagnosticID, page.Items[0].ID)
	require.Equal(t, int64(1), page.Total)

	mock.ExpectQuery(`(?s)FROM desktop_diagnostics x.*WHERE x.public_id = \$1::uuid.*x.expires_at > CURRENT_TIMESTAMP`).
		WithArgs(diagnosticID).
		WillReturnRows(sqlmock.NewRows([]string{
			"public_id", "device_id", "user_id", "app_version", "platform", "architecture", "os_version",
			"gateway_status", "codex_config_status", "request_sample_count", "request_error_count", "created_at", "expires_at", "device_internal_id", "encrypted_payload",
		}))

	_, err = repo.GetDiagnostic(context.Background(), diagnosticID)
	require.ErrorIs(t, err, desktop.ErrDiagnosticNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDesktopRepositoryCleanupExpiredIsRepeatableAndBatchLocked(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &desktopRepository{db: db}
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	cutoff := now.Add(-7 * 24 * time.Hour)

	pendingQuery := `(?s)` + regexp.QuoteMeta("WITH candidates AS (") + `.*` +
		regexp.QuoteMeta("FROM desktop_devices") + `.*` +
		regexp.QuoteMeta("status = 'pending' AND pairing_expires_at <= $1") + `.*` +
		regexp.QuoteMeta("FOR UPDATE SKIP LOCKED") + `.*` +
		regexp.QuoteMeta("DELETE FROM desktop_devices")
	sessionQuery := `(?s)` + regexp.QuoteMeta("WITH candidates AS (") + `.*` +
		regexp.QuoteMeta("FROM desktop_device_sessions") + `.*` +
		regexp.QuoteMeta("expires_at <= $1") + `.*` +
		regexp.QuoteMeta("status = 'revoked'") + `.*` +
		regexp.QuoteMeta("FOR UPDATE SKIP LOCKED") + `.*` +
		regexp.QuoteMeta("DELETE FROM desktop_device_sessions")

	diagnosticQuery := `(?s)` + regexp.QuoteMeta("WITH candidates AS (") + `.*` +
		regexp.QuoteMeta("FROM desktop_diagnostics") + `.*` +
		regexp.QuoteMeta("expires_at <= $1") + `.*` +
		regexp.QuoteMeta("FOR UPDATE SKIP LOCKED") + `.*` +
		regexp.QuoteMeta("DELETE FROM desktop_diagnostics")

	for _, deleted := range []struct{ pending, sessions, diagnostics int64 }{{2, 3, 4}, {0, 0, 0}} {
		mock.ExpectBegin()
		mock.ExpectExec(pendingQuery).WithArgs(now, 500).
			WillReturnResult(sqlmock.NewResult(0, deleted.pending))
		mock.ExpectExec(sessionQuery).WithArgs(cutoff, 500).
			WillReturnResult(sqlmock.NewResult(0, deleted.sessions))
		mock.ExpectExec(diagnosticQuery).WithArgs(now, 500).
			WillReturnResult(sqlmock.NewResult(0, deleted.diagnostics))
		mock.ExpectCommit()

		result, err := repo.CleanupExpired(context.Background(), now, cutoff, 500)
		require.NoError(t, err)
		require.Equal(t, deleted.pending, result.PendingDevicesDeleted)
		require.Equal(t, deleted.sessions, result.SessionsDeleted)
		require.Equal(t, deleted.diagnostics, result.DiagnosticsDeleted)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

type desktopGroupAccessStub struct {
	groups []service.Group
	rates  map[int64]float64
}

func (s desktopGroupAccessStub) GetAvailableGroups(context.Context, int64) ([]service.Group, error) {
	return s.groups, nil
}

func (s desktopGroupAccessStub) GetUserGroupRates(context.Context, int64) (map[int64]float64, error) {
	return s.rates, nil
}

func TestDesktopRouteCatalogReturnsResolvedUserGroupRateMultiplier(t *testing.T) {
	catalog := &desktopRouteCatalog{apiKeys: desktopGroupAccessStub{
		groups: []service.Group{
			{
				ID: 9, Name: "OpenAI default", Platform: service.PlatformOpenAI,
				Status: service.StatusActive, RateMultiplier: 1.25,
				ModelsListConfig: service.GroupModelsListConfig{Enabled: true, Models: []string{"gpt-5"}},
			},
			{
				ID: 10, Name: "OpenAI override", Platform: service.PlatformOpenAI,
				Status: service.StatusActive, RateMultiplier: 1.5,
			},
			{ID: 11, Name: "Not OpenAI", Platform: service.PlatformAnthropic, Status: service.StatusActive},
		},
		rates: map[int64]float64{10: 0.75},
	}}

	routes, err := catalog.ListRoutes(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, []desktop.Route{
		{GroupID: 9, Name: "OpenAI default", RateMultiplier: 1.25, Models: []string{"gpt-5"}},
		{GroupID: 10, Name: "OpenAI override", RateMultiplier: 0.75, Models: desktopDefaultOpenAIModels()},
	}, routes)
}

func TestDesktopRouteCatalogUsesChatCapableDefaultsForUnrestrictedOpenAIGroups(t *testing.T) {
	catalog := &desktopRouteCatalog{apiKeys: desktopGroupAccessStub{
		groups: []service.Group{{
			ID: 12, Name: "OpenAI unrestricted", Platform: service.PlatformOpenAI,
			Status: service.StatusActive,
		}},
	}}

	routes, err := catalog.ListRoutes(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.Equal(t, "gpt-5.6", routes[0].Models[0])
	require.Contains(t, routes[0].Models, "gpt-5.6-sol")
	require.NotContains(t, routes[0].Models, "codex-auto-review")
	require.NotContains(t, routes[0].Models, "gpt-image-1")
}

func TestDesktopRouteCatalogUsesCuratedModelsWithoutEnforcingAllowlist(t *testing.T) {
	catalog := &desktopRouteCatalog{apiKeys: desktopGroupAccessStub{
		groups: []service.Group{{
			ID: 14, Name: "OpenAI catalog", Platform: service.PlatformOpenAI,
			Status: service.StatusActive,
			ModelsListConfig: service.GroupModelsListConfig{
				Enabled: false,
				Models:  []string{"gpt-5.6-sol", "gpt-image-2", "codex-auto-review", "gpt-5.6-terra"},
			},
		}},
	}}

	routes, err := catalog.ListRoutes(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.Equal(t, []string{"gpt-5.6-sol", "gpt-5.6-terra"}, routes[0].Models)
}

func TestDesktopRouteCatalogPreservesExplicitlyEmptyModelAllowlist(t *testing.T) {
	catalog := &desktopRouteCatalog{apiKeys: desktopGroupAccessStub{
		groups: []service.Group{{
			ID: 13, Name: "OpenAI blocked", Platform: service.PlatformOpenAI,
			Status: service.StatusActive, ModelsListConfig: service.GroupModelsListConfig{Enabled: true},
		}},
	}}

	routes, err := catalog.ListRoutes(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.Empty(t, routes[0].Models)
}

func desktopDeviceRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "public_id", "user_id", "name", "platform", "architecture",
		"os_version", "app_version", "status", "token_version",
		"approved_at", "activated_at", "last_seen_at",
	})
}
