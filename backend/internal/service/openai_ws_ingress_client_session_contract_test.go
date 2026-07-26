package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func newOpenAIWSIngressClientSessionContractPair(
	t *testing.T,
	options OpenAIWSIngressClientSessionOptions,
) (*OpenAIWSIngressClientSession, *coderws.Conn) {
	t.Helper()

	accepted := make(chan *coderws.Conn, 1)
	releaseHandler := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, nil)
		if err != nil {
			accepted <- nil
			return
		}
		accepted <- conn
		<-releaseHandler
	}))

	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(
		dialCtx,
		"ws"+strings.TrimPrefix(server.URL, "http"),
		nil,
	)
	cancelDial()
	require.NoError(t, err)

	var serverConn *coderws.Conn
	select {
	case serverConn = <-accepted:
		require.NotNil(t, serverConn)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server websocket accept")
	}
	serverConn.SetReadLimit(openAIWSClientReadLimitBytesDefault)

	session := NewOpenAIWSIngressClientSession(serverConn, options)
	t.Cleanup(func() {
		session.CloseAndWait()
		_ = clientConn.CloseNow()
		close(releaseHandler)
		server.Close()
	})
	return session, clientConn
}

func writeOpenAIWSIngressClientSessionContractFrame(
	t *testing.T,
	conn *coderws.Conn,
	msgType coderws.MessageType,
	payload []byte,
) {
	t.Helper()
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelWrite()
	require.NoError(t, conn.Write(writeCtx, msgType, payload))
}

func readOpenAIWSIngressClientSessionContractFrame(
	t *testing.T,
	conn *coderws.Conn,
) (coderws.MessageType, []byte) {
	t.Helper()
	readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelRead()
	msgType, payload, err := conn.Read(readCtx)
	require.NoError(t, err)
	return msgType, payload
}

func commitOpenAIWSIngressClientSessionContractAttempt(
	t *testing.T,
	attempt *OpenAIWSIngressClientAttempt,
	clientConn *coderws.Conn,
	marker string,
) {
	t.Helper()
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelWrite()
	require.NoError(t, attempt.WriteFrame(
		writeCtx,
		coderws.MessageText,
		[]byte(marker),
	))
	msgType, payload := readOpenAIWSIngressClientSessionContractFrame(t, clientConn)
	require.Equal(t, coderws.MessageText, msgType)
	require.Equal(t, marker, string(payload))
}

func TestOpenAIWSIngressClientSessionContract_AttemptCancelPreservesPhysicalConnection(t *testing.T) {
	session, clientConn := newOpenAIWSIngressClientSessionContractPair(
		t,
		OpenAIWSIngressClientSessionOptions{},
	)

	attemptA := session.BeginAttempt()
	commitOpenAIWSIngressClientSessionContractAttempt(t, attemptA, clientConn, "attempt-a-ready")

	readCtxA, cancelReadA := context.WithCancel(context.Background())
	readDoneA := make(chan error, 1)
	go func() {
		_, _, err := attemptA.ReadFrame(readCtxA)
		readDoneA <- err
	}()
	cancelReadA()
	select {
	case err := <-readDoneA:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("attempt A read did not stop after its consumer context was canceled")
	}
	require.NoError(t, session.Err())

	attemptB := session.BeginAttempt()
	commitOpenAIWSIngressClientSessionContractAttempt(t, attemptB, clientConn, "attempt-b-ready")
	writeOpenAIWSIngressClientSessionContractFrame(
		t,
		clientConn,
		coderws.MessageBinary,
		[]byte("frame-after-attempt-cancel"),
	)

	readCtxB, cancelReadB := context.WithTimeout(context.Background(), 3*time.Second)
	msgType, payload, err := attemptB.ReadFrame(readCtxB)
	cancelReadB()
	require.NoError(t, err)
	require.Equal(t, coderws.MessageBinary, msgType)
	require.Equal(t, []byte("frame-after-attempt-cancel"), payload)
	require.NoError(t, session.Err())

	noDuplicateCtx, cancelNoDuplicate := context.WithTimeout(context.Background(), 50*time.Millisecond)
	_, _, err = attemptB.ReadFrame(noDuplicateCtx)
	cancelNoDuplicate()
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, session.Err(), "consumer timeout must not cancel the physical websocket reader")
}

