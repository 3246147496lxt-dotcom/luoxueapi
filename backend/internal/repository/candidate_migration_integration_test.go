//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"
	"testing/fstest"
	"time"

	migrationfs "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestCandidateMigrationsUpgradeProduction246To255AndReplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	files, err := collectValidatedMigrationFiles(migrationfs.FS)
	require.NoError(t, err)
	const productionBaselineCount = 246
	require.Len(t, files, productionBaselineCount+9, "candidate binary must embed the reviewed 255-file manifest")

	wantTail := []MigrationManifestEntry{
		{Filename: "201_library_files.sql", SHA256: "03f6a53d92e93fbfee37b9e5dd78dc11cae49dd921812253d0425154f1a9c23e"},
		{Filename: "201a_library_alias_unique_index_notx.sql", SHA256: "ba15a71ce63180c21f8addda85351b13171a0e6c22f7427bcb4b8c955499e564"},
		{Filename: "201b_library_alias_constraints.sql", SHA256: "f52ac96a80583b4e7a3c7c5f9923eee5d95a47c4a2b2d9844c864f31abe83833"},
		{Filename: "202_chat_message_activities.sql", SHA256: "e2ee8b4480af916327f132d378eb70b2291c85efba0ced4555452147b56fdb8f"},
		{Filename: "231_add_users_email_alias_dedup_index_notx.sql", SHA256: "fd103466b72b14919fc7a0b02135f019f9fe7a409a434726467c3649551321e4"},
		{Filename: "232_add_users_email_normalized_index_notx.sql", SHA256: "052a61bf4bdc89a5215970059a61096f4eaea5c244b6781f3ec42d6ac8e8bb5d"},
		{Filename: "233_group_profit_control.sql", SHA256: "afd79e417fc16d34da93df95de87abef407f7cd21e16dae037e0aa59c048d52b"},
		{Filename: "234_add_usage_log_upstream_response_model.sql", SHA256: "cad520cbfcf7af7ea9acae92e5bcbe27501fd9e3ad5b02e306f4f97be4410a82"},
		{Filename: "235_add_usage_log_upstream_model_mismatch_index_notx.sql", SHA256: "692f2a75f0c62670b4d68986912bf24eb92f6377ec904d3806ff7d62b0da8355"},
	}
	require.Len(t, files, productionBaselineCount+len(wantTail))
	for index, want := range wantTail {
		got := files[productionBaselineCount+index]
		require.Equal(t, want.Filename, got.name)
		require.Equal(t, want.SHA256, got.checksum)
	}

	productionBaseline := make(fstest.MapFS, productionBaselineCount)
	for _, file := range files[:productionBaselineCount] {
		productionBaseline[file.name] = &fstest.MapFile{Data: []byte(file.content)}
	}

	db := openIsolatedMigrationIntegrationDB(t, "sub2api_candidate_246_to_255")
	require.NoError(t, applyMigrationsFSWithPolicy(
		ctx,
		db,
		productionBaseline,
		migrationRunnerPolicy{},
	))
	requireSchemaMigrationCount(t, ctx, db, productionBaselineCount)

	identity, err := queryDatabaseIdentity(ctx, db)
	require.NoError(t, err)
	require.NoError(t, applyMigrationsFSWithExpectedDatabaseIdentity(
		ctx,
		db,
		migrationfs.FS,
		&identity,
	))
	requireSchemaMigrationCount(t, ctx, db, len(files))
	requireCandidateMigrationRows(t, ctx, db, wantTail)
	requireCandidateLibrarySchema(t, ctx, db)
	requireCandidateChatActivitySchema(t, ctx, db)
	requireCandidateProfitControlSchema(t, ctx, db)
	requireCandidateUpstreamResponseModelSchema(t, ctx, db)

	beforeReplay := schemaMigrationsFingerprint(t, ctx, db)
	require.NoError(t, applyMigrationsFSWithExpectedDatabaseIdentity(
		ctx,
		db,
		migrationfs.FS,
		&identity,
	), "the second migrate-only execution must be idempotent")
	requireSchemaMigrationCount(t, ctx, db, len(files))
	require.Equal(t, beforeReplay, schemaMigrationsFingerprint(t, ctx, db),
		"migrate-only replay must not rewrite migration evidence")

	// The production application revision embeds the first 246 migrations. Its
	// startup runner must accept (and ignore) newer forward-applied rows rather
	// than treating them as a downgrade or checksum mismatch.
	require.NoError(t, applyMigrationsFSWithPolicy(
		ctx,
		db,
		productionBaseline,
		migrationRunnerPolicy{},
	), "the old 246-migration application contract must start on the forward schema")
	requireSchemaMigrationCount(t, ctx, db, len(files))
	require.Equal(t, beforeReplay, schemaMigrationsFingerprint(t, ctx, db))
}

