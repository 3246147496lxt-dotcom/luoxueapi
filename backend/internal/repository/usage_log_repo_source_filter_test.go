package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendUsageLogRequestIDWhereConditionAcceptsBareReceiptID(t *testing.T) {
	t.Parallel()

	conditions, args := appendUsageLogRequestIDWhereCondition(nil, nil, "receipt-123")

	require.Len(t, conditions, 1)
	require.Contains(t, conditions[0], "request_id = $1")
	require.Contains(t, conditions[0], "request_id = $2")
	require.Equal(t, []any{"receipt-123", "client:receipt-123"}, args)
}

func TestAppendUsageLogSourceWhereConditionUsesAPIKeyPurpose(t *testing.T) {
	t.Parallel()

	for _, source := range []string{"web_chat", "api"} {
		t.Run(source, func(t *testing.T) {
			conditions, args := appendUsageLogSourceWhereCondition(nil, nil, source)
			require.Len(t, conditions, 1)
			require.Contains(t, conditions[0], "usage_source_key.purpose = $1")
			require.Equal(t, []any{"web_chat"}, args)
			if source == "api" {
				require.True(t, strings.Contains(conditions[0], "NOT EXISTS"))
			} else {
				require.NotContains(t, conditions[0], "NOT EXISTS")
			}
		})
	}
}
