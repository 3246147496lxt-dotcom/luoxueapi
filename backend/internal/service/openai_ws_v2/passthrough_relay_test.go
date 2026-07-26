package openai_ws_v2

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

type passthroughTestFrame struct {
	msgType coderws.MessageType
	payload []byte
}

type passthroughTestFrameConn struct {
	mu     sync.Mutex
	writes []passthroughTestFrame
	readCh chan passthroughTestFrame
	once   sync.Once
}

type delayedReadFrameConn struct {
	base       FrameConn
	firstDelay time.Duration
	once       sync.Once
}

type delayedReadErrorFrameConn struct {
	delay time.Duration
	err   error
}

type multiTurnEOFFrameConn struct {
	firstTerminal    []byte
	firstRead        atomic.Bool
	writeCount       atomic.Int32
	secondTurnSent   chan struct{}
	secondTurnSignal sync.Once
}

type terminalBeforeWriteReturnFrameConn struct {
	firstTerminal  []byte
	secondTerminal []byte
	readCount      atomic.Int32
	writeStarted   chan struct{}
	writeReturned  chan struct{}
	writeStartOnce sync.Once
	writeDoneOnce  sync.Once
}

type closeSpyFrameConn struct {
	closeCalls atomic.Int32
}

func newPassthroughTestFrameConn(frames []passthroughTestFrame, autoClose bool) *passthroughTestFrameConn {
	c := &passthroughTestFrameConn{
		readCh: make(chan passthroughTestFrame, len(frames)+1),
	}
	for _, frame := range frames {
		copied := passthroughTestFrame{msgType: frame.msgType, payload: append([]byte(nil), frame.payload...)}
		c.readCh <- copied
	}
	if autoClose {
		close(c.readCh)
	}
	return c
}

func (c *passthroughTestFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return coderws.MessageText, nil, ctx.Err()
	case frame, ok := <-c.readCh:
		if !ok {
			return coderws.MessageText, nil, io.EOF
		}
		return frame.msgType, append([]byte(nil), frame.payload...), nil
	}
}

func (c *passthroughTestFrameConn) WriteFrame(ctx context.Context, msgType coderws.MessageType, payload []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes = append(c.writes, passthroughTestFrame{msgType: msgType, payload: append([]byte(nil), payload...)})
	return nil
}

func (c *passthroughTestFrameConn) Close() error {
	c.once.Do(func() {
		defer func() { _ = recover() }()
		close(c.readCh)
	})
	return nil
}

func (c *passthroughTestFrameConn) Writes() []passthroughTestFrame {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]passthroughTestFrame, len(c.writes))
	copy(out, c.writes)
	return out
}

func (c *delayedReadFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	if c == nil || c.base == nil {
		return coderws.MessageText, nil, io.EOF
	}
	c.once.Do(func() {
		if c.firstDelay > 0 {
			timer := time.NewTimer(c.firstDelay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
			case <-timer.C:
			}
		}
	})
	return c.base.ReadFrame(ctx)
}

func (c *delayedReadFrameConn) WriteFrame(ctx context.Context, msgType coderws.MessageType, payload []byte) error {
	if c == nil || c.base == nil {
		return io.EOF
	}
	return c.base.WriteFrame(ctx, msgType, payload)
}

func (c *delayedReadFrameConn) Close() error {
	if c == nil || c.base == nil {
		return nil
	}
	return c.base.Close()
}

func (c *delayedReadErrorFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	timer := time.NewTimer(c.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return coderws.MessageText, nil, ctx.Err()
	case <-timer.C:
		return coderws.MessageText, nil, c.err
	}
}

func (c *delayedReadErrorFrameConn) WriteFrame(context.Context, coderws.MessageType, []byte) error {
	return nil
}

func (c *delayedReadErrorFrameConn) Close() error {
	return nil
}

func (c *multiTurnEOFFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	if c.firstRead.CompareAndSwap(false, true) {
		return coderws.MessageText, append([]byte(nil), c.firstTerminal...), nil
	}
	select {
	case <-ctx.Done():
		return coderws.MessageText, nil, ctx.Err()
	case <-c.secondTurnSent:
		return coderws.MessageText, nil, io.EOF
	}
}

