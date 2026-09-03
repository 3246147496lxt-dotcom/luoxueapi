package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newServiceTierManifestRequest(account *Account) codexModelsManifestRequest {
	return codexModelsManifestRequest{
		account:             account,
		credentialAccountID: account.ID,
		credentialAccount:   account,
	}
}

func TestObserveCodexModelsManifestShadowSelectedAlias(t *testing.T) {
	s := newCapabilityManifestTestService()
	parentID := int64(7)
	parent := newCapabilityManifestTestAccount()
	parent.ID = parentID
	parent.Type = AccountTypeOAuth
	parent.Credentials = map[string]any{
		"access_token": "token",
		"base_url":     "https://credential-upstream.example/v1",
	}
	shadow := &Account{
		ID:              42,
		Platform:        PlatformOpenAI,
		Type:            AccountTypeOAuth,
		ParentAccountID: &parentID,
		Credentials:     map[string]any{},
	}
	request := codexModelsManifestRequest{
		account:             shadow,
		credentialAccount:   parent,
		credentialAccountID: parent.ID,
		accountID:           shadow.ID,
	}
	s.observeCodexModelsManifest(request, []byte(`{"models":[{"slug":"gpt-5.6-sol","service_tiers":[{"id":"priority"}]}]}`))
	require.Equal(t, OpenAIServiceTierSupportSupported, s.serviceTierSupport(context.Background(), shadow, openAIServiceTierModel))
}

func newCapabilityManifestTestService() *OpenAIGatewayService {
	return &OpenAIGatewayService{billingService: NewBillingService(nil, nil)}
}

func newCapabilityManifestTestAccount() *Account {
	return &Account{
		ID:          42,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://upstream.example/v1"},
	}
}

func capabilityStateForTest(s *OpenAIGatewayService, account *Account) OpenAIServiceTierSupportState {
	key := openAIServiceTierCapabilityKey(account, openAIServiceTierModel)
	state, ok := s.openAIServiceTierCapabilities.get(key, time.Now())
	if !ok {
		return OpenAIServiceTierSupportUnknown
	}
	return state
}

func TestObserveCodexModelsManifestServiceTierCapability(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		state OpenAIServiceTierSupportState
	}{
		{
			name:  "service tiers priority",
			body:  `{"models":[{"slug":"gpt-5.6-sol","service_tiers":[{"id":"priority"}]}]}`,
			state: OpenAIServiceTierSupportSupported,
		},
		{
			name:  "legacy additional speed tier",
			body:  `{"models":[{"slug":"gpt-5.6","additional_speed_tiers":["fast"]}]}`,
			state: OpenAIServiceTierSupportSupported,
		},
		{
			name:  "target without priority",
			body:  `{"models":[{"slug":"gpt-5.6-sol","service_tiers":[{"id":"standard"}]}]}`,
			state: OpenAIServiceTierSupportUnsupported,
		},
		{
			name:  "target omitted",
			body:  `{"models":[{"slug":"gpt-5.5"}]}`,
			state: OpenAIServiceTierSupportUnsupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newCapabilityManifestTestService()
			account := newCapabilityManifestTestAccount()
			s.observeCodexModelsManifest(newServiceTierManifestRequest(account), []byte(tt.body))
			require.Equal(t, tt.state, capabilityStateForTest(s, account))
		})
	}
}

func TestObserveCodexModelsManifestDoesNotPolluteOnInvalidEnvelope(t *testing.T) {
	s := newCapabilityManifestTestService()
	account := newCapabilityManifestTestAccount()
	s.observeCodexModelsManifest(newServiceTierManifestRequest(account), []byte(`{"object":"list","data":[]}`))
	require.Equal(t, OpenAIServiceTierSupportUnknown, capabilityStateForTest(s, account))

	s.observeCodexModelsManifest(newServiceTierManifestRequest(account), []byte(`{"models":`))
	require.Equal(t, OpenAIServiceTierSupportUnknown, capabilityStateForTest(s, account))
}

func TestOpenAIServiceTierCapabilityKeyIsolatesAccountAndUpstream(t *testing.T) {
	first := newCapabilityManifestTestAccount()
	second := newCapabilityManifestTestAccount()
	second.ID++
	third := newCapabilityManifestTestAccount()
	third.Credentials["base_url"] = "https://other-upstream.example/v1"
	queryTenant := newCapabilityManifestTestAccount()
	queryTenant.Credentials["base_url"] = "https://upstream.example/v1?tenant=other"

	firstKey := openAIServiceTierCapabilityKey(first, openAIServiceTierModel)
	secondKey := openAIServiceTierCapabilityKey(second, openAIServiceTierModel)
	thirdKey := openAIServiceTierCapabilityKey(third, openAIServiceTierModel)
	queryTenantKey := openAIServiceTierCapabilityKey(queryTenant, openAIServiceTierModel)
	require.NotEqual(t, firstKey, secondKey)
	require.NotEqual(t, firstKey, thirdKey)
	require.NotEqual(t, secondKey, thirdKey)
	require.NotEqual(t, firstKey, queryTenantKey, "custom upstream tenant queries must remain isolated")
	require.Equal(t, firstKey, openAIServiceTierCapabilityKey(first, "gpt-5.6"), "model aliases must use the normalized cache key")
}

