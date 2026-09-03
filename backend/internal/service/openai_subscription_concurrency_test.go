package service

import (
	"context"
	"net/http"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type subscriptionCASTestRepo struct {
	AccountRepository
	account *Account

	casResult bool
	casErr    error
	casCalls  int
	casID     int64
	casBefore map[string]any
	casAfter  map[string]any
}

func (r *subscriptionCASTestRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	if r.account == nil || r.account.ID != id {
		return nil, nil
	}
	return r.account, nil
}

func (r *subscriptionCASTestRepo) UpdateOpenAISubscriptionCredentialsIfUnchanged(
	_ context.Context,
	id int64,
	expectedCredentials map[string]any,
	credentials map[string]any,
) (bool, error) {
	r.casCalls++
	r.casID = id
	r.casBefore = cloneSubscriptionCASTestMap(expectedCredentials)
	r.casAfter = cloneSubscriptionCASTestMap(credentials)
	return r.casResult, r.casErr
}

func TestPersistSubscriptionInfoCASUpdatesUnchangedCredentials(t *testing.T) {
	expectedCredentials := map[string]any{
		"access_token":            "token-before",
		"chatgpt_account_id":      "acc-before",
		"refresh_token":           "refresh-before",
		"plan_type":               "plus",
		"subscription_expires_at": "2026-08-01T00:00:00Z",
		"subscription_will_renew": true,
	}
	repo := &subscriptionCASTestRepo{casResult: true}
	quotaService := &OpenAIQuotaService{accountRepo: repo}
	willRenew := false

	err := quotaService.persistSubscriptionInfo(context.Background(), 41, expectedCredentials, &OpenAISubscriptionInfo{
		PlanType:    " pro ",
		ActiveUntil: "2026-09-01T00:00:00Z",
		WillRenew:   &willRenew,
		CheckedAt:   "2026-08-04T10:00:00Z",
		Source:      "live",
	})

	require.NoError(t, err)
	require.Equal(t, 1, repo.casCalls)
	require.Equal(t, int64(41), repo.casID)
	require.Equal(t, expectedCredentials, repo.casBefore)
	require.Equal(t, map[string]any{
		"access_token":            "token-before",
		"chatgpt_account_id":      "acc-before",
		"refresh_token":           "refresh-before",
		"plan_type":               "pro",
		"subscription_expires_at": "2026-09-01T00:00:00Z",
		"subscription_will_renew": false,
		"subscription_checked_at": "2026-08-04T10:00:00Z",
	}, repo.casAfter)
	require.Equal(t, "2026-08-01T00:00:00Z", expectedCredentials["subscription_expires_at"], "persist must not mutate the captured snapshot")
}

func TestPersistSubscriptionInfoCASClearsSuccessfulEmptySubscription(t *testing.T) {
	expectedCredentials := map[string]any{
		"access_token":            "token-before",
		"chatgpt_account_id":      "acc-before",
		"plan_type":               "plus",
		"subscription_expires_at": "2026-08-01T00:00:00Z",
		"subscription_will_renew": true,
	}
	repo := &subscriptionCASTestRepo{casResult: true}
	quotaService := &OpenAIQuotaService{accountRepo: repo}

	err := quotaService.persistSubscriptionInfo(context.Background(), 41, expectedCredentials, &OpenAISubscriptionInfo{
		CheckedAt: "2026-08-04T10:00:00Z",
		Source:    "live",
	})

	require.NoError(t, err)
	require.Equal(t, 1, repo.casCalls)
	require.Equal(t, map[string]any{
		"access_token":            "token-before",
		"chatgpt_account_id":      "acc-before",
		"subscription_checked_at": "2026-08-04T10:00:00Z",
	}, repo.casAfter)
	require.Equal(t, "plus", expectedCredentials["plan_type"], "persist must not mutate the captured snapshot")
}

func TestPersistSubscriptionInfoCASRejectsChangedCredentials(t *testing.T) {
	expectedCredentials := map[string]any{
		"access_token":       "token-before",
		"chatgpt_account_id": "acc-before",
	}
	repo := &subscriptionCASTestRepo{casResult: false}
	quotaService := &OpenAIQuotaService{accountRepo: repo}

	err := quotaService.persistSubscriptionInfo(context.Background(), 41, expectedCredentials, &OpenAISubscriptionInfo{
		ActiveUntil: "2026-09-01T00:00:00Z",
		CheckedAt:   "2026-08-04T10:00:00Z",
	})

	require.Error(t, err)
	require.Equal(t, http.StatusConflict, infraerrors.Code(err))
	require.Equal(t, "OPENAI_SUBSCRIPTION_IDENTITY_CHANGED", infraerrors.Reason(err))
	require.Equal(t, 1, repo.casCalls)
	require.Equal(t, expectedCredentials, repo.casBefore)
}

func TestLoadSubscriptionCredentialSnapshotRejectsIdentityChanges(t *testing.T) {
	tests := []struct {
		name                     string
		storedChatGPTAccountID   string
		storedAccessToken        string
		expectedChatGPTAccountID string
		expectedAccessToken      string
	}{
		{
			name:                     "chatgpt account id changed",
			storedChatGPTAccountID:   "acc-after",
			storedAccessToken:        "token-same",
			expectedChatGPTAccountID: "acc-before",
			expectedAccessToken:      "token-same",
		},
		{
			name:                     "access token changed",
			storedChatGPTAccountID:   "acc-same",
			storedAccessToken:        "token-after",
			expectedChatGPTAccountID: "acc-same",
			expectedAccessToken:      "token-before",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &subscriptionCASTestRepo{account: &Account{
				ID:       41,
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"chatgpt_account_id": tt.storedChatGPTAccountID,
					"access_token":       tt.storedAccessToken,
				},
			}}
			quotaService := &OpenAIQuotaService{accountRepo: repo}

			account, snapshot, err := quotaService.loadSubscriptionCredentialSnapshot(
				context.Background(),
				41,
				tt.expectedChatGPTAccountID,
				tt.expectedAccessToken,
			)

			require.Error(t, err)
			require.Nil(t, account)
			require.Nil(t, snapshot)
			require.Equal(t, http.StatusConflict, infraerrors.Code(err))
			require.Equal(t, "OPENAI_SUBSCRIPTION_IDENTITY_CHANGED", infraerrors.Reason(err))
		})
	}
}

func TestLoadSubscriptionCredentialSnapshotReturnsIndependentCopy(t *testing.T) {
	account := &Account{
		ID:       41,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"chatgpt_account_id": "acc-same",
			"access_token":       "token-same",
		},
	}
	quotaService := &OpenAIQuotaService{accountRepo: &subscriptionCASTestRepo{account: account}}

	gotAccount, snapshot, err := quotaService.loadSubscriptionCredentialSnapshot(
		context.Background(),
		41,
		"acc-same",
		"token-same",
	)

	require.NoError(t, err)
	require.Same(t, account, gotAccount)
	require.Equal(t, account.Credentials, snapshot)
	snapshot["access_token"] = "mutated-test-copy"
	require.Equal(t, "token-same", account.Credentials["access_token"])
}

func cloneSubscriptionCASTestMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
