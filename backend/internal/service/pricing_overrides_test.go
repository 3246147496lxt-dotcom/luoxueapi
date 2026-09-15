package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPricingOverridesReplacePricesAndPreserveMetadata(t *testing.T) {
	svc := pricingOverrideTestService(t)
	base, err := svc.parsePricingData([]byte(`{
		"hy3": {
			"input_cost_per_token": 9, "output_cost_per_token": 8,
			"cache_creation_input_token_cost": 7, "cache_read_input_token_cost": 6,
			"input_cost_per_token_priority": 10, "output_cost_per_token_priority": 11,
			"cache_creation_input_token_cost_priority": 12, "cache_read_input_token_cost_priority": 13,
			"cache_creation_input_token_cost_above_1hr": 14,
			"long_context_input_token_threshold": 200000, "long_context_input_cost_multiplier": 2,
			"long_context_output_cost_multiplier": 3, "supports_service_tier": true,
			"litellm_provider": "tencent", "mode": "chat", "max_input_tokens": 256000, "max_output_tokens": 128000,
			"supports_vision": true, "supports_reasoning": true, "supports_function_calling": true,
			"supports_prompt_caching": true
		},
		"hy3-preview": {"input_cost_per_token": 5}
	}`))
	require.NoError(t, err)
	writePricingOverrideTestFile(t, svc, `{
		"$schema": "https://example.test/pricing.schema.json",
		"hy3": {"input_cost_per_token": 0, "cache_read_input_token_cost": 0, "source": {"url": "https://example.test/official-prices"}},
		"image-only": {"output_cost_per_image": 0.04, "sources": ["https://example.test/images"]}
	}`)

	merged, err := svc.applyPricingOverrides(base)
	require.NoError(t, err)
	require.Len(t, merged, 3)
	require.NotSame(t, base["hy3"], merged["hy3"])
	require.Same(t, base["hy3-preview"], merged["hy3-preview"], "no alias or family replacement")
	require.Equal(t, 9.0, base["hy3"].InputCostPerToken, "base snapshot must remain untouched")
	require.True(t, merged["hy3"].InputCostPerTokenSet)
	require.Zero(t, merged["hy3"].InputCostPerToken)
	require.True(t, merged["hy3"].CacheReadInputTokenCostSet)
	require.Zero(t, merged["hy3"].CacheReadInputTokenCost)
	require.False(t, merged["hy3"].OutputCostPerTokenSet)
	require.False(t, merged["hy3"].CacheCreationInputTokenCostSet)
	require.Zero(t, merged["hy3"].OutputCostPerToken)
	require.Zero(t, merged["hy3"].CacheCreationInputTokenCost)
	require.Zero(t, merged["hy3"].InputCostPerTokenPriority)
	require.Zero(t, merged["hy3"].OutputCostPerTokenPriority)
	require.Zero(t, merged["hy3"].CacheCreationInputTokenCostPriority)
	require.Zero(t, merged["hy3"].CacheReadInputTokenCostPriority)
	require.Zero(t, merged["hy3"].CacheCreationInputTokenCostAbove1hr)
	require.Zero(t, merged["hy3"].LongContextInputTokenThreshold)
	require.Zero(t, merged["hy3"].LongContextInputCostMultiplier)
	require.Zero(t, merged["hy3"].LongContextOutputCostMultiplier)
	require.False(t, merged["hy3"].SupportsServiceTier, "pricing flags must not inherit stale values")
	require.Equal(t, "tencent", merged["hy3"].LiteLLMProvider)
	require.Equal(t, "chat", merged["hy3"].Mode)
	require.Equal(t, int64(256000), merged["hy3"].MaxInputTokens)
	require.Equal(t, int64(128000), merged["hy3"].MaxOutputTokens)
	require.True(t, merged["hy3"].SupportsVision, "omitted capabilities must survive a price-only override")
	require.True(t, merged["hy3"].SupportsReasoning)
	require.True(t, merged["hy3"].SupportsFunctionCalling)
	require.True(t, merged["hy3"].SupportsPromptCaching)
	require.True(t, merged["image-only"].TokenPricingAbsent)
	require.True(t, merged["image-only"].OutputCostPerImageSet)
}

