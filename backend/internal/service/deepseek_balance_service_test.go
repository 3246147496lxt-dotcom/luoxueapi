package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

type deepSeekBalanceTestRepo struct {
	AccountRepository
	account     *Account
	updateCalls int
	updates     map[string]any
}

func (r *deepSeekBalanceTestRepo) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

func (r *deepSeekBalanceTestRepo) UpdateExtra(_ context.Context, _ int64, updates map[string]any) error {
	r.updateCalls++
	r.updates = updates
	return nil
}

type deepSeekBalanceTestUpstream struct {
	status  int
	body    string
	request *http.Request
	err     error
}

func (u *deepSeekBalanceTestUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.request = req.Clone(req.Context())
	if u.err != nil {
		return nil, u.err
	}
	return &http.Response{
		StatusCode: u.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(u.body)),
	}, nil
}

func (u *deepSeekBalanceTestUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func deepSeekBalanceTestAccount() *Account {
	return &Account{
		ID:       7,
		Platform: PlatformDeepseek,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "sk-test-balance",
			"base_url": "https://api.deepseek.com",
		},
	}
}

func TestDeepSeekBalanceServiceQueriesAndPersistsMultiCurrencyBalance(t *testing.T) {
	repo := &deepSeekBalanceTestRepo{account: deepSeekBalanceTestAccount()}
	upstream := &deepSeekBalanceTestUpstream{
		status: http.StatusOK,
		body:   `{"is_available":true,"balance_infos":[{"currency":"CNY","total_balance":"12.50"},{"currency":"USD","total_balance":3.25}]}`,
	}
	svc := NewDeepSeekBalanceService(repo, nil, upstream, &config.Config{})

	result, err := svc.QueryBalance(context.Background(), repo.account.ID)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.True(t, result.Available)
	require.Equal(t, 12.5, result.Balance)
	require.Equal(t, "CNY", result.Currency)
	require.Equal(t, []DeepSeekBalanceEntry{{Currency: "CNY", Balance: 12.5}, {Currency: "USD", Balance: 3.25}}, result.Balances)
	require.True(t, result.Persisted)
	require.Equal(t, 1, repo.updateCalls)
	require.Equal(t, "https://api.deepseek.com/user/balance", upstream.request.URL.String())
	require.Equal(t, "Bearer sk-test-balance", upstream.request.Header.Get("Authorization"))
	require.Equal(t, 12.5, repo.updates[deepSeekBalanceExtraKey])
}

func TestDeepSeekBalanceServiceDoesNotPersistInvalidPayload(t *testing.T) {
	repo := &deepSeekBalanceTestRepo{account: deepSeekBalanceTestAccount()}
	upstream := &deepSeekBalanceTestUpstream{
		status: http.StatusOK,
		body:   `{"is_available":true,"balance_infos":[{"currency":"CNY","total_balance":"not-a-number"}]}`,
	}
	svc := NewDeepSeekBalanceService(repo, nil, upstream, &config.Config{})

	result, err := svc.QueryBalance(context.Background(), repo.account.ID)

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "no valid balance entries")
	require.Zero(t, repo.updateCalls)
}

func TestDeepSeekBalanceServiceRejectsNonAPIKeyAndCodingAccounts(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*Account)
		reason string
	}{
		{
			name: "wrong platform",
			mutate: func(account *Account) {
				account.Platform = PlatformOpenAI
			},
			reason: "DEEPSEEK_BALANCE_INVALID_PLATFORM",
		},
		{
			name: "oauth type",
			mutate: func(account *Account) {
				account.Type = AccountTypeOAuth
			},
			reason: "DEEPSEEK_BALANCE_INVALID_TYPE",
		},
		{
			name: "coding mode",
			mutate: func(account *Account) {
				account.Credentials["account_mode"] = AccountModeCoding
			},
			reason: "DEEPSEEK_BALANCE_CODING_PLAN",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			account := deepSeekBalanceTestAccount()
			test.mutate(account)
			repo := &deepSeekBalanceTestRepo{account: account}
			upstream := &deepSeekBalanceTestUpstream{status: http.StatusOK, body: `{}`}
			svc := NewDeepSeekBalanceService(repo, nil, upstream, &config.Config{})

			_, err := svc.QueryBalance(context.Background(), account.ID)

			require.Error(t, err)
			require.Contains(t, err.Error(), test.reason)
			require.Nil(t, upstream.request)
		})
	}
}