func (c *multiTurnEOFFrameConn) WriteFrame(
	_ context.Context,
	_ coderws.MessageType,
	_ []byte,
) error {
	if c.writeCount.Add(1) == 2 {
		c.secondTurnSignal.Do(func() {
			close(c.secondTurnSent)
		})
	}
	return nil
}

func (c *multiTurnEOFFrameConn) Close() error {
	return nil
}

func (c *terminalBeforeWriteReturnFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	switch c.readCount.Add(1) {
	case 1:
		return coderws.MessageText, append([]byte(nil), c.firstTerminal...), nil
	case 2:
		select {
		case <-ctx.Done():
			return coderws.MessageText, nil, ctx.Err()
		case <-c.writeStarted:
			return coderws.MessageText, append([]byte(nil), c.secondTerminal...), nil
		}
	default:
		select {
		case <-ctx.Done():
			return coderws.MessageText, nil, ctx.Err()
		case <-c.writeReturned:
			return coderws.MessageText, nil, io.EOF
		}
	}
}

func (c *terminalBeforeWriteReturnFrameConn) WriteFrame(
	ctx context.Context,
	_ coderws.MessageType,
	_ []byte,
) error {
	c.writeStartOnce.Do(func() {
		close(c.writeStarted)
	})
	timer := time.NewTimer(50 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	c.writeDoneOnce.Do(func() {
		close(c.writeReturned)
	})
	return nil
}

func (c *terminalBeforeWriteReturnFrameConn) Close() error {
	c.writeDoneOnce.Do(func() {
		close(c.writeReturned)
	})
	return nil
}

func (c *closeSpyFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	<-ctx.Done()
	return coderws.MessageText, nil, ctx.Err()
}

func (c *closeSpyFrameConn) WriteFrame(ctx context.Context, _ coderws.MessageType, _ []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (c *closeSpyFrameConn) Close() error {
	if c != nil {
		c.closeCalls.Add(1)
	}
	return nil
}

func (c *closeSpyFrameConn) CloseCalls() int32 {
	if c == nil {
		return 0
	}
	return c.closeCalls.Load()
}

func TestRelay_BasicRelayAndUsage(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_123","usage":{"input_tokens":7,"output_tokens":3,"input_tokens_details":{"cached_tokens":2}}}}`),
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-5.3-codex","input":[{"type":"input_text","text":"hello"}]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{})
	require.Nil(t, relayExit)
	require.Equal(t, "gpt-5.3-codex", result.RequestModel)
	require.Equal(t, "resp_123", result.RequestID)
	require.Equal(t, "response.completed", result.TerminalEventType)
	require.Equal(t, RelayExitSourceTerminal, result.ExitSource)
	require.Equal(t, 7, result.Usage.InputTokens)
	require.Equal(t, 3, result.Usage.OutputTokens)
	require.Equal(t, 2, result.Usage.CacheReadInputTokens)
	require.NotNil(t, result.FirstTokenMs)
	require.Equal(t, int64(1), result.ClientToUpstreamFrames)
	require.Equal(t, int64(1), result.UpstreamToClientFrames)
	require.Equal(t, int64(0), result.DroppedDownstreamFrames)

	upstreamWrites := upstreamConn.Writes()
	require.Len(t, upstreamWrites, 1)
	require.Equal(t, coderws.MessageText, upstreamWrites[0].msgType)
	require.JSONEq(t, string(firstPayload), string(upstreamWrites[0].payload))

	clientWrites := clientConn.Writes()
	require.Len(t, clientWrites, 1)
	require.Equal(t, coderws.MessageText, clientWrites[0].msgType)
	require.JSONEq(t, `{"type":"response.completed","response":{"id":"resp_123","usage":{"input_tokens":7,"output_tokens":3,"input_tokens_details":{"cached_tokens":2}}}}`, string(clientWrites[0].payload))
}

func TestRelay_FunctionCallOutputBytesPreserved(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_func","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-5.3-codex","input":[{"type":"function_call_output","call_id":"call_abc123","output":"{\"ok\":true}"}]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{})
	require.Nil(t, relayExit)

	upstreamWrites := upstreamConn.Writes()
	require.Len(t, upstreamWrites, 1)
	require.Equal(t, coderws.MessageText, upstreamWrites[0].msgType)
	require.Equal(t, firstPayload, upstreamWrites[0].payload)
}

