package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	openaiwsv2 "github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2"
	coderws "github.com/coder/websocket"
)

const (
	defaultOpenAIWSIngressClientMaxQueuedFrames = 64
	defaultOpenAIWSIngressClientMaxQueuedBytes  = int(openAIWSClientReadLimitBytesDefault)
)

var (
	ErrOpenAIWSIngressClientSessionClosed = errors.New("openai websocket ingress client session closed")
	ErrOpenAIWSIngressClientAttemptClosed = errors.New("openai websocket ingress client attempt closed")
	ErrOpenAIWSIngressClientQueueOverflow = errors.New("openai websocket ingress client queue overflow")
)

type OpenAIWSIngressClientSessionOptions struct {
	MaxQueuedFrames int
	MaxQueuedBytes  int
}

type openAIWSIngressClientFrame struct {
	arrivalID uint64
	msgType   coderws.MessageType
	payload   []byte
}

// OpenAIWSIngressClientSession owns the only physical reader for an ingress
// client connection. Attempt contexts only bound consumption from its queue.
type OpenAIWSIngressClientSession struct {
	conn *coderws.Conn

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	maxQueuedFrames int
	maxQueuedBytes  int

	mu          sync.Mutex
	queue       []openAIWSIngressClientFrame
	queuedBytes int
	lastArrival uint64
	terminalErr error
	notify      chan struct{}

	closeOnce sync.Once
}

var _ openaiwsv2.FrameConn = (*OpenAIWSIngressClientAttempt)(nil)

func NewOpenAIWSIngressClientSession(
	conn *coderws.Conn,
	options OpenAIWSIngressClientSessionOptions,
) *OpenAIWSIngressClientSession {
	maxQueuedFrames := options.MaxQueuedFrames
	if maxQueuedFrames <= 0 {
		maxQueuedFrames = defaultOpenAIWSIngressClientMaxQueuedFrames
	}
	maxQueuedBytes := options.MaxQueuedBytes
	if maxQueuedBytes <= 0 {
		maxQueuedBytes = defaultOpenAIWSIngressClientMaxQueuedBytes
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &OpenAIWSIngressClientSession{
		conn:            conn,
		ctx:             ctx,
		cancel:          cancel,
		done:            make(chan struct{}),
		maxQueuedFrames: maxQueuedFrames,
		maxQueuedBytes:  maxQueuedBytes,
		notify:          make(chan struct{}),
	}
	if conn == nil {
		session.setTerminalError(ErrOpenAIWSIngressClientSessionClosed)
		close(session.done)
		return session
	}

	go session.readPump()
	return session
}

func (s *OpenAIWSIngressClientSession) readPump() {
	defer close(s.done)

	for {
		msgType, payload, err := s.conn.Read(s.ctx)
		if err != nil {
			if s.ctx.Err() != nil {
				s.setTerminalError(ErrOpenAIWSIngressClientSessionClosed)
			} else {
				s.setTerminalError(err)
			}
			return
		}

		payload = append([]byte(nil), payload...)
		if err := s.enqueue(msgType, payload); err != nil {
			s.setTerminalError(err)
			_ = s.conn.CloseNow()
			return
		}
	}
}

func (s *OpenAIWSIngressClientSession) enqueue(
	msgType coderws.MessageType,
	payload []byte,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.terminalErr != nil {
		return s.terminalErr
	}
	if len(s.queue) >= s.maxQueuedFrames {
		return fmt.Errorf(
			"%w: frame limit %d exceeded",
			ErrOpenAIWSIngressClientQueueOverflow,
			s.maxQueuedFrames,
		)
	}
	if len(payload) > s.maxQueuedBytes-s.queuedBytes {
		return fmt.Errorf(
			"%w: byte limit %d exceeded",
			ErrOpenAIWSIngressClientQueueOverflow,
			s.maxQueuedBytes,
		)
	}

	s.queue = append(s.queue, openAIWSIngressClientFrame{
		arrivalID: s.lastArrival + 1,
		msgType:   msgType,
		payload:   payload,
	})
	s.lastArrival++
	s.queuedBytes += len(payload)
	s.signalLocked()
	return nil
}

func (s *OpenAIWSIngressClientSession) ReadNext(
	ctx context.Context,
) (coderws.MessageType, []byte, error) {
	frame, err := s.readNextFrame(ctx, nil)
	if err != nil {
		return coderws.MessageText, nil, err
	}
	return frame.msgType, frame.payload, nil
}

func (s *OpenAIWSIngressClientSession) readNextFrame(
	ctx context.Context,
	abort <-chan struct{},
) (openAIWSIngressClientFrame, error) {
	if s == nil {
		return openAIWSIngressClientFrame{}, ErrOpenAIWSIngressClientSessionClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		if abort != nil {
			select {
			case <-abort:
				return openAIWSIngressClientFrame{}, ErrOpenAIWSIngressClientAttemptClosed
			default:
			}
		}

		s.mu.Lock()
		if len(s.queue) != 0 {
			frame := s.queue[0]
			s.queue[0] = openAIWSIngressClientFrame{}
			s.queue = s.queue[1:]
			s.queuedBytes -= len(frame.payload)
			s.mu.Unlock()
			frame.payload = append([]byte(nil), frame.payload...)
			return frame, nil
		}
		if s.terminalErr != nil {
			err := s.terminalErr
			s.mu.Unlock()
			return openAIWSIngressClientFrame{}, err
		}
		notify := s.notify
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return openAIWSIngressClientFrame{}, ctx.Err()
		case <-abort:
			return openAIWSIngressClientFrame{}, ErrOpenAIWSIngressClientAttemptClosed
		case <-notify:
		}
	}
}

