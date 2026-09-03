//go:build unit

package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBatchImageLibraryIngestStateBeginRetryAndComplete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	repo := &batchImageRepository{db: db, sql: db}
	batchID := "imgbatch_library_state"

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO batch_image_library_ingests").
		WithArgs(batchID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("FROM batch_image_library_ingests.*FOR UPDATE").
		WithArgs(batchID).
		WillReturnRows(sqlmock.NewRows(batchImageLibraryStateColumns()).
			AddRow(batchID, "pending", 0, 0, 0, 0, "", ""))
	mock.ExpectQuery("UPDATE batch_image_library_ingests.*attempts=attempts\\+1").
		WithArgs(batchID).
		WillReturnRows(sqlmock.NewRows(batchImageLibraryStateColumns()).
			AddRow(batchID, "pending", 1, 0, 0, 0, "", ""))
	mock.ExpectCommit()

	state, started, err := repo.BeginBatchImageLibraryIngest(context.Background(), batchID, 3)
	require.NoError(t, err)
	require.True(t, started)
	require.Equal(t, 1, state.Attempts)

	mock.ExpectExec("UPDATE batch_image_library_ingests.*status='retrying'").
		WithArgs(batchID, 1, 0, 1, "LIBRARY_GENERATED_SAVE_RETRYABLE", "temporary").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.RetryBatchImageLibraryIngest(context.Background(), batchID, service.BatchImageLibraryIngestAttempt{
		ImportedCount: 1, FailedCount: 1, ErrorCode: "LIBRARY_GENERATED_SAVE_RETRYABLE", ErrorMessage: "temporary",
	}))

	mock.ExpectExec("UPDATE batch_image_library_ingests.*status=\\$2").
		WithArgs(batchID, service.BatchImageLibraryIngestCompleted, 2, 0, 0, "", "").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.CompleteBatchImageLibraryIngest(context.Background(), batchID, service.BatchImageLibraryIngestCompleted,
		service.BatchImageLibraryIngestAttempt{ImportedCount: 2}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchImageLibraryIngestStateExhaustedBecomesTerminal(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	repo := &batchImageRepository{db: db, sql: db}
	batchID := "imgbatch_library_exhausted"

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO batch_image_library_ingests").
		WithArgs(batchID).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("FROM batch_image_library_ingests.*FOR UPDATE").
		WithArgs(batchID).
		WillReturnRows(sqlmock.NewRows(batchImageLibraryStateColumns()).
			AddRow(batchID, "retrying", 3, 1, 0, 1, "LIBRARY_GENERATED_SAVE_RETRYABLE", "temporary"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE batch_image_library_ingests")).
		WithArgs(batchID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	state, started, err := repo.BeginBatchImageLibraryIngest(context.Background(), batchID, 3)
	require.NoError(t, err)
	require.False(t, started)
	require.Equal(t, service.BatchImageLibraryIngestCompletedWithErrors, state.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func batchImageLibraryStateColumns() []string {
	return []string{"job_id", "status", "attempts", "imported_count", "suppressed_count", "failed_count", "last_error_code", "last_error_message"}
}
