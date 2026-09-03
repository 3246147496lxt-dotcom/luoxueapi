package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUpdateOpenAISubscriptionCredentialsIfUnchangedCAS(t *testing.T) {
	const (
		updatedJSON  = `{"access_token":"token-before","chatgpt_account_id":"acc-before","subscription_checked_at":"2026-08-04T10:00:00Z","subscription_expires_at":"2026-09-01T00:00:00Z","subscription_will_renew":false}`
		expectedJSON = `{"access_token":"token-before","chatgpt_account_id":"acc-before"}`
	)

	tests := []struct {
		name         string
		rowsAffected int64
		wantUpdated  bool
	}{
		{name: "credentials unchanged", rowsAffected: 1, wantUpdated: true},
		{name: "credentials changed", rowsAffected: 0, wantUpdated: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			repo := newAccountRepositoryWithSQL(nil, db, nil)

			mock.ExpectExec(`(?s)WITH updated AS \(.*UPDATE accounts AS a.*a\.credentials = \$5::jsonb.*INSERT INTO scheduler_outbox`).
				WithArgs(
					updatedJSON,
					int64(41),
					service.PlatformOpenAI,
					service.AccountTypeOAuth,
					expectedJSON,
					service.SchedulerOutboxEventAccountChanged,
				).
				WillReturnResult(sqlmock.NewResult(0, tt.rowsAffected))

			updated, err := repo.UpdateOpenAISubscriptionCredentialsIfUnchanged(
				context.Background(),
				41,
				map[string]any{
					"access_token":       "token-before",
					"chatgpt_account_id": "acc-before",
				},
				map[string]any{
					"access_token":            "token-before",
					"chatgpt_account_id":      "acc-before",
					"subscription_checked_at": "2026-08-04T10:00:00Z",
					"subscription_expires_at": "2026-09-01T00:00:00Z",
					"subscription_will_renew": false,
				},
			)

			require.NoError(t, err)
			require.Equal(t, tt.wantUpdated, updated)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
