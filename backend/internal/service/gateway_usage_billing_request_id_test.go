package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestResolveUsageBillingRequestIDWebChatCannotReuseClientXRequestID(t *testing.T) {
	newRequestContext := func(clientRequestID string) context.Context {
		ctx := context.WithValue(context.Background(), ctxkey.RequestID, "attacker-reused-request-id")
		ctx = context.WithValue(ctx, ctxkey.ClientRequestID, clientRequestID)
		return context.WithValue(ctx, ctxkey.WebChat, true)
	}

	first := resolveUsageBillingRequestID(newRequestContext("server-generated-1"), "upstream-id")
	second := resolveUsageBillingRequestID(newRequestContext("server-generated-2"), "upstream-id")

	require.Equal(t, "client:server-generated-1", first)
	require.Equal(t, "client:server-generated-2", second)
	require.NotEqual(t, first, second)
}
