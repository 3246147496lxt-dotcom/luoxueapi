package handler

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	defaultLibraryDownloadGlobalLimit       = 4
	defaultLibraryDownloadPerUserLimit      = 2
	defaultLibraryDownloadReservedByteLimit = int64(256 << 20)
	defaultLibraryDownloadAcquireTimeout    = 15 * time.Second
	defaultLibraryDownloadOperationTimeout  = 15 * time.Minute
	libraryDownloadZIPReservationOverhead   = int64(2 << 20)
)

var errLibraryDownloadBusy = infraerrors.TooManyRequests(
	"LIBRARY_DOWNLOAD_BUSY",
	"The library download service is busy; please retry shortly",
)

type libraryDownloadGateLimits struct {
	globalLimit       int
	perUserLimit      int
	reservedByteLimit int64
	acquireTimeout    time.Duration
	operationTimeout  time.Duration
}

// libraryDownloadGate bounds the process-local temporary-disk and file-copy
// work used by atomic library downloads. It deliberately reserves bytes before
// creating a temporary file, so concurrency cannot multiply the configured
// per-request batch limit into unbounded disk use.
type libraryDownloadGate struct {
	mu sync.Mutex

	limits        libraryDownloadGateLimits
	notify        chan struct{}
	active        int
	activeByUser  map[int64]int
	reservedBytes int64
}

func newDefaultLibraryDownloadGate() *libraryDownloadGate {
	return newLibraryDownloadGate(libraryDownloadGateLimits{
		globalLimit:       defaultLibraryDownloadGlobalLimit,
		perUserLimit:      defaultLibraryDownloadPerUserLimit,
		reservedByteLimit: defaultLibraryDownloadReservedByteLimit,
		acquireTimeout:    defaultLibraryDownloadAcquireTimeout,
		operationTimeout:  defaultLibraryDownloadOperationTimeout,
	})
}

func newLibraryDownloadGate(limits libraryDownloadGateLimits) *libraryDownloadGate {
	if limits.globalLimit <= 0 {
		limits.globalLimit = defaultLibraryDownloadGlobalLimit
	}
	if limits.perUserLimit <= 0 {
		limits.perUserLimit = defaultLibraryDownloadPerUserLimit
	}
	if limits.reservedByteLimit <= 0 {
		limits.reservedByteLimit = defaultLibraryDownloadReservedByteLimit
	}
	if limits.acquireTimeout <= 0 {
		limits.acquireTimeout = defaultLibraryDownloadAcquireTimeout
	}
	if limits.operationTimeout <= 0 {
		limits.operationTimeout = defaultLibraryDownloadOperationTimeout
	}
	return &libraryDownloadGate{
		limits:       limits,
		notify:       make(chan struct{}),
		activeByUser: make(map[int64]int),
	}
}

// acquire waits for all three resource dimensions (global, per-user and
// reserved bytes) at once. Waiting is capped even when the caller supplies a
// context without a deadline. Every capacity/cancellation timeout is exposed
// as the same stable 429 response so clients can safely retry.
func (g *libraryDownloadGate) acquire(ctx context.Context, userID, reservedBytes int64) (*libraryDownloadLease, error) {
	if g == nil {
		return nil, errLibraryDownloadBusy
	}
	if userID <= 0 || reservedBytes <= 0 {
		return nil, service.ErrLibraryInvalidRequest
	}
	if reservedBytes > g.limits.reservedByteLimit {
		return nil, service.ErrLibraryBatchTooLarge.WithMetadata(map[string]string{
			"limit_bytes": fmt.Sprintf("%d", g.limits.reservedByteLimit),
		})
	}
	if ctx == nil {
		ctx = context.Background()
	}

	waitCtx, cancelWait := context.WithTimeout(ctx, g.limits.acquireTimeout)
	defer cancelWait()
	for {
		g.mu.Lock()
		if g.canAcquireLocked(userID, reservedBytes) {
			g.active++
			g.activeByUser[userID]++
			g.reservedBytes += reservedBytes
			g.mu.Unlock()

			operationCtx, cancelOperation := context.WithTimeout(ctx, g.limits.operationTimeout)
			lease := &libraryDownloadLease{
				ctx:           operationCtx,
				cancel:        cancelOperation,
				gate:          g,
				userID:        userID,
				reservedBytes: reservedBytes,
			}
			// Release automatically on disconnect/deadline as a fail-safe. The
			// handler should still defer release so normal completion is prompt.
			go func() {
				<-operationCtx.Done()
				lease.release()
			}()
			return lease, nil
		}
		notify := g.notify
		g.mu.Unlock()

		select {
		case <-notify:
		case <-waitCtx.Done():
			return nil, errLibraryDownloadBusy.WithCause(waitCtx.Err())
		}
	}
}

func (g *libraryDownloadGate) canAcquireLocked(userID, reservedBytes int64) bool {
	return g.active < g.limits.globalLimit &&
		g.activeByUser[userID] < g.limits.perUserLimit &&
		g.reservedBytes <= g.limits.reservedByteLimit-reservedBytes
}

func (g *libraryDownloadGate) release(userID, reservedBytes int64) {
	g.mu.Lock()
	if g.active > 0 {
		g.active--
	}
	if active := g.activeByUser[userID]; active <= 1 {
		delete(g.activeByUser, userID)
	} else {
		g.activeByUser[userID] = active - 1
	}
	if reservedBytes >= g.reservedBytes {
		g.reservedBytes = 0
	} else {
		g.reservedBytes -= reservedBytes
	}
	close(g.notify)
	g.notify = make(chan struct{})
	g.mu.Unlock()
}

// libraryDownloadLease owns both admission counters and the hard operation
// deadline. Its context must be used for staging and the final client copy.
type libraryDownloadLease struct {
	ctx    context.Context
	cancel context.CancelFunc

	gate          *libraryDownloadGate
	userID        int64
	reservedBytes int64
	once          sync.Once
}

func (l *libraryDownloadLease) context() context.Context {
	if l == nil || l.ctx == nil {
		return context.Background()
	}
	return l.ctx
}

func (l *libraryDownloadLease) release() {
	if l == nil {
		return
	}
	l.once.Do(func() {
		if l.cancel != nil {
			l.cancel()
		}
		if l.gate != nil {
			l.gate.release(l.userID, l.reservedBytes)
		}
	})
}

// libraryDownloadReservationBytes mirrors the maximum size written by the
// atomic staging path: the exact stored bytes for one file, or those bytes plus
// the bounded ZIP envelope for a batch.
func libraryDownloadReservationBytes(files []service.LibraryFile) (int64, error) {
	if len(files) == 0 {
		return 0, service.ErrLibraryInvalidRequest
	}
	var reserved int64
	if len(files) > 1 {
		reserved = libraryDownloadZIPReservationOverhead
	}
	for i := range files {
		if files[i].StoredSize <= 0 {
			return 0, service.ErrLibraryDownloadFailed.WithCause(
				fmt.Errorf("library file %q has invalid stored size", files[i].ID),
			)
		}
		if reserved > math.MaxInt64-files[i].StoredSize {
			return 0, service.ErrLibraryBatchTooLarge
		}
		reserved += files[i].StoredSize
	}
	return reserved, nil
}