func TestRelay_UpstreamDisconnect(t *testing.T) {
	t.Parallel()

	// 上游立即关闭（EOF），客户端不发送额外帧
	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn(nil, true) // 立即 close -> EOF

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{})
	require.NotNil(t, relayExit, "上游 EOF 且未见 terminal 应返回失败")
	require.Equal(t, "read_upstream", relayExit.Stage)
	require.ErrorIs(t, relayExit.Err, io.EOF)
	require.Equal(t, "gpt-4o", result.RequestModel)
	require.Equal(t, RelayExitSourceUpstreamDisconnect, result.ExitSource)
}

func TestRelay_ClientDisconnect(t *testing.T) {
	t.Parallel()

	// 客户端立即关闭（EOF），上游阻塞读取直到 context 取消
	clientConn := newPassthroughTestFrameConn(nil, true) // 立即 close -> EOF
	upstreamConn := newPassthroughTestFrameConn(nil, false)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		UpstreamDrainTimeout: 20 * time.Millisecond,
	})
	require.Nil(t, relayExit, "客户端 graceful disconnect 且 drain 超时应为中性退出")
	require.Equal(t, "gpt-4o", result.RequestModel)
	require.Equal(t, RelayExitSourceClientDisconnect, result.ExitSource)
}

func TestRelay_ClientDisconnect_DrainCapturesLateUsage(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, true)
	upstreamBase := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_drain","usage":{"input_tokens":6,"output_tokens":4,"input_tokens_details":{"cached_tokens":1}}}}`),
		},
	}, true)
	upstreamConn := &delayedReadFrameConn{
		base:       upstreamBase,
		firstDelay: 80 * time.Millisecond,
	}

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		FirstMessageSent:     true,
		UpstreamDrainTimeout: 400 * time.Millisecond,
	})
	require.Nil(t, relayExit)
	require.Equal(t, "resp_drain", result.RequestID)
	require.Equal(t, "response.completed", result.TerminalEventType)
	require.Equal(t, RelayExitSourceTerminal, result.ExitSource)
	require.Equal(t, 6, result.Usage.InputTokens)
	require.Equal(t, 4, result.Usage.OutputTokens)
	require.Equal(t, 1, result.Usage.CacheReadInputTokens)
	require.Equal(t, int64(1), result.ClientToUpstreamFrames)
	require.Equal(t, int64(0), result.UpstreamToClientFrames)
	require.Equal(t, int64(1), result.DroppedDownstreamFrames)
}

func TestRelay_ClientDisconnect_DrainCapturesUpstreamExit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "eof", err: io.EOF},
		{name: "error", err: errors.New("upstream read failed")},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientConn := newPassthroughTestFrameConn(nil, true)
			upstreamConn := &delayedReadErrorFrameConn{
				delay: 50 * time.Millisecond,
				err:   tt.err,
			}
			firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
				FirstMessageSent:     true,
				UpstreamDrainTimeout: 400 * time.Millisecond,
			})
			require.NotNil(t, relayExit, "FirstMessageSent 不应吞掉 drain 捕获的上游退出")
			require.Equal(t, "read_upstream", relayExit.Stage)
			require.ErrorIs(t, relayExit.Err, tt.err)
			require.Equal(t, RelayExitSourceUpstreamDisconnect, result.ExitSource)
			require.Empty(t, result.TerminalEventType)
		})
	}
}

func TestRelay_ClientDisconnect_DrainErrorEventIsUpstreamFailure(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, true)
	upstreamConn := &delayedReadFrameConn{
		firstDelay: 50 * time.Millisecond,
		base: newPassthroughTestFrameConn([]passthroughTestFrame{
			{
				msgType: coderws.MessageText,
				payload: []byte(`{"type":"error","error":{"type":"server_error","code":"server_error","message":"upstream failed"}}`),
			},
		}, false),
	}
	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		FirstMessageSent:     true,
		UpstreamDrainTimeout: 400 * time.Millisecond,
	})

	require.NotNil(t, relayExit)
	require.Equal(t, "upstream_message", relayExit.Stage)
	require.ErrorContains(t, relayExit.Err, "upstream websocket error event")
	require.Equal(t, RelayExitSourceUpstreamDisconnect, result.ExitSource)
	require.Empty(t, result.TerminalEventType)
}