func TestOpenAIWSIngressClientSessionContract_DisconnectIsStickyAcrossAttempts(t *testing.T) {
	session, clientConn := newOpenAIWSIngressClientSessionContractPair(
		t,
		OpenAIWSIngressClientSessionOptions{},
	)
	attemptA := session.BeginAttempt()
	commitOpenAIWSIngressClientSessionContractAttempt(t, attemptA, clientConn, "attempt-a-ready")

	writeOpenAIWSIngressClientSessionContractFrame(t, clientConn, coderws.MessageText, []byte("queued-1"))
	writeOpenAIWSIngressClientSessionContractFrame(t, clientConn, coderws.MessageBinary, []byte("queued-2"))
	require.NoError(t, clientConn.CloseNow())

	select {
	case <-session.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("session read pump did not observe the physical client disconnect")
	}
	stickyErr := session.Err()
	require.Error(t, stickyErr)
	require.NotErrorIs(t, stickyErr, ErrOpenAIWSIngressClientSessionClosed)

	readCtx, cancelRead := context.WithTimeout(context.Background(), time.Second)
	msgType, payload, err := attemptA.ReadFrame(readCtx)
	require.NoError(t, err)
	require.Equal(t, coderws.MessageText, msgType)
	require.Equal(t, []byte("queued-1"), payload)

	msgType, payload, err = attemptA.ReadFrame(readCtx)
	require.NoError(t, err)
	require.Equal(t, coderws.MessageBinary, msgType)
	require.Equal(t, []byte("queued-2"), payload)

	_, _, err = attemptA.ReadFrame(readCtx)
	require.Equal(t, stickyErr, err)
	cancelRead()

	attemptB := session.BeginAttempt()
	readCtxB, cancelReadB := context.WithTimeout(context.Background(), time.Second)
	_, _, err = attemptB.ReadFrame(readCtxB)
	cancelReadB()
	require.Equal(t, stickyErr, err, "a later failover attempt must observe the same disconnect")
	require.Equal(t, stickyErr, session.Err())
}

func TestOpenAIWSIngressClientSessionContract_CloseAndWaitUnblocksAndIsIdempotent(t *testing.T) {
	session, _ := newOpenAIWSIngressClientSessionContractPair(
		t,
		OpenAIWSIngressClientSessionOptions{},
	)
	attempt := session.BeginAttempt()

	readCtx, cancelRead := context.WithCancel(context.Background())
	defer cancelRead()
	readDone := make(chan error, 1)
	go func() {
		_, _, err := attempt.ReadFrame(readCtx)
		readDone <- err
	}()

	const closeCallers = 8
	var closeWG sync.WaitGroup
	closeWG.Add(closeCallers)
	for range closeCallers {
		go func() {
			defer closeWG.Done()
			session.CloseAndWait()
		}()
	}
	closeDone := make(chan struct{})
	go func() {
		closeWG.Wait()
		close(closeDone)
	}()

	select {
	case <-closeDone:
	case <-time.After(3 * time.Second):
		t.Fatal("concurrent CloseAndWait calls did not unblock the physical reader")
	}
	select {
	case err := <-readDone:
		require.ErrorIs(t, err, ErrOpenAIWSIngressClientSessionClosed)
	case <-time.After(time.Second):
		t.Fatal("CloseAndWait did not unblock the pending attempt consumer")
	}
	require.ErrorIs(t, session.Err(), ErrOpenAIWSIngressClientSessionClosed)

	secondCloseDone := make(chan struct{})
	go func() {
		session.CloseAndWait()
		close(secondCloseDone)
	}()
	select {
	case <-secondCloseDone:
	case <-time.After(time.Second):
		t.Fatal("repeated CloseAndWait was not idempotent")
	}
}

