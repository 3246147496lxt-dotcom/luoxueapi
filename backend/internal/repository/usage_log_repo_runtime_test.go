package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogBatchRuntimeStopDrainsQueuedRequests(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.MatchExpectationsInOrder(false)

	createdAt := time.Date(2026, time.July, 23, 10, 30, 0, 0, time.UTC)
	payload := fmt.Sprintf(
		`[{"request_id":"drain-create","api_key_id":11,"id":101,"created_at":%q,"inserted":true}]`,
		createdAt.Format(time.RFC3339Nano),
	)
	mock.ExpectQuery(`WITH input`).
		WillDelayFor(40 * time.Millisecond).
		WillReturnRows(sqlmock.NewRows([]string{"payload"}).AddRow([]byte(payload)))
	mock.ExpectExec(`WITH input`).
		WillDelayFor(40 * time.Millisecond).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := newUsageLogRepositoryWithSQL(nil, db)
	createLog := usageLogRuntimeTestLog("drain-create", 11)
	bestEffortLog := usageLogRuntimeTestLog("drain-best-effort", 12)

	type createResult struct {
		inserted bool
		err      error
	}
	createResultCh := make(chan createResult, 1)
	bestEffortResultCh := make(chan error, 1)
	go func() {
		inserted, createErr := repo.Create(context.Background(), createLog)
		createResultCh <- createResult{inserted: inserted, err: createErr}
	}()
	go func() {
		bestEffortResultCh <- repo.CreateBestEffort(context.Background(), bestEffortLog)
	}()

	require.Eventually(t, func() bool {
		repo.batchMu.Lock()
		defer repo.batchMu.Unlock()
		return repo.createBatchCh != nil && repo.bestEffortBatchCh != nil
	}, time.Second, time.Millisecond)
	require.NoError(t, repo.Stop(context.Background()))

	createResultValue := <-createResultCh
	require.NoError(t, createResultValue.err)
	require.True(t, createResultValue.inserted)
	require.Equal(t, int64(101), createLog.ID)
	require.Equal(t, createdAt, createLog.CreatedAt)
	require.NoError(t, <-bestEffortResultCh)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogBatchRuntimeConcurrentEnqueueAndStop(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	repo := newUsageLogRepositoryWithSQL(nil, db)
	createBatchCh := make(chan usageLogCreateRequest, 128)
	bestEffortBatchCh := make(chan usageLogBestEffortRequest, 128)
	createReceived := make(chan struct{})
	bestEffortReceived := make(chan struct{})
	releaseWorkers := make(chan struct{})

	repo.batchMu.Lock()
	repo.createBatchCh = createBatchCh
	repo.bestEffortBatchCh = bestEffortBatchCh
	repo.batchWorkerWG.Add(2)
	repo.batchMu.Unlock()

	go func() {
		defer repo.batchWorkerWG.Done()
		first := true
		for req := range createBatchCh {
			if first {
				close(createReceived)
				<-releaseWorkers
				first = false
			}
			if req.log != nil {
				req.log.ID = 7001
			}
			completeUsageLogCreateRequest(req, usageLogCreateResult{inserted: true})
		}
	}()
	go func() {
		defer repo.batchWorkerWG.Done()
		first := true
		for req := range bestEffortBatchCh {
			if first {
				close(bestEffortReceived)
				<-releaseWorkers
				first = false
			}
			sendUsageLogBestEffortResult(req.resultCh, nil)
		}
	}()

	initialCreateResult := make(chan error, 1)
	go func() {
		inserted, createErr := repo.Create(context.Background(), usageLogRuntimeTestLog("initial-create", 1))
		if createErr == nil && !inserted {
			createErr = errors.New("initial create was not inserted")
		}
		initialCreateResult <- createErr
	}()
	initialBestEffortResult := make(chan error, 1)
	go func() {
		initialBestEffortResult <- repo.CreateBestEffort(context.Background(), usageLogRuntimeTestLog("initial-best-effort", 2))
	}()
	<-createReceived
	<-bestEffortReceived

	const racers = 96
	start := make(chan struct{})
	results := make([]error, racers)
	inserted := make([]bool, racers)
	var callers sync.WaitGroup
	callers.Add(racers)
	for i := 0; i < racers; i++ {
		i := i
		go func() {
			defer callers.Done()
			<-start
			log := usageLogRuntimeTestLog(fmt.Sprintf("race-%d", i), int64(i+10))
			if i%2 == 0 {
				inserted[i], results[i] = repo.Create(context.Background(), log)
				return
			}
			results[i] = repo.CreateBestEffort(context.Background(), log)
		}()
	}
	stopResult := make(chan error, 1)
	go func() {
		<-start
		stopResult <- repo.Stop(context.Background())
	}()
	close(start)

	require.Eventually(t, func() bool {
		repo.batchMu.Lock()
		defer repo.batchMu.Unlock()
		return !repo.batchAccepting
	}, time.Second, time.Millisecond)
	close(releaseWorkers)
	callers.Wait()

	require.NoError(t, <-initialCreateResult)
	require.NoError(t, <-initialBestEffortResult)
	require.NoError(t, <-stopResult)
	for i, resultErr := range results {
		if resultErr == nil {
			if i%2 == 0 {
				require.True(t, inserted[i])
			}
			continue
		}
		if i%2 == 0 {
			require.True(t, service.IsUsageLogCreateNotPersisted(resultErr), "create result %d: %v", i, resultErr)
		} else {
			require.True(t, service.IsUsageLogCreateDropped(resultErr), "best-effort result %d: %v", i, resultErr)
		}
	}

	_, err = repo.Create(context.Background(), usageLogRuntimeTestLog("after-stop-create", 1001))
	require.Error(t, err)
	require.True(t, service.IsUsageLogCreateNotPersisted(err))
	err = repo.CreateBestEffort(context.Background(), usageLogRuntimeTestLog("after-stop-best-effort", 1002))
	require.Error(t, err)
	require.True(t, service.IsUsageLogCreateDropped(err))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogBatchRuntimeStopBeforeLazyStartIsIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	repo := newUsageLogRepositoryWithSQL(nil, db)
	runtime, err := ProvideUsageLogBatchRuntime(repo)
	require.NoError(t, err)

	const callers = 16
	results := make(chan error, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			results <- runtime.Stop(context.Background())
		}()
	}
	wg.Wait()
	close(results)
	for stopErr := range results {
		require.NoError(t, stopErr)
	}
	require.NoError(t, runtime.Stop(context.Background()))

	repo.batchMu.Lock()
	require.Nil(t, repo.createBatchCh)
	require.Nil(t, repo.bestEffortBatchCh)
	require.False(t, repo.batchAccepting)
	repo.batchMu.Unlock()
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUsageLogBatchRuntimeStopTimeoutContinuesDrain(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	repo := newUsageLogRepositoryWithSQL(nil, db)
	createBatchCh := make(chan usageLogCreateRequest)
	requestReceived := make(chan struct{})
	releaseWorker := make(chan struct{})
	repo.batchMu.Lock()
	repo.createBatchCh = createBatchCh
	repo.batchWorkerWG.Add(1)
	repo.batchMu.Unlock()
	go func() {
		defer repo.batchWorkerWG.Done()
		for req := range createBatchCh {
			close(requestReceived)
			<-releaseWorker
			if req.log != nil {
				req.log.ID = 909
			}
			completeUsageLogCreateRequest(req, usageLogCreateResult{inserted: true})
		}
	}()

	log := usageLogRuntimeTestLog("timeout-create", 91)
	type createResult struct {
		inserted bool
		err      error
	}
	createResultCh := make(chan createResult, 1)
	go func() {
		inserted, createErr := repo.Create(context.Background(), log)
		createResultCh <- createResult{inserted: inserted, err: createErr}
	}()
	<-requestReceived

	stopCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = repo.Stop(stopCtx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	_, rejectedErr := repo.Create(context.Background(), usageLogRuntimeTestLog("timeout-rejected", 92))
	require.Error(t, rejectedErr)
	require.True(t, service.IsUsageLogCreateNotPersisted(rejectedErr))

	secondStopResult := make(chan error, 1)
	go func() { secondStopResult <- repo.Stop(context.Background()) }()
	close(releaseWorker)
	require.NoError(t, <-secondStopResult)

	result := <-createResultCh
	require.NoError(t, result.err)
	require.True(t, result.inserted)
	require.Equal(t, int64(909), log.ID)
	require.NoError(t, repo.Stop(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func usageLogRuntimeTestLog(requestID string, apiKeyID int64) *service.UsageLog {
	return &service.UsageLog{
		UserID:       apiKeyID + 1000,
		APIKeyID:     apiKeyID,
		AccountID:    apiKeyID + 2000,
		RequestID:    requestID,
		Model:        "claude-test",
		InputTokens:  10,
		OutputTokens: 20,
		TotalCost:    0.5,
		ActualCost:   0.4,
		CreatedAt:    time.Now().UTC(),
	}
}
