package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

func TestFetchChatGPTSubscriptionInfoReturnsActiveUntil(t *testing.T) {
	const wantActiveUntil = "2026-08-12T02:52:15Z"

	server := newChatGPTSubscriptionInfoServer(t, map[string]any{
		"plan_type":    " plus ",
		"active_until": wantActiveUntil,
		"will_renew":   true,
		"id":           "sub_123",
	})
	defer server.Close()

	info := fetchChatGPTSubscriptionInfo(
		context.Background(),
		testChatGPTSubscriptionClientFactory,
		"access-token",
		"",
		"acc_123",
	)

	require.NotNil(t, info)
	require.Equal(t, "plus", info.PlanType)
	require.Equal(t, wantActiveUntil, info.ActiveUntil)
	require.Equal(t, "live", info.Source)
	require.NotEmpty(t, info.CheckedAt)
	_, err := time.Parse(time.RFC3339, info.CheckedAt)
	require.NoError(t, err)
}

func TestFetchChatGPTSubscriptionInfoPreservesWillRenewTriState(t *testing.T) {
	for _, tt := range []struct {
		name      string
		payload   map[string]any
		wantSet   bool
		wantRenew bool
	}{
		{
			name:      "true",
			payload:   map[string]any{"will_renew": true},
			wantSet:   true,
			wantRenew: true,
		},
		{
			name:      "false is present",
			payload:   map[string]any{"will_renew": false},
			wantSet:   true,
			wantRenew: false,
		},
		{
			name:    "omitted is unknown",
			payload: map[string]any{},
			wantSet: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := newChatGPTSubscriptionInfoServer(t, tt.payload)
			defer server.Close()

			info := fetchChatGPTSubscriptionInfo(
				context.Background(),
				testChatGPTSubscriptionClientFactory,
				"access-token",
				"",
				"acc_123",
			)

			require.NotNil(t, info)
			if tt.wantSet {
				require.NotNil(t, info.WillRenew)
				require.Equal(t, tt.wantRenew, *info.WillRenew)
			} else {
				require.Nil(t, info.WillRenew)
			}
		})
	}
}

func TestFetchChatGPTSubscriptionInfoReturnsEmptyResultOnSuccessfulEmptySubscription(t *testing.T) {
	server := newChatGPTSubscriptionInfoServer(t, map[string]any{})
	defer server.Close()

	info := fetchChatGPTSubscriptionInfo(
		context.Background(),
		testChatGPTSubscriptionClientFactory,
		"access-token",
		"",
		"acc_123",
	)

	require.NotNil(t, info, "a successful empty response must differ from a failed lookup")
	require.Empty(t, info.PlanType)
	require.Empty(t, info.ActiveUntil)
	require.Nil(t, info.WillRenew)
	require.Equal(t, "live", info.Source)
	require.NotEmpty(t, info.CheckedAt)
}

func newChatGPTSubscriptionInfoServer(t *testing.T, payload map[string]any) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/backend-api/subscriptions", r.URL.Path)
		require.Equal(t, "acc_123", r.URL.Query().Get("account_id"))
		require.Equal(t, "Bearer access-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(payload))
	}))

	previousURL := chatGPTSubscriptionsURL
	chatGPTSubscriptionsURL = server.URL + "/backend-api/subscriptions"
	t.Cleanup(func() { chatGPTSubscriptionsURL = previousURL })
	return server
}

func testChatGPTSubscriptionClientFactory(string) (*req.Client, error) {
	return req.C().SetTimeout(5 * time.Second), nil
}