func TestOpenAIWSIngressClientSessionContract_QueueOverflowIsStickyAfterQueuedFramesDrain(t *testing.T) {
	tests := []struct {
		name          string
		options       OpenAIWSIngressClientSessionOptions
		acceptedFrame []byte
		overflowFrame []byte
	}{
		{
			name: "frame limit",
			options: OpenAIWSIngressClientSessionOptions{
				MaxQueuedFrames: 1,
				MaxQueuedBytes:  1024,
			},
			acceptedFrame: []byte("first"),
			overflowFrame: []byte("second"),
		},
		{
			name: "byte limit",
			options: OpenAIWSIngressClientSessionOptions{
				MaxQueuedFrames: 4,
				MaxQueuedBytes:  5,
			},
			acceptedFrame: []byte("12345"),
			overflowFrame: []byte("6"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, clientConn := newOpenAIWSIngressClientSessionContractPair(t, tt.options)
			writeOpenAIWSIngressClientSessionContractFrame(
				t,
				clientConn,
				coderws.MessageText,
				tt.acceptedFrame,
			)
			writeOpenAIWSIngressClientSessionContractFrame(
				t,
				clientConn,
				coderws.MessageBinary,
				tt.overflowFrame,
			)

			select {
			case <-session.Done():
			case <-time.After(3 * time.Second):
				t.Fatal("queue overflow did not stop the session read pump")
			}
			require.ErrorIs(t, session.Err(), ErrOpenAIWSIngressClientQueueOverflow)

			readCtx, cancelRead := context.WithTimeout(context.Background(), time.Second)
			msgType, payload, err := session.ReadNext(readCtx)
			require.NoError(t, err)
			require.Equal(t, coderws.MessageText, msgType)
			require.Equal(t, tt.acceptedFrame, payload)

			_, _, err = session.ReadNext(readCtx)
			require.ErrorIs(t, err, ErrOpenAIWSIngressClientQueueOverflow)
			_, _, repeatedErr := session.ReadNext(readCtx)
			cancelRead()
			require.ErrorIs(t, repeatedErr, ErrOpenAIWSIngressClientQueueOverflow)
			require.Equal(t, session.Err(), repeatedErr)
		})
	}
}

func TestOpenAIWSIngressClientSessionContract_PreCommitGateReleasesAllowedFrameAfterCommit(t *testing.T) {
	session, clientConn := newOpenAIWSIngressClientSessionContractPair(
		t,
		OpenAIWSIngressClientSessionOptions{},
	)
	attempt := session.BeginAttempt()
	preCommitFrame := []byte(`{"type":"session.update","session":{"model":"gpt-5.5"}}`)
	writeOpenAIWSIngressClientSessionContractFrame(
		t,
		clientConn,
		coderws.MessageText,
		preCommitFrame,
	)
	require.Eventually(t, func() bool {
		session.mu.Lock()
		defer session.mu.Unlock()
		return session.lastArrival == 1
	}, time.Second, 5*time.Millisecond, "pre-commit frame did not reach the session queue")

	type readResult struct {
		msgType coderws.MessageType
		payload []byte
		err     error
	}
	readDone := make(chan readResult, 1)
	go func() {
		msgType, payload, err := attempt.ReadFrame(context.Background())
		readDone <- readResult{msgType: msgType, payload: payload, err: err}
	}()
	select {
	case result := <-readDone:
		t.Fatalf("pre-commit gate released a frame early: %+v", result)
	case <-time.After(100 * time.Millisecond):
	}

	commitOpenAIWSIngressClientSessionContractAttempt(t, attempt, clientConn, "attempt-committed")
	select {
	case result := <-readDone:
		require.NoError(t, result.err)
		require.Equal(t, coderws.MessageText, result.msgType)
		require.Equal(t, preCommitFrame, result.payload)
	case <-time.After(time.Second):
		t.Fatal("allowed pre-commit frame was not released after attempt commit")
	}
}

func TestOpenAIWSIngressClientSessionContract_PreCommitInvalidFramesArePolicyViolations(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		reason  string
	}{
		{
			name:    "response create",
			payload: `{"type":"response.create","model":"gpt-5.5","input":"pipelined"}`,
			reason:  "response.create arrived before websocket attempt was committed",
		},
		{
			name:    "legacy response create without type",
			payload: `{"model":"gpt-5.5","input":"pipelined"}`,
			reason:  "response.create arrived before websocket attempt was committed",
		},
		{
			name:    "malformed json",
			payload: `{"type":`,
			reason:  "invalid websocket request payload",
		},
		{
			name:    "duplicate type",
			payload: `{"type":"session.update","type":"session.update","session":{}}`,
			reason:  "invalid websocket request payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, clientConn := newOpenAIWSIngressClientSessionContractPair(
				t,
				OpenAIWSIngressClientSessionOptions{},
			)
			attempt := session.BeginAttempt()
			writeOpenAIWSIngressClientSessionContractFrame(
				t,
				clientConn,
				coderws.MessageText,
				[]byte(tt.payload),
			)
			require.Eventually(t, func() bool {
				session.mu.Lock()
				defer session.mu.Unlock()
				return session.lastArrival == 1
			}, time.Second, 5*time.Millisecond, "pre-commit invalid frame did not reach the session queue")

			commitOpenAIWSIngressClientSessionContractAttempt(t, attempt, clientConn, "attempt-committed")
			readCtx, cancelRead := context.WithTimeout(context.Background(), time.Second)
			_, _, err := attempt.ReadFrame(readCtx)
			cancelRead()
			require.Error(t, err)
			var closeErr *OpenAIWSClientCloseError
			require.ErrorAs(t, err, &closeErr)
			require.Equal(t, coderws.StatusPolicyViolation, closeErr.StatusCode())
			require.Equal(t, tt.reason, closeErr.Reason())
		})
	}
}

