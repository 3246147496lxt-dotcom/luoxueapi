package service

import (
	"context"
	"errors"
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
	"github.com/tidwall/gjson"
)

type openAIWSPassthroughModelLockRelayHarness struct {
	clientConn   *coderws.Conn
	upstreamConn *openAIWSPassthroughDelayedTerminalConn
	dialer       *openAIWSQueueDialer
	serverErrCh  chan error
}

func newOpenAIWSPassthroughModelLockRelayHarness(
	t *testing.T,
	hooks *OpenAIWSIngressHooks,
) *openAIWSPassthroughModelLockRelayHarness {
	t.Helper()
	setGinTestMode()

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
	cfg.Gateway.OpenAIWS.IngressModeDefault = OpenAIWSIngressModeCtxPool
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	upstreamConn := newOpenAIWSPassthroughDelayedTerminalConn([]byte(
		`{"type":"response.completed","response":{"id":"resp_model_lock_turn_1","model":"gpt-5.4","usage":{"input_tokens":2,"output_tokens":1}}}`,
	))
	dialer := &openAIWSQueueDialer{
		conns: []openAIWSClientConn{upstreamConn},
	}
	svc := &OpenAIGatewayService{
		cfg:                       cfg,
		httpUpstream:              &httpUpstreamRecorder{},
		cache:                     &stubGatewayCache{},
		openaiWSResolver:          NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:             NewCodexToolCorrector(),
		openaiWSPassthroughDialer: dialer,
	}
	account := &Account{
		ID:          456,
		Name:        "openai-ingress-passthrough-model-lock",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
		Extra: map[string]any{
			"openai_apikey_responses_websockets_v2_mode": OpenAIWSIngressModePassthrough,
		},
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

		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		req := r.Clone(r.Context())
		req.Header = req.Header.Clone()
		req.Header.Set("User-Agent", "unit-test-agent/1.0")
		ginCtx.Request = req

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		msgType, firstMessage, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
			serverErrCh <- errors.New("unsupported websocket client message type")
			return
		}

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

	harness := &openAIWSPassthroughModelLockRelayHarness{
		clientConn:   clientConn,
		upstreamConn: upstreamConn,
		dialer:       dialer,
		serverErrCh:  serverErrCh,
	}
	t.Cleanup(func() {
		_ = clientConn.CloseNow()
		_ = upstreamConn.Close()
		wsServer.Close()
	})
	return harness
}

func (h *openAIWSPassthroughModelLockRelayHarness) writeMessage(t *testing.T, payload string) {
	t.Helper()
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelWrite()
	require.NoError(t, h.clientConn.Write(writeCtx, coderws.MessageText, []byte(payload)))
}

func (h *openAIWSPassthroughModelLockRelayHarness) readMessage(t *testing.T) []byte {
	t.Helper()
	readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelRead()
	msgType, payload, err := h.clientConn.Read(readCtx)
	require.NoError(t, err)
	require.Equal(t, coderws.MessageText, msgType)
	return payload
}

func (h *openAIWSPassthroughModelLockRelayHarness) readFirstTurn(t *testing.T) {
	t.Helper()
	require.Equal(t, "response.created", gjson.GetBytes(h.readMessage(t), "type").String())
	terminal := h.readMessage(t)
	require.Equal(t, "response.completed", gjson.GetBytes(terminal, "type").String())
	require.Equal(t, "resp_model_lock_turn_1", gjson.GetBytes(terminal, "response.id").String())
}

func (h *openAIWSPassthroughModelLockRelayHarness) waitProxyResult(t *testing.T) error {
	t.Helper()
	select {
	case err := <-h.serverErrCh:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("等待 passthrough model-lock relay 结束超时")
		return nil
	}
}

func TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_PassthroughRewritesOriginalAliasToLockedForwardModel(t *testing.T) {
	const (
		originalModel = "gpt-5.4-xhigh"
		forwardModel  = "gpt-5.4"
	)
	var beforeTurnCalls atomic.Int32
	results := make(chan *OpenAIForwardResult, 2)
	beforeRequestModels := make(chan string, 1)
	hooks := &OpenAIWSIngressHooks{
		InitialRequestModel: originalModel,
		BeforeRequest: func(_ int, _ []byte, original string) error {
			beforeRequestModels <- original
			return nil
		},
		BeforeTurn: func(int) error {
			beforeTurnCalls.Add(1)
			return nil
		},
		AfterTurn: func(_ int, result *OpenAIForwardResult, turnErr error) {
			if turnErr == nil && result != nil {
				results <- result
			}
		},
	}
	harness := newOpenAIWSPassthroughModelLockRelayHarness(t, hooks)
	waitResult := func() *OpenAIForwardResult {
		t.Helper()
		select {
		case result := <-results:
			return result
		case <-time.After(3 * time.Second):
			t.Fatal("等待 passthrough model-lock turn 结果超时")
			return nil
		}
	}

	// This is the handler-facing service contract: the first frame has already
	// been channel-mapped, while InitialRequestModel retains the public alias.
	harness.writeMessage(t, `{"type":"response.create","model":"gpt-5.4","stream":true}`)
	harness.readFirstTurn(t)
	firstResult := waitResult()
	require.Equal(t, forwardModel, firstResult.Model)
	require.NotNil(t, firstResult.ReasoningEffort)
	require.Equal(t, "xhigh", *firstResult.ReasoningEffort)

	harness.writeMessage(t, `{"type":"session.update","session":{"model":"gpt-5.4-xhigh","voice":"alloy"}}`)
	require.Eventually(t, func() bool {
		return len(harness.upstreamConn.Writes()) == 2
	}, 2*time.Second, 10*time.Millisecond, "合法同模 session.update 未写入上游")
	sessionUpdate := harness.upstreamConn.Writes()[1]
	require.Equal(t, "session.update", gjson.GetBytes(sessionUpdate, "type").String())
	require.Equal(t, forwardModel, gjson.GetBytes(sessionUpdate, "session.model").String())

	harness.writeMessage(t, `{"type":"response.create","model":"gpt-5.4-xhigh","stream":true,"previous_response_id":"resp_model_lock_turn_1"}`)
	require.Eventually(t, func() bool {
		return len(harness.upstreamConn.Writes()) == 3
	}, 2*time.Second, 10*time.Millisecond, "合法同模 response.create 未写入上游")
	secondRequest := harness.upstreamConn.Writes()[2]
	require.Equal(t, "response.create", gjson.GetBytes(secondRequest, "type").String())
	require.Equal(t, forwardModel, gjson.GetBytes(secondRequest, "model").String())

	pushCtx, cancelPush := context.WithTimeout(context.Background(), time.Second)
	require.NoError(t, harness.upstreamConn.PushEvent(
		pushCtx,
		[]byte(`{"type":"response.completed","response":{"id":"resp_model_lock_turn_2","model":"gpt-5.4","usage":{"input_tokens":3,"output_tokens":2}}}`),
	))
	cancelPush()
	secondTerminal := harness.readMessage(t)
	require.Equal(t, "response.completed", gjson.GetBytes(secondTerminal, "type").String())
	require.Equal(t, "resp_model_lock_turn_2", gjson.GetBytes(secondTerminal, "response.id").String())

	secondResult := waitResult()
	require.Equal(t, forwardModel, secondResult.Model)
	require.NotNil(t, secondResult.ReasoningEffort)
	require.Equal(t, "xhigh", *secondResult.ReasoningEffort)
	select {
	case beforeRequestModel := <-beforeRequestModels:
		require.Equal(t, originalModel, beforeRequestModel)
	case <-time.After(3 * time.Second):
		t.Fatal("等待 passthrough model-lock BeforeRequest 回调超时")
	}
	require.Equal(t, int32(1), beforeTurnCalls.Load())
	require.Equal(t, 1, harness.dialer.DialCount())

	require.NoError(t, harness.upstreamConn.Close())
	_ = harness.waitProxyResult(t)
}

func TestOpenAIGatewayService_ProxyResponsesWebSocketFromClient_PassthroughRejectsConnectionModelSwitch(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{
			name:    "response create",
			payload: `{"type":"response.create","model":"gpt-5.5","stream":true}`,
		},
		{
			name:    "session update",
			payload: `{"type":"session.update","session":{"model":"gpt-5.5"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var beforeTurnCalls atomic.Int32
			var successfulAfterTurnCalls atomic.Int32
			hooks := &OpenAIWSIngressHooks{
				InitialRequestModel: "gpt-5.4",
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

			harness.writeMessage(t, tt.payload)
			proxyErr := harness.waitProxyResult(t)
			require.Error(t, proxyErr)
			var closeErr *OpenAIWSClientCloseError
			require.ErrorAs(t, proxyErr, &closeErr)
			require.Equal(t, coderws.StatusPolicyViolation, closeErr.StatusCode())
			require.Equal(t, "changing model within a websocket connection is not supported", closeErr.Reason())

			require.Zero(t, beforeTurnCalls.Load(), "切模帧不得启动下一 turn")
			require.Equal(t, int32(1), successfulAfterTurnCalls.Load(), "切模帧不得产生额外成功 usage turn")
			upstreamWrites := harness.upstreamConn.Writes()
			require.Len(t, upstreamWrites, 1, "切模帧必须在写入上游前被拒绝")
			require.Equal(t, "response.create", gjson.GetBytes(upstreamWrites[0], "type").String())
			require.Equal(t, 1, harness.dialer.DialCount())
		})
	}
}
