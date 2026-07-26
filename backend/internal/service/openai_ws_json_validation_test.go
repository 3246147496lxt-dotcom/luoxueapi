package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestValidateJSONNoDuplicateObjectKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		payload   string
		duplicate bool
	}{
		{
			name:    "unique nested objects",
			payload: `{"type":"response.create","input":[{"type":"message","content":[{"type":"input_text","text":"hello"}]}]}`,
		},
		{
			name:      "ordinary duplicate root key",
			payload:   `{"type":"response.create","type":"response.cancel"}`,
			duplicate: true,
		},
		{
			name:      "escaped equivalent root key",
			payload:   `{"type":"response.create","t\u0079pe":"response.cancel"}`,
			duplicate: true,
		},
		{
			name:      "escaped equivalent nested key",
			payload:   `{"type":"session.update","session":{"model":"gpt-5.4","m\u006fdel":"gpt-5.5"}}`,
			duplicate: true,
		},
		{
			name:      "duplicate inside array object",
			payload:   `{"items":[{"name":"first","na\u006de":"second"}]}`,
			duplicate: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateJSONNoDuplicateObjectKeys([]byte(tt.payload))
			if tt.duplicate {
				require.ErrorIs(t, err, ErrJSONDuplicateObjectKey)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateOpenAIWSFirstClientFrameJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		wantErr string
	}{
		{
			name:    "valid response create",
			payload: `{"type":"response.create","model":"gpt-5.4"}`,
		},
		{
			name:    "missing type",
			payload: `{"model":"gpt-5.4"}`,
			wantErr: "type response.create",
		},
		{
			name:    "wrong type",
			payload: `{"type":"session.update","model":"gpt-5.4"}`,
			wantErr: "type response.create",
		},
		{
			name:    "type must match exactly",
			payload: `{"type":" response.create ","model":"gpt-5.4"}`,
			wantErr: "type response.create",
		},
		{
			name:    "non string type",
			payload: `{"type":null,"model":"gpt-5.4"}`,
			wantErr: "must be a string",
		},
		{
			name:    "duplicate type",
			payload: `{"type":"response.create","type":"session.update","model":"gpt-5.4"}`,
			wantErr: "duplicate JSON object key",
		},
		{
			name:    "escaped duplicate model",
			payload: `{"type":"response.create","model":"gpt-5.4","m\u006fdel":"gpt-5.5"}`,
			wantErr: "duplicate JSON object key",
		},
		{
			name:    "multiple JSON values",
			payload: `{"type":"response.create"} {"type":"response.create"}`,
			wantErr: "trailing token",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateOpenAIWSFirstClientFrameJSON([]byte(tt.payload))
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestOpenAIGatewayService_PassthroughRejectsDuplicateKeysBeforeStartingTurn(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "duplicate type",
			payload: `{"type":"response.create","t\u0079pe":"response.cancel","model":"gpt-5.4"}`,
		},
		{
			name:    "duplicate response model",
			payload: `{"type":"response.create","model":"gpt-5.4","m\u006fdel":"gpt-5.5"}`,
		},
		{
			name:    "duplicate session model",
			payload: `{"type":"session.update","session":{"model":"gpt-5.4","m\u006fdel":"gpt-5.5"}}`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var beforeRequestCalls atomic.Int32
			var beforeTurnCalls atomic.Int32
			var successfulAfterTurnCalls atomic.Int32
			hooks := &OpenAIWSIngressHooks{
				InitialRequestModel: "gpt-5.4",
				BeforeRequest: func(int, []byte, string) error {
					beforeRequestCalls.Add(1)
					return nil
				},
				BeforeTurn: func(int) error {
					beforeTurnCalls.Add(1)
					return nil
				},
				AfterTurn: func(_ int, result *OpenAIForwardResult, turnErr error) {
					if turnErr == nil && result != nil {
						successfulAfterTurnCalls.Add(1)
					}
				},
			}
			harness := newOpenAIWSPassthroughModelLockRelayHarness(t, hooks)
			harness.writeMessage(t, `{"type":"response.create","model":"gpt-5.4","stream":true}`)
			harness.readFirstTurn(t)
			require.Equal(t, int32(1), successfulAfterTurnCalls.Load())

			writesBefore := len(harness.upstreamConn.Writes())
			harness.writeMessage(t, tt.payload)
			proxyErr := harness.waitProxyResult(t)
			require.Error(t, proxyErr)
			var closeErr *OpenAIWSClientCloseError
			require.ErrorAs(t, proxyErr, &closeErr)
			require.Equal(t, coderws.StatusPolicyViolation, closeErr.StatusCode())
			require.ErrorIs(t, proxyErr, ErrJSONDuplicateObjectKey)

			require.Zero(t, beforeRequestCalls.Load(), "重复键帧不得进入 BeforeRequest")
			require.Zero(t, beforeTurnCalls.Load(), "重复键帧不得启动下一 turn")
			require.Equal(t, int32(1), successfulAfterTurnCalls.Load(), "重复键帧不得产生额外 usage turn")
			require.Len(t, harness.upstreamConn.Writes(), writesBefore, "重复键帧不得写入上游")
		})
	}
}

func TestOpenAIGatewayService_CtxPoolRejectsDuplicateKeysBeforeStartingTurn(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "duplicate type",
			payload: `{"type":"response.create","t\u0079pe":"response.cancel","model":"gpt-5.1"}`,
		},
		{
			name:    "duplicate model",
			payload: `{"type":"response.create","model":"gpt-5.1","m\u006fdel":"gpt-5.5"}`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var beforeRequestCalls atomic.Int32
			var beforeTurnCalls atomic.Int32
			var successfulAfterTurnCalls atomic.Int32
			hooks := &OpenAIWSIngressHooks{
				InitialRequestModel: "gpt-5.1",
				BeforeRequest: func(int, []byte, string) error {
					beforeRequestCalls.Add(1)
					return nil
				},
				BeforeTurn: func(int) error {
					beforeTurnCalls.Add(1)
					return nil
				},
				AfterTurn: func(_ int, result *OpenAIForwardResult, turnErr error) {
					if turnErr == nil && result != nil {
						successfulAfterTurnCalls.Add(1)
					}
				},
			}
			clientConn, upstreamConn, serverErrCh := newOpenAIWSCtxPoolJSONValidationHarness(t, hooks)

			writeOpenAIWSJSONValidationMessage(t, clientConn, `{"type":"response.create","model":"gpt-5.1","stream":false}`)
			readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
			_, _, err := clientConn.Read(readCtx)
			cancelRead()
			require.NoError(t, err)

			require.Eventually(t, func() bool {
				return successfulAfterTurnCalls.Load() == 1
			}, 3*time.Second, 10*time.Millisecond, "首轮 terminal 应先完成 turn 结算")
			beforeRequestBaseline := beforeRequestCalls.Load()
			beforeTurnBaseline := beforeTurnCalls.Load()
			afterTurnBaseline := successfulAfterTurnCalls.Load()
			require.Equal(t, int32(1), afterTurnBaseline)
			require.Equal(t, 1, openAIWSCaptureWriteCount(upstreamConn))

			writeOpenAIWSJSONValidationMessage(t, clientConn, tt.payload)
			select {
			case proxyErr := <-serverErrCh:
				require.Error(t, proxyErr)
				var closeErr *OpenAIWSClientCloseError
				require.ErrorAs(t, proxyErr, &closeErr)
				require.Equal(t, coderws.StatusPolicyViolation, closeErr.StatusCode())
				require.ErrorIs(t, proxyErr, ErrJSONDuplicateObjectKey)
			case <-time.After(5 * time.Second):
				t.Fatal("等待 ctx-pool 重复键拒绝结果超时")
			}

			require.Equal(t, beforeRequestBaseline, beforeRequestCalls.Load(), "重复键帧不得进入 BeforeRequest")
			require.Equal(t, beforeTurnBaseline, beforeTurnCalls.Load(), "重复键帧不得启动下一 turn")
			require.Equal(t, afterTurnBaseline, successfulAfterTurnCalls.Load(), "重复键帧不得产生额外 usage turn")
			require.Equal(t, 1, openAIWSCaptureWriteCount(upstreamConn), "重复键帧不得写入上游")
		})
	}
}