func TestOpenAIWSIngressClientSessionContract_DefaultByteLimitAcceptsLargeFollowUpFrame(t *testing.T) {
	session, clientConn := newOpenAIWSIngressClientSessionContractPair(
		t,
		OpenAIWSIngressClientSessionOptions{},
	)
	require.Equal(
		t,
		int(openAIWSClientReadLimitBytesDefault),
		session.maxQueuedBytes,
		"session queue must permit one frame allowed by the ingress read limit",
	)

	attempt := session.BeginAttempt()
	commitOpenAIWSIngressClientSessionContractAttempt(t, attempt, clientConn, "large-frame-ready")

	paddingBytes := int(openAIWSMessageReadLimitBytes) + 1
	largeFrame := []byte(
		`{"type":"session.update","padding":"` +
			strings.Repeat("x", paddingBytes) +
			`"}`,
	)
	require.Greater(t, len(largeFrame), int(openAIWSMessageReadLimitBytes))
	require.LessOrEqual(t, len(largeFrame), int(openAIWSClientReadLimitBytesDefault))

	writeOpenAIWSIngressClientSessionContractFrame(
		t,
		clientConn,
		coderws.MessageText,
		largeFrame,
	)
	readCtx, cancelRead := context.WithTimeout(context.Background(), 5*time.Second)
	msgType, payload, err := attempt.ReadFrame(readCtx)
	cancelRead()
	require.NoError(t, err)
	require.Equal(t, coderws.MessageText, msgType)
	require.Equal(t, largeFrame, payload)
	require.NoError(t, session.Err())
}

func TestOpenAIWSIngressClientSessionContract_FailedWriteDoesNotCommitAttempt(t *testing.T) {
	session, clientConn := newOpenAIWSIngressClientSessionContractPair(
		t,
		OpenAIWSIngressClientSessionOptions{},
	)
	attempt := session.BeginAttempt()
	require.NoError(t, clientConn.CloseNow())
	select {
	case <-session.Done():
	case <-time.After(3 * time.Second):
		t.Fatal("session did not observe client close before failed attempt write")
	}

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), time.Second)
	err := attempt.WriteFrame(writeCtx, coderws.MessageText, []byte("must-not-commit"))
	cancelWrite()
	require.Error(t, err)
	require.False(t, attempt.didCommit.Load())
	select {
	case <-attempt.committed:
		t.Fatal("failed WriteFrame closed the attempt commit gate")
	default:
	}
}

