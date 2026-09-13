package service

import (
	"context"
	"testing"
)

func TestOpenAICompatibleAccountEligibility_AllowsUnknownModelForPassthrough(t *testing.T) {
	base := &Account{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.6": "gpt-5.6"}},
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Extra:       map[string]any{"openai_passthrough": true},
	}

	ctx := context.Background()
	if !isOpenAICompatibleAccountEligibleForRequest(ctx, base, PlatformOpenAI, "gpt-6-astra", false, "") {
		t.Fatal("passthrough account should accept an unknown model")
	}

	base.Extra["openai_passthrough"] = false
	if isOpenAICompatibleAccountEligibleForRequest(ctx, base, PlatformOpenAI, "gpt-6-astra", false, "") {
		t.Fatal("non-passthrough account should enforce its model mapping")
	}
}
