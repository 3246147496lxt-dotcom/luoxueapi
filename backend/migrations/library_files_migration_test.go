package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLibraryFilesMigrationCreatesIndependentDurableMetadata(t *testing.T) {
	content, err := FS.ReadFile("201_library_files.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	lower := strings.ToLower(sql)

	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS library_files")
	require.Contains(t, sql, "user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE")
	require.Contains(t, sql, "CONSTRAINT library_files_public_unique UNIQUE (public_id)")
	require.Contains(t, sql, "source IN ('uploaded', 'generated')")
	require.Contains(t, sql, "source_key VARCHAR(512)")
	require.Contains(t, sql, "source_key IS NULL OR source = 'generated'")
	require.Contains(t, sql, "file_type IN ('image', 'pdf', 'document', 'spreadsheet', 'presentation', 'other')")
	require.Contains(t, sql, "status IN ('pending', 'ready', 'deleted')")
	require.Contains(t, sql, "extension ~ '^[a-z0-9]{1,10}$'")
	require.NotContains(t, sql, "extension ~ '^\\.[a-z0-9]{1,10}$'")
	require.NotContains(t, lower, "bytea")
	require.NotContains(t, lower, "base64")
	require.NotContains(t, lower, "extracted_text")

	createTableEnd := strings.Index(sql, "ALTER TABLE chat_attachments")
	require.Positive(t, createTableEnd)
	libraryTable := sql[:createTableEnd]
	require.NotContains(t, libraryTable, "conversation_id")
	require.NotContains(t, libraryTable, "REFERENCES chat_conversations")
}

func TestLibraryFilesMigrationCreatesReusableAliasAndLifecycleIndexes(t *testing.T) {
	content, err := FS.ReadFile("201_library_files.sql")
	require.NoError(t, err)
	sql := strings.Join(strings.Fields(string(content)), " ")
	indexContent, err := FS.ReadFile("201a_library_alias_unique_index_notx.sql")
	require.NoError(t, err)
	indexSQL := strings.Join(strings.Fields(string(indexContent)), " ")
	validationContent, err := FS.ReadFile("201b_library_alias_constraints.sql")
	require.NoError(t, err)
	validationSQL := strings.Join(strings.Fields(string(validationContent)), " ")

	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS library_file_id BIGINT")
	require.Contains(t, sql, "CONSTRAINT library_files_id_user_unique UNIQUE (id, user_id)")
	require.Contains(t, sql, "FOREIGN KEY (library_file_id, user_id) REFERENCES library_files(id, user_id) ON DELETE SET NULL (library_file_id) NOT VALID")
	require.Contains(t, sql, "library_file_id IS NULL OR conversation_id IS NULL ) NOT VALID")
	require.Contains(t, sql, "library_file_id IS NULL OR storage_key IS NULL ) NOT VALID")
	require.NotContains(t, sql, "idx_chat_attachments_library_file_unique")
	require.NotContains(t, sql, "VALIDATE CONSTRAINT")

	require.Contains(t, indexSQL, "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_chat_attachments_library_file_unique")
	require.Contains(t, indexSQL, "WHERE library_file_id IS NOT NULL")
	require.Equal(t, 1, strings.Count(indexSQL, "CREATE UNIQUE INDEX"))

	require.Contains(t, validationSQL, "VALIDATE CONSTRAINT chat_attachments_library_file_fkey")
	require.Contains(t, validationSQL, "VALIDATE CONSTRAINT chat_attachments_library_alias_conversation_check")
	require.Contains(t, validationSQL, "VALIDATE CONSTRAINT chat_attachments_library_alias_storage_check")
	require.Equal(t, 3, strings.Count(validationSQL, "VALIDATE CONSTRAINT"))
	require.Contains(t, sql, "idx_library_files_user_status_updated")
	require.Contains(t, sql, "idx_library_files_user_category_updated")
	require.Contains(t, sql, "idx_library_files_user_type_updated")
	require.Contains(t, sql, "idx_library_files_user_source_updated")
	require.Contains(t, sql, "idx_library_files_usage")
	require.Contains(t, sql, "WHERE status IN ('pending', 'ready')")
	require.Contains(t, sql, "idx_library_files_cleanup")
	require.Contains(t, sql, "WHERE status = 'deleted' AND storage_key IS NOT NULL")
	require.Contains(t, sql, "idx_library_files_pending_cleanup")
	require.Contains(t, sql, "WHERE status = 'pending' AND storage_key IS NOT NULL")
	require.Contains(t, sql, "idx_library_files_source_key_unique")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS batch_image_library_ingests")
	require.Contains(t, sql, "status IN ('pending', 'retrying', 'completed', 'completed_with_errors')")
	require.NotContains(t, sql, "is_library")
	require.NotContains(t, sql, "DROP CONSTRAINT IF EXISTS chat_attachments_kind_check")
}
