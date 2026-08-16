//go:build unit

package repository

import (
	"context"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestLibraryFileCreatePendingLocksAndReservesCapacityAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	repo := NewLibraryFileRepository(db, &config.Config{ChatAttachments: config.ChatAttachmentConfig{
		UploadsPerMinute: 10, DailyUploadBytes: 100 << 20,
	}})

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtext('library_files'), hashint8($1::bigint))")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"lock"}).AddRow(nil))
	mock.ExpectQuery("COUNT\\(\\*\\) FILTER.*SUM\\(stored_size\\) FILTER").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "daily", "used"}).AddRow(0, 0, int64(90)))
	mock.ExpectRollback()

	_, err = repo.CreatePending(context.Background(), &service.CreateLibraryFileInput{
		UserID: 7,
		File: service.LibraryFile{
			ID: "file_capacity", Name: "report.pdf", Format: "pdf", Category: "file", Type: "pdf",
			Source: "uploaded", MIMEType: "application/pdf", Extension: ".pdf", Size: 20,
			StoredSize: 20, Digest: strings.Repeat("a", 64), StorageKey: "library/7/file_capacity.pdf",
			Status: "pending",
		},
	}, 100)
	require.ErrorIs(t, err, service.ErrLibraryStorageLimit)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLibraryFileGetOwnedDistinguishesAccessDenied(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	repo := NewLibraryFileRepository(db, nil)

	mock.ExpectQuery("FROM library_files f.*f.user_id=\\$1.*f.public_id=\\$2").
		WithArgs(int64(7), "file_other").
		WillReturnRows(sqlmock.NewRows(libraryFileTestColumns()))
	mock.ExpectQuery("SELECT user_id FROM library_files WHERE public_id=\\$1 AND status='ready'").
		WithArgs("file_other").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(8)))

	_, err = repo.GetOwned(context.Background(), 7, "file_other")
	require.ErrorIs(t, err, service.ErrLibraryFileAccessDenied)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLibraryFileMarkDeletedDistinguishesAccessDenied(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	db.SetMaxOpenConns(1)
	repo := NewLibraryFileRepository(db, nil)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtext('library_files'), hashint8($1::bigint))")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"lock"}).AddRow(nil))
	mock.ExpectQuery("UPDATE library_files AS f.*status IN \\('pending','ready'\\)").
		WithArgs(int64(7), "file_other").
		WillReturnRows(sqlmock.NewRows(libraryFileTestColumns()))
	mock.ExpectQuery("SELECT user_id FROM library_files WHERE public_id=\\$1 AND status='ready'").
		WithArgs("file_other").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(8)))
	mock.ExpectRollback()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = repo.MarkDeleted(ctx, 7, "file_other")
	require.ErrorIs(t, err, service.ErrLibraryFileAccessDenied)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLibraryFileAbortNoRowLookupReusesTransactionConnection(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	// A regression to r.db.QueryRowContext while the transaction holds this sole
	// connection waits for a second pool connection until the context expires.
	db.SetMaxOpenConns(1)
	repo := NewLibraryFileRepository(db, nil)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtext('library_files'), hashint8($1::bigint))")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"lock"}).AddRow(nil))
	mock.ExpectQuery("UPDATE library_files AS f.*f.status='pending'").
		WithArgs(int64(7), "file_other").
		WillReturnRows(sqlmock.NewRows(libraryFileTestColumns()))
	mock.ExpectQuery("SELECT user_id FROM library_files WHERE public_id=\\$1 AND status='ready'").
		WithArgs("file_other").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(int64(8)))
	mock.ExpectRollback()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = repo.Abort(ctx, 7, "file_other")
	require.ErrorIs(t, err, service.ErrLibraryFileAccessDenied)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLibraryFileUsageCountsOnlyReady(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	repo := NewLibraryFileRepository(db, nil)

	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(stored_size\\), 0\\).*status='ready'").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"used"}).AddRow(int64(321)))
	used, err := repo.Usage(context.Background(), 7)
	require.NoError(t, err)
	require.EqualValues(t, 321, used)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLibraryFileResolveStorageLimitUsesLargestActiveOverride(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	repo := NewLibraryFileRepository(db, nil)

	mock.ExpectQuery("SELECT group_id.*FROM user_subscriptions.*status='active'").
		WithArgs(int64(7), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"group_id"}).AddRow(int64(3)).AddRow(int64(5)))
	limit, err := repo.ResolveStorageLimit(context.Background(), 7, 100, map[int64]int64{3: 200, 5: 500})
	require.NoError(t, err)
	require.EqualValues(t, 500, limit)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLibraryFileClaimCleanupReclaimsStalePendingUploads(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	repo := NewLibraryFileRepository(db, nil)
	deletedBefore := time.Now().UTC().Add(-30 * 24 * time.Hour)
	pendingBefore := time.Now().UTC().Add(-time.Hour)

	mock.ExpectBegin()
	mock.ExpectQuery("status='pending' AND created_at <= \\$2").
		WithArgs(deletedBefore, pendingBefore, 10).
		WillReturnRows(sqlmock.NewRows(libraryFileTestColumns()).AddRow(libraryFileTestRow()...))
	mock.ExpectExec("UPDATE chat_attachments AS a").
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	files, err := repo.ClaimCleanup(context.Background(), deletedBefore, pendingBefore, 10)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLibraryFileCheckUploadQuotaCountsUploadedSourceOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	repo := NewLibraryFileRepository(db, &config.Config{ChatAttachments: config.ChatAttachmentConfig{
		UploadsPerMinute: 10, DailyUploadBytes: 100 << 20,
	}})

	mock.ExpectQuery("FROM library_files.*WHERE user_id=\\$1 AND source='uploaded'").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "bytes"}).AddRow(0, int64(0)))
	require.NoError(t, repo.CheckUploadQuota(context.Background(), 7, 123))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLibraryFileCreatePendingGeneratedBypassesInteractiveQuotaButReservesStorage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { mock.ExpectClose(); require.NoError(t, db.Close()) }()
	repo := NewLibraryFileRepository(db, &config.Config{ChatAttachments: config.ChatAttachmentConfig{
		UploadsPerMinute: 1, DailyUploadBytes: 1,
	}})
	sourceKey := "batch_image/imgbatch_test/Y292ZXI/0"
	now := time.Now().UTC()
	row := libraryFileTestRow()
	row[3] = "cover.png"
	row[4] = "png"
	row[5] = "image"
	row[6] = "image"
	row[7] = "generated"
	row[8] = sourceKey
	row[9] = "image/png"
	row[10] = "png"
	row[11] = int64(10)
	row[12] = int64(10)
	row[13] = "pending"
	row[19] = now
	row[20] = now

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtext('library_files'), hashint8($1::bigint))")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"lock"}).AddRow(nil))
	mock.ExpectQuery("FROM library_files f.*f.source_key=\\$1.*FOR UPDATE").
		WithArgs(sourceKey).
		WillReturnRows(sqlmock.NewRows(libraryFileTestColumns()))
	// Interactive counts are deliberately above the configured thresholds; the
	// trusted generated path ignores them, while used storage is still checked.
	mock.ExpectQuery("COUNT\\(\\*\\) FILTER.*SUM\\(stored_size\\) FILTER").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "daily", "used"}).AddRow(999, int64(999), int64(40)))
	mock.ExpectQuery("INSERT INTO library_files AS f").
		WithArgs(
			"file_generated", int64(7), "cover.png", "png", "image", "image", "generated", sourceKey,
			"image/png", "png", int64(10), int64(10), strings.Repeat("a", 64), "library/7/generated",
			nil, nil, nil, "pending",
		).
		WillReturnRows(sqlmock.NewRows(libraryFileTestColumns()).AddRow(row...))
	mock.ExpectExec("INSERT INTO chat_attachments").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	created, err := repo.CreatePending(context.Background(), &service.CreateLibraryFileInput{
		UserID: 7, SourceKey: sourceKey, BypassUploadQuota: true,
		File: service.LibraryFile{
			ID: "file_generated", Name: "cover.png", Format: "png", Category: "image", Type: "image",
			Source: "generated", MIMEType: "image/png", Extension: "png", Size: 10, StoredSize: 10,
			Digest: strings.Repeat("a", 64), StorageKey: "library/7/generated", Status: "pending",
		},
	}, 100)
	require.NoError(t, err)
	require.Equal(t, sourceKey, created.SourceKey)
	require.NoError(t, mock.ExpectationsWereMet())
}

func libraryFileTestColumns() []string {
	return []string{
		"id", "public_id", "user_id", "original_name", "storage_kind", "category", "file_type",
		"source", "source_key", "mime_type", "extension", "byte_size", "stored_size", "status", "page_count",
		"width", "height", "storage_key", "sha256", "created_at", "updated_at", "last_used_at",
		"deleted_at", "cleaned_at",
	}
}

func libraryFileTestRow() []driver.Value {
	now := time.Now().UTC()
	return []driver.Value{
		int64(1), "file_test", int64(7), "test.pdf", "pdf", "file", "pdf", "uploaded",
		"", "application/pdf", ".pdf", int64(10), int64(10), "ready", nil, nil, nil,
		"library/7/file_test.pdf", strings.Repeat("a", 64), now, now, nil, nil, nil,
	}
}
