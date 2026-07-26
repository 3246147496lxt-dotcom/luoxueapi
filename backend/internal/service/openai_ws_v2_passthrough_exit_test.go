package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIWSPassthroughExitHookCall struct {
	turn   int
	result *OpenAIForwardResult
	err    error
}

type openAIWSPassthroughExitDialer struct {
	conn openAIWSClientConn
}

func (d *openAIWSPassthroughExitDialer) Dial(
	context.Context,
	string,
	http.Header,
	string,
) (openAIWSClientConn, int, http.Header, error) {
	return d.conn, http.StatusSwitchingProtocols, nil, nil
}

type openAIWSPassthroughBlockingExitConn struct {
	closed    chan struct{}
	closeOnce sync.Once
}

func newOpenAIWSPassthroughBlockingExitConn() *openAIWSPassthroughBlockingExitConn {
	return &openAIWSPassthroughBlockingExitConn{closed: make(chan struct{})}
}

func (c *openAIWSPassthroughBlockingExitConn) WriteJSON(context.Context, any) error {
	return errors.New("unexpected WriteJSON call")
}

func (c *openAIWSPassthroughBlockingExitConn) ReadMessage(ctx context.Context) ([]byte, error) {
	_, payload, err := c.ReadFrame(ctx)
	return payload, err
}

func (c *openAIWSPassthroughBlockingExitConn) ReadFrame(
	ctx context.Context,
) (coderws.MessageType, []byte, error) {
	select {
	case <-ctx.Done():
		return coderws.MessageText, nil, ctx.Err()
	case <-c.closed:
		return coderws.MessageText, nil, io.EOF
	}
}

func (c *openAIWSPassthroughBlockingExitConn) WriteFrame(
	ctx context.Context,
	_ coderws.MessageType,
	_ []byte,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return io.EOF
	default:
		return nil
	}
}

func (c *openAIWSPassthroughBlockingExitConn) Ping(context.Context) error {
	return nil
}

func (c *openAIWSPassthroughBlockingExitConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

type openAIWSPassthroughReadErrorConn struct {
	err error
}

func (c *openAIWSPassthroughReadErrorConn) WriteJSON(context.Context, any) error {
	return errors.New("unexpected WriteJSON call")
}

func (c *openAIWSPassthroughReadErrorConn) ReadMessage(ctx context.Context) ([]byte, error) {
	_, payload, err := c.ReadFrame(ctx)
	return payload, err
}

func (c *openAIWSPassthroughReadErrorConn) ReadFrame(
	context.Context,
) (coderws.MessageType, []byte, error) {
	return coderws.MessageText, nil, c.err
}

func (c *openAIWSPassthroughReadErrorConn) WriteFrame(
	context.Context,
	coderws.MessageType,
	[]byte,
) error {
	return nil
}

func (c *openAIWSPassthroughReadErrorConn) Ping(context.Context) error {
	return nil
}

func (c *openAIWSPassthroughReadErrorConn) Close() error {
	return nil
}

type openAIWSPassthroughDelayedTerminalExitConn struct {
	terminal []byte
	delay    time.Duration

	mu       sync.Mutex
	readOnce bool
	closed   chan struct{}
	once     sync.Once
}

func newOpenAIWSPassthroughDelayedTerminalExitConn(
	delay time.Duration,
	terminal []byte,
) *openAIWSPassthroughDelayedTerminalExitConn {
	return &openAIWSPassthroughDelayedTerminalExitConn{
		terminal: append([]byte(nil), terminal...),
		delay:    delay,
		closed:   make(chan struct{}),
	}
}

func (c *openAIWSPassthroughDelayedTerminalExitConn) WriteJSON(context.Context, any) error {
	return errors.New("unexpected WriteJSON call")
}

func (c *openAIWSPassthroughDelayedTerminalExitConn) ReadMessage(ctx context.Context) ([]byte, error) {
	_, payload, err := c.ReadFrame(ctx)
	return payload, err
}

func (c *openAIWSPassthroughDelayedTerminalExitConn) ReadFrame(
	ctx context.Context,
) (coderws.MessageType, []byte, error) {
	c.mu.Lock()
	firstRead := !c.readOnce
	c.readOnce = true
	c.mu.Unlock()

	if firstRead {
		timer := time.NewTimer(c.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return coderws.MessageText, nil, ctx.Err()
		case <-c.closed:
			return coderws.MessageText, nil, io.EOF
		case <-timer.C:
			return coderws.MessageText, append([]byte(nil), c.terminal...), nil
		}
	}
	select {
	case <-ctx.Done():
		return coderws.MessageText, nil, ctx.Err()
	case <-c.closed:
		return coderws.MessageText, nil, io.EOF
	}
}

func (c *openAIWSPassthroughDelayedTerminalExitConn) WriteFrame(
	ctx context.Context,
	_ coderws.MessageType,
	_ []byte,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return io.EOF
	default:
		return nil
	}
}

