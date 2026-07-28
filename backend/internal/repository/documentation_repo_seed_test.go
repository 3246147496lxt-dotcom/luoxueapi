package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDocumentationSeedRefreshesOnlyUntouchedBundledContent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	content := []byte(`{"schema_version":1,"tutorials":[{"id":"desktop"}]}`)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)INSERT INTO documentation_documents.*draft_version = 1.*published_version = 1.*draft_updated_by IS NULL.*published_by IS NULL`).
		WithArgs(string(content)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(`(?s)INSERT INTO documentation_revisions.*ON CONFLICT \(version\) DO UPDATE.*published_by IS NULL`).
		WithArgs(string(content)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := &documentationRepository{db: db}
	require.NoError(t, repo.EnsureSeed(context.Background(), content))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentationSeedPreservesAdminManagedContent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	content := []byte(`{"schema_version":1,"tutorials":[{"id":"desktop"}]}`)
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)INSERT INTO documentation_documents.*draft_updated_by IS NULL.*published_by IS NULL`).
		WithArgs(string(content)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectCommit()

	repo := &documentationRepository{db: db}
	require.NoError(t, repo.EnsureSeed(context.Background(), content))
	require.NoError(t, mock.ExpectationsWereMet())
}
