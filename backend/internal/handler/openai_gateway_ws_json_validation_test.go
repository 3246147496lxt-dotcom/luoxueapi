//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesWebSocket_RejectsDuplicateFirstFrameKeysBeforeAccountSelection(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "duplicate type",
			payload: `{"type":"response.create","t\u0079pe":"session.update","model":"grok"}`,
		},
		{
			name:    "duplicate model",
			payload: `{"type":"response.create","model":"grok","m\u006fdel":"other"}`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, repo, upstream, router, cleanup := newGrokCredentialFailoverHandler(t, "missing_row")
			defer cleanup()

			server, clientConn := dialOpenAIWSJSONValidationHandler(t, router)
			defer server()
			defer func() {
				_ = clientConn.CloseNow()
			}()

			writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
			err := clientConn.Write(writeCtx, coderws.MessageText, []byte(tt.payload))
			cancelWrite()
			require.NoError(t, err)

			readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
			_, _, err = clientConn.Read(readCtx)
			cancelRead()
			require.Error(t, err)
			var closeErr coderws.CloseError
			require.ErrorAs(t, err, &closeErr)
			require.Equal(t, coderws.StatusPolicyViolation, closeErr.Code)
			require.Zero(t, repo.selectorCalls(), "重复键首包不得进入账号选择")
			require.Empty(t, upstream.accountHits(), "重复键首包不得连接或请求上游")
		})
	}
}

func dialOpenAIWSJSONValidationHandler(t *testing.T, router http.Handler) (func(), *coderws.Conn) {
	t.Helper()
	server := httptest.NewServer(router)
	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(
		dialCtx,
		"ws"+strings.TrimPrefix(server.URL, "http")+"/openai/v1/responses",
		nil,
	)
	cancelDial()
	require.NoError(t, err)
	return server.Close, clientConn
}
