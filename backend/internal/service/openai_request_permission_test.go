//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

const imagePermissionTestBody = `{"error":{"type":"permission_error","message":"Image generation is not enabled for this group"}}`

func TestIsOpenAIRequestPermissionError(t *testing.T) {
	for _, tc := range []struct {
		name     string
		platform string
		status   int
		body     string
		want     bool
	}{
		{name: "openai", platform: PlatformOpenAI, status: 403, body: imagePermissionTestBody, want: true},
		{name: "zhipu", platform: PlatformZhipu, status: 403, body: imagePermissionTestBody, want: true},
		{name: "deepseek", platform: PlatformDeepseek, status: 403, body: imagePermissionTestBody, want: true},
		{name: "message", platform: PlatformOpenAI, status: 403, body: `{"message":"Image generation is not enabled for this group"}`, want: true},
		{name: "detail", platform: PlatformOpenAI, status: 403, body: `{"detail":"Image generation is not enabled for this group"}`, want: true},
		{name: "detail_message", platform: PlatformOpenAI, status: 403, body: `{"detail":{"message":"Image generation is not enabled for this group"}}`, want: true},
		{name: "response_error_message", platform: PlatformOpenAI, status: 403, body: `{"response":{"error":{"message":"Image generation is not enabled for this group"}}}`, want: true},
		{name: "plain_text", platform: PlatformOpenAI, status: 403, body: " \nImage generation is not enabled for this group\n", want: true},
		{name: "other_platform", platform: PlatformAnthropic, status: 403, body: imagePermissionTestBody},
		{name: "not_forbidden", platform: PlatformOpenAI, status: 401, body: imagePermissionTestBody},
		{name: "real_forbidden", platform: PlatformOpenAI, status: 403, body: `{"error":{"message":"account suspended"}}`},
		{name: "substring_only", platform: PlatformOpenAI, status: 403, body: `{"error":{"message":"account suspended: Image generation is not enabled for this group"}}`},
		{name: "unrelated_metadata", platform: PlatformOpenAI, status: 403, body: `{"error":{"message":"account suspended"},"metadata":{"message":"Image generation is not enabled for this group"}}`},
		{name: "html_body", platform: PlatformOpenAI, status: 403, body: `<html>Image generation is not enabled for this group</html>`},
		{name: "empty_body", platform: PlatformOpenAI, status: 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account := &Account{ID: 148, Platform: tc.platform, Type: AccountTypeAPIKey}
			require.Equal(t, tc.want, isOpenAIRequestPermissionError(account, tc.status, []byte(tc.body)))
		})
	}
	require.False(t, isOpenAIRequestPermissionError(nil, http.StatusForbidden, []byte(imagePermissionTestBody)))
}

func TestRateLimitService_RequestPermissionErrorSkipsAccountPolicies(t *testing.T) {
	for _, tc := range []struct {
		name        string
		credentials map[string]any
	}{
		{name: "default"},
		{name: "custom_403", credentials: map[string]any{
			"custom_error_codes_enabled": true,
			"custom_error_codes":         []any{float64(403)},
		}},
		{name: "broad_temp_rule", credentials: map[string]any{
			"temp_unschedulable_enabled": true,
			"temp_unschedulable_rules": []any{map[string]any{
				"error_code": float64(403), "keywords": []any{"not enabled"}, "duration_minutes": float64(10),
			}},
		}},
		{name: "custom_and_temp_rule", credentials: map[string]any{
			"custom_error_codes_enabled": true,
			"custom_error_codes":         []any{float64(403)},
			"temp_unschedulable_enabled": true,
			"temp_unschedulable_rules": []any{map[string]any{
				"error_code": float64(403), "keywords": []any{"not enabled"}, "duration_minutes": float64(10),
			}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &rateLimitAccountRepoStub{}
			counter := &openAI403CounterCacheStub{counts: []int64{1, 2, 3}}
			blocker := &runtimeBlockRecorder{}
			svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
			svc.SetOpenAI403CounterCache(counter)
			svc.SetAccountRuntimeBlocker(blocker)
			account := &Account{ID: 148, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: tc.credentials}

			for i := 0; i < 3; i++ {
				require.Equal(t, ErrorPolicySkipped, svc.CheckErrorPolicy(context.Background(), account, http.StatusForbidden, []byte(imagePermissionTestBody)))
				require.False(t, svc.HandleTempUnschedulable(context.Background(), account, http.StatusForbidden, []byte(imagePermissionTestBody)))
				require.False(t, svc.HandleUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, []byte(imagePermissionTestBody), "gpt-5.5"))
			}

			require.Zero(t, repo.setErrorCalls)
			require.Zero(t, repo.tempCalls)
			require.Empty(t, blocker.accounts)
			require.Empty(t, blocker.clearedIDs)
			require.Equal(t, []int64{1, 2, 3}, counter.counts, "request permission errors must not increment account 403 counts")
			require.Empty(t, counter.resetCalls, "request permission errors must not erase existing account failures")
		})
	}
}