func TestRelay_MultiTurnEOFDoesNotReusePriorTerminal(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.create","model":"gpt-4o","input":[{"type":"input_text","text":"turn 2"}]}`),
		},
	}, true)
	upstreamConn := &multiTurnEOFFrameConn{
		firstTerminal:  []byte(`{"type":"response.completed","response":{"id":"resp_turn_1","usage":{"input_tokens":2,"output_tokens":1}}}`),
		secondTurnSent: make(chan struct{}),
	}
	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[{"type":"input_text","text":"turn 1"}]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		StartClientAfterFirstDownstream: true,
		UpstreamDrainTimeout:            400 * time.Millisecond,
	})
	require.NotNil(t, relayExit)
	require.Equal(t, "read_upstream", relayExit.Stage)
	require.ErrorIs(t, relayExit.Err, io.EOF)
	require.Equal(t, RelayExitSourceUpstreamDisconnect, result.ExitSource)
	require.Equal(t, "response.completed", result.TerminalEventType, "历史 terminal 仍保留用于观测")
	require.Equal(t, int64(2), result.ClientToUpstreamFrames)
}

func TestRelay_TerminalCannotRaceAheadOfSuccessfulTurnRegistration(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.create","model":"gpt-4o","input":[{"type":"input_text","text":"turn 2"}]}`),
		},
	}, false)
	upstreamConn := &terminalBeforeWriteReturnFrameConn{
		firstTerminal:  []byte(`{"type":"response.completed","response":{"id":"resp_race_turn_1","usage":{"input_tokens":2,"output_tokens":1}}}`),
		secondTerminal: []byte(`{"type":"response.completed","response":{"id":"resp_race_turn_2","usage":{"input_tokens":3,"output_tokens":2}}}`),
		writeStarted:   make(chan struct{}),
		writeReturned:  make(chan struct{}),
	}
	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[{"type":"input_text","text":"turn 1"}]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var completedTurns atomic.Int32
	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		FirstMessageSent:                true,
		StartClientAfterFirstDownstream: true,
		OnTurnComplete: func(RelayTurnResult) {
			completedTurns.Add(1)
		},
	})

	require.Nil(t, relayExit)
	require.Equal(t, RelayExitSourceTerminal, result.ExitSource)
	require.Equal(t, "resp_race_turn_2", result.RequestID)
	require.Equal(t, int32(2), completedTurns.Load())
	require.Equal(t, int64(2), result.ClientToUpstreamFrames)
}

func TestRelay_DeduplicatesTerminalResponseID(t *testing.T) {
	t.Parallel()

	terminal := []byte(`{"type":"response.completed","response":{"id":"resp_duplicate","usage":{"input_tokens":3,"output_tokens":2}}}`)
	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{msgType: coderws.MessageText, payload: terminal},
		{msgType: coderws.MessageText, payload: terminal},
	}, true)
	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var completedTurns atomic.Int32
	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		OnTurnComplete: func(RelayTurnResult) {
			completedTurns.Add(1)
		},
	})
	require.Nil(t, relayExit)
	require.Equal(t, RelayExitSourceTerminal, result.ExitSource)
	require.Equal(t, int32(1), completedTurns.Load())
	require.Equal(t, 3, result.Usage.InputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens)
}

func TestRelay_ResponseIDLessTerminalStillCompletesTurn(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"usage":{"input_tokens":3,"output_tokens":2}}}`),
		},
	}, false)
	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	completed := make(chan RelayTurnResult, 1)
	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		OnTurnComplete: func(turn RelayTurnResult) {
			completed <- turn
		},
	})

	require.NoError(t, ctx.Err(), "response-id-less terminal must end the physical relay before the context deadline")
	require.Nil(t, relayExit)
	require.Equal(t, RelayExitSourceTerminal, result.ExitSource)
	select {
	case turn := <-completed:
		require.Empty(t, turn.RequestID)
		require.Equal(t, 3, turn.Usage.InputTokens)
		require.Equal(t, 2, turn.Usage.OutputTokens)
	case <-time.After(time.Second):
		t.Fatal("response-id-less terminal did not complete the active turn")
	}
}

func TestRelay_ClientFrameErrorAfterPriorTerminalIsNotSwallowed(t *testing.T) {
	t.Parallel()

	clientErr := errors.New("invalid next client frame")
	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_turn_1","usage":{"input_tokens":2,"output_tokens":1}}}`),
		},
	}, false)
	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		StartClientAfterFirstDownstream: true,
		ReadClientFrame: func(context.Context, FrameConn) (coderws.MessageType, []byte, error) {
			return coderws.MessageText, nil, clientErr
		},
	})

	require.NotNil(t, relayExit)
	require.Equal(t, "read_client", relayExit.Stage)
	require.ErrorIs(t, relayExit.Err, clientErr)
	require.Equal(t, RelayExitSourceUnknown, result.ExitSource)
	require.Equal(t, "response.completed", result.TerminalEventType, "历史 terminal 只保留用于观测")
}

