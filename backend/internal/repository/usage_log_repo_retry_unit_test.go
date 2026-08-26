//go:build unit

package repository

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

func TestPreferSuccessfulUsageLogPromotesZeroCostPlaceholder(t *testing.T) {
	failure := prepareUsageLogInsert(&service.UsageLog{
		RequestID:  "retry-request",
		APIKeyID:   7,
		ActualCost: 0,
	})
	success := prepareUsageLogInsert(&service.UsageLog{
		RequestID:  "retry-request",
		APIKeyID:   7,
		ActualCost: 1.25,
	})

	got := preferSuccessfulUsageLog(failure, success)
	require.Equal(t, 1.25, got.actualCost)
	require.Equal(t, success.actualCost, preferSuccessfulUsageLog(success, failure).actualCost)
}

func TestUsageLogRepositoryCreateBestEffortRecentFailureDoesNotHideSuccessRetry(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	requestID := "retry-recent"
	apiKeyID := int64(11)
	repo := &usageLogRepository{
		sql:               db,
		db:                db,
		batchAccepting:    true,
		bestEffortBatchCh: make(chan usageLogBestEffortRequest, 1),
		bestEffortRecent:  cache.New(time.Minute, time.Minute),
	}
	key := usageLogBatchKey(requestID, apiKeyID)
	repo.bestEffortRecent.SetDefault(key, struct{}{})

	// A duplicate failure remains a cheap no-op while its recent marker is hot.
	require.NoError(t, repo.CreateBestEffort(context.Background(), &service.UsageLog{
		RequestID:  requestID,
		APIKeyID:   apiKeyID,
		ActualCost: 0,
	}))
	require.Len(t, repo.bestEffortBatchCh, 0)

	// A successful retry must bypass that marker and reach the batch queue.
	done := make(chan error, 1)
	go func() {
		done <- repo.CreateBestEffort(context.Background(), &service.UsageLog{
			RequestID:  requestID,
			APIKeyID:   apiKeyID,
			ActualCost: 1.25,
		})
	}()

	var req usageLogBestEffortRequest
	select {
	case req = <-repo.bestEffortBatchCh:
	case <-time.After(time.Second):
		t.Fatal("successful retry was hidden by recent best-effort marker")
	}
	require.Equal(t, 1.25, req.prepared.actualCost)
	sendUsageLogBestEffortResult(req.resultCh, nil)
	require.NoError(t, <-done)
}

func TestUsageLogInsertQueriesPromoteSuccessfulRetry(t *testing.T) {
	prepared := prepareUsageLogInsert(&service.UsageLog{
		RequestID:  "retry-query",
		APIKeyID:   5,
		ActualCost: 2,
	})
	key := usageLogBatchKey(prepared.requestID, 5)

	batchQuery, _ := buildUsageLogBatchInsertQuery([]string{key}, map[string]usageLogInsertPrepared{key: prepared})
	require.Contains(t, batchQuery, "ON CONFLICT (request_id, api_key_id) DO UPDATE")
	require.Contains(t, batchQuery, "SET actual_cost = EXCLUDED.actual_cost")
	require.Contains(t, batchQuery, "usage_logs.actual_cost <= 0 AND EXCLUDED.actual_cost > 0")

	bestEffortQuery, _ := buildUsageLogBestEffortInsertQuery([]usageLogInsertPrepared{prepared})
	require.Contains(t, bestEffortQuery, "ON CONFLICT (request_id, api_key_id) DO UPDATE")
	require.Contains(t, bestEffortQuery, "SET actual_cost = EXCLUDED.actual_cost")
	require.Contains(t, bestEffortQuery, "usage_logs.actual_cost <= 0 AND EXCLUDED.actual_cost > 0")
}

