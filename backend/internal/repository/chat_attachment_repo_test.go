package repository

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestChatAttachmentQuotaPreflightReturns429BeforeParsing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	}()
	repo := NewChatAttachmentRepository(db, &config.Config{ChatAttachments: config.ChatAttachmentConfig{UploadsPerMinute: 10, DailyUploadBytes: 100 << 20}})
	mock.ExpectQuery("SELECT\\s+COUNT\\(\\*\\) FILTER").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "bytes"}).AddRow(10, 1))
	err = repo.CheckUploadQuota(context.Background(), 7, 1024)
	require.ErrorIs(t, err, service.ErrChatAttachmentRateLimit)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatAttachmentCreateUsesNamespacedLockAndAtomicQuota(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	}()
	repo := NewChatAttachmentRepository(db, &config.Config{ChatAttachments: config.ChatAttachmentConfig{UploadsPerMinute: 10, DailyUploadBytes: 100 << 20}})
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtext('chat_attachments'), hashint8($1::bigint))")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"lock"}).AddRow(nil))
	mock.ExpectQuery("SELECT\\s+COUNT\\(\\*\\) FILTER").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "bytes"}).AddRow(10, 0))
	mock.ExpectRollback()
	_, err = repo.Create(context.Background(), &service.CreateChatAttachmentInput{UserID: 7, Attachment: service.ChatAttachment{Size: 1}})
	require.ErrorIs(t, err, service.ErrChatAttachmentRateLimit)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatAttachmentCreateAliasesInsertForReturningColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	}()
	repo := NewChatAttachmentRepository(db, &config.Config{ChatAttachments: config.ChatAttachmentConfig{UploadsPerMinute: 10, DailyUploadBytes: 100 << 20}})
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_advisory_xact_lock(hashtext('chat_attachments'), hashint8($1::bigint))")).
		WithArgs(int64(7)).WillReturnRows(sqlmock.NewRows([]string{"lock"}).AddRow(nil))
	mock.ExpectQuery("SELECT\\s+COUNT\\(\\*\\) FILTER").WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "bytes"}).AddRow(0, 0))
	mock.ExpectQuery("INSERT INTO chat_attachments AS a").
		WillReturnRows(sqlmock.NewRows([]string{
			"public_id", "original_name", "kind", "mime_type", "byte_size", "stored_size", "status", "expires_at",
			"page_count", "width", "height", "storage_key", "sha256", "extracted_text", "library_file_id",
			"is_library", "source", "file_type", "created_at", "updated_at", "last_used_at",
		}).AddRow(
			"att_success", "logo.png", "image", "image/png", int64(10), int64(8), "pending", expiresAt,
			nil, 2, 2, "7/att_success.png", strings.Repeat("0", 64), "", int64(0), false, "", "",
			time.Now(), time.Now(), nil,
		))
	mock.ExpectCommit()

	created, err := repo.Create(context.Background(), &service.CreateChatAttachmentInput{
		UserID: 7,
		Attachment: service.ChatAttachment{
			ID: "att_success", Name: "logo.png", Kind: service.ChatAttachmentKindImage,
			Format: service.ChatAttachmentKindImage, MIMEType: "image/png", Size: 10, StoredSize: 8,
			Status: service.ChatAttachmentStatusPending, ExpiresAt: expiresAt,
			Width: 2, Height: 2, StorageKey: "7/att_success.png", Digest: strings.Repeat("0", 64),
		},
	})
	require.NoError(t, err)
	require.Equal(t, "att_success", created.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChatAttachmentCleanupNeverClaimsFreshPendingRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	}()
	repo := NewChatAttachmentRepository(db, nil)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE chat_attachments\\s+SET status='expired'").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("UPDATE chat_attachments\\s+SET status='deleted'.*status='pending'").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("WHERE a.status IN \\('expired','deleted'\\).*library_file_id IS NULL.*storage_key IS NOT NULL").
		WithArgs(100).WillReturnRows(sqlmock.NewRows([]string{
		"public_id", "original_name", "kind", "mime_type", "byte_size", "stored_size", "status", "expires_at",
		"page_count", "width", "height", "storage_key", "sha256", "extracted_text", "library_file_id",
		"is_library", "source", "file_type", "created_at", "updated_at", "last_used_at",
	}))
	mock.ExpectCommit()
	items, err := repo.ClaimCleanup(context.Background(), time.Now(), 100)
	require.NoError(t, err)
	require.Empty(t, items)
	require.NoError(t, mock.ExpectationsWereMet())
}