func TestRelay_IdleTimeout(t *testing.T) {
	t.Parallel()

	// 客户端和上游都不发送帧，idle timeout 应触发
	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn(nil, false)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 使用快进时间来加速 idle timeout
	now := time.Now()
	callCount := 0
	nowFn := func() time.Time {
		callCount++
		// 前几次调用返回正常时间（初始化阶段），之后快进
		if callCount <= 5 {
			return now
		}
		return now.Add(time.Hour) // 快进到超时
	}

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		IdleTimeout: 2 * time.Second,
		Now:         nowFn,
	})
	require.NotNil(t, relayExit, "应因 idle timeout 退出")
	require.Equal(t, "idle_timeout", relayExit.Stage)
	require.Equal(t, "gpt-4o", result.RequestModel)
}

func TestRelay_IdleTimeoutDoesNotCloseClientOnError(t *testing.T) {
	t.Parallel()

	clientConn := &closeSpyFrameConn{}
	upstreamConn := &closeSpyFrameConn{}

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	callCount := 0
	nowFn := func() time.Time {
		callCount++
		if callCount <= 5 {
			return now
		}
		return now.Add(time.Hour)
	}

	_, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		IdleTimeout: 2 * time.Second,
		Now:         nowFn,
	})
	require.NotNil(t, relayExit, "应因 idle timeout 退出")
	require.Equal(t, "idle_timeout", relayExit.Stage)
	require.Zero(t, clientConn.CloseCalls(), "错误路径不应提前关闭客户端连接，交给上层决定 close code")
	require.GreaterOrEqual(t, upstreamConn.CloseCalls(), int32(1))
}

func TestRelay_TerminalThenIdleTimeoutPreservesIdleExit(t *testing.T) {
	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_before_idle","usage":{"input_tokens":3,"output_tokens":2}}}`),
		},
	}, false)
	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var completedTurns atomic.Int32
	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		IdleTimeout: 50 * time.Millisecond,
		OnTurnComplete: func(RelayTurnResult) {
			completedTurns.Add(1)
		},
	})

	require.NotNil(t, relayExit)
	require.Equal(t, "idle_timeout", relayExit.Stage)
	require.ErrorIs(t, relayExit.Err, context.DeadlineExceeded)
	require.Equal(t, RelayExitSourceUnknown, result.ExitSource)
	require.Equal(t, "response.completed", result.TerminalEventType)
	require.Equal(t, "resp_before_idle", result.RequestID)
	require.Equal(t, int32(1), completedTurns.Load(), "idle timeout must not settle historical usage again")
	require.Len(t, clientConn.Writes(), 1)
}

func TestRelay_ClientDisconnectDrainIgnoresIdleTimeout(t *testing.T) {
	clientConn := newPassthroughTestFrameConn(nil, true)
	upstreamConn := newPassthroughTestFrameConn(nil, false)
	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var sawDrainIdle atomic.Bool
	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		FirstMessageSent:     true,
		IdleTimeout:          50 * time.Millisecond,
		UpstreamDrainTimeout: 1300 * time.Millisecond,
		OnTrace: func(event RelayTraceEvent) {
			if event.Stage == "second_exit" && event.Direction == "watchdog" {
				sawDrainIdle.Store(true)
			}
		},
	})

	require.Nil(t, relayExit)
	require.Equal(t, RelayExitSourceClientDisconnect, result.ExitSource)
	require.True(t, sawDrainIdle.Load(), "test must observe the drain-time idle signal")
}

