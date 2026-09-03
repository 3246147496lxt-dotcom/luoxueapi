package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func newPriorityInjectionTestService() *OpenAIGatewayService {
	return &OpenAIGatewayService{
		billingService: NewBillingService(nil, nil),
	}
}

func newPriorityInjectionTestAccount(platform string) *Account {
	return &Account{
		ID:          42,
		Platform:    platform,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://upstream.example/v1"},
	}
}

func markPriorityCapability(s *OpenAIGatewayService, account *Account, state OpenAIServiceTierSupportState) {
	key := openAIServiceTierCapabilityKey(account, openAIServiceTierModel)
	s.openAIServiceTierCapabilities.set(key, state, time.Now())
}

func TestInjectDefaultOpenAIServiceTierMatrix(t *testing.T) {
	service := newPriorityInjectionTestService()
	account := newPriorityInjectionTestAccount(PlatformOpenAI)
	priorityContext := context.WithValue(context.Background(), ctxkey.OpenAIServiceTierPreference, ServiceTierPreferencePriority)
	standardContext := context.WithValue(context.Background(), ctxkey.OpenAIServiceTierPreference, ServiceTierPreferenceStandard)

	t.Run("priority preference and verified capability injects", func(t *testing.T) {
		markPriorityCapability(service, account, OpenAIServiceTierSupportSupported)
		body, injected, err := service.injectDefaultOpenAIServiceTier(priorityContext, nil, account, "gpt-5.6-sol", []byte(`{"model":"gpt-5.6-sol"}`))
		require.NoError(t, err)
		require.True(t, injected)
		require.JSONEq(t, `{"model":"gpt-5.6-sol","service_tier":"priority"}`, string(body))
	})

	t.Run("standard preference leaves body unchanged", func(t *testing.T) {
		markPriorityCapability(service, account, OpenAIServiceTierSupportSupported)
		original := []byte(`{"model":"gpt-5.6-sol"}`)
		body, injected, err := service.injectDefaultOpenAIServiceTier(standardContext, nil, account, "gpt-5.6-sol", original)
		require.NoError(t, err)
		require.False(t, injected)
		require.Equal(t, string(original), string(body))
	})

	t.Run("unknown and unsupported capability leave body unchanged", func(t *testing.T) {
		for _, state := range []OpenAIServiceTierSupportState{
			OpenAIServiceTierSupportUnknown,
			OpenAIServiceTierSupportUnsupported,
		} {
			stateService := newPriorityInjectionTestService()
			markPriorityCapability(stateService, account, state)
			original := []byte(`{"model":"gpt-5.6-sol"}`)
			body, injected, err := stateService.injectDefaultOpenAIServiceTier(priorityContext, nil, account, "gpt-5.6-sol", original)
			require.NoError(t, err)
			require.False(t, injected)
			require.Equal(t, string(original), string(body))
		}
	})

	t.Run("explicit service tier including null is never overwritten", func(t *testing.T) {
		markPriorityCapability(service, account, OpenAIServiceTierSupportSupported)
		for _, raw := range []string{`"priority"`, `"fast"`, `"flex"`, `"auto"`, `"default"`, `"scale"`, `""`, `null`, `false`, `0`, `{}`} {
			original := []byte(`{"model":"gpt-5.6-sol","service_tier":` + raw + `}`)
			body, injected, err := service.injectDefaultOpenAIServiceTier(priorityContext, nil, account, "gpt-5.6-sol", original)
			require.NoError(t, err)
			require.False(t, injected)
			require.Equal(t, string(original), string(body))
		}
	})

	t.Run("non-target models and platforms are excluded", func(t *testing.T) {
		markPriorityCapability(service, account, OpenAIServiceTierSupportSupported)
		for _, test := range []struct {
			name    string
			account *Account
			model   string
		}{
			{name: "other model", account: account, model: "gpt-5.5"},
			{name: "grok", account: newPriorityInjectionTestAccount(PlatformGrok), model: "gpt-5.6-sol"},
		} {
			t.Run(test.name, func(t *testing.T) {
				body, injected, err := service.injectDefaultOpenAIServiceTier(priorityContext, nil, test.account, test.model, []byte(`{"model":"`+test.model+`"}`))
				require.NoError(t, err)
				require.False(t, injected)
				require.Contains(t, string(body), `"model"`)
			})
		}
	})
}