func (s *OpenAIWSIngressClientSession) BeginAttempt() *OpenAIWSIngressClientAttempt {
	return &OpenAIWSIngressClientAttempt{
		session:   s,
		committed: make(chan struct{}),
		closed:    make(chan struct{}),
	}
}

func (s *OpenAIWSIngressClientSession) Done() <-chan struct{} {
	if s == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return s.done
}

func (s *OpenAIWSIngressClientSession) Err() error {
	if s == nil {
		return ErrOpenAIWSIngressClientSessionClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terminalErr
}

func (s *OpenAIWSIngressClientSession) CloseAndWait() {
	if s == nil {
		return
	}
	s.closeTransport(0, "")
	<-s.done
}

// Close sends an application close frame before stopping the physical reader.
// The handler remains the connection owner; this method exists for protocol
// exits, such as the inter-turn idle timeout, that need a specific close code.
func (s *OpenAIWSIngressClientSession) Close(status coderws.StatusCode, reason string) {
	if s == nil {
		return
	}
	s.closeTransport(status, reason)
	<-s.done
}

func (s *OpenAIWSIngressClientSession) closeTransport(status coderws.StatusCode, reason string) {
	s.closeOnce.Do(func() {
		s.setTerminalError(ErrOpenAIWSIngressClientSessionClosed)
		if s.conn != nil && status != 0 {
			_ = s.conn.Close(status, reason)
		}
		s.cancel()
		if s.conn != nil {
			_ = s.conn.CloseNow()
		}
	})
}

func (s *OpenAIWSIngressClientSession) setTerminalError(err error) {
	if err == nil {
		err = ErrOpenAIWSIngressClientSessionClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.terminalErr != nil {
		return
	}
	s.terminalErr = err
	s.signalLocked()
}

func (s *OpenAIWSIngressClientSession) signalLocked() {
	close(s.notify)
	s.notify = make(chan struct{})
}

type OpenAIWSIngressClientAttempt struct {
	session *OpenAIWSIngressClientSession

	commitOnce sync.Once
	committed  chan struct{}
	didCommit  atomic.Bool
	// committedThrough is the last frame already queued when the first
	// downstream write succeeds.
	committedThrough atomic.Uint64

	closeMu   sync.Mutex
	closeOnce sync.Once
	closed    chan struct{}
}

func (a *OpenAIWSIngressClientAttempt) ReadFrame(
	ctx context.Context,
) (coderws.MessageType, []byte, error) {
	if a == nil || a.session == nil {
		return coderws.MessageText, nil, ErrOpenAIWSIngressClientSessionClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if !a.didCommit.Load() {
		for !a.didCommit.Load() {
			a.session.mu.Lock()
			if a.session.terminalErr != nil {
				err := a.session.terminalErr
				a.session.mu.Unlock()
				return coderws.MessageText, nil, err
			}
			notify := a.session.notify
			a.session.mu.Unlock()

			select {
			case <-a.committed:
			case <-notify:
			case <-a.closed:
				return coderws.MessageText, nil, ErrOpenAIWSIngressClientAttemptClosed
			case <-ctx.Done():
				return coderws.MessageText, nil, ctx.Err()
			}
		}
	}

	frame, err := a.session.readNextFrame(ctx, a.closed)
	if err != nil {
		return coderws.MessageText, nil, err
	}
	if frame.arrivalID <= a.committedThrough.Load() {
		eventType, validationErr := ValidateOpenAIWSClientFrameJSON(frame.payload)
		if validationErr != nil {
			return coderws.MessageText, nil, NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				"invalid websocket request payload",
				validationErr,
			)
		}
		// The ctx_pool/http_bridge parser treats a missing type as the legacy
		// response.create shorthand, so the pre-commit gate must do the same.
		if eventType == "" || eventType == "response.create" {
			return coderws.MessageText, nil, NewOpenAIWSClientCloseError(
				coderws.StatusPolicyViolation,
				"response.create arrived before websocket attempt was committed",
				nil,
			)
		}
	}
	return frame.msgType, frame.payload, nil
}

func (a *OpenAIWSIngressClientAttempt) WriteFrame(
	ctx context.Context,
	msgType coderws.MessageType,
	payload []byte,
) error {
	if a == nil || a.session == nil || a.session.conn == nil {
		return ErrOpenAIWSIngressClientSessionClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}

	a.closeMu.Lock()
	defer a.closeMu.Unlock()
	select {
	case <-a.closed:
		return ErrOpenAIWSIngressClientAttemptClosed
	default:
	}

	// Keep the physical read pump from enqueueing a frame between the
	// successful socket write and the commit boundary snapshot. A client can
	// receive the frame and immediately send its next response.create before
	// Conn.Write returns to this goroutine; without this lock that valid frame
	// can be misclassified as pre-commit input.
	a.session.mu.Lock()
	defer a.session.mu.Unlock()
	if a.session.terminalErr != nil {
		return a.session.terminalErr
	}
	if err := a.session.conn.Write(ctx, msgType, payload); err != nil {
		return err
	}
	a.commitOnce.Do(func() {
		a.committedThrough.Store(a.session.lastArrival)
		a.didCommit.Store(true)
		close(a.committed)
	})
	return nil
}

func (a *OpenAIWSIngressClientAttempt) Close() error {
	if a == nil {
		return nil
	}
	a.closeMu.Lock()
	defer a.closeMu.Unlock()
	a.closeOnce.Do(func() {
		close(a.closed)
	})
	return nil
}