func TestRelay_NilConnections(t *testing.T) {
	t.Parallel()

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx := context.Background()

	t.Run("nil client conn", func(t *testing.T) {
		upstreamConn := newPassthroughTestFrameConn(nil, true)
		_, relayExit := Relay(ctx, nil, upstreamConn, firstPayload, RelayOptions{})
		require.NotNil(t, relayExit)
		require.Equal(t, "relay_init", relayExit.Stage)
		require.Contains(t, relayExit.Err.Error(), "nil")
	})

	t.Run("nil upstream conn", func(t *testing.T) {
		clientConn := newPassthroughTestFrameConn(nil, true)
		_, relayExit := Relay(ctx, clientConn, nil, firstPayload, RelayOptions{})
		require.NotNil(t, relayExit)
		require.Equal(t, "relay_init", relayExit.Stage)
		require.Contains(t, relayExit.Err.Error(), "nil")
	})
}

func TestRelay_MultipleUpstreamMessages(t *testing.T) {
	t.Parallel()

	// 上游发送多个事件（delta + completed），验证多帧中继和 usage 聚合
	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.output_text.delta","delta":"Hello"}`),
		},
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.output_text.delta","delta":" world"}`),
		},
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_multi","usage":{"input_tokens":10,"output_tokens":5,"input_tokens_details":{"cached_tokens":3}}}}`),
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[{"type":"input_text","text":"hi"}]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{})
	require.Nil(t, relayExit)
	require.Equal(t, "resp_multi", result.RequestID)
	require.Equal(t, "response.completed", result.TerminalEventType)
	require.Equal(t, 10, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
	require.Equal(t, 3, result.Usage.CacheReadInputTokens)
	require.NotNil(t, result.FirstTokenMs)

	// 验证所有 3 个上游帧都转发给了客户端
	clientWrites := clientConn.Writes()
	require.Len(t, clientWrites, 3)
}

func TestRelay_OnTurnComplete_PerTerminalEvent(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_turn_1","usage":{"input_tokens":2,"output_tokens":1}}}`),
		},
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.failed","response":{"id":"resp_turn_2","usage":{"input_tokens":3,"output_tokens":4}}}`),
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-5.3-codex","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	turns := make([]RelayTurnResult, 0, 2)
	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		OnTurnComplete: func(turn RelayTurnResult) {
			turns = append(turns, turn)
		},
	})
	require.Nil(t, relayExit)
	require.Len(t, turns, 2)
	require.Equal(t, "resp_turn_1", turns[0].RequestID)
	require.Equal(t, "response.completed", turns[0].TerminalEventType)
	require.Equal(t, 2, turns[0].Usage.InputTokens)
	require.Equal(t, 1, turns[0].Usage.OutputTokens)
	require.Equal(t, "resp_turn_2", turns[1].RequestID)
	require.Equal(t, "response.failed", turns[1].TerminalEventType)
	require.Equal(t, 3, turns[1].Usage.InputTokens)
	require.Equal(t, 4, turns[1].Usage.OutputTokens)
	require.Equal(t, 5, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
}

func TestRelay_OnTurnComplete_ProvidesTurnMetrics(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.output_text.delta","response_id":"resp_metric","delta":"hi"}`),
		},
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_metric","usage":{"input_tokens":2,"output_tokens":1}}}`),
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-5.3-codex","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	base := time.Unix(0, 0)
	var nowTick atomic.Int64
	nowFn := func() time.Time {
		step := nowTick.Add(1)
		return base.Add(time.Duration(step) * 5 * time.Millisecond)
	}

	var turn RelayTurnResult
	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		Now: nowFn,
		OnTurnComplete: func(current RelayTurnResult) {
			turn = current
		},
	})
	require.Nil(t, relayExit)
	require.Equal(t, "resp_metric", turn.RequestID)
	require.Equal(t, "response.completed", turn.TerminalEventType)
	require.NotNil(t, turn.FirstTokenMs)
	require.GreaterOrEqual(t, *turn.FirstTokenMs, 0)
	require.Greater(t, turn.Duration.Milliseconds(), int64(0))
	require.NotNil(t, result.FirstTokenMs)
	require.Greater(t, result.Duration.Milliseconds(), int64(0))
}