func requireSchemaMigrationCount(t *testing.T, ctx context.Context, db *sql.DB, want int) {
	t.Helper()
	var got int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&got))
	require.Equal(t, want, got)
}

func requireCandidateProfitControlSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'groups'
  AND column_name IN (
    'profit_control_enabled',
    'profit_min_margin',
    'profit_safety_buffer'
  )`).Scan(&count))
	require.Equal(t, 3, count)
}

func requireCandidateUpstreamResponseModelSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	var responseModelType, mismatchType, responseModelNullable, mismatchNullable string
	var responseModelMaxLength sql.NullInt64
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT
  MAX(CASE WHEN column_name = 'upstream_response_model' THEN data_type END),
  MAX(CASE WHEN column_name = 'upstream_model_mismatch' THEN data_type END),
  MAX(CASE WHEN column_name = 'upstream_response_model' THEN is_nullable END),
  MAX(CASE WHEN column_name = 'upstream_model_mismatch' THEN is_nullable END),
  MAX(CASE WHEN column_name = 'upstream_response_model' THEN character_maximum_length END)
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'usage_logs'
  AND column_name IN ('upstream_response_model', 'upstream_model_mismatch')`).Scan(
		&responseModelType,
		&mismatchType,
		&responseModelNullable,
		&mismatchNullable,
		&responseModelMaxLength,
	))
	require.Equal(t, "character varying", responseModelType)
	require.Equal(t, "boolean", mismatchType)
	require.Equal(t, "YES", responseModelNullable)
	require.Equal(t, "YES", mismatchNullable)
	require.Equal(t, sql.NullInt64{Int64: 200, Valid: true}, responseModelMaxLength)

	var valid, ready bool
	var definition string
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT index_state.indisvalid, index_state.indisready,
       pg_get_indexdef(index_state.indexrelid)
FROM pg_class AS index_class
JOIN pg_index AS index_state ON index_state.indexrelid = index_class.oid
JOIN pg_namespace AS index_namespace ON index_namespace.oid = index_class.relnamespace
WHERE index_namespace.nspname = 'public'
  AND index_class.relname = 'idx_usage_logs_upstream_model_mismatch_created_at'`).Scan(
		&valid,
		&ready,
		&definition,
	))
	require.True(t, valid)
	require.True(t, ready)
	require.Contains(t, definition, "(created_at DESC, id DESC)")
	require.Contains(t, definition, "WHERE (upstream_model_mismatch IS TRUE)")
}

func requireCandidateMigrationRows(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
	want []MigrationManifestEntry,
) {
	t.Helper()
	rows, err := db.QueryContext(ctx, `
