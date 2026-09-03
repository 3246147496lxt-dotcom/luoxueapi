package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestDefaultLibraryDownloadGateHasBoundedProductionBudgets(t *testing.T) {
	gate := newDefaultLibraryDownloadGate()

	require.Equal(t, 4, gate.limits.globalLimit)
	require.Equal(t, 2, gate.limits.perUserLimit)
	require.Equal(t, int64(256<<20), gate.limits.reservedByteLimit)
	require.Equal(t, 15*time.Second, gate.limits.acquireTimeout)
	require.Equal(t, 15*time.Minute, gate.limits.operationTimeout)
}

func TestLibraryDownloadGateEnforcesGlobalLimitAndRecoversAfterRelease(t *testing.T) {
	gate := newTestLibraryDownloadGate(2, 2, 100, 20*time.Millisecond, time.Second)
	first := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 1, 10)
	second := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 2, 10)

	blocked, err := gate.acquire(context.Background(), 3, 10)
	require.Nil(t, blocked)
	requireLibraryDownloadBusy(t, err)

	first.release()
	replacement := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 3, 10)
	replacement.release()
	second.release()
}

func TestLibraryDownloadGateEnforcesPerUserLimitWithoutBlockingOtherUsers(t *testing.T) {
	gate := newTestLibraryDownloadGate(4, 2, 100, 20*time.Millisecond, time.Second)
	first := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 7, 10)
	second := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 7, 10)

	blocked, err := gate.acquire(context.Background(), 7, 10)
	require.Nil(t, blocked)
	requireLibraryDownloadBusy(t, err)

	otherUser := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 8, 10)
	otherUser.release()
	first.release()
	second.release()
}

func TestLibraryDownloadGateEnforcesReservedByteLimit(t *testing.T) {
	gate := newTestLibraryDownloadGate(4, 2, 100, 20*time.Millisecond, time.Second)
	first := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 1, 60)
	second := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 2, 40)

	blocked, err := gate.acquire(context.Background(), 3, 1)
	require.Nil(t, blocked)
	requireLibraryDownloadBusy(t, err)

	second.release()
	replacement := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 3, 40)
	replacement.release()
	first.release()
}

func TestLibraryDownloadGateRejectsReservationThatCanNeverFit(t *testing.T) {
	gate := newTestLibraryDownloadGate(4, 2, 100, time.Second, time.Second)
	started := time.Now()

	lease, err := gate.acquire(context.Background(), 1, 101)

	require.Nil(t, lease)
	require.Equal(t, http.StatusRequestEntityTooLarge, infraerrors.Code(err))
	require.Equal(t, "LIBRARY_BATCH_TOO_LARGE", infraerrors.Reason(err))
	require.Less(t, time.Since(started), 100*time.Millisecond)
}

func TestLibraryDownloadGateWaitsForRelease(t *testing.T) {
	gate := newTestLibraryDownloadGate(1, 1, 100, time.Second, time.Second)
	first := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 1, 10)
	type acquireResult struct {
		lease *libraryDownloadLease
		err   error
	}
	result := make(chan acquireResult, 1)
	go func() {
		lease, err := gate.acquire(context.Background(), 2, 10)
		result <- acquireResult{lease: lease, err: err}
	}()

	select {
	case early := <-result:
		if early.lease != nil {
			early.lease.release()
		}
		t.Fatalf("acquire returned before capacity was released: %v", early.err)
	case <-time.After(20 * time.Millisecond):
	}
	first.release()

	select {
	case acquired := <-result:
		require.NoError(t, acquired.err)
		require.NotNil(t, acquired.lease)
		acquired.lease.release()
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for released download capacity")
	}
}

func TestLibraryDownloadGateCancellationReturnsStableBusyError(t *testing.T) {
	gate := newTestLibraryDownloadGate(1, 1, 100, time.Second, time.Second)
	first := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 1, 10)
	defer first.release()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()

	lease, err := gate.acquire(cancelled, 2, 10)

	require.Nil(t, lease)
	requireLibraryDownloadBusy(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, time.Since(started), 100*time.Millisecond)
}