func TestOpenAIServiceTierOAuthCapabilityKeyIgnoresRequestQuery(t *testing.T) {
	previous := chatgptCodexModelsURL
	t.Cleanup(func() { chatgptCodexModelsURL = previous })
	account := &Account{ID: 9, Platform: PlatformOpenAI, Type: AccountTypeOAuth}
	chatgptCodexModelsURL = "https://chatgpt.example/backend-api/codex/models?client_version=old"
	first := openAIServiceTierCapabilityKey(account, openAIServiceTierModel)
	chatgptCodexModelsURL = "https://chatgpt.example/backend-api/codex/models?client_version=new"
	second := openAIServiceTierCapabilityKey(account, openAIServiceTierModel)
	require.Equal(t, first, second)
}

func TestOpenAIServiceTierCapabilityCacheUsesUnknownAndNegativeTTLs(t *testing.T) {
	cache := openAIServiceTierCapabilityCache{}
	now := time.Now()
	cache.set("unknown", OpenAIServiceTierSupportUnknown, now)
	cache.set("unsupported", OpenAIServiceTierSupportUnsupported, now)
	_, ok := cache.get("unknown", now.Add(openAIServiceTierCapabilityUnknown-time.Nanosecond))
	require.True(t, ok)
	_, ok = cache.get("unknown", now.Add(openAIServiceTierCapabilityUnknown+time.Nanosecond))
	require.False(t, ok)
	_, ok = cache.get("unsupported", now.Add(openAIServiceTierCapabilityNegative-time.Nanosecond))
	require.True(t, ok)
	_, ok = cache.get("unsupported", now.Add(openAIServiceTierCapabilityNegative+time.Nanosecond))
	require.False(t, ok)
}

func TestServiceTierSupportUnknownProbeDoesNotBlockRequest(t *testing.T) {
	probeStarted := make(chan struct{})
	releaseProbe := make(chan struct{})
	upstream := &codexModelsHTTPUpstreamStub{do: func(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		close(probeStarted)
		<-releaseProbe
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(
				`{"models":[{"slug":"gpt-5.6-sol","service_tiers":[{"id":"priority"}]}]}`,
			)),
		}, nil
	}}
	s := newCodexModelsAPIKeyTestService(upstream)
	s.billingService = NewBillingService(nil, nil)
	account := newCodexModelsAPIKeyTestAccount("https://upstream.example/v1")

	result := make(chan OpenAIServiceTierSupportState, 1)
	go func() {
		result <- s.serviceTierSupport(context.Background(), account, openAIServiceTierModel)
	}()
	select {
	case state := <-result:
		require.Equal(t, OpenAIServiceTierSupportUnknown, state)
	case <-time.After(time.Second):
		t.Fatal("unknown capability lookup waited for the manifest network probe")
	}

	select {
	case <-probeStarted:
	case <-time.After(time.Second):
		t.Fatal("background manifest probe did not start")
	}
	close(releaseProbe)
	require.Eventually(t, func() bool {
		return capabilityStateForTest(s, account) == OpenAIServiceTierSupportSupported
	}, time.Second, 10*time.Millisecond)
}

func TestOpenAIServiceTierCapabilityCacheExpiryDoesNotDeleteConcurrentRefresh(t *testing.T) {
	cache := openAIServiceTierCapabilityCache{}
	now := time.Now()
	cache.set("account", OpenAIServiceTierSupportUnsupported, now)
	cache.mu.Lock()
	cache.entries["account"] = openAIServiceTierCapabilityEntry{
		state:     OpenAIServiceTierSupportUnsupported,
		expiresAt: now.Add(-time.Second),
	}
	cache.mu.Unlock()

	// Simulate a refresh racing with an expired reader. The conditional cleanup
	// must leave the refreshed supported value intact.
	cache.set("account", OpenAIServiceTierSupportSupported, time.Now())
	state, ok := cache.get("account", now.Add(time.Second))
	require.True(t, ok)
	require.Equal(t, OpenAIServiceTierSupportSupported, state)
}

func TestOpenAIServiceTierCapabilityCacheProbeFailureDoesNotClobberObservation(t *testing.T) {
	cache := openAIServiceTierCapabilityCache{}
	now := time.Now()
	cache.set("account", OpenAIServiceTierSupportSupported, now)
	cache.setUnknownIfNotKnown("account", now.Add(time.Second))
	state, ok := cache.get("account", now.Add(time.Second))
	require.True(t, ok)
	require.Equal(t, OpenAIServiceTierSupportSupported, state)
}

func TestObserveCodexModelsManifestRecognizesModelFieldAliases(t *testing.T) {
	for _, field := range []string{"model", "model_slug"} {
		t.Run(field, func(t *testing.T) {
			s := newCapabilityManifestTestService()
			account := newCapabilityManifestTestAccount()
			body := `{"models":[{"` + field + `":"gpt-5.6-sol","additional_speed_tiers":["fast"]}]}`
			s.observeCodexModelsManifest(newServiceTierManifestRequest(account), []byte(body))
			require.Equal(t, OpenAIServiceTierSupportSupported, capabilityStateForTest(s, account))
		})
	}
}
