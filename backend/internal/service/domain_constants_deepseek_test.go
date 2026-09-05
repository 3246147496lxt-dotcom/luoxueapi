//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/stretchr/testify/require"
)

func TestDeepseekPlatformIsExposedAcrossDomainBoundaries(t *testing.T) {
	require.Equal(t, "deepseek", domain.PlatformDeepseek)
	require.Equal(t, domain.PlatformDeepseek, PlatformDeepseek)
	require.True(t, IsAllowedQuotaPlatform(PlatformDeepseek))
	require.Contains(t, AllowedQuotaPlatforms, PlatformDeepseek)
	require.Contains(t, model.AllPlatforms(), model.PlatformDeepseek)
}

func TestDeepSeekIgnoresLegacyOpenAICapabilityMarkers(t *testing.T) {
	account := &Account{
		Platform: PlatformDeepseek,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"openai_responses_supported": false,
			"openai_capabilities":        []any{"embeddings"},
		},
	}

	if !account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions) {
		t.Fatal("DeepSeek Chat capability must not be gated by OpenAI markers")
	}
	if !account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityResponses) {
		t.Fatal("DeepSeek Responses capability must not be gated by OpenAI markers")
	}
}