func TestDeepSeekBalanceServiceHonorsURLAllowlist(t *testing.T) {
	repo := &deepSeekBalanceTestRepo{account: deepSeekBalanceTestAccount()}
	repo.account.Credentials["base_url"] = "https://relay.example.test"
	upstream := &deepSeekBalanceTestUpstream{status: http.StatusOK, body: `{}`}
	cfg := &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{
		Enabled:       true,
		UpstreamHosts: []string{"api.deepseek.com"},
	}}}
	svc := NewDeepSeekBalanceService(repo, nil, upstream, cfg)

	_, err := svc.QueryBalance(context.Background(), repo.account.ID)

	require.Error(t, err)
	require.Contains(t, err.Error(), "DEEPSEEK_BALANCE_URL_REJECTED")
	require.Nil(t, upstream.request)
}

func TestBuildDeepSeekBalanceURLNormalizesAnthropicBasePaths(t *testing.T) {
	for _, test := range []struct {
		name string
		base string
		want string
	}{
		{name: "anthropic", base: "https://api.deepseek.com/anthropic", want: "https://api.deepseek.com/user/balance"},
		{name: "anthropic trailing slash", base: "https://api.deepseek.com/anthropic/", want: "https://api.deepseek.com/user/balance"},
		{name: "anthropic v1", base: "https://api.deepseek.com/anthropic/v1", want: "https://api.deepseek.com/user/balance"},
		{name: "anthropic v1 messages", base: "https://api.deepseek.com/anthropic/v1/messages/", want: "https://api.deepseek.com/user/balance"},
		{name: "custom anthropic path preserved", base: "https://relay.example/ANTHROPIC/V1/", want: "https://relay.example/ANTHROPIC/V1/user/balance"},
		{name: "custom nested anthropic path preserved", base: "https://relay.example/proxy/anthropic", want: "https://relay.example/proxy/anthropic/user/balance"},
		{name: "official nested path preserved", base: "https://api.deepseek.com/proxy/anthropic", want: "https://api.deepseek.com/proxy/anthropic/user/balance"},
		{name: "already balance endpoint", base: "https://api.deepseek.com/user/balance", want: "https://api.deepseek.com/user/balance"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildDeepSeekBalanceURL(test.base, &config.Config{})
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestDeepSeekOpenAIFormatBaseURLNormalizesOfficialAnthropicOnly(t *testing.T) {
	for _, test := range []struct {
		name string
		base string
		want string
	}{
		{name: "official anthropic", base: "https://api.deepseek.com/anthropic", want: "https://api.deepseek.com"},
		{name: "official anthropic v1", base: "https://api.deepseek.com/anthropic/v1", want: "https://api.deepseek.com"},
		{name: "official messages endpoint", base: "https://api.deepseek.com/anthropic/v1/messages", want: "https://api.deepseek.com"},
		{name: "custom relay path preserved", base: "https://relay.example/proxy/anthropic", want: "https://relay.example/proxy/anthropic"},
	} {
		t.Run(test.name, func(t *testing.T) {
			account := &Account{
				Platform: PlatformDeepseek,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"api_protocol": APIProtocolAnthropic,
					"base_url":     test.base,
				},
			}
			require.Equal(t, test.want, account.GetOpenAIFormatBaseURL())
		})
	}
}

func TestDeepSeekBalanceServiceReturnsUpstreamErrorsWithoutSyntheticZero(t *testing.T) {
	repo := &deepSeekBalanceTestRepo{account: deepSeekBalanceTestAccount()}
	upstream := &deepSeekBalanceTestUpstream{status: http.StatusUnauthorized, body: `{"error":"bad key"}`}
	svc := NewDeepSeekBalanceService(repo, nil, upstream, &config.Config{})

	result, err := svc.QueryBalance(context.Background(), repo.account.ID)

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Equal(t, http.StatusUnauthorized, result.StatusCode)
	require.Contains(t, result.Error, "HTTP 401")
	require.Zero(t, result.Balance)
	require.Zero(t, repo.updateCalls)
}

func TestDeepSeekBalanceServicePropagatesTransportError(t *testing.T) {
	repo := &deepSeekBalanceTestRepo{account: deepSeekBalanceTestAccount()}
	upstream := &deepSeekBalanceTestUpstream{err: errors.New("timeout")}
	svc := NewDeepSeekBalanceService(repo, nil, upstream, &config.Config{})

	_, err := svc.QueryBalance(context.Background(), repo.account.ID)

	require.Error(t, err)
	require.Contains(t, err.Error(), "DEEPSEEK_BALANCE_REQUEST_FAILED")
	require.Zero(t, repo.updateCalls)
}
