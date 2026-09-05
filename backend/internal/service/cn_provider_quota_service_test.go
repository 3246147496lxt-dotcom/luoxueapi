package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type cnQuotaHardeningRepo struct {
	AccountRepository
	account     *Account
	updateCalls int
}

func (r *cnQuotaHardeningRepo) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

func (r *cnQuotaHardeningRepo) UpdateExtra(context.Context, int64, map[string]any) error {
	r.updateCalls++
	return nil
}

type cnQuotaHardeningUpstream struct {
	body    string
	request *http.Request
}

func (u *cnQuotaHardeningUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.request = req.Clone(req.Context())
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(u.body)),
	}, nil
}

func (u *cnQuotaHardeningUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func cnQuotaHardeningAccount() *Account {
	return &Account{
		ID:       8801,
		Platform: PlatformZhipu,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"account_mode": AccountModeCoding,
			"api_key":      "sk-zhipu-hardening",
			"base_url":     DefaultZhipuCodingBaseURL,
		},
	}
}

func TestCNProviderQuotaQueryUsageNilContextIsSafe(t *testing.T) {
	upstream := &cnQuotaHardeningUpstream{body: `{"success":true,"data":{"limits":[]}}`}
	repo := &cnQuotaHardeningRepo{account: cnQuotaHardeningAccount()}
	svc := NewCNProviderQuotaService(repo, nil, upstream, &config.Config{})

	result, err := svc.QueryUsage(nil, repo.account.ID)

	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, 1, repo.updateCalls)
	require.NotNil(t, upstream.request)
}

func TestCNProviderQuotaRequestUsesOpenAIProfileAndDisablesRedirects(t *testing.T) {
	upstream := &cnQuotaHardeningUpstream{body: `{"success":true,"data":{"limits":[]}}`}
	repo := &cnQuotaHardeningRepo{account: cnQuotaHardeningAccount()}
	svc := NewCNProviderQuotaService(repo, nil, upstream, &config.Config{})

	_, err := svc.QueryUsage(context.Background(), repo.account.ID)

	require.NoError(t, err)
	require.NotNil(t, upstream.request)
	require.Equal(t, HTTPUpstreamProfileOpenAI, HTTPUpstreamProfileFromContext(upstream.request.Context()))
	require.True(t, HTTPUpstreamRedirectsDisabled(upstream.request.Context()))
	require.Equal(t, "sk-zhipu-hardening", upstream.request.Header.Get("Authorization"))
}

func TestParseZhipuTokenTiersSortsStringResetTimesByTimestamp(t *testing.T) {
	data := gjson.Parse(`{
		"limits": [
			{"type":"TOKENS_LIMIT","percentage":80,"nextResetTime":"2026-09-05T20:00:00Z"},
			{"type":"TOKENS_LIMIT","percentage":20,"nextResetTime":"2026-09-05T10:00:00Z"}
		]
	}`)

	tiers := parseZhipuTokenTiers(data)

	require.Len(t, tiers, 2)
	require.Equal(t, "5h", tiers[0].Window)
	require.Equal(t, 20.0, tiers[0].UsedPercent)
	require.Equal(t, "weekly", tiers[1].Window)
	require.Equal(t, 80.0, tiers[1].UsedPercent)
}
