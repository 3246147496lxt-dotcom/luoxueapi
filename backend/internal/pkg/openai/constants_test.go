package openai

import (
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
