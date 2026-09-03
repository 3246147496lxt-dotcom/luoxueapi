package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func newPrioritySchedulingTestAccount(id int64, priority int, baseURL string) Account {
	return Account{
		ID:          id,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    priority,
		Credentials: map[string]any{"base_url": baseURL, "api_key": "test-key"},
	}
}

func prioritySchedulingContext() context.Context {
	return context.WithValue(context.Background(), ctxkey.OpenAIServiceTierPreference, ServiceTierPreferencePriority)
}

// The legacy/load-aware path must prefer a verified Priority account even when
// the ordinary account priority field would select a Standard account first.
func TestOpenAIPrioritySchedulingLegacyPrefersVerifiedAccount(t *testing.T) {
	fallback := newPrioritySchedulingTestAccount(88001, 0, "https://standard.example/v1")
	supported := newPrioritySchedulingTestAccount(88002, 100, "https://priority.example/v1")
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{fallback, supported}},
		billingService:     NewBillingService(nil, nil),
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
	markPriorityCapability(svc, &supported, OpenAIServiceTierSupportSupported)
	markPriorityCapability(svc, &fallback, OpenAIServiceTierSupportUnsupported)

	selection, err := svc.SelectAccountWithLoadAwareness(
		prioritySchedulingContext(), nil, "", openAIServiceTierModel, nil,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, supported.ID, selection.Account.ID)
}

// The advanced scheduler uses the same capability preference while retaining
// its normal scoring within each capability pool.
func TestOpenAIPrioritySchedulingAdvancedPrefersVerifiedAccount(t *testing.T) {
	fallback := newPrioritySchedulingTestAccount(88101, 0, "https://standard-advanced.example/v1")
	supported := newPrioritySchedulingTestAccount(88102, 100, "https://priority-advanced.example/v1")
	cfg := newSchedulerTestSubscriptionPriorityConfig()
	settings := &openAIAdvancedSchedulerSettingRepoStub{values: map[string]string{
		openAIAdvancedSchedulerSettingKey: "true",
	}}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{fallback, supported}},
		billingService:     NewBillingService(nil, nil),
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		rateLimitService:   &RateLimitService{settingService: NewSettingService(settings, cfg)},
	}
	markPriorityCapability(svc, &supported, OpenAIServiceTierSupportSupported)
	markPriorityCapability(svc, &fallback, OpenAIServiceTierSupportUnsupported)
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)

	selection, _, err := svc.SelectAccountWithScheduler(
		prioritySchedulingContext(), nil, "", "", openAIServiceTierModel, nil,
		OpenAIUpstreamTransportAny, false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, supported.ID, selection.Account.ID)
}

// Account-specific model mappings are resolved before the capability gate.
// The client alias itself does not need to be a built-in GPT-5.6 Sol alias.
func TestOpenAIPrioritySchedulingPrefersVerifiedAccountAfterModelMapping(t *testing.T) {
	const clientAlias = "team-fast-model"
	fallback := newPrioritySchedulingTestAccount(88201, 0, "https://standard-mapped.example/v1")
	supported := newPrioritySchedulingTestAccount(88202, 100, "https://priority-mapped.example/v1")
	fallback.Credentials["model_mapping"] = map[string]any{clientAlias: clientAlias}
	supported.Credentials["model_mapping"] = map[string]any{clientAlias: openAIServiceTierModel}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{fallback, supported}},
		billingService:     NewBillingService(nil, nil),
		cache:              &schedulerTestGatewayCache{},
		cfg:                &config.Config{},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
	markPriorityCapability(svc, &supported, OpenAIServiceTierSupportSupported)

	selection, err := svc.SelectAccountWithLoadAwareness(
		prioritySchedulingContext(), nil, "", clientAlias, nil,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, supported.ID, selection.Account.ID)
}