func (c *openAIWSPassthroughDelayedTerminalExitConn) Ping(context.Context) error {
	return nil
}

func (c *openAIWSPassthroughDelayedTerminalExitConn) Close() error {
	c.once.Do(func() {
		close(c.closed)
	})
	return nil
}

type openAIWSPassthroughSecondTurnEOFConn struct {
	firstTerminal  []byte
	secondTerminal []byte

	firstRead      sync.Once
	secondRead     sync.Once
	secondWritten  chan struct{}
	secondObserved chan struct{}
	writeCount     int
	writeMu        sync.Mutex
	writeOnce      sync.Once
	observeOnce    sync.Once
	closed         chan struct{}
	closeOnce      sync.Once
}

func newOpenAIWSPassthroughSecondTurnEOFConn(
	firstTerminal []byte,
	secondTerminal ...[]byte,
) *openAIWSPassthroughSecondTurnEOFConn {
	var second []byte
	if len(secondTerminal) > 0 {
		second = append([]byte(nil), secondTerminal[0]...)
	}
	return &openAIWSPassthroughSecondTurnEOFConn{
		firstTerminal:  append([]byte(nil), firstTerminal...),
		secondTerminal: second,
		secondWritten:  make(chan struct{}),
		secondObserved: make(chan struct{}),
		closed:         make(chan struct{}),
	}
}

func (c *openAIWSPassthroughSecondTurnEOFConn) WriteJSON(context.Context, any) error {
	return errors.New("unexpected WriteJSON call")
}

func (c *openAIWSPassthroughSecondTurnEOFConn) ReadMessage(ctx context.Context) ([]byte, error) {
	_, payload, err := c.ReadFrame(ctx)
	return payload, err
}

func (c *openAIWSPassthroughSecondTurnEOFConn) ReadFrame(
	ctx context.Context,
) (coderws.MessageType, []byte, error) {
	var terminal []byte
	c.firstRead.Do(func() {
		terminal = append([]byte(nil), c.firstTerminal...)
	})
	if terminal != nil {
		return coderws.MessageText, terminal, nil
	}
	select {
	case <-ctx.Done():
		return coderws.MessageText, nil, ctx.Err()
	case <-c.closed:
		return coderws.MessageText, nil, io.EOF
	case <-c.secondWritten:
		c.observeOnce.Do(func() {
			close(c.secondObserved)
		})
		var terminal []byte
		c.secondRead.Do(func() {
			terminal = append([]byte(nil), c.secondTerminal...)
		})
		if len(terminal) > 0 {
			return coderws.MessageText, terminal, nil
		}
		return coderws.MessageText, nil, io.EOF
	}
}

func (c *openAIWSPassthroughSecondTurnEOFConn) WriteFrame(
	ctx context.Context,
	_ coderws.MessageType,
	_ []byte,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return io.EOF
	default:
	}
	c.writeMu.Lock()
	c.writeCount++
	writeCount := c.writeCount
	c.writeMu.Unlock()
	if writeCount == 2 {
		c.writeOnce.Do(func() {
			close(c.secondWritten)
		})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closed:
			return io.EOF
		case <-c.secondObserved:
		}
	}
	return nil
}

