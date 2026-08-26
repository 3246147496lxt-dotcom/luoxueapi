package dto

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupProfitControlFieldsAreAdminOnly(t *testing.T) {
	t.Parallel()

	src := &service.Group{
		ID:                   42,
		Name:                 "profit-controlled",
		Platform:             service.PlatformOpenAI,
		ProfitControlEnabled: true,
		ProfitMinMargin:      0.25,
		ProfitSafetyBuffer:   0.10,
	}

	userPayload := marshalJSONObject(t, GroupFromService(src))
	require.NotContains(t, userPayload, "profit_control_enabled")
	require.NotContains(t, userPayload, "profit_min_margin")
	require.NotContains(t, userPayload, "profit_safety_buffer")

	adminPayload := marshalJSONObject(t, GroupFromServiceAdmin(src))
	require.Equal(t, true, adminPayload["profit_control_enabled"])
	require.InDelta(t, 0.25, adminPayload["profit_min_margin"], 1e-12)
	require.InDelta(t, 0.10, adminPayload["profit_safety_buffer"], 1e-12)
}

func marshalJSONObject(t *testing.T, value any) map[string]any {
	t.Helper()

	raw, err := json.Marshal(value)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	return payload
}