func TestPricingOverridesExplicitMetadataWins(t *testing.T) {
	svc := pricingOverrideTestService(t)
	base := map[string]*LiteLLMModelPricing{"model": {
		InputCostPerToken: 9, LiteLLMProvider: "old", Mode: "chat",
		MaxInputTokens: 100, MaxOutputTokens: 200, SupportsVision: true,
		SupportsFunctionCalling: true, SupportsReasoning: true, SupportsPromptCaching: true,
	}}
	writePricingOverrideTestFile(t, svc, `{"model":{
		"input_cost_per_token":1, "litellm_provider":"", "mode":"completion",
		"max_input_tokens":0, "max_output_tokens":300,
		"supports_vision":false, "supports_function_calling":false, "supports_prompt_caching":false,
		"supports_service_tier":true
	}}`)
	merged, err := svc.applyPricingOverrides(base)
	require.NoError(t, err)
	pricing := merged["model"]
	require.Empty(t, pricing.LiteLLMProvider)
	require.Equal(t, "completion", pricing.Mode)
	require.Zero(t, pricing.MaxInputTokens)
	require.Equal(t, int64(300), pricing.MaxOutputTokens)
	require.False(t, pricing.SupportsVision)
	require.False(t, pricing.SupportsFunctionCalling)
	require.False(t, pricing.SupportsPromptCaching)
	require.True(t, pricing.SupportsReasoning, "unspecified capability still inherits")
	require.True(t, pricing.SupportsServiceTier)
	require.True(t, base["model"].SupportsVision, "original entry remains immutable")
}

func TestPricingOverridesMetadataSourceUsesStableCaseMatch(t *testing.T) {
	for _, exact := range []bool{false, true} {
		t.Run(map[bool]string{false: "sorted fallback", true: "exact key"}[exact], func(t *testing.T) {
			svc := pricingOverrideTestService(t)
			base := map[string]*LiteLLMModelPricing{
				"MINIMAX-M2.7": {LiteLLMProvider: "sorted-first", SupportsVision: true},
				"MiniMax-M2.7": {LiteLLMProvider: "sorted-second"},
			}
			want := "sorted-first"
			if exact {
				base["minimax-m2.7"] = &LiteLLMModelPricing{LiteLLMProvider: "exact"}
				want = "exact"
			}
			writePricingOverrideTestFile(t, svc, `{"minimax-m2.7":{"input_cost_per_token":1}}`)
			for i := 0; i < 20; i++ {
				merged, err := svc.applyPricingOverrides(base)
				require.NoError(t, err)
				require.Len(t, merged, 1)
				require.Equal(t, want, merged["minimax-m2.7"].LiteLLMProvider)
			}
		})
	}
}

