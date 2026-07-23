package repository

import (
	"context"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var errUsageLogBatchRuntimeStopped = errors.New("usage log batch runtime stopped")

// UsageLogBatchRuntime exposes repository-owned batcher shutdown without
// widening the business-facing service.UsageLogRepository interface.
type UsageLogBatchRuntime interface {
	Stop(context.Context) error
}

// ProvideUsageLogBatchRuntime keeps the lifecycle dependency narrow while
// reusing the exact repository instance injected into usage producers.
func ProvideUsageLogBatchRuntime(repo service.UsageLogRepository) (UsageLogBatchRuntime, error) {
	runtime, ok := repo.(UsageLogBatchRuntime)
	if !ok {
		return nil, errors.New("usage log repository does not provide batch runtime lifecycle")
	}
	return runtime, nil
}

func (r *usageLogRepository) enqueueCreateBatchRequest(ctx context.Context, req usageLogCreateRequest) error {
	batchCh, ok := r.acquireCreateBatchChannel()
	if !ok {
		return errUsageLogBatchRuntimeStopped
	}
	defer r.batchEnqueueWG.Done()
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case batchCh <- req:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *usageLogRepository) enqueueBestEffortBatchRequest(ctx context.Context, req usageLogBestEffortRequest) error {
	batchCh, ok := r.acquireBestEffortBatchChannel()
	if !ok {
		return errUsageLogBatchRuntimeStopped
	}
	defer r.batchEnqueueWG.Done()
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case batchCh <- req:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *usageLogRepository) acquireCreateBatchChannel() (chan usageLogCreateRequest, bool) {
	if r == nil || r.db == nil {
		return nil, false
	}
	r.batchMu.Lock()
	defer r.batchMu.Unlock()
	if !r.batchAccepting {
		return nil, false
	}
	if r.createBatchCh == nil {
		r.createBatchCh = make(chan usageLogCreateRequest, usageLogCreateBatchQueueCap)
		batchCh := r.createBatchCh
		r.batchWorkerWG.Add(1)
		go func() {
			defer r.batchWorkerWG.Done()
			r.runCreateBatcher(r.db, batchCh)
		}()
	}
	r.batchEnqueueWG.Add(1)
	return r.createBatchCh, true
}

func (r *usageLogRepository) acquireBestEffortBatchChannel() (chan usageLogBestEffortRequest, bool) {
	if r == nil || r.db == nil {
		return nil, false
	}
	r.batchMu.Lock()
	defer r.batchMu.Unlock()
	if !r.batchAccepting {
		return nil, false
	}
	if r.bestEffortBatchCh == nil {
		r.bestEffortBatchCh = make(chan usageLogBestEffortRequest, usageLogBestEffortBatchQueueCap)
		batchCh := r.bestEffortBatchCh
		r.batchWorkerWG.Add(1)
		go func() {
			defer r.batchWorkerWG.Done()
			r.runBestEffortBatcher(r.db, batchCh)
		}()
	}
	r.batchEnqueueWG.Add(1)
	return r.bestEffortBatchCh, true
}

// Stop rejects new batch enqueues, lets already-admitted senders finish, then
// closes both queues and waits for their workers to flush every queued request.
// A caller timeout does not abort the shared drain; a later Stop can keep
// waiting for the same completion signal.
func (r *usageLogRepository) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	r.batchStopOnce.Do(func() {
		r.batchMu.Lock()
		r.batchAccepting = false
		if r.batchStopDone == nil {
			r.batchStopDone = make(chan struct{})
		}
		stopDone := r.batchStopDone
		r.batchMu.Unlock()

		go r.drainUsageLogBatchers(stopDone)
	})

	select {
	case <-r.batchStopDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *usageLogRepository) drainUsageLogBatchers(stopDone chan struct{}) {
	// Admission increments while holding batchMu. Stop flips batchAccepting
	// under the same lock before waiting, so no Add can race this Wait.
	r.batchEnqueueWG.Wait()

	r.batchMu.Lock()
	if r.createBatchCh != nil {
		close(r.createBatchCh)
	}
	if r.bestEffortBatchCh != nil {
		close(r.bestEffortBatchCh)
	}
	r.batchMu.Unlock()

	r.batchWorkerWG.Wait()
	close(stopDone)
}
