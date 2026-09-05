//go:build unit

package schema

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserPlatformQuotaPlatformValidator_AllowsDeepseek(t *testing.T) {
	var validators []any
	for _, f := range (UserPlatformQuota{}).Fields() {
		if d := f.Descriptor(); d.Name == "platform" {
			validators = d.Validators
			break
		}
	}
	require.NotEmpty(t, validators, "platform field should expose validators")

	for _, candidate := range []string{"deepseek", "anthropic", "grok"} {
		for i, raw := range validators {
			validator, ok := raw.(func(string) error)
			require.Truef(t, ok, "validator %d has unexpected type %T", i, raw)
			require.NoErrorf(t, validator(candidate), "validator %d rejected %q", i, candidate)
		}
	}

	// Keep the schema-side allowlist strict: an unknown platform must still be
	// rejected even though the field itself is otherwise a regular string.
	rejected := false
	for _, raw := range validators {
		validator := raw.(func(string) error)
		if validator("deepseek-unknown") != nil {
			rejected = true
			break
		}
	}
	require.True(t, rejected, "unknown platform should be rejected by the allowlist validator")
}