SELECT filename, checksum
FROM schema_migrations
WHERE filename IN (
  '201_library_files.sql',
  '201a_library_alias_unique_index_notx.sql',
  '201b_library_alias_constraints.sql',
  '202_chat_message_activities.sql',
  '231_add_users_email_alias_dedup_index_notx.sql',
  '232_add_users_email_normalized_index_notx.sql',
  '233_group_profit_control.sql',
  '234_add_usage_log_upstream_response_model.sql',
  '235_add_usage_log_upstream_model_mismatch_index_notx.sql'
)
ORDER BY filename`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	index := 0
	for rows.Next() {
		var got MigrationManifestEntry
		require.NoError(t, rows.Scan(&got.Filename, &got.SHA256))
		require.Less(t, index, len(want))
		require.Equal(t, want[index], got)
		index++
	}
	require.NoError(t, rows.Err())
	require.Equal(t, len(want), index)
}

func requireCandidateLibrarySchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	var unique, valid, ready bool
	var definition string
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT index_state.indisunique, index_state.indisvalid, index_state.indisready,
       pg_get_indexdef(index_state.indexrelid)
FROM pg_class AS index_class
JOIN pg_index AS index_state ON index_state.indexrelid=index_class.oid
JOIN pg_namespace AS index_namespace ON index_namespace.oid=index_class.relnamespace
WHERE index_namespace.nspname='public'
  AND index_class.relname='idx_chat_attachments_library_file_unique'`).Scan(
		&unique, &valid, &ready, &definition,
	))
	require.True(t, unique)
	require.True(t, valid)
	require.True(t, ready)
	require.Contains(t, definition, "library_file_id")
	require.Contains(t, definition, "WHERE (library_file_id IS NOT NULL)")

	rows, err := db.QueryContext(ctx, `
SELECT constraint_state.conname, constraint_state.convalidated,
       pg_get_constraintdef(constraint_state.oid)
FROM pg_constraint AS constraint_state
JOIN pg_class AS constrained_table ON constrained_table.oid=constraint_state.conrelid
JOIN pg_namespace AS constrained_namespace ON constrained_namespace.oid=constrained_table.relnamespace
WHERE constrained_namespace.nspname='public'
  AND constrained_table.relname='chat_attachments'
  AND constraint_state.conname IN (
    'chat_attachments_library_alias_conversation_check',
    'chat_attachments_library_alias_storage_check',
    'chat_attachments_library_file_fkey'
  )
ORDER BY constraint_state.conname`)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	seen := 0
	for rows.Next() {
		var name, constraintDefinition string
		var validated bool
		require.NoError(t, rows.Scan(&name, &validated, &constraintDefinition))
		require.Truef(t, validated, "constraint %s must be validated", name)
		require.NotEmpty(t, constraintDefinition)
		seen++
	}
	require.NoError(t, rows.Err())
	require.Equal(t, 3, seen)
}

func requireCandidateChatActivitySchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	var tableExists, columnExists, functionExists bool
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT to_regclass('public.chat_message_activities') IS NOT NULL,
       EXISTS (
         SELECT 1 FROM information_schema.columns
         WHERE table_schema='public'
           AND table_name='chat_request_attempts'
           AND column_name='stop_requested_at'
       ),
       to_regprocedure('public.enforce_chat_message_activity_assistant()') IS NOT NULL`).Scan(
		&tableExists, &columnExists, &functionExists,
	))
	require.True(t, tableExists)
	require.True(t, columnExists)
	require.True(t, functionExists)

	var triggerDefinition, triggerEnabled string
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT pg_get_triggerdef(trigger_state.oid), trigger_state.tgenabled::TEXT
FROM pg_trigger AS trigger_state
JOIN pg_class AS trigger_table ON trigger_table.oid=trigger_state.tgrelid
JOIN pg_namespace AS trigger_namespace ON trigger_namespace.oid=trigger_table.relnamespace
WHERE trigger_namespace.nspname='public'
  AND trigger_table.relname='chat_message_activities'
  AND trigger_state.tgname='trg_chat_message_activities_assistant'
  AND NOT trigger_state.tgisinternal`).Scan(&triggerDefinition, &triggerEnabled))
	require.Contains(t, triggerDefinition, "BEFORE INSERT OR UPDATE OF message_id")
	require.Contains(t, triggerDefinition, "enforce_chat_message_activity_assistant()")
	require.Equal(t, "O", triggerEnabled)

	for _, indexName := range []string{
		"idx_chat_message_activities_message_order",
		"idx_chat_message_activities_response",
	} {
		var exists bool
		require.NoError(t, db.QueryRowContext(ctx, `
SELECT to_regclass('public.' || $1) IS NOT NULL`, indexName).Scan(&exists))
		require.Truef(t, exists, "index %s must exist", indexName)
	}
}

func schemaMigrationsFingerprint(t *testing.T, ctx context.Context, db *sql.DB) string {
	t.Helper()
	var fingerprint string
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT encode(
  sha256(
    convert_to(
      string_agg(
        filename || chr(31) || checksum || chr(31) ||
        to_char(applied_at AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US'),
        E'\n' ORDER BY filename
      ),
      'UTF8'
    )
  ),
  'hex'
)
FROM schema_migrations`).Scan(&fingerprint))
	require.Len(t, fingerprint, 64)
	return fingerprint
}