func TestCreateSingleUsesSuccessfulRetryPromotionClause(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	createdAt := time.Date(2026, time.August, 26, 1, 2, 3, 0, time.UTC)
	log := &service.UsageLog{
		UserID:     1,
		APIKeyID:   5,
		AccountID:  9,
		RequestID:  "retry-single",
		ActualCost: 2,
		CreatedAt:  createdAt,
	}
	prepared := prepareUsageLogInsert(log)
	mock.ExpectQuery(`(?s)INSERT INTO usage_logs.*ON CONFLICT \(request_id, api_key_id\) DO UPDATE.*usage_logs\.actual_cost <= 0 AND EXCLUDED\.actual_cost > 0`).
		WithArgs(anySliceToDriverValues(prepared.args)...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(77), createdAt))

	inserted, err := repo.Create(context.Background(), log)
	require.NoError(t, err)
	require.True(t, inserted)
	require.Equal(t, int64(77), log.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFlushCreateBatchAssignsPromotionToSuccessfulRetry(t *testing.T) {
	db, mock := newSQLMock(t)
	createdAt := time.Date(2026, time.August, 26, 2, 3, 4, 0, time.UTC)
	requestID := "retry-create-batch"
	apiKeyID := int64(7)
	failureLog := &service.UsageLog{
		UserID:     1,
		APIKeyID:   apiKeyID,
		AccountID:  9,
		RequestID:  requestID,
		CreatedAt:  createdAt,
		ActualCost: 0,
	}
	successLog := &service.UsageLog{
		UserID:     1,
		APIKeyID:   apiKeyID,
		AccountID:  9,
		RequestID:  requestID,
		CreatedAt:  createdAt,
		ActualCost: 1.25,
	}
	failurePrepared := prepareUsageLogInsert(failureLog)
	successPrepared := prepareUsageLogInsert(successLog)
	expectedArgs := append([]driver.Value{0}, anySliceToDriverValues(successPrepared.args)...)
	mock.ExpectQuery("WITH input").
		WithArgs(expectedArgs...).
		WillReturnRows(sqlmock.NewRows([]string{"payload"}).AddRow([]byte(fmt.Sprintf(
			`[{"request_id":%q,"api_key_id":%d,"id":42,"created_at":%q,"inserted":true}]`,
			requestID, apiKeyID, createdAt.Format(time.RFC3339Nano),
		))))

	failureResult := make(chan usageLogCreateResult, 1)
	successResult := make(chan usageLogCreateResult, 1)
	(&usageLogRepository{}).flushCreateBatch(db, []usageLogCreateRequest{
		{log: failureLog, prepared: failurePrepared, resultCh: failureResult},
		{log: successLog, prepared: successPrepared, resultCh: successResult},
	})

	require.False(t, (<-failureResult).inserted, "failed placeholder must not receive the promotion")
	require.True(t, (<-successResult).inserted, "successful retry must be the promoting representative")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFlushBestEffortBatchPrefersSuccessfulRetry(t *testing.T) {
	db, mock := newSQLMock(t)
	createdAt := time.Date(2026, time.August, 26, 3, 4, 5, 0, time.UTC)
	requestID := "retry-best-effort-batch"
	apiKeyID := int64(8)
	failurePrepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:     1,
		APIKeyID:   apiKeyID,
		AccountID:  9,
		RequestID:  requestID,
		CreatedAt:  createdAt,
		ActualCost: 0,
	})
	successPrepared := prepareUsageLogInsert(&service.UsageLog{
		UserID:     1,
		APIKeyID:   apiKeyID,
		AccountID:  9,
		RequestID:  requestID,
		CreatedAt:  createdAt,
		ActualCost: 1.25,
	})
	mock.ExpectExec("WITH input").
		WithArgs(anySliceToDriverValues(successPrepared.args)...).
		WillReturnResult(sqlmock.NewResult(0, 1))

	failureResult := make(chan error, 1)
	successResult := make(chan error, 1)
	repo := &usageLogRepository{}
	repo.flushBestEffortBatch(db, []usageLogBestEffortRequest{
		{prepared: failurePrepared, apiKeyID: apiKeyID, resultCh: failureResult},
		{prepared: successPrepared, apiKeyID: apiKeyID, resultCh: successResult},
	})

	require.NoError(t, <-failureResult)
	require.NoError(t, <-successResult)
	require.NoError(t, mock.ExpectationsWereMet())
}
