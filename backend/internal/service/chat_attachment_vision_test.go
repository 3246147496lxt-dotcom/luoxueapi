package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelPricingResolverSupportsVisionIsExactAndFailClosed(t *testing.T) {
	pricing := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"vision-exact": {SupportsVision: true},
		"family":       {SupportsVision: true},
	}}
	resolver := &ModelPricingResolver{billingService: &BillingService{pricingService: pricing}}
	require.True(t, resolver.SupportsVision("vision-exact"))
	require.False(t, resolver.SupportsVision("vision-exact-20260807"))
	require.False(t, resolver.SupportsVision("family-child"))
	require.False(t, resolver.SupportsVision("unknown"))
}

func TestAccountSupportsVisionChecksFinalAccountMappedModel(t *testing.T) {
	pricing := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"vision-model": {SupportsVision: true},
		"text-model":   {SupportsVision: false},
	}}
	resolver := &ModelPricingResolver{billingService: &BillingService{pricingService: pricing}}
	gateway := &OpenAIGatewayService{resolver: resolver}
	visionAccount := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"model_mapping": map[string]any{"alias": "vision-model"},
	}}
	textAccount := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"model_mapping": map[string]any{"alias": "text-model"},
	}}
	require.True(t, gateway.AccountSupportsVision(visionAccount, "alias"))
	require.False(t, gateway.AccountSupportsVision(textAccount, "alias"))
}

func TestOpenAIRequestBodyHasImageInputScansAllHistoryMessages(t *testing.T) {
	body := []byte(`{"model":"vision-exact","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,QQ=="}}]},{"role":"user","content":"latest text"}]}`)
	require.True(t, OpenAIRequestBodyHasImageInput(body))
}