func newOpenAIWSCtxPoolJSONValidationHarness(
	t *testing.T,
	hooks *OpenAIWSIngressHooks,
) (*coderws.Conn, *openAIWSCaptureConn, <-chan error) {
	t.Helper()
	setGinTestMode()

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.MaxConnsPerAccount = 1
	cfg.Gateway.OpenAIWS.QueueLimitPerConn = 8
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	upstreamConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.completed","response":{"id":"resp_strict_json_turn_1","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}
	dialer := &openAIWSCaptureDialer{conn: upstreamConn}
	pool := newOpenAIWSConnPool(cfg)
	pool.setClientDialerForTest(dialer)
	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     &httpUpstreamRecorder{},
		cache:            &stubGatewayCache{},
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
		openaiWSPool:     pool,
	}
	account := &Account{
		ID:          9191,
		Name:        "openai-ingress-strict-json",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra:       map[string]any{"responses_websockets_v2_enabled": true},
	}

	serverErrCh := make(chan error, 1)
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
			CompressionMode: coderws.CompressionContextTakeover,
		})
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		_, firstMessage, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		ginCtx.Request = r.Clone(r.Context())
		serverErrCh <- svc.ProxyResponsesWebSocketFromClient(
			r.Context(),
			ginCtx,
			conn,
			account,
			"sk-test",
			firstMessage,
			hooks,
		)
	}))

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(
		dialCtx,
		"ws"+strings.TrimPrefix(wsServer.URL, "http"),
		nil,
	)
	cancelDial()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = clientConn.CloseNow()
		wsServer.Close()
		pool.Close()
	})
	return clientConn, upstreamConn, serverErrCh
}

func writeOpenAIWSJSONValidationMessage(t *testing.T, conn *coderws.Conn, payload string) {
	t.Helper()
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelWrite()
	require.NoError(t, conn.Write(writeCtx, coderws.MessageText, []byte(payload)))
}

func openAIWSCaptureWriteCount(conn *openAIWSCaptureConn) int {
	if conn == nil {
		return 0
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	return len(conn.writes)
}