func TestRateLimitService_RequestPermissionErrorSkipsUnavailable403Counter(t *testing.T) {
	for _, tc := range []struct {
		name    string
		counter *openAI403CounterCacheStub
	}{
		{name: "missing_counter"},
		{name: "failed_counter", counter: &openAI403CounterCacheStub{err: errors.New("redis unavailable")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &rateLimitAccountRepoStub{}
			svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
			if tc.counter != nil {
				svc.SetOpenAI403CounterCache(tc.counter)
			}
			account := &Account{ID: 149, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

			require.False(t, svc.HandleUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, []byte(imagePermissionTestBody)))
			require.Zero(t, repo.setErrorCalls)
			require.Zero(t, repo.tempCalls)

			// Unknown 403s still fail closed when their counter cannot be used.
			require.True(t, svc.HandleUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, []byte(`{"error":{"message":"account suspended"}}`)))
			require.Equal(t, 1, repo.setErrorCalls)
		})
	}
}

func TestOpenAIGateway_RequestPermissionErrorPreservesRuntimeHealth(t *testing.T) {
	for _, alreadyBlocked := range []bool{false, true} {
		name := "healthy_account"
		if alreadyBlocked {
			name = "existing_account_failure"
		}
		t.Run(name, func(t *testing.T) {
			repo := &rateLimitAccountRepoStub{}
			counter := &openAI403CounterCacheStub{counts: []int64{2}}
			rateLimiter := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
			rateLimiter.SetOpenAI403CounterCache(counter)
			gateway := &OpenAIGatewayService{rateLimitService: rateLimiter}
			rateLimiter.SetAccountRuntimeBlocker(gateway)
			account := &Account{ID: 151, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}
			until := time.Now().Add(30 * time.Minute)
			if alreadyBlocked {
				account.TempUnschedulableUntil = &until
				account.TempUnschedulableReason = "existing authentication failure"
				gateway.BlockAccountScheduling(account, until, "oauth_401")
			}

			require.False(t, gateway.handleOpenAIAccountUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, []byte(imagePermissionTestBody), "gpt-5.5"))
			require.Equal(t, alreadyBlocked, gateway.isOpenAIAccountRuntimeBlocked(account))
			require.Zero(t, repo.setErrorCalls)
			require.Zero(t, repo.tempCalls)
			require.Equal(t, []int64{2}, counter.counts)
			require.Empty(t, counter.resetCalls)
			if alreadyBlocked {
				require.Equal(t, &until, account.TempUnschedulableUntil)
				require.Equal(t, "existing authentication failure", account.TempUnschedulableReason)
				stored, ok := gateway.openaiAccountRuntimeBlockUntil.Load(account.ID)
				require.True(t, ok)
				require.Equal(t, until, stored)
			} else {
				require.True(t, account.IsSchedulable())
			}

			// A different upstream may allow images; preserve the bounded failover
			// path while keeping this account usable for its text requests.
			require.True(t, gateway.shouldFailoverOpenAIUpstreamResponse(http.StatusForbidden, ImageGenerationPermissionMessage(), []byte(imagePermissionTestBody)))
		})
	}
}

func TestRateLimitService_RequestPermissionErrorDoesNotResetReal403Escalation(t *testing.T) {
	repo := &rateLimitAccountRepoStub{}
	counter := &openAI403CounterCacheStub{counts: []int64{1, 2, 3}}
	svc := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	svc.SetOpenAI403CounterCache(counter)
	account := &Account{ID: 148, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	for i := 0; i < 3; i++ {
		require.False(t, svc.HandleUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, []byte(imagePermissionTestBody)))
		require.True(t, svc.HandleUpstreamError(context.Background(), account, http.StatusForbidden, http.Header{}, []byte(`{"error":{"message":"workspace forbidden by policy"}}`)))
	}

	require.Equal(t, 2, repo.tempCalls)
	require.Equal(t, 1, repo.setErrorCalls)
	require.Contains(t, repo.lastErrorMsg, "consecutive_403=3/3")
	require.Empty(t, counter.counts)
	require.Empty(t, counter.resetCalls)
}