func TestRelay_BinaryFramePassthrough(t *testing.T) {
	t.Parallel()

	// 验证 binary frame 被透传但不进行 usage 解析
	binaryPayload := []byte{0x00, 0x01, 0x02, 0x03}
	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageBinary,
			payload: binaryPayload,
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{})
	require.NotNil(t, relayExit)
	require.Equal(t, "read_upstream", relayExit.Stage)
	require.Equal(t, RelayExitSourceUpstreamDisconnect, result.ExitSource)
	// binary frame 不解析 usage
	require.Equal(t, 0, result.Usage.InputTokens)

	clientWrites := clientConn.Writes()
	require.Len(t, clientWrites, 1)
	require.Equal(t, coderws.MessageBinary, clientWrites[0].msgType)
	require.Equal(t, binaryPayload, clientWrites[0].payload)
}

func TestRelay_BinaryJSONFrameSkipsObservation(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageBinary,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_binary","usage":{"input_tokens":7,"output_tokens":3}}}`),
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{})
	require.NotNil(t, relayExit)
	require.Equal(t, "read_upstream", relayExit.Stage)
	require.Equal(t, RelayExitSourceUpstreamDisconnect, result.ExitSource)
	require.Equal(t, 0, result.Usage.InputTokens)
	require.Equal(t, "", result.RequestID)
	require.Equal(t, "", result.TerminalEventType)

	clientWrites := clientConn.Writes()
	require.Len(t, clientWrites, 1)
	require.Equal(t, coderws.MessageBinary, clientWrites[0].msgType)
}

func TestRelay_UpstreamErrorEventPassthroughRaw(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	errorEvent := []byte(`{"type":"error","error":{"type":"invalid_request_error","message":"No tool call found"}}`)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: errorEvent,
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{})
	require.NotNil(t, relayExit)
	require.Equal(t, "read_upstream", relayExit.Stage)
	require.Equal(t, RelayExitSourceUpstreamDisconnect, result.ExitSource)

	clientWrites := clientConn.Writes()
	require.Len(t, clientWrites, 1)
	require.Equal(t, coderws.MessageText, clientWrites[0].msgType)
	require.Equal(t, errorEvent, clientWrites[0].payload)
}

func TestRelay_PreservesFirstMessageType(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn(nil, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		FirstMessageType: coderws.MessageBinary,
	})
	require.NotNil(t, relayExit)
	require.Equal(t, "read_upstream", relayExit.Stage)
	require.Equal(t, RelayExitSourceUpstreamDisconnect, result.ExitSource)

	upstreamWrites := upstreamConn.Writes()
	require.Len(t, upstreamWrites, 1)
	require.Equal(t, coderws.MessageBinary, upstreamWrites[0].msgType)
	require.Equal(t, firstPayload, upstreamWrites[0].payload)
}

func TestRelay_UsageParseFailureDoesNotBlockRelay(t *testing.T) {
	baseline := SnapshotMetrics().UsageParseFailureTotal

	// 上游发送无效 JSON（非 usage 格式），不应影响透传
	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_bad","usage":"not_an_object"}}`),
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{})
	require.Nil(t, relayExit)
	// usage 解析失败，值为 0 但不影响透传
	require.Equal(t, 0, result.Usage.InputTokens)
	require.Equal(t, "response.completed", result.TerminalEventType)

	// 帧仍然被转发
	clientWrites := clientConn.Writes()
	require.Len(t, clientWrites, 1)
	require.GreaterOrEqual(t, SnapshotMetrics().UsageParseFailureTotal, baseline+1)
}

func TestRelay_WriteUpstreamFirstMessageFails(t *testing.T) {
	t.Parallel()

	// 上游连接立即关闭，首包写入失败
	upstreamConn := newPassthroughTestFrameConn(nil, true)
	_ = upstreamConn.Close()

	// 覆盖 WriteFrame 使其返回错误
	errConn := &errorOnWriteFrameConn{}
	clientConn := newPassthroughTestFrameConn(nil, false)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, relayExit := Relay(ctx, clientConn, errConn, firstPayload, RelayOptions{})
	require.NotNil(t, relayExit)
	require.Equal(t, "write_upstream", relayExit.Stage)
}

func TestRelay_ContextCanceled(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn(nil, false)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)

	// 立即取消 context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{})
	// context 取消导致写首包失败
	require.NotNil(t, relayExit)
}

