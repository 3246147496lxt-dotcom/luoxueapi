package openai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultModelsIncludeBareGPT56Alias(t *testing.T) {
	require.Contains(t, DefaultModelIDs(), "gpt-5.6")
}

func TestDefaultModelsExcludeRetiredGPT52Family(t *testing.T) {
	for _, model := range DefaultModelIDs() {
		normalized := strings.ToLower(strings.TrimSpace(model))
		require.False(t, normalized == "gpt-5.2" || strings.HasPrefix(normalized, "gpt-5.2-"))
	}
}

func TestGPT56ReasoningCapabilities(t *testing.T) {
	tests := []struct {
		model         string
		wantPro       bool
		wantMaxEffort bool
	}{
		{model: "gpt-5.6", wantPro: true},
		{model: "gpt-5.6-sol", wantPro: true},
		{model: "gpt-5.6-terra", wantPro: true, wantMaxEffort: true},
		{model: "gpt-5.6-luna", wantPro: true, wantMaxEffort: true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			model, ok := DefaultModelByID(tt.model)
			require.True(t, ok)
			require.True(t, model.SupportsResponses)
			require.True(t, model.SupportsReasoningSummary)
			require.Equal(t, tt.wantPro, model.SupportsReasoningProMode)
			require.True(t, model.SupportsReasoningEffort("medium"))
			require.Equal(t, tt.wantMaxEffort, model.SupportsReasoningEffort("max"))
		})
	}
}

func TestDefaultModelByIDProviderPrefixAndUnknownVariant(t *testing.T) {
	model, ok := DefaultModelByID(" OpenAI/GPT-5.6-Sol ")
	require.True(t, ok)
	require.Equal(t, "gpt-5.6-sol", model.ID)

	_, ok = DefaultModelByID("gpt-5.6-sol-unverified")
	require.False(t, ok, "unknown variants must not inherit Pro capability")
}

func TestModelCapabilitiesUseSnakeCaseJSON(t *testing.T) {
	model, ok := DefaultModelByID("gpt-5.6-sol")
	require.True(t, ok)
	payload, err := json.Marshal(model)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, true, decoded["supports_responses"])
	require.Equal(t, true, decoded["supports_reasoning_summary"])
	require.Equal(t, true, decoded["supports_reasoning_pro_mode"])
	require.NotEmpty(t, decoded["supported_reasoning_efforts"])
	require.NotContains(t, decoded, "supportsResponses")
}