func (c *openAIWSPassthroughSecondTurnEOFConn) Ping(context.Context) error {
	return nil
}

func (c *openAIWSPassthroughSecondTurnEOFConn) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func newOpenAIWSPassthroughExitTestClient(
	t *testing.T,
	upstream openAIWSClientConn,
	hooks *OpenAIWSIngressHooks,
) (*coderws.Conn, <-chan error, *OpenAIGatewayService) {
	t.Helper()
	setGinTestMode()

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3

	svc := &OpenAIGatewayService{
		cfg:                       cfg,
		rateLimitService:          NewRateLimitService(transientCooldownAccountRepo{}, nil, cfg, nil, nil),
		httpUpstream:              &httpUpstreamRecorder{},
		cache:                     &stubGatewayCache{},
		openaiWSResolver:          NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:             NewCodexToolCorrector(),
		openaiWSPassthroughDialer: &openAIWSPassthroughExitDialer{conn: upstream},
	}
	account := &Account{
		ID:          4951,
		Name:        "passthrough-exit-contract",
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
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			serverErrCh <- err
			return
		}
		defer func() {
			_ = conn.CloseNow()
		}()

		rec := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(rec)
		ginCtx.Request = r.Clone(r.Context())

		readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
		msgType, firstMessage, readErr := conn.Read(readCtx)
		cancelRead()
		if readErr != nil {
			serverErrCh <- readErr
			return
		}
		if msgType != coderws.MessageText {
			serverErrCh <- errors.New("unexpected websocket client message type")
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
	t.Cleanup(wsServer.Close)

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
	})
	return clientConn, serverErrCh, svc
}

func writeOpenAIWSPassthroughExitTestTurn(
	t *testing.T,
	clientConn *coderws.Conn,
	payload string,
) {
	t.Helper()
	writeOpenAIWSPassthroughExitTestFrame(t, clientConn, coderws.MessageText, payload)
}

func writeOpenAIWSPassthroughExitTestFrame(
	t *testing.T,
	clientConn *coderws.Conn,
	msgType coderws.MessageType,
	payload string,
) {
	t.Helper()
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelWrite()
	require.NoError(t, clientConn.Write(writeCtx, msgType, []byte(payload)))
}

func readOpenAIWSPassthroughExitTestEvent(
	t *testing.T,
	clientConn *coderws.Conn,
) []byte {
	t.Helper()
	readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelRead()
	msgType, payload, err := clientConn.Read(readCtx)
	require.NoError(t, err)
	require.Equal(t, coderws.MessageText, msgType)
	return payload
}