func TestPricingOverridesRejectWholeInvalidFile(t *testing.T) {
	tests := map[string]string{
		"null root":            `null`,
		"array root":           `[]`,
		"empty file":           ``,
		"truncated object":     `{"bad":`,
		"trailing document":    `{} {}`,
		"null model":           `{"bad":null}`,
		"array model":          `{"bad":[]}`,
		"string model":         `{"bad":"invalid"}`,
		"empty model":          `{"bad":{}}`,
		"metadata only":        `{"bad":{"source":"https://example.test"}}`,
		"cache price only":     `{"bad":{"cache_read_input_token_cost":0}}`,
		"null price":           `{"bad":{"input_cost_per_token":1,"output_cost_per_token":null}}`,
		"string price":         `{"bad":{"input_cost_per_token":"1"}}`,
		"overflow price":       `{"bad":{"input_cost_per_token":1e999}}`,
		"invalid boolean":      `{"bad":{"input_cost_per_token":1,"supports_vision":"true"}}`,
		"negative limit":       `{"bad":{"input_cost_per_token":1,"max_input_tokens":-1}}`,
		"negative multiplier":  `{"bad":{"input_cost_per_token":1,"long_context_input_cost_multiplier":-1}}`,
		"fractional limit":     `{"bad":{"input_cost_per_token":1,"long_context_input_token_threshold":1.5}}`,
		"empty model ID":       `{"":{"input_cost_per_token":1}}`,
		"padded model ID":      `{" bad":{"input_cost_per_token":1}}`,
		"reserved model ID":    `{"sample_spec":{"input_cost_per_token":1}}`,
		"nonstring schema":     `{"$schema":{},"bad":{"input_cost_per_token":1}}`,
		"null schema":          `{"$schema":null}`,
		"duplicate model":      `{"bad":{"input_cost_per_token":-1},"bad":{"input_cost_per_token":1}}`,
		"case duplicate model": `{"MiniMax-M2.7":{"input_cost_per_token":1},"minimax-m2.7":{"input_cost_per_token":2}}`,
		"duplicate field":      `{"bad":{"input_cost_per_token":-1,"input_cost_per_token":1}}`,
		"case alias duplicate": `{"bad":{"input_cost_per_token":-1,"INPUT_COST_PER_TOKEN":1}}`,
		"case alias null":      `{"bad":{"input_cost_per_token":1,"OUTPUT_COST_PER_TOKEN":null}}`,
		"mixed valid invalid":  `{"good":{"input_cost_per_token":0},"bad":{"input_cost_per_token":-1}}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			svc := pricingOverrideTestService(t)
			original := &LiteLLMModelPricing{InputCostPerToken: 9}
			base := map[string]*LiteLLMModelPricing{"good": original}
			writePricingOverrideTestFile(t, svc, body)
			merged, err := svc.applyPricingOverrides(base)
			require.Error(t, err)
			require.Len(t, base, 1)
			require.Len(t, merged, 1)
			require.Same(t, original, merged["good"])
			require.Equal(t, 9.0, original.InputCostPerToken)
		})
	}
}

func TestPricingOverridesPersistAcrossLoadDownloadAndRestart(t *testing.T) {
	svc := pricingOverrideTestService(t)
	svc.cfg.Pricing.RemoteURL = "https://pricing.example.test/models.json"
	svc.cfg.Pricing.HashURL = "https://pricing.example.test/models.sha256"
	svc.cfg.Pricing.FallbackFile = filepath.Join(t.TempDir(), "fallback.json")
	require.NoError(t, os.WriteFile(svc.cfg.Pricing.FallbackFile, []byte(`{"fallback-only":{"input_cost_per_token":0.7}}`), 0600))
	localBody := []byte(`{"MiniMax-M2.7":{"input_cost_per_token":9,"cache_read_input_token_cost":7},"unchanged":{"input_cost_per_token":1}}`)
	localDigest := sha256.Sum256(localBody)
	localHash := hex.EncodeToString(localDigest[:])
	require.NoError(t, os.WriteFile(svc.getPricingFilePath(), localBody, 0600))
	require.NoError(t, os.WriteFile(svc.getHashFilePath(), []byte(localHash+"\n"), 0600))
	overrideBody := `{"minimax-m2.7":{"input_cost_per_token":0,"output_cost_per_token":0.000001}}`
	writePricingOverrideTestFile(t, svc, overrideBody)

	require.NoError(t, svc.loadPricingData(svc.getPricingFilePath()))
	require.Equal(t, localHash, svc.localHash)
	assertPricingOverrideIntegrationSnapshot(t, svc, 1)
	loadedRaw, err := os.ReadFile(svc.getPricingFilePath())
	require.NoError(t, err)
	require.Equal(t, localBody, loadedRaw)
	loadedHash, err := os.ReadFile(svc.getHashFilePath())
	require.NoError(t, err)
	require.Equal(t, localHash+"\n", string(loadedHash))

	remoteBody := []byte(`{"MINIMAX-M2.7":{"input_cost_per_token":99,"output_cost_per_token":88},"unchanged":{"input_cost_per_token":2}}`)
	remoteDigest := sha256.Sum256(remoteBody)
	remoteHash := hex.EncodeToString(remoteDigest[:])
	remote := &pricingOverridesRemoteClient{body: remoteBody, hash: remoteHash}
	svc.remoteClient = remote
	require.NoError(t, svc.downloadPricingData())
	assertPricingOverrideIntegrationSnapshot(t, svc, 2)
	require.Equal(t, remoteHash, svc.localHash)
	cachedRaw, err := os.ReadFile(svc.getPricingFilePath())
	require.NoError(t, err)
	require.Equal(t, remoteBody, cachedRaw, "cache must contain original remote bytes, not merged overrides")
	cachedHash, err := os.ReadFile(svc.getHashFilePath())
	require.NoError(t, err)
	require.Equal(t, remoteHash+"\n", string(cachedHash))
	storedOverrides, err := os.ReadFile(filepath.Join(svc.cfg.Pricing.DataDir, pricingOverridesFileName))
	require.NoError(t, err)
	require.Equal(t, overrideBody, string(storedOverrides))

	restarted := NewPricingService(svc.cfg, remote)
	require.NoError(t, restarted.loadPricingData(restarted.getPricingFilePath()))
	assertPricingOverrideIntegrationSnapshot(t, restarted, 2)

	// An invalid operator edit must not publish a partially replaced snapshot
	// or advance the remote cache/hash ahead of the last valid snapshot.
	previous := svc.pricingData["minimax-m2.7"]
	writePricingOverrideTestFile(t, svc, `{"minimax-m2.7":{"input_cost_per_token":3},"bad":{"input_cost_per_token":-1}}`)
	remote.body = []byte(`{"minimax-m2.7":{"input_cost_per_token":100}}`)
	require.Error(t, svc.downloadPricingData())
	require.Error(t, svc.loadPricingData(svc.getPricingFilePath()))
	require.Same(t, previous, svc.pricingData["minimax-m2.7"])
	assertPricingOverrideIntegrationSnapshot(t, svc, 2)
	stillRaw, err := os.ReadFile(svc.getPricingFilePath())
	require.NoError(t, err)
	require.Equal(t, cachedRaw, stillRaw)
	stillHash, err := os.ReadFile(svc.getHashFilePath())
	require.NoError(t, err)
	require.Equal(t, cachedHash, stillHash)
	require.Equal(t, remoteHash, svc.localHash)
}

func assertPricingOverrideIntegrationSnapshot(t *testing.T, svc *PricingService, otherPrice float64) {
	t.Helper()
	require.Len(t, svc.pricingData, 3, "only one case-insensitive copy of the overridden model should remain")
	require.NotContains(t, svc.pricingData, "MiniMax-M2.7")
	require.NotContains(t, svc.pricingData, "MINIMAX-M2.7")
	pricing := svc.GetModelPricing("MiniMax-M2.7")
	require.NotNil(t, pricing)
	require.Zero(t, pricing.InputCostPerToken)
	require.True(t, pricing.InputCostPerTokenSet)
	require.Equal(t, 0.000001, pricing.OutputCostPerToken)
	require.False(t, pricing.CacheReadInputTokenCostSet)
	require.Equal(t, otherPrice, svc.pricingData["unchanged"].InputCostPerToken)
	require.Equal(t, 0.7, svc.pricingData["fallback-only"].InputCostPerToken)
}

type pricingOverridesRemoteClient struct {
	body []byte
	hash string
}

func (c *pricingOverridesRemoteClient) FetchPricingJSON(context.Context, string) ([]byte, error) {
	return c.body, nil
}

func (c *pricingOverridesRemoteClient) FetchHashText(context.Context, string) (string, error) {
	return c.hash, nil
}

func TestPricingOverridesValidateEveryPriceField(t *testing.T) {
	fields := []string{
		"input_cost_per_token", "output_cost_per_token",
		"input_cost_per_token_priority", "output_cost_per_token_priority",
		"cache_creation_input_token_cost", "cache_creation_input_token_cost_priority",
		"cache_creation_input_token_cost_above_1hr", "cache_read_input_token_cost",
		"cache_read_input_token_cost_priority", "output_cost_per_image",
		"input_cost_per_image_token", "output_cost_per_image_token",
	}
	for _, field := range fields {
		t.Run(field, func(t *testing.T) {
			body := `{"model":{"output_cost_per_image":0.04,"` + field + `":-0.01}}`
			if field == "output_cost_per_image" {
				body = `{"model":{"input_cost_per_token":0.01,"output_cost_per_image":-0.01}}`
			}
			_, err := (&PricingService{}).parsePricingOverrides([]byte(body))
			require.ErrorContains(t, err, "must be finite and non-negative")
		})
	}
	for _, number := range []float64{math.Inf(1), math.Inf(-1), math.NaN()} {
		err := validatePricingOverrideNumbers(LiteLLMRawEntry{InputCostPerToken: &number}, nil)
		require.Error(t, err)
	}
}

func TestPricingOverridesReloadWithoutChangingRemoteCache(t *testing.T) {
	svc := pricingOverrideTestService(t)
	remoteJSON, remoteHash := []byte(`{"model":{"input_cost_per_token":9}}`), []byte("remote-hash\n")
	require.NoError(t, os.WriteFile(svc.getPricingFilePath(), remoteJSON, 0600))
	require.NoError(t, os.WriteFile(svc.getHashFilePath(), remoteHash, 0600))
	svc.localHash = "memory-remote-hash"
	base := map[string]*LiteLLMModelPricing{"model": {InputCostPerToken: 9}}
	writePricingOverrideTestFile(t, svc, `{"model":{"input_cost_per_token":1}}`)
	first, err := svc.applyPricingOverrides(base)
	require.NoError(t, err)
	writePricingOverrideTestFile(t, svc, `{"model":{"input_cost_per_token":2}}`)
	second, err := svc.applyPricingOverrides(base)
	require.NoError(t, err)
	require.Equal(t, 1.0, first["model"].InputCostPerToken)
	require.Equal(t, 2.0, second["model"].InputCostPerToken)
	actualJSON, err := os.ReadFile(svc.getPricingFilePath())
	require.NoError(t, err)
	actualHash, err := os.ReadFile(svc.getHashFilePath())
	require.NoError(t, err)
	require.Equal(t, remoteJSON, actualJSON)
	require.Equal(t, remoteHash, actualHash)
	require.Equal(t, "memory-remote-hash", svc.localHash)

	require.NoError(t, os.Remove(filepath.Join(svc.cfg.Pricing.DataDir, pricingOverridesFileName)))
	reverted, err := svc.applyPricingOverrides(base)
	require.NoError(t, err)
	require.Same(t, base["model"], reverted["model"])
}

func TestPricingOverridesMissingEmptyAndReadError(t *testing.T) {
	svc := pricingOverrideTestService(t)
	base := map[string]*LiteLLMModelPricing{"existing": {InputCostPerToken: 9}}
	for _, service := range []*PricingService{nil, {}, {cfg: &config.Config{}}, svc} {
		merged, err := service.applyPricingOverrides(base)
		require.NoError(t, err)
		require.Same(t, base["existing"], merged["existing"])
	}
	for _, body := range []string{`{}`, `{"$schema":"https://example.test/schema"}`} {
		writePricingOverrideTestFile(t, svc, body)
		merged, err := svc.applyPricingOverrides(nil)
		require.NoError(t, err)
		require.Nil(t, merged)
	}
	filePath := filepath.Join(svc.cfg.Pricing.DataDir, pricingOverridesFileName)
	require.NoError(t, os.Remove(filePath))
	require.NoError(t, os.Mkdir(filePath, 0700))
	merged, err := svc.applyPricingOverrides(base)
	require.ErrorContains(t, err, "read pricing overrides")
	require.Same(t, base["existing"], merged["existing"])
}

func pricingOverrideTestService(t *testing.T) *PricingService {
	t.Helper()
	cfg := &config.Config{}
	cfg.Pricing.DataDir = t.TempDir()
	return &PricingService{cfg: cfg}
}

func writePricingOverrideTestFile(t *testing.T, svc *PricingService, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(svc.cfg.Pricing.DataDir, pricingOverridesFileName), []byte(body), 0600))
}

func TestPricingOverridesInvalidStartupPreservesLastUsableCache(t *testing.T) {
	svc := pricingOverrideTestService(t)
	original := []byte(`{"last-good":{"input_cost_per_token":1,"output_cost_per_token":2}}`)
	hash := []byte("last-good-hash\n")
	require.NoError(t, os.WriteFile(svc.getPricingFilePath(), original, 0600))
	require.NoError(t, os.WriteFile(svc.getHashFilePath(), hash, 0600))
	svc.cfg.Pricing.FallbackFile = filepath.Join(t.TempDir(), "fallback.json")
	require.NoError(t, os.WriteFile(svc.cfg.Pricing.FallbackFile, []byte(`{"fallback":{"input_cost_per_token":9}}`), 0600))
	writePricingOverrideTestFile(t, svc, `{"bad":{"input_cost_per_token":-1}}`)
	require.Error(t, svc.useFallbackPricing())
	raw, err := os.ReadFile(svc.getPricingFilePath())
	require.NoError(t, err)
	require.Equal(t, original, raw)
	savedHash, err := os.ReadFile(svc.getHashFilePath())
	require.NoError(t, err)
	require.Equal(t, hash, savedHash)
	require.Nil(t, svc.pricingData)
}