func TestOpenAIWSIngressClientSessionContract_AttemptCloseUnblocksWithoutClosingSocket(t *testing.T) {
	session, clientConn := newOpenAIWSIngressClientSessionContractPair(
		t,
		OpenAIWSIngressClientSessionOptions{},
	)

	uncommitted := session.BeginAttempt()
	uncommittedRead := make(chan error, 1)
	go func() {
		_, _, err := uncommitted.ReadFrame(context.Background())
		uncommittedRead <- err
	}()
	require.NoError(t, uncommitted.Close())
	select {
	case err := <-uncommittedRead:
		require.ErrorIs(t, err, ErrOpenAIWSIngressClientAttemptClosed)
	case <-time.After(time.Second):
		t.Fatal("closing an uncommitted attempt did not unblock ReadFrame")
	}
	require.ErrorIs(
		t,
		uncommitted.WriteFrame(context.Background(), coderws.MessageText, []byte("closed")),
		ErrOpenAIWSIngressClientAttemptClosed,
	)
	require.NoError(t, session.Err())

	writeOpenAIWSIngressClientSessionContractFrame(
		t,
		clientConn,
		coderws.MessageText,
		[]byte("socket-still-open"),
	)
	readCtx, cancelRead := context.WithTimeout(context.Background(), time.Second)
	_, payload, err := session.ReadNext(readCtx)
	cancelRead()
	require.NoError(t, err)
	require.Equal(t, []byte("socket-still-open"), payload)

	committed := session.BeginAttempt()
	commitOpenAIWSIngressClientSessionContractAttempt(t, committed, clientConn, "committed-ready")
	committedRead := make(chan error, 1)
	go func() {
		_, _, err := committed.ReadFrame(context.Background())
		committedRead <- err
	}()
	require.NoError(t, committed.Close())
	select {
	case err := <-committedRead:
		require.ErrorIs(t, err, ErrOpenAIWSIngressClientAttemptClosed)
	case <-time.After(time.Second):
		t.Fatal("closing a committed attempt did not unblock ReadFrame")
	}
	require.NoError(t, session.Err())

	finalAttempt := session.BeginAttempt()
	commitOpenAIWSIngressClientSessionContractAttempt(t, finalAttempt, clientConn, "final-ready")
	writeOpenAIWSIngressClientSessionContractFrame(
		t,
		clientConn,
		coderws.MessageBinary,
		[]byte("read-after-attempt-close"),
	)
	finalReadCtx, cancelFinalRead := context.WithTimeout(context.Background(), time.Second)
	msgType, payload, err := finalAttempt.ReadFrame(finalReadCtx)
	cancelFinalRead()
	require.NoError(t, err)
	require.Equal(t, coderws.MessageBinary, msgType)
	require.Equal(t, []byte("read-after-attempt-close"), payload)
}

func TestOpenAIWSIngressClientSessionContract_FIFOAcrossMessageTypes(t *testing.T) {
	session, clientConn := newOpenAIWSIngressClientSessionContractPair(
		t,
		OpenAIWSIngressClientSessionOptions{},
	)
	attempt := session.BeginAttempt()
	commitOpenAIWSIngressClientSessionContractAttempt(t, attempt, clientConn, "fifo-ready")

	frames := []struct {
		msgType coderws.MessageType
		payload []byte
	}{
		{msgType: coderws.MessageText, payload: []byte("first")},
		{msgType: coderws.MessageBinary, payload: []byte("second")},
		{msgType: coderws.MessageText, payload: []byte("third")},
	}
	for _, frame := range frames {
		writeOpenAIWSIngressClientSessionContractFrame(
			t,
			clientConn,
			frame.msgType,
			frame.payload,
		)
	}

	readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelRead()
	for _, want := range frames {
		msgType, payload, err := attempt.ReadFrame(readCtx)
		require.NoError(t, err)
		require.Equal(t, want.msgType, msgType)
		require.Equal(t, want.payload, payload)
	}
}

func TestOpenAIWSIngressClientSessionContract_GracefulClosePreservesStatusAndJoins(t *testing.T) {
	session, clientConn := newOpenAIWSIngressClientSessionContractPair(
		t,
		OpenAIWSIngressClientSessionOptions{},
	)

	closeDone := make(chan struct{})
	go func() {
		session.Close(coderws.StatusNormalClosure, "websocket idle timeout")
		close(closeDone)
	}()

	readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
	_, _, err := clientConn.Read(readCtx)
	cancelRead()
	var closeErr coderws.CloseError
	require.True(t, errors.As(err, &closeErr))
	require.Equal(t, coderws.StatusNormalClosure, closeErr.Code)
	require.Equal(t, "websocket idle timeout", closeErr.Reason)

	select {
	case <-closeDone:
	case <-time.After(3 * time.Second):
		t.Fatal("graceful close did not join the physical read pump")
	}
	select {
	case <-session.Done():
	default:
		t.Fatal("graceful close returned before the physical read pump exited")
	}

	secondCloseDone := make(chan struct{})
	go func() {
		session.CloseAndWait()
		close(secondCloseDone)
	}()
	select {
	case <-secondCloseDone:
	case <-time.After(time.Second):
		t.Fatal("CloseAndWait was not idempotent after graceful close")
	}
}