func TestRelay_TraceEvents_ContainsLifecycleStages(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_trace","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stages := make([]string, 0, 8)
	var stagesMu sync.Mutex
	_, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		OnTrace: func(event RelayTraceEvent) {
			stagesMu.Lock()
			stages = append(stages, event.Stage)
			stagesMu.Unlock()
		},
	})
	require.Nil(t, relayExit)
	stagesMu.Lock()
	capturedStages := append([]string(nil), stages...)
	stagesMu.Unlock()
	require.Contains(t, capturedStages, "relay_start")
	require.Contains(t, capturedStages, "write_first_message_ok")
	require.Contains(t, capturedStages, "first_exit")
	require.Contains(t, capturedStages, "relay_complete")
}

func TestRelay_TraceEvents_IdleTimeout(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn(nil, false)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-4o","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	callCount := 0
	nowFn := func() time.Time {
		callCount++
		if callCount <= 5 {
			return now
		}
		return now.Add(time.Hour)
	}

	stages := make([]string, 0, 8)
	var stagesMu sync.Mutex
	_, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		IdleTimeout: 2 * time.Second,
		Now:         nowFn,
		OnTrace: func(event RelayTraceEvent) {
			stagesMu.Lock()
			stages = append(stages, event.Stage)
			stagesMu.Unlock()
		},
	})
	require.NotNil(t, relayExit)
	require.Equal(t, "idle_timeout", relayExit.Stage)
	stagesMu.Lock()
	capturedStages := append([]string(nil), stages...)
	stagesMu.Unlock()
	require.Contains(t, capturedStages, "idle_timeout_triggered")
	require.Contains(t, capturedStages, "relay_exit")
}

// errorOnWriteFrameConn 是一个写入总是失败的 FrameConn 实现，用于测试首包写入失败。
type errorOnWriteFrameConn struct{}

func (c *errorOnWriteFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	<-ctx.Done()
	return coderws.MessageText, nil, ctx.Err()
}

func (c *errorOnWriteFrameConn) WriteFrame(_ context.Context, _ coderws.MessageType, _ []byte) error {
	return errors.New("write failed: connection refused")
}

func (c *errorOnWriteFrameConn) Close() error {
	return nil
}

func TestRelay_OnTurnComplete_RealOpenAIStream_FirstTokenMs(t *testing.T) {
	t.Parallel()

	clientConn := newPassthroughTestFrameConn(nil, false)
	upstreamConn := newPassthroughTestFrameConn([]passthroughTestFrame{
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.created","response":{"id":"resp_real"}}`),
		},
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.output_text.delta","delta":"He"}`),
		},
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.output_text.delta","delta":"llo"}`),
		},
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.output_text.delta","delta":" world"}`),
		},
		{
			msgType: coderws.MessageText,
			payload: []byte(`{"type":"response.completed","response":{"id":"resp_real","usage":{"input_tokens":2,"output_tokens":3}}}`),
		},
	}, true)

	firstPayload := []byte(`{"type":"response.create","model":"gpt-5.3-codex","input":[]}`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	base := time.Unix(0, 0)
	var nowTick atomic.Int64
	nowFn := func() time.Time {
		step := nowTick.Add(1)
		return base.Add(time.Duration(step) * 10 * time.Millisecond)
	}

	var turn RelayTurnResult
	result, relayExit := Relay(ctx, clientConn, upstreamConn, firstPayload, RelayOptions{
		Now: nowFn,
		OnTurnComplete: func(current RelayTurnResult) {
			turn = current
		},
	})
	require.Nil(t, relayExit)
	require.Equal(t, "resp_real", turn.RequestID)
	require.Equal(t, "response.completed", turn.TerminalEventType)

	require.NotNil(t, turn.FirstTokenMs, "per-turn FirstTokenMs must be captured for real OpenAI streams")
	require.Greater(t, turn.Duration.Milliseconds(), int64(0))

	require.Less(t,
		int64(*turn.FirstTokenMs),
		turn.Duration.Milliseconds(),
		"per-turn FirstTokenMs (%dms) should be strictly less than Duration (%dms); "+
			"equality indicates the bug where first_token is mistakenly stamped on the terminal event",
		*turn.FirstTokenMs, turn.Duration.Milliseconds(),
	)

	require.NotNil(t, result.FirstTokenMs)
	require.Greater(t, *result.FirstTokenMs, 0)
}
