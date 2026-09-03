package service

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeGroupPlatformDefaultsOnlyEmptyPlatform(t *testing.T) {
	t.Parallel()

	require.Equal(t, PlatformAnthropic, NormalizeGroupPlatform(""))
	for _, platform := range []string{
		PlatformOpenAI,
		PlatformAnthropic,
		PlatformGemini,
		PlatformGrok,
		PlatformAntigravity,
	} {
		platform := platform
		t.Run(platform, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, platform, NormalizeGroupPlatform(platform))
		})
	}
}

func TestValidateProfitControlConfigBoundaries(t *testing.T) {
	t.Parallel()

	valid := []struct {
		name         string
		platform     string
		minMargin    float64
		safetyBuffer float64
	}{
		{name: "zero ratios", platform: PlatformOpenAI},
		{name: "sum below one", platform: PlatformAnthropic, minMargin: 0.75, safetyBuffer: 0.2499},
		{name: "gemini", platform: PlatformGemini, minMargin: 0.2, safetyBuffer: 0.1},
		{name: "grok", platform: PlatformGrok, minMargin: 0.2, safetyBuffer: 0.1},
		{name: "antigravity", platform: PlatformAntigravity, minMargin: 0.2, safetyBuffer: 0.1},
	}
	for _, tc := range valid {
		tc := tc
		t.Run("valid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ValidateProfitControlConfig(tc.platform, true, tc.minMargin, tc.safetyBuffer))
		})
	}

	invalid := []struct {
		name         string
		platform     string
		minMargin    float64
		safetyBuffer float64
	}{
		{name: "unsupported platform", platform: "sora", minMargin: 0.2, safetyBuffer: 0.1},
		{name: "negative margin", platform: PlatformOpenAI, minMargin: -0.01},
		{name: "margin equals one", platform: PlatformOpenAI, minMargin: 1},
		{name: "negative buffer", platform: PlatformOpenAI, safetyBuffer: -0.01},
		{name: "buffer equals one", platform: PlatformOpenAI, safetyBuffer: 1},
		{name: "sum equals one", platform: PlatformOpenAI, minMargin: 0.8, safetyBuffer: 0.2},
		{name: "nan margin", platform: PlatformOpenAI, minMargin: math.NaN()},
		{name: "infinite buffer", platform: PlatformOpenAI, safetyBuffer: math.Inf(1)},
	}
	for _, tc := range invalid {
		tc := tc
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, ValidateProfitControlConfig(tc.platform, true, tc.minMargin, tc.safetyBuffer))
		})
	}

	// Disabled policy is inert; dormant values are normalized separately.
	require.NoError(t, ValidateProfitControlConfig("unsupported", false, math.NaN(), math.Inf(1)))
}

func TestNormalizeProfitControlConfigClearsUnsupportedAndSanitizesDormantValues(t *testing.T) {
	t.Parallel()

	enabled, margin, buffer := NormalizeProfitControlConfig("sora", true, 0.2, 0.1)
	require.False(t, enabled)
	require.Zero(t, margin)
	require.Zero(t, buffer)

	enabled, margin, buffer = NormalizeProfitControlConfig(PlatformOpenAI, false, math.NaN(), math.Inf(1))
	require.False(t, enabled)
	require.Zero(t, margin)
	require.Zero(t, buffer)

	enabled, margin, buffer = NormalizeProfitControlConfig(PlatformGemini, false, 0.2, 0.1)
	require.False(t, enabled)
	require.Equal(t, 0.2, margin)
	require.Equal(t, 0.1, buffer)

	// Normalization mirrors NUMERIC(10,4), so validation applies to the values
	// that PostgreSQL will actually return.
	enabled, margin, buffer = NormalizeProfitControlConfig(PlatformOpenAI, true, 0.99994, 0.00005)
	require.True(t, enabled)
	require.Equal(t, 0.9999, margin)
	require.Equal(t, 0.0001, buffer)
	require.Error(t, ValidateProfitControlConfig(PlatformOpenAI, enabled, margin, buffer))

	enabled, margin, buffer = NormalizeProfitControlConfig(PlatformOpenAI, false, 0.99996, 0)
	require.False(t, enabled)
	require.Zero(t, margin, "a dormant ratio that rounds to 1 must be sanitized")
	require.Zero(t, buffer)

	// Enabled invalid values are deliberately preserved for validation to
	// reject; normalization must not silently turn a bad active policy inert.
	enabled, margin, buffer = NormalizeProfitControlConfig(PlatformGrok, true, -0.1, 0.2)
	require.True(t, enabled)
	require.Equal(t, -0.1, margin)
	require.Equal(t, 0.2, buffer)
}