func requireNoExtraOpenAIWSPassthroughExitHookCall(
	t *testing.T,
	hookCalls <-chan openAIWSPassthroughExitHookCall,
) {
	t.Helper()
	select {
	case call := <-hookCalls:
		t.Fatalf("unexpected extra AfterTurn call: turn=%d result=%+v err=%v", call.turn, call.result, call.err)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestOpenAIWSPassthroughExit_ClientGracefulDisconnectWithoutTerminalIsNeutral(t *testing.T) {
	upstream := newOpenAIWSPassthroughBlockingExitConn()
	hookCalls := make(chan openAIWSPassthroughExitHookCall, 2)
	hooks := &OpenAIWSIngressHooks{
		AfterTurn: func(turn int, result *OpenAIForwardResult, turnErr error) {
			hookCalls <- openAIWSPassthroughExitHookCall{turn: turn, result: result, err: turnErr}
		},
	}
	clientConn, serverErrCh, svc := newOpenAIWSPassthroughExitTestClient(t, upstream, hooks)
	svc.recordOpenAIAccountModelTransientFailure(
		&Account{ID: 4951, Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
		"gpt-5.5",
		time.Now(),
	)
	require.Equal(t, 1, svc.getOpenAIAccountModelTransientState().size())
	writeOpenAIWSPassthroughExitTestTurn(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"disconnect"}`,
	)

	_ = clientConn.Close(coderws.StatusNormalClosure, "done")

	select {
	case serverErr := <-serverErrCh:
		require.NoError(t, serverErr)
	case <-time.After(4 * time.Second):
		t.Fatal("neutral client disconnect did not finish after bounded upstream drain")
	}
	select {
	case call := <-hookCalls:
		require.Equal(t, 1, call.turn)
		require.NoError(t, call.err)
		require.NotNil(t, call.result)
		require.True(t, call.result.ClientDisconnect)
		require.Empty(t, call.result.UpstreamTerminalEvent)
		require.Zero(t, call.result.Usage.InputTokens)
		require.Zero(t, call.result.Usage.OutputTokens)
	case <-time.After(time.Second):
		t.Fatal("neutral client disconnect did not release the active turn")
	}
	requireNoExtraOpenAIWSPassthroughExitHookCall(t, hookCalls)
	require.Equal(t, 1, svc.getOpenAIAccountModelTransientState().size(), "neutral disconnect must not clear transient state")
}

func TestOpenAIWSPassthroughExit_UpstreamFailureFailsActiveTurn(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "eof", err: io.EOF},
		{name: "read error", err: errors.New("upstream read failed")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := &openAIWSPassthroughReadErrorConn{err: tt.err}
			hookCalls := make(chan openAIWSPassthroughExitHookCall, 2)
			hooks := &OpenAIWSIngressHooks{
				AfterTurn: func(turn int, result *OpenAIForwardResult, turnErr error) {
					hookCalls <- openAIWSPassthroughExitHookCall{turn: turn, result: result, err: turnErr}
				},
			}
			clientConn, serverErrCh, _ := newOpenAIWSPassthroughExitTestClient(t, upstream, hooks)
			writeOpenAIWSPassthroughExitTestTurn(
				t,
				clientConn,
				`{"type":"response.create","model":"gpt-5.5","input":"upstream failure"}`,
			)

			select {
			case serverErr := <-serverErrCh:
				require.Error(t, serverErr)
				var turnErr *openAIWSIngressTurnError
				require.ErrorAs(t, serverErr, &turnErr)
				require.Equal(t, "read_upstream", turnErr.stage)
				require.ErrorIs(t, turnErr.cause, tt.err)
			case <-time.After(3 * time.Second):
				t.Fatal("upstream failure did not fail the active passthrough turn")
			}
			select {
			case call := <-hookCalls:
				require.Equal(t, 1, call.turn)
				require.Nil(t, call.result)
				require.Error(t, call.err)
				var turnErr *openAIWSIngressTurnError
				require.ErrorAs(t, call.err, &turnErr)
				require.Equal(t, "read_upstream", turnErr.stage)
			case <-time.After(time.Second):
				t.Fatal("upstream failure did not release the active turn")
			}
			requireNoExtraOpenAIWSPassthroughExitHookCall(t, hookCalls)
		})
	}
}

func TestOpenAIWSPassthroughExit_ClientDisconnectDrainCapturesTerminalOnce(t *testing.T) {
	upstream := newOpenAIWSPassthroughDelayedTerminalExitConn(
		100*time.Millisecond,
		[]byte(`{"type":"response.completed","response":{"id":"resp_drain_terminal","model":"gpt-5.5","usage":{"input_tokens":6,"output_tokens":4}}}`),
	)
	hookCalls := make(chan openAIWSPassthroughExitHookCall, 2)
	hooks := &OpenAIWSIngressHooks{
		AfterTurn: func(turn int, result *OpenAIForwardResult, turnErr error) {
			hookCalls <- openAIWSPassthroughExitHookCall{turn: turn, result: result, err: turnErr}
		},
	}
	clientConn, serverErrCh, _ := newOpenAIWSPassthroughExitTestClient(t, upstream, hooks)
	writeOpenAIWSPassthroughExitTestTurn(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"drain terminal"}`,
	)

	_ = clientConn.Close(coderws.StatusNormalClosure, "done")

	select {
	case serverErr := <-serverErrCh:
		require.NoError(t, serverErr)
	case <-time.After(3 * time.Second):
		t.Fatal("terminal captured during drain did not finish passthrough")
	}
	select {
	case call := <-hookCalls:
		require.Equal(t, 1, call.turn)
		require.NoError(t, call.err)
		require.NotNil(t, call.result)
		require.False(t, call.result.ClientDisconnect)
		require.Equal(t, "resp_drain_terminal", call.result.RequestID)
		require.Equal(t, "response.completed", call.result.UpstreamTerminalEvent)
		require.Equal(t, 6, call.result.Usage.InputTokens)
		require.Equal(t, 4, call.result.Usage.OutputTokens)
	case <-time.After(time.Second):
		t.Fatal("terminal captured during drain did not settle the active turn")
	}
	requireNoExtraOpenAIWSPassthroughExitHookCall(t, hookCalls)
}

func TestOpenAIWSPassthroughExit_ClientDisconnectDrainErrorEventIsFailure(t *testing.T) {
	upstream := newOpenAIWSPassthroughDelayedTerminalExitConn(
		100*time.Millisecond,
		[]byte(`{"type":"error","error":{"type":"server_error","code":"server_error","message":"upstream failed"}}`),
	)
	hookCalls := make(chan openAIWSPassthroughExitHookCall, 2)
	hooks := &OpenAIWSIngressHooks{
		AfterTurn: func(turn int, result *OpenAIForwardResult, turnErr error) {
			hookCalls <- openAIWSPassthroughExitHookCall{turn: turn, result: result, err: turnErr}
		},
	}
	clientConn, serverErrCh, svc := newOpenAIWSPassthroughExitTestClient(t, upstream, hooks)
	writeOpenAIWSPassthroughExitTestTurn(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"drain error"}`,
	)

	clientCloseDone := make(chan error, 1)
	go func() {
		clientCloseDone <- clientConn.Close(coderws.StatusNormalClosure, "done")
	}()

	select {
	case serverErr := <-serverErrCh:
		require.Error(t, serverErr)
		var turnErr *openAIWSIngressTurnError
		require.ErrorAs(t, serverErr, &turnErr)
		require.Equal(t, "upstream_message", turnErr.stage)
	case <-time.After(3 * time.Second):
		t.Fatal("drain error event did not fail the active turn")
	}
	select {
	case call := <-hookCalls:
		require.Equal(t, 1, call.turn)
		require.Nil(t, call.result)
		require.Error(t, call.err)
		var turnErr *openAIWSIngressTurnError
		require.ErrorAs(t, call.err, &turnErr)
		require.Equal(t, "upstream_message", turnErr.stage)
	case <-time.After(time.Second):
		t.Fatal("drain error event did not settle the active turn")
	}
	require.Equal(t, 1, svc.getOpenAIAccountModelTransientState().size())
	requireNoExtraOpenAIWSPassthroughExitHookCall(t, hookCalls)
	select {
	case <-clientCloseDone:
	case <-time.After(2 * time.Second):
		t.Fatal("drain error client close did not finish")
	}
}

func TestOpenAIWSPassthroughExit_SecondTurnEOFDoesNotReuseFirstTerminal(t *testing.T) {
	testOpenAIWSPassthroughExitSecondTurnEOFDoesNotReuseFirstTerminal(t, coderws.MessageText)
}

func TestOpenAIWSPassthroughExit_BinarySecondTurnEOFDoesNotReuseFirstTerminal(t *testing.T) {
	testOpenAIWSPassthroughExitSecondTurnEOFDoesNotReuseFirstTerminal(t, coderws.MessageBinary)
}

func testOpenAIWSPassthroughExitSecondTurnEOFDoesNotReuseFirstTerminal(
	t *testing.T,
	secondTurnMessageType coderws.MessageType,
) {
	t.Helper()
	upstream := newOpenAIWSPassthroughSecondTurnEOFConn(
		[]byte(`{"type":"response.completed","response":{"id":"resp_exit_turn_1","model":"gpt-5.5","usage":{"input_tokens":2,"output_tokens":1}}}`),
	)
	hookCalls := make(chan openAIWSPassthroughExitHookCall, 3)
	hooks := &OpenAIWSIngressHooks{
		AfterTurn: func(turn int, result *OpenAIForwardResult, turnErr error) {
			hookCalls <- openAIWSPassthroughExitHookCall{turn: turn, result: result, err: turnErr}
		},
	}
	clientConn, serverErrCh, _ := newOpenAIWSPassthroughExitTestClient(t, upstream, hooks)
	writeOpenAIWSPassthroughExitTestTurn(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"turn 1"}`,
	)
	firstTerminal := readOpenAIWSPassthroughExitTestEvent(t, clientConn)
	require.Contains(t, string(firstTerminal), `"id":"resp_exit_turn_1"`)

	select {
	case call := <-hookCalls:
		require.Equal(t, 1, call.turn)
		require.NoError(t, call.err)
		require.NotNil(t, call.result)
		require.Equal(t, "resp_exit_turn_1", call.result.RequestID)
	case <-time.After(time.Second):
		t.Fatal("first terminal did not settle before second turn")
	}

	writeOpenAIWSPassthroughExitTestFrame(
		t,
		clientConn,
		secondTurnMessageType,
		`{"type":"response.create","model":"gpt-5.5","previous_response_id":"resp_exit_turn_1","input":"turn 2"}`,
	)

	select {
	case serverErr := <-serverErrCh:
		require.Error(t, serverErr)
		var turnErr *openAIWSIngressTurnError
		require.ErrorAs(t, serverErr, &turnErr)
		require.Equal(t, "read_upstream", turnErr.stage)
		require.ErrorIs(t, turnErr.cause, io.EOF)
	case <-time.After(3 * time.Second):
		t.Fatal("second-turn upstream EOF did not fail the active turn")
	}
	select {
	case call := <-hookCalls:
		require.Equal(t, 2, call.turn)
		require.Nil(t, call.result, "second turn must not inherit the first terminal result")
		require.Error(t, call.err)
		var turnErr *openAIWSIngressTurnError
		require.ErrorAs(t, call.err, &turnErr)
		require.Equal(t, "read_upstream", turnErr.stage)
	case <-time.After(time.Second):
		t.Fatal("second-turn upstream EOF did not release the active turn")
	}
	requireNoExtraOpenAIWSPassthroughExitHookCall(t, hookCalls)
}

func TestOpenAIWSPassthroughExit_TerminalThenIdleTimeoutDoesNotSettleTwice(t *testing.T) {
	upstream := newOpenAIWSPassthroughSecondTurnEOFConn(
		[]byte(`{"type":"response.completed","response":{"id":"resp_idle_turn_1","model":"gpt-5.5","usage":{"input_tokens":2,"output_tokens":1}}}`),
	)
	hookCalls := make(chan openAIWSPassthroughExitHookCall, 2)
	hooks := &OpenAIWSIngressHooks{
		AfterTurn: func(turn int, result *OpenAIForwardResult, turnErr error) {
			hookCalls <- openAIWSPassthroughExitHookCall{turn: turn, result: result, err: turnErr}
		},
	}
	clientConn, serverErrCh, svc := newOpenAIWSPassthroughExitTestClient(t, upstream, hooks)
	svc.cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 1

	writeOpenAIWSPassthroughExitTestTurn(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"turn 1"}`,
	)
	firstTerminal := readOpenAIWSPassthroughExitTestEvent(t, clientConn)
	require.Contains(t, string(firstTerminal), `"id":"resp_idle_turn_1"`)

	select {
	case call := <-hookCalls:
		require.Equal(t, 1, call.turn)
		require.NoError(t, call.err)
		require.NotNil(t, call.result)
		require.Equal(t, "resp_idle_turn_1", call.result.RequestID)
	case <-time.After(time.Second):
		t.Fatal("first terminal did not settle before idle timeout")
	}

	select {
	case serverErr := <-serverErrCh:
		require.Error(t, serverErr)
		var turnErr *openAIWSIngressTurnError
		require.ErrorAs(t, serverErr, &turnErr)
		require.Equal(t, "idle_timeout", turnErr.stage)
		var closeErr *OpenAIWSClientCloseError
		require.ErrorAs(t, serverErr, &closeErr)
		require.Equal(t, coderws.StatusPolicyViolation, closeErr.StatusCode())
		require.Equal(t, "client websocket idle timeout", closeErr.Reason())
	case <-time.After(3 * time.Second):
		t.Fatal("terminal followed by idle timeout did not close the session")
	}
	requireNoExtraOpenAIWSPassthroughExitHookCall(t, hookCalls)
}

func TestOpenAIWSPassthroughExit_ResponseIDLessTerminalSettlesOnlyCurrentTurnUsage(t *testing.T) {
	upstream := newOpenAIWSPassthroughSecondTurnEOFConn(
		[]byte(`{"type":"response.completed","response":{"id":"resp_idless_turn_1","model":"gpt-5.5","usage":{"input_tokens":2,"output_tokens":1}}}`),
		[]byte(`{"type":"response.completed","response":{"model":"gpt-5.5","usage":{"input_tokens":3,"output_tokens":2}}}`),
	)
	hookCalls := make(chan openAIWSPassthroughExitHookCall, 3)
	hooks := &OpenAIWSIngressHooks{
		AfterTurn: func(turn int, result *OpenAIForwardResult, turnErr error) {
			hookCalls <- openAIWSPassthroughExitHookCall{turn: turn, result: result, err: turnErr}
		},
	}
	clientConn, serverErrCh, _ := newOpenAIWSPassthroughExitTestClient(t, upstream, hooks)
	writeOpenAIWSPassthroughExitTestTurn(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","input":"turn 1"}`,
	)
	_ = readOpenAIWSPassthroughExitTestEvent(t, clientConn)

	firstCall := <-hookCalls
	require.Equal(t, 1, firstCall.turn)
	require.NoError(t, firstCall.err)
	require.NotNil(t, firstCall.result)
	require.Equal(t, "resp_idless_turn_1", firstCall.result.RequestID)
	require.Equal(t, 2, firstCall.result.Usage.InputTokens)
	require.Equal(t, 1, firstCall.result.Usage.OutputTokens)

	writeOpenAIWSPassthroughExitTestTurn(
		t,
		clientConn,
		`{"type":"response.create","model":"gpt-5.5","previous_response_id":"resp_idless_turn_1","input":"turn 2"}`,
	)
	secondTerminal := readOpenAIWSPassthroughExitTestEvent(t, clientConn)
	require.Contains(t, string(secondTerminal), `"type":"response.completed"`)

	secondCall := <-hookCalls
	require.Equal(t, 2, secondCall.turn)
	require.NoError(t, secondCall.err)
	require.NotNil(t, secondCall.result)
	require.Empty(t, secondCall.result.RequestID)
	require.Equal(t, 3, secondCall.result.Usage.InputTokens)
	require.Equal(t, 2, secondCall.result.Usage.OutputTokens)

	select {
	case serverErr := <-serverErrCh:
		require.NoError(t, serverErr)
	case <-time.After(3 * time.Second):
		t.Fatal("response-id-less terminal relay did not finish")
	}
	requireNoExtraOpenAIWSPassthroughExitHookCall(t, hookCalls)
}