func TestLibraryDownloadGateOperationDeadlineAutomaticallyReleases(t *testing.T) {
	gate := newTestLibraryDownloadGate(1, 1, 100, 200*time.Millisecond, 25*time.Millisecond)
	first := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 1, 10)
	deadline, ok := first.context().Deadline()
	require.True(t, ok)
	require.WithinDuration(t, time.Now().Add(25*time.Millisecond), deadline, 15*time.Millisecond)

	select {
	case <-first.context().Done():
		require.ErrorIs(t, first.context().Err(), context.DeadlineExceeded)
	case <-time.After(time.Second):
		t.Fatal("download operation context did not reach its hard deadline")
	}

	second := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 2, 10)
	second.release()
	first.release() // The caller's required defer remains safe after auto-release.
}

func TestLibraryDownloadGateReleaseIsIdempotentAndRestoresAllCounters(t *testing.T) {
	gate := newTestLibraryDownloadGate(1, 1, 10, time.Second, time.Second)
	lease := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 1, 10)

	lease.release()
	lease.release()

	gate.mu.Lock()
	require.Zero(t, gate.active)
	require.Zero(t, gate.reservedBytes)
	require.Empty(t, gate.activeByUser)
	gate.mu.Unlock()
	replacement := mustAcquireLibraryDownloadLease(t, gate, context.Background(), 1, 10)
	replacement.release()
}

func TestLibraryDownloadGateProtectsHandlerBeforeCreatingTemporaryFile(t *testing.T) {
	data := []byte("private download")
	files := map[string]service.LibraryFile{
		"file_alpha": newHandlerLibraryFile("file_alpha", "private.txt", "library/7/alpha", data),
	}
	handler, stagedPaths := newHandlerLibraryDownloadFixture(t, files, &handlerLibraryStore{
		objects: map[string][]byte{"library/7/alpha": data},
	})
	handler.downloadGate = newTestLibraryDownloadGate(1, 1, 100, 20*time.Millisecond, time.Second)
	held := mustAcquireLibraryDownloadLease(t, handler.downloadGate, context.Background(), 8, 10)
	defer held.release()

	recorder := performLibraryBatchDownload(
		t,
		handler,
		`{"file_ids":["file_alpha"]}`,
		context.Background(),
	)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	assertLibraryErrorEnvelope(t, recorder, "LIBRARY_DOWNLOAD_BUSY")
	require.Equal(t, "15", recorder.Header().Get("Retry-After"))
	require.Empty(t, *stagedPaths, "admission must fail before a temporary file is created")
}

func TestLibraryDownloadReservationBytesAccountsForZIPEnvelope(t *testing.T) {
	single, err := libraryDownloadReservationBytes([]service.LibraryFile{{ID: "one", StoredSize: 11}})
	require.NoError(t, err)
	require.Equal(t, int64(11), single)

	batch, err := libraryDownloadReservationBytes([]service.LibraryFile{
		{ID: "one", StoredSize: 11},
		{ID: "two", StoredSize: 17},
	})
	require.NoError(t, err)
	require.Equal(t, int64(28+libraryDownloadZIPReservationOverhead), batch)

	_, err = libraryDownloadReservationBytes(nil)
	require.ErrorIs(t, err, service.ErrLibraryInvalidRequest)
	_, err = libraryDownloadReservationBytes([]service.LibraryFile{{ID: "bad", StoredSize: 0}})
	require.ErrorIs(t, err, service.ErrLibraryDownloadFailed)
}

func newTestLibraryDownloadGate(
	globalLimit, perUserLimit int,
	reservedByteLimit int64,
	acquireTimeout, operationTimeout time.Duration,
) *libraryDownloadGate {
	return newLibraryDownloadGate(libraryDownloadGateLimits{
		globalLimit:       globalLimit,
		perUserLimit:      perUserLimit,
		reservedByteLimit: reservedByteLimit,
		acquireTimeout:    acquireTimeout,
		operationTimeout:  operationTimeout,
	})
}

func mustAcquireLibraryDownloadLease(
	t *testing.T,
	gate *libraryDownloadGate,
	ctx context.Context,
	userID, reservedBytes int64,
) *libraryDownloadLease {
	t.Helper()
	lease, err := gate.acquire(ctx, userID, reservedBytes)
	require.NoError(t, err)
	require.NotNil(t, lease)
	return lease
}

func requireLibraryDownloadBusy(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	require.Equal(t, http.StatusTooManyRequests, infraerrors.Code(err))
	require.Equal(t, "LIBRARY_DOWNLOAD_BUSY", infraerrors.Reason(err))
	require.True(t, errors.Is(err, errLibraryDownloadBusy))
}
