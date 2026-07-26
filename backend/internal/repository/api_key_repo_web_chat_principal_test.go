package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestInsertWebChatPrincipalUsesConflictSafeReturningInsert(t *testing.T) {
	query := `(?s)` + regexp.QuoteMeta("INSERT INTO api_keys") + `.*` +
		regexp.QuoteMeta("deleted_at") + `.*` + regexp.QuoteMeta("ON CONFLICT DO NOTHING") + `.*` + regexp.QuoteMeta("RETURNING id")

	t.Run("inserted", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		mock.ExpectQuery(query).
			WithArgs(int64(7), int64(11), "sk-chat-new", service.StatusActive, service.APIKeyPurposeWebChat).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(31)))

		id, inserted, err := insertWebChatPrincipal(context.Background(), db, 7, 11, "sk-chat-new")
		require.NoError(t, err)
		require.True(t, inserted)
		require.Equal(t, int64(31), id)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("conflict returns no row without transaction-aborting error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		mock.ExpectQuery(query).
			WithArgs(int64(7), int64(11), "sk-chat-conflict", service.StatusActive, service.APIKeyPurposeWebChat).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		id, inserted, err := insertWebChatPrincipal(context.Background(), db, 7, 11, "sk-chat-conflict")
		require.NoError(t, err)
		require.False(t, inserted)
		require.Zero(t, id)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAPIKeyRepositoryWebChatPrincipalIsReusableAndHidden(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "web-chat-principal@test.com")
	group := mustCreateWebChatGroup(t, ctx, client, "web-chat-principal")

	principal, err := repo.GetOrCreateWebChatPrincipal(ctx, user.ID, group.ID, "sk-web-chat-principal")
	require.NoError(t, err)
	require.NotZero(t, principal.ID)
	require.Equal(t, service.APIKeyPurposeWebChat, principal.Purpose)
	require.Equal(t, user.ID, principal.UserID)
	require.Equal(t, group.ID, *principal.GroupID)

	stored, err := client.APIKey.Query().
		Where(apikey.IDEQ(principal.ID)).
		Only(mixins.SkipSoftDelete(ctx))
	require.NoError(t, err)
	require.NotNil(t, stored.DeletedAt, "web-chat principals must be tombstoned for rollback compatibility")
	require.Equal(t, service.StatusActive, stored.Status)

	// Simulate the pre-purpose repository/auth query. Old binaries only know the
	// soft-delete boundary, so the internal key must remain invisible there.
	_, err = client.APIKey.Query().
		Where(apikey.DeletedAtIsNil(), apikey.KeyEQ(principal.Key)).
		Only(ctx)
	require.True(t, dbent.IsNotFound(err))

	reused, err := repo.GetOrCreateWebChatPrincipal(ctx, user.ID, group.ID, "sk-unused-second-key")
	require.NoError(t, err)
	require.Equal(t, principal.ID, reused.ID)
	require.Equal(t, principal.Key, reused.Key)

	_, err = repo.GetByID(ctx, principal.ID)
	require.ErrorIs(t, err, service.ErrAPIKeyNotFound)
	_, err = repo.GetByKey(ctx, principal.Key)
	require.ErrorIs(t, err, service.ErrAPIKeyNotFound)

	// Soft deletion preserves the FK row used by billing and usage attribution.
	accountID := mustCreateAPIKeyRepoAccount(t, ctx, client, "web-chat-principal-account")
	mustCreateAPIKeyRepoUsageLog(t, ctx, client, user.ID, principal.ID, accountID, "req-web-chat-principal", time.Now().UTC(), nil)
}

func TestAPIKeyRepositoryWebChatPrincipalKeyConflictIsRetryable(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "web-chat-key-conflict@test.com")
	group := mustCreateWebChatGroup(t, ctx, client, "web-chat-key-conflict")
	conflictingKey := &service.APIKey{
		UserID: user.ID,
		Key:    "sk-web-chat-key-conflict",
		Name:   "User key",
		Status: service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, conflictingKey))

	_, err := repo.GetOrCreateWebChatPrincipal(ctx, user.ID, group.ID, conflictingKey.Key)
	require.ErrorIs(t, err, service.ErrAPIKeyExists)
}

func mustCreateWebChatGroup(t *testing.T, ctx context.Context, client *dbent.Client, name string) *dbent.Group {
	t.Helper()
	group, err := client.Group.Create().
		SetName(name).
		SetPlatform(service.PlatformOpenAI).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)
	return group
}
