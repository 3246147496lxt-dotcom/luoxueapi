package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/opsruntime"
	"github.com/Wei-Shaw/sub2api/migrations"
)

// schemaMigrationsTableDDL 定义迁移记录表的 DDL。
// 该表用于跟踪已应用的迁移文件及其校验和。
// - filename: 迁移文件名，作为主键唯一标识每个迁移
// - checksum: 文件内容的 SHA256 哈希值，用于检测迁移文件是否被篡改
// - applied_at: 迁移应用时间戳
const schemaMigrationsTableDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	filename   TEXT PRIMARY KEY,
	checksum   TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

// schemaMigrationRunnerStateTableDDL stores migration-runner metadata that
// must survive a partially completed first migration run. Keep this separate
// from schema_migrations so its rows can never be mistaken for SQL files.
const schemaMigrationRunnerStateTableDDL = `
CREATE TABLE IF NOT EXISTS schema_migration_runner_state (
	state_key   TEXT PRIMARY KEY,
	state_value TEXT NOT NULL,
	recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

const atlasSchemaRevisionsTableDDL = `
CREATE TABLE IF NOT EXISTS atlas_schema_revisions (
	version TEXT PRIMARY KEY,
	description TEXT NOT NULL,
	type INTEGER NOT NULL,
	applied INTEGER NOT NULL DEFAULT 0,
	total INTEGER NOT NULL DEFAULT 0,
	executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	execution_time BIGINT NOT NULL DEFAULT 0,
	error TEXT NULL,
	error_stmt TEXT NULL,
	hash TEXT NOT NULL DEFAULT '',
	partial_hashes TEXT[] NULL,
	operator_version TEXT NULL
);
`

// migrationsAdvisoryLockID 是用于序列化迁移操作的 PostgreSQL Advisory Lock ID。
// 在多实例部署场景下，该锁确保同一时间只有一个实例执行迁移。
// 任何稳定的 int64 值都可以，只要不与同一数据库中的其他锁冲突即可。
const migrationsAdvisoryLockID int64 = 694208311321144027
const migrationsLockRetryInterval = 500 * time.Millisecond
const migrationsUnlockTimeout = 5 * time.Second
const nonTransactionalMigrationSuffix = "_notx.sql"
const paymentOrdersOutTradeNoUniqueMigration = "120_enforce_payment_orders_out_trade_no_unique_notx.sql"
const paymentOrdersOutTradeNoUniqueIndex = "paymentorder_out_trade_no_unique"
const schedulerOutboxPendingDedupKeyMigration = "153_scheduler_outbox_pending_dedup_key_index_notx.sql"
const schedulerOutboxPendingDedupKeyIndex = "idx_scheduler_outbox_pending_dedup_key"
const latestAPIKeyIPIndexMigration = "174_add_usage_logs_api_key_latest_ip_index_notx.sql"
const latestAPIKeyIPIndex = "idx_usage_logs_api_key_latest_ip"
const apiKeyPurposeWebChatIndexesMigration = "183a_api_key_purpose_web_chat_indexes_notx.sql"
const apiKeyPurposeIndex = "idx_api_keys_purpose"
const apiKeyWebChatActiveUserGroupIndex = "idx_api_keys_web_chat_active_user_group"
const libraryAliasUniqueIndexMigration = "201a_library_alias_unique_index_notx.sql"
const libraryAliasUniqueIndex = "idx_chat_attachments_library_file_unique"

// These indexes are created by the local email-alias migrations.  Keep the
// migration-to-index mapping explicit: an interrupted CREATE INDEX
// CONCURRENTLY leaves an invalid relation behind, and IF NOT EXISTS would
// otherwise skip it on the next run and incorrectly mark the migration as
// applied.  Do not derive names from arbitrary migration SQL before dropping
// anything; only these reviewed index names may be cleaned up automatically.
const usersEmailAliasDedupIndexMigration = "231_add_users_email_alias_dedup_index_notx.sql"
const usersEmailAliasDedupIndex = "idx_users_email_dot_stripped"
const usersEmailNormalizedIndexMigration = "232_add_users_email_normalized_index_notx.sql"
const usersEmailNormalizedIndex = "idx_users_email_normalized"
const upstreamModelMismatchIndexMigration = "235_add_usage_log_upstream_model_mismatch_index_notx.sql"
const upstreamModelMismatchIndex = "idx_usage_logs_upstream_model_mismatch_created_at"
const subscriptionAnchoredMonthlyQuotaMigration = "195_subscription_anchored_monthly_quota.sql"
const schemaMigrationOriginStateKey = "schema_origin"

var maintenanceOnlyMigrations = []string{
	subscriptionAnchoredMonthlyQuotaMigration,
}

type schemaMigrationOrigin string

const (
	schemaMigrationOriginFresh  schemaMigrationOrigin = "fresh"
	schemaMigrationOriginLegacy schemaMigrationOrigin = "legacy"
)

var canonicalMigrationFilename = regexp.MustCompile(`^[0-9]{3}[a-z]?_[a-z0-9][a-z0-9_]*\.sql$`)

type migrationQueryExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type validatedMigrationFile struct {
	name     string
	content  string
	checksum string
}

// expectedConcurrentIndex describes the schema contract for a named index
// created by a non-transactional migration.  CREATE INDEX IF NOT EXISTS only
// checks the relation name; it does not verify the table or definition.  Keep
// these expectations explicit and reviewed rather than deriving them from
// arbitrary migration SQL before issuing any cleanup.
type expectedConcurrentIndex struct {
	migration     string
	indexName     string
	tableName     string
	accessMethod  string
	operatorClass string
	unique        bool
	keyAttributes int
	attributes    int
	keyDefinition string
	expression    string
	predicate     string
}

var (
	usersEmailAliasDedupIndexSpec = expectedConcurrentIndex{
		migration:     usersEmailAliasDedupIndexMigration,
		indexName:     usersEmailAliasDedupIndex,
		tableName:     "users",
		accessMethod:  "btree",
		operatorClass: "text_pattern_ops",
		unique:        false,
		keyAttributes: 1,
		attributes:    1,
		expression:    "REPLACE(LOWER(TRIM(email)), '.', '')",
		predicate:     "deleted_at IS NULL",
	}
	usersEmailNormalizedIndexSpec = expectedConcurrentIndex{
		migration:     usersEmailNormalizedIndexMigration,
		indexName:     usersEmailNormalizedIndex,
		tableName:     "users",
		accessMethod:  "btree",
		operatorClass: "text_pattern_ops",
		unique:        false,
		keyAttributes: 1,
		attributes:    1,
		expression:    "RTRIM(LOWER(TRIM(email)), '.')",
		predicate:     "deleted_at IS NULL",
	}
	upstreamModelMismatchIndexSpec = expectedConcurrentIndex{
		migration:     upstreamModelMismatchIndexMigration,
		indexName:     upstreamModelMismatchIndex,
		tableName:     "usage_logs",
		accessMethod:  "btree",
		operatorClass: "",
		unique:        false,
		keyAttributes: 2,
		attributes:    2,
		keyDefinition: "created_at DESC, id DESC",
		expression:    "",
		predicate:     "upstream_model_mismatch IS TRUE",
	}
)

type migrationRunnerPolicy struct {
	expectedIdentity     *ConfiguredDatabaseIdentity
	allowMaintenanceOnly bool
}

const EmbeddedMigrationManifestContract = "sub2api-migration-manifest/v1"

// MigrationManifestEntry identifies one embedded SQL migration by the exact
// checksum used by the migration runner.
type MigrationManifestEntry struct {
	Filename string `json:"filename"`
	SHA256   string `json:"sha256"`
}

// MigrationManifest is a deterministic, database-free projection of the SQL
// migrations embedded in this binary. Release tooling uses it to bind a
// reviewed checkout to the immutable candidate image before maintenance begins.
type MigrationManifest struct {
	Contract  string                   `json:"contract"`
	SetSHA256 string                   `json:"set_sha256"`
	Entries   []MigrationManifestEntry `json:"migrations"`
}

type migrationChecksumCompatibilityRule struct {
	fileChecksum       string
	acceptedDBChecksum map[string]struct{}
	acceptedChecksums  map[string]struct{}
}

// migrationChecksumCompatibilityRules 仅用于兼容历史上误修改过的迁移文件 checksum。
// 规则必须同时匹配「迁移名 + 数据库 checksum + 当前文件 checksum」且两者都落在该迁移的已知版本集合内才会放行，
// 避免放宽全局校验，也允许将误改的历史 migration 回滚为已发布版本而不要求人工修 checksum。
var migrationChecksumCompatibilityRules = map[string]migrationChecksumCompatibilityRule{
	"054_drop_legacy_cache_columns.sql":                       newMigrationChecksumCompatibilityRule("82de761156e03876653e7a6a4eee883cd927847036f779b0b9f34c42a8af7a7d", "182c193f3359946cf094090cd9e57d5c3fd9abaffbc1e8fc378646b8a6fa12b4"),
	"061_add_usage_log_request_type.sql":                      newMigrationChecksumCompatibilityRule("66207e7aa5dd0429c2e2c0fabdaf79783ff157fa0af2e81adff2ee03790ec65c", "08a248652cbab7cfde147fc6ef8cda464f2477674e20b718312faa252e0481c0", "222b4a09c797c22e5922b6b172327c824f5463aaa8760e4f621bc5c22e2be0f3"),
	"109_auth_identity_compat_backfill.sql":                   newMigrationChecksumCompatibilityRule("0580b4602d85435edf9aca1633db580bb3932f26517f75134106f80275ec2ace", "551e498aa5616d2d91096e9d72cf9fb36e418ee22eacc557f8811cadbc9e20ee"),
	"110_pending_auth_and_provider_default_grants.sql":        newMigrationChecksumCompatibilityRule("32cf87ee787b1bb36b5c691367c96eee37518fa3eed6f3322cf68795e3745279", "e3d1f433be2b564cfbdc549adf98fce13c5c7b363ebc20fd05b765d0563b0925"),
	"112_add_payment_order_provider_key_snapshot.sql":         newMigrationChecksumCompatibilityRule("b75f8f56d39455682787696a3d92ad25b055444ca328fb7fca9a460a15d68d99", "ffd3e8a2c9295fa9cbefefd629a78268877e5b51bc970a82d9b3f46ec4ebd15e"),
	"115_auth_identity_legacy_external_backfill.sql":          newMigrationChecksumCompatibilityRule("022aadd97bb53e755f0cf7a3a957e0cb1a1353b0c39ec4de3234acd2871fd04f", "4cf39e508be9fd1a5aa41610cbbebeb80385c9adda45bf78a706de9db4f1385f"),
	"116_auth_identity_legacy_external_safety_reports.sql":    newMigrationChecksumCompatibilityRule("07edb09fa8d04ffb172b0621e3c22f4d1757d20a24ae267b3b36b087ab72d488", "f7757bd929ac67ffb08ce69fa4cf20fad39dbff9d5a5085fb2adabb7607e5877"),
	"118_wechat_dual_mode_and_auth_source_defaults.sql":       newMigrationChecksumCompatibilityRule("b54194d7a3e4fbf710e0a3590d22a2fe7966804c487052a356e0b55f53ef96b0", "e0cdf835d6c688d64100f483d31bc02ac9ebad414bf1837af239a84bf75b8227", "a38243ca0a72c3a01c0a92b7986423054d6133c0399441f853b99802852720fb"),
	"119_enforce_payment_orders_out_trade_no_unique.sql":      newMigrationChecksumCompatibilityRule("0bbe809ae48a9d811dabda1ba1c74955bd71c4a9cc610f9128816818dfa6c11e", "ebd2c67cce0116393fb4f1b5d5116a67c6aceb73820dfb5133d1ff6f36d72d34"),
	"120_enforce_payment_orders_out_trade_no_unique_notx.sql": newMigrationChecksumCompatibilityRule("34aadc0db59a4e390f92a12b73bd74642d9724f33124f73638ae00089ea5e074", "e77921f79d539bc24575cb9c16cbe566d2b23ce816190343d0a7568f6a3fcf61", "707431450603e70a43ce9fbd61e0c12fa67da4875158ccefabacea069587ab22", "04b082b5a239c525154fe9185d324ee2b05ff90da9297e10dba19f9be79aa59a"),
	"123_fix_legacy_auth_source_grant_on_signup_defaults.sql": newMigrationChecksumCompatibilityRule("2ce43c2cd89e9f9e1febd34a407ed9e84d177386c5544b6f02c1f58a21129f57", "6cd33422f215dcd1f486ab6f35c0ea5805d9ca69bb25906d94bc649156657145"),
	"159_batch_image_foundation.sql":                          newMigrationChecksumCompatibilityRule("d902b70982025ec519749faf058aab7631e82c3f48167b9a4ae4db718eb72cce", "82da85b5d98e67a0507647b873a40373e84538e4adafdeed6767c0ac8b6570b2"),
	"161_batch_image_pricing_snapshot.sql":                    newMigrationChecksumCompatibilityRule("4012af3e43636cb6af22e0176d59d1fcc70615c0f310194329461ae462c4fbd6", "96d915c9b7a6941ae99039e0ff3f1a61481eb9bddd933d11c6fadb2274554e87"),
}

// ApplyMigrations 将嵌入的 SQL 迁移文件应用到指定的数据库。
//
// 该函数可以在每次应用启动时安全调用：
// - 已应用的迁移会被自动跳过（通过校验 filename 判断）
// - 如果迁移文件内容被修改（checksum 不匹配），会返回错误
// - 使用 PostgreSQL Advisory Lock 确保多实例并发安全
//
// 参数：
//   - ctx: 上下文，用于超时控制和取消
//   - db: 数据库连接
//
// 返回：
//   - error: 迁移过程中的任何错误
func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("nil sql db")
	}
	return applyMigrationsFSWithPolicy(ctx, db, migrations.FS, migrationRunnerPolicy{})
}

// rejectPendingMaintenanceOnlyMigrations keeps ordinary application startup
// from applying migrations that require an externally verified maintenance
// window. Only a database with no existing public tables may initialize the
// complete schema. Once application tables or schema_migrations exist, even a
// historically fresh-origin database must use the explicit migration entrypoint.
func rejectPendingMaintenanceOnlyMigrations(ctx context.Context, db migrationQueryExecer) error {
	if db == nil {
		return errors.New("check maintenance-only migrations: nil database session")
	}

	hasSchemaMigrations, err := tableExists(ctx, db, "schema_migrations")
	if err != nil {
		return fmt.Errorf("check maintenance-only migrations table: %w", err)
	}
	if !hasSchemaMigrations {
		var hasExistingTables bool
		if err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_catalog.pg_class c
				JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = 'public'
				  AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
				  AND c.relname <> 'schema_migration_runner_state'
			)
		`).Scan(&hasExistingTables); err != nil {
			return fmt.Errorf("check existing public tables before migration bootstrap: %w", err)
		}
		if !hasExistingTables {
			return nil
		}
		return fmt.Errorf(
			"maintenance-only migration %s may be pending on a database with existing public tables but no schema_migrations metadata; refusing automatic application startup migration; reconcile the migration state and complete the reviewed maintenance runbook with --migrate-only",
			subscriptionAnchoredMonthlyQuotaMigration,
		)
	}

	for _, name := range maintenanceOnlyMigrations {
		var applied bool
		if err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM schema_migrations
				WHERE filename = $1
			)
		`, name).Scan(&applied); err != nil {
			return fmt.Errorf("check maintenance-only migration %s: %w", name, err)
		}
		if !applied {
			return fmt.Errorf(
				"maintenance-only migration %s is pending on an existing database; refusing automatic application startup migration; complete the reviewed maintenance runbook with --migrate-only",
				name,
			)
		}
	}

	return nil
}

// EmbeddedMigrationManifest returns the validated embedded migration set
// without loading application configuration or opening a database connection.
func EmbeddedMigrationManifest() (MigrationManifest, error) {
	files, err := collectValidatedMigrationFiles(migrations.FS)
	if err != nil {
		return MigrationManifest{}, fmt.Errorf("validate embedded migrations: %w", err)
	}

	entries := make([]MigrationManifestEntry, 0, len(files))
	setDigest := sha256.New()
	for _, file := range files {
		entries = append(entries, MigrationManifestEntry{
			Filename: file.name,
			SHA256:   file.checksum,
		})
		_, _ = fmt.Fprintf(setDigest, "%s\x00%s\n", file.name, file.checksum)
	}

	return MigrationManifest{
		Contract:  EmbeddedMigrationManifestContract,
		SetSHA256: hex.EncodeToString(setDigest.Sum(nil)),
		Entries:   entries,
	}, nil
}

// applyMigrationsFS 是迁移执行的核心实现。
// 它从指定的文件系统读取 SQL 迁移文件并按顺序应用。
//
// 迁移执行流程：
//  1. 在任何数据库 SQL 前校验并读取所有迁移文件
//  2. 固定一个 PostgreSQL session 并获取 Advisory Lock
//  3. 持久化首次采用 runner 时的 fresh/legacy 状态并确保 schema_migrations 表存在
//  4. 对于每个迁移文件：
//     - 计算文件内容的 SHA256 校验和
//     - 检查该迁移是否已应用（通过 filename 查询）
//     - 如果已应用，验证校验和是否匹配
//     - 如果未应用，在事务中执行迁移并记录
//  5. 全部成功后仅为启动前 legacy schema 对齐 Atlas baseline
//  6. 在同一 session 释放 Advisory Lock
//
// 参数：
//   - ctx: 上下文
//   - db: 数据库连接
//   - fsys: 包含迁移文件的文件系统（通常是 embed.FS）
func applyMigrationsFS(ctx context.Context, db *sql.DB, fsys fs.FS) (retErr error) {
	return applyMigrationsFSWithPolicy(ctx, db, fsys, migrationRunnerPolicy{
		allowMaintenanceOnly: true,
	})
}

func applyMigrationsFSWithExpectedDatabaseIdentity(
	ctx context.Context,
	db *sql.DB,
	fsys fs.FS,
	expectedIdentity *ConfiguredDatabaseIdentity,
) (retErr error) {
	if expectedIdentity == nil {
		return errors.New("maintenance migrations require an expected database identity")
	}
	return applyMigrationsFSWithPolicy(ctx, db, fsys, migrationRunnerPolicy{
		expectedIdentity:     expectedIdentity,
		allowMaintenanceOnly: true,
	})
}

func applyMigrationsFSWithPolicy(
	ctx context.Context,
	db *sql.DB,
	fsys fs.FS,
	policy migrationRunnerPolicy,
) (retErr error) {
	if db == nil {
		return errors.New("nil sql db")
	}

	// 文件清单必须在任何数据库 SQL 之前完成严格校验。除了阻止拼写错误的迁移，
	// 这也会显式拒绝 macOS AppleDouble 文件（例如 ._001_init.sql），避免它们被
	// 当成一条真实迁移记录进 schema_migrations。
	files, err := collectValidatedMigrationFiles(fsys)
	if err != nil {
		return fmt.Errorf("validate migrations: %w", err)
	}
	if policy.allowMaintenanceOnly && policy.expectedIdentity == nil &&
		containsMaintenanceOnlyMigration(files) {
		return errors.New("maintenance-only migrations require an expected database identity")
	}

	// PostgreSQL advisory lock 是 session-scoped。整个锁生命周期及所有迁移 SQL
	// 必须固定在同一个物理连接上，不能经 *sql.DB 重新从池中选择连接。
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire migrations connection: %w", err)
	}
	locked := false
	discardConn := false
	defer func() {
		if locked {
			// 原 ctx 可能已经取消；给同一 session 一个独立、有限的解锁窗口。
			unlockCtx, cancel := context.WithTimeout(context.Background(), migrationsUnlockTimeout)
			unlockErr := pgAdvisoryUnlock(unlockCtx, conn)
			cancel()
			if unlockErr != nil {
				discardConn = true
				retErr = errors.Join(retErr, unlockErr)
			}
		}

		if discardConn {
			// 解锁结果不可信时绝不能让该物理 session 回到连接池；它可能仍持有锁。
			discardSQLConn(conn)
			_ = conn.Close()
			return
		}
		if closeErr := conn.Close(); closeErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("release migrations connection: %w", closeErr))
		}
	}()

	// A maintenance migration must verify the operator-approved identity on
	// this exact physical session before taking a lock or issuing schema SQL.
	// This closes the DNS/load-balancer gap between a separate identity probe
	// and the connection that actually performs the migration.
	if policy.expectedIdentity != nil {
		if err := verifyExpectedDatabaseIdentity(ctx, conn, *policy.expectedIdentity); err != nil {
			return fmt.Errorf("verify migration database identity: %w", err)
		}
	}

	// 获取分布式锁，确保多实例部署时只有一个实例执行迁移。
	// 锁查询本身失败时结果可能不确定，因此同样丢弃该 session。
	if err := pgAdvisoryLock(ctx, conn); err != nil {
		discardConn = true
		return err
	}
	locked = true

	// The ordinary-startup maintenance gate must run on the exact session that
	// holds the migration lock. A pooled precheck can observe a different
	// backend after DNS or proxy failover and must never authorize schema writes.
	if !policy.allowMaintenanceOnly {
		if err := rejectPendingMaintenanceOnlyMigrations(ctx, conn); err != nil {
			return err
		}
	}

	// origin 必须在创建 schema_migrations 之前持久化。这样即使新库首次迁移失败，
	// 后续重试或等待锁的第二实例也仍会读到 fresh，不会因迁移记录表已创建而误判
	// 为 legacy 并写入 Atlas baseline。
	schemaOrigin, err := ensureSchemaMigrationOrigin(ctx, conn)
	if err != nil {
		return err
	}

	// 创建迁移记录表（如果不存在）。
	// 该表记录所有已应用的迁移及其校验和。
	if _, err := conn.ExecContext(ctx, schemaMigrationsTableDDL); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, file := range files {
		name := file.name
		content := file.content
		if content == "" {
			continue // 跳过空文件
		}

		// 检查该迁移是否已经应用
		var existing string
		rowErr := conn.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename = $1", name).Scan(&existing)
		if rowErr == nil {
			// 迁移已应用，验证校验和是否匹配
			if existing != file.checksum {
				// 兼容特定历史误改场景（仅白名单规则），其余仍保持严格不可变约束。
				if isMigrationChecksumCompatible(name, existing, file.checksum) {
					continue
				}
				// 校验和不匹配意味着迁移文件在应用后被修改，这是危险的。
				// 正确的做法是创建新的迁移文件来进行变更。
				return fmt.Errorf(
					"migration %s checksum mismatch (db=%s file=%s)\n"+
						"This means the migration file was modified after being applied to the database.\n"+
						"Solutions:\n"+
						"  1. Revert to original: git log --oneline -- migrations/%s && git checkout <commit> -- migrations/%s\n"+
						"  2. For new changes, create a new migration file instead of modifying existing ones\n"+
						"Note: Modifying applied migrations breaks the immutability principle and can cause inconsistencies across environments",
					name, existing, file.checksum, name, name,
				)
			}
			continue // 迁移已应用且校验和匹配，跳过
		}
		if !errors.Is(rowErr, sql.ErrNoRows) {
			return fmt.Errorf("check migration %s: %w", name, rowErr)
		}

		nonTx, err := validateMigrationExecutionMode(name, content)
		if err != nil {
			return fmt.Errorf("validate migration %s: %w", name, err)
		}

		if nonTx {
			if err := prepareNonTransactionalMigration(ctx, conn, name); err != nil {
				return fmt.Errorf("prepare migration %s: %w", name, err)
			}

			// *_notx.sql：用于 CREATE/DROP INDEX CONCURRENTLY 场景，必须非事务执行。
			// 逐条语句执行，避免将多条 CONCURRENTLY 语句放入同一个隐式事务块。
			statements := splitSQLStatements(content)
			for i, stmt := range statements {
				trimmed := strings.TrimSpace(stmt)
				if trimmed == "" {
					continue
				}
				if stripSQLLineComment(trimmed) == "" {
					continue
				}
				if _, err := conn.ExecContext(ctx, trimmed); err != nil {
					return fmt.Errorf("apply migration %s (non-tx statement %d): %w", name, i+1, err)
				}
			}
			if _, err := conn.ExecContext(ctx, "INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)", name, file.checksum); err != nil {
				return fmt.Errorf("record migration %s (non-tx): %w", name, err)
			}
			continue
		}

		// 默认迁移在事务中执行，确保原子性：要么完全成功，要么完全回滚。
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}

		// 执行迁移 SQL
		if _, err := tx.ExecContext(ctx, content); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		// 记录迁移已完成，保存文件名和校验和
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)", name, file.checksum); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}

	// 仅对首次采用 runner 时已经存在 schema_migrations 的 legacy 数据库补 Atlas
	// baseline，且必须在全部 SQL migrations 成功后执行，避免记录未达到的版本。
	if schemaOrigin == schemaMigrationOriginLegacy {
		if err := ensureAtlasBaselineAlignedWithFiles(ctx, conn, files); err != nil {
			return err
		}
	}

	return nil
}

func containsMaintenanceOnlyMigration(files []validatedMigrationFile) bool {
	for _, file := range files {
		for _, maintenanceName := range maintenanceOnlyMigrations {
			if file.name == maintenanceName {
				return true
			}
		}
	}
	return false
}

func ensureSchemaMigrationOrigin(ctx context.Context, db migrationQueryExecer) (schemaMigrationOrigin, error) {
	hasSchemaMigrations, err := tableExists(ctx, db, "schema_migrations")
	if err != nil {
		return "", fmt.Errorf("check schema_migrations before bootstrap: %w", err)
	}

	candidate := schemaMigrationOriginFresh
	if hasSchemaMigrations {
		candidate = schemaMigrationOriginLegacy
	}

	if _, err := db.ExecContext(ctx, schemaMigrationRunnerStateTableDDL); err != nil {
		return "", fmt.Errorf("create schema_migration_runner_state: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO schema_migration_runner_state (state_key, state_value)
		VALUES ($1, $2)
		ON CONFLICT (state_key) DO NOTHING
	`, schemaMigrationOriginStateKey, candidate); err != nil {
		return "", fmt.Errorf("record schema migration origin: %w", err)
	}

	return readSchemaMigrationOrigin(ctx, db)
}

// readSchemaMigrationOrigin is deliberately read-only. Missing or invalid state
// must fail closed; callers must never reconstruct origin from schema_migrations.
func readSchemaMigrationOrigin(ctx context.Context, db migrationQueryExecer) (schemaMigrationOrigin, error) {
	var persisted string
	err := db.QueryRowContext(ctx, `
		SELECT state_value
		FROM schema_migration_runner_state
		WHERE state_key = $1
	`, schemaMigrationOriginStateKey).Scan(&persisted)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("schema migration origin is missing")
	}
	if err != nil {
		return "", fmt.Errorf("read schema migration origin: %w", err)
	}

	origin := schemaMigrationOrigin(persisted)
	switch origin {
	case schemaMigrationOriginFresh, schemaMigrationOriginLegacy:
		return origin, nil
	default:
		return "", fmt.Errorf("invalid schema migration origin %q", persisted)
	}
}

func collectValidatedMigrationFiles(fsys fs.FS) ([]validatedMigrationFile, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("list migration directory: %w", err)
	}

	files := make([]validatedMigrationFile, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !strings.EqualFold(path.Ext(name), ".sql") {
			continue
		}
		if !canonicalMigrationFilename.MatchString(name) {
			return nil, fmt.Errorf(
				"invalid migration filename %q: expected NNN[a]_description.sql (lowercase letters, digits, and underscores only); remove metadata files such as ._*.sql",
				name,
			)
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("inspect migration %s: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("invalid migration %q: expected a regular file", name)
		}

		contentBytes, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}
		content := strings.TrimSpace(string(contentBytes))
		sum := sha256.Sum256([]byte(content))
		files = append(files, validatedMigrationFile{
			name:     name,
			content:  content,
			checksum: hex.EncodeToString(sum[:]),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].name < files[j].name
	})
	return files, nil
}

func prepareNonTransactionalMigration(ctx context.Context, db migrationQueryExecer, name string) error {
	switch name {
	case paymentOrdersOutTradeNoUniqueMigration:
		return preparePaymentOrdersOutTradeNoUniqueMigration(ctx, db)
	case schedulerOutboxPendingDedupKeyMigration:
		return dropInvalidIndexIfPresent(ctx, db, schedulerOutboxPendingDedupKeyIndex)
	case latestAPIKeyIPIndexMigration:
		return dropInvalidIndexIfPresent(ctx, db, latestAPIKeyIPIndex)
	case apiKeyPurposeWebChatIndexesMigration:
		for _, indexName := range []string{apiKeyPurposeIndex, apiKeyWebChatActiveUserGroupIndex} {
			if err := dropInvalidIndexIfPresent(ctx, db, indexName); err != nil {
				return err
			}
		}
		return nil
	case libraryAliasUniqueIndexMigration:
		return dropInvalidIndexIfPresent(ctx, db, libraryAliasUniqueIndex)
	case usersEmailAliasDedupIndexMigration:
		if err := validateExpectedConcurrentIndex(ctx, db, usersEmailAliasDedupIndexSpec); err != nil {
			return err
		}
		return dropInvalidIndexIfPresent(ctx, db, usersEmailAliasDedupIndex)
	case usersEmailNormalizedIndexMigration:
		if err := validateExpectedConcurrentIndex(ctx, db, usersEmailNormalizedIndexSpec); err != nil {
			return err
		}
		return dropInvalidIndexIfPresent(ctx, db, usersEmailNormalizedIndex)
	case upstreamModelMismatchIndexMigration:
		if err := validateExpectedConcurrentIndex(ctx, db, upstreamModelMismatchIndexSpec); err != nil {
			return err
		}
		if err := validateUpstreamResponseModelColumns(ctx, db); err != nil {
			return err
		}
		return dropInvalidIndexIfPresent(ctx, db, upstreamModelMismatchIndex)
	default:
		return nil
	}
}

type observedConcurrentIndex struct {
	tableSchema   string
	tableName     string
	valid         bool
	ready         bool
	unique        bool
	accessMethod  string
	keyAttributes int
	attributes    int
	expression    sql.NullString
	predicate     sql.NullString
	definition    string
}

// validateExpectedConcurrentIndex prevents CREATE INDEX IF NOT EXISTS from
// silently accepting a valid relation with the same name but a different
// table/definition. Only structurally matching invalid concurrent-index
// leftovers are allowed through to dropInvalidIndexIfPresent; every mismatch
// fails closed and requires explicit operator remediation.
func validateExpectedConcurrentIndex(ctx context.Context, db migrationQueryExecer, expected expectedConcurrentIndex) error {
	var observed observedConcurrentIndex
	err := db.QueryRowContext(ctx, `
		SELECT table_namespace.nspname,
		       table_class.relname,
		       index_state.indisvalid,
		       index_state.indisready,
		       index_state.indisunique,
		       access_method.amname,
		       index_state.indnkeyatts,
		       index_state.indnatts,
		       pg_get_expr(index_state.indexprs, index_state.indrelid),
		       pg_get_expr(index_state.indpred, index_state.indrelid),
		       pg_get_indexdef(index_state.indexrelid)
		FROM pg_class AS index_class
		JOIN pg_namespace AS index_namespace ON index_namespace.oid = index_class.relnamespace
		JOIN pg_index AS index_state ON index_state.indexrelid = index_class.oid
		JOIN pg_class AS table_class ON table_class.oid = index_state.indrelid
		JOIN pg_namespace AS table_namespace ON table_namespace.oid = table_class.relnamespace
		JOIN pg_am AS access_method ON access_method.oid = index_class.relam
		WHERE index_namespace.nspname = 'public'
		  AND index_class.relname = $1
	`, expected.indexName).Scan(
		&observed.tableSchema,
		&observed.tableName,
		&observed.valid,
		&observed.ready,
		&observed.unique,
		&observed.accessMethod,
		&observed.keyAttributes,
		&observed.attributes,
		&observed.expression,
		&observed.predicate,
		&observed.definition,
	)
	if errors.Is(err, sql.ErrNoRows) {
		// The metadata query intentionally joins pg_index. Distinguish a genuinely
		// absent name from a table/view/sequence using the same relation name;
		// CREATE INDEX IF NOT EXISTS would otherwise skip and the runner could mark
		// a migration applied without creating the required index.
		var relationKind string
		relationErr := db.QueryRowContext(ctx, `
			SELECT index_class.relkind::text
			FROM pg_class AS index_class
			JOIN pg_namespace AS index_namespace ON index_namespace.oid = index_class.relnamespace
			WHERE index_namespace.nspname = 'public'
			  AND index_class.relname = $1
		`, expected.indexName).Scan(&relationKind)
		if errors.Is(relationErr, sql.ErrNoRows) {
			return nil
		}
		if relationErr != nil {
			return fmt.Errorf("inspect existing relation %s for migration %s: %w", expected.indexName, expected.migration, relationErr)
		}
		return fmt.Errorf(
			"existing public relation %s has relkind %q, not an index; refusing migration %s because CREATE INDEX IF NOT EXISTS would not create the required index",
			expected.indexName,
			relationKind,
			expected.migration,
		)
	}
	if err != nil {
		return fmt.Errorf("inspect existing index %s for migration %s: %w", expected.indexName, expected.migration, err)
	}

	if observed.tableSchema != "public" || observed.tableName != expected.tableName {
		return incompatibleConcurrentIndexError(expected, "table", observed.tableSchema+"."+observed.tableName, "public."+expected.tableName)
	}
	if observed.unique != expected.unique {
		return incompatibleConcurrentIndexError(expected, "uniqueness", fmt.Sprintf("%t", observed.unique), fmt.Sprintf("%t", expected.unique))
	}
	if !strings.EqualFold(observed.accessMethod, expected.accessMethod) {
		return incompatibleConcurrentIndexError(expected, "access method", observed.accessMethod, expected.accessMethod)
	}
	if observed.keyAttributes != expected.keyAttributes || observed.attributes != expected.attributes {
		return incompatibleConcurrentIndexError(
			expected,
			"attribute count",
			fmt.Sprintf("key=%d total=%d", observed.keyAttributes, observed.attributes),
			fmt.Sprintf("key=%d total=%d", expected.keyAttributes, expected.attributes),
		)
	}

	observedExpression := ""
	if observed.expression.Valid {
		observedExpression = canonicalizeIndexSQL(observed.expression.String)
	}
	if observedExpression != canonicalizeIndexSQL(expected.expression) {
		return incompatibleConcurrentIndexError(expected, "expression", observedExpression, canonicalizeIndexSQL(expected.expression))
	}
	observedPredicate := ""
	if observed.predicate.Valid {
		observedPredicate = canonicalizeIndexSQL(observed.predicate.String)
	}
	if observedPredicate != canonicalizeIndexSQL(expected.predicate) {
		return incompatibleConcurrentIndexError(expected, "predicate", observedPredicate, canonicalizeIndexSQL(expected.predicate))
	}

	// pg_get_expr does not expose operator classes.  Check the full generated
	// definition for the reviewed class as well, so a default btree index with
	// the same expression cannot silently satisfy the migration contract.
	definition := canonicalizeIndexSQL(observed.definition)
	if expected.keyDefinition != "" {
		keyDefinition := "using" + canonicalizeIndexSQL(expected.accessMethod) + "(" + canonicalizeIndexSQL(expected.keyDefinition) + ")"
		if !strings.Contains(definition, keyDefinition) {
			return incompatibleConcurrentIndexError(expected, "key definition", definition, keyDefinition)
		}
	}
	if expected.operatorClass != "" && !strings.Contains(definition, canonicalizeIndexSQL(expected.operatorClass)) {
		return incompatibleConcurrentIndexError(expected, "operator class", definition, expected.operatorClass)
	}

	if !observed.valid {
		// An interrupted CREATE INDEX CONCURRENTLY leaves an invalid relation.
		// It is safe to remove automatically only after every structural property
		// above has matched the reviewed migration contract.
		return nil
	}
	if !observed.ready {
		return fmt.Errorf(
			"existing index %s is valid but not ready for migration %s; refusing to apply until an operator repairs or removes the index",
			expected.indexName,
			expected.migration,
		)
	}
	return nil
}

func validateUpstreamResponseModelColumns(ctx context.Context, db migrationQueryExecer) error {
	type expectedColumn struct {
		name      string
		dataType  string
		maxLength sql.NullInt64
	}
	for _, expected := range []expectedColumn{
		{name: "upstream_response_model", dataType: "character varying", maxLength: sql.NullInt64{Int64: 200, Valid: true}},
		{name: "upstream_model_mismatch", dataType: "boolean"},
	} {
		var dataType, nullable string
		var maxLength sql.NullInt64
		err := db.QueryRowContext(ctx, `
			SELECT data_type, character_maximum_length, is_nullable
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'usage_logs'
			  AND column_name = $1
		`, expected.name).Scan(&dataType, &maxLength, &nullable)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("required column usage_logs.%s is missing after migration 234; refusing migration %s", expected.name, upstreamModelMismatchIndexMigration)
		}
		if err != nil {
			return fmt.Errorf("inspect required column usage_logs.%s: %w", expected.name, err)
		}
		if dataType != expected.dataType || maxLength != expected.maxLength || nullable != "YES" {
			return fmt.Errorf(
				"existing column usage_logs.%s is incompatible (type=%q max_length=%v nullable=%q, want type=%q max_length=%v nullable=\"YES\"); refusing migration %s",
				expected.name,
				dataType,
				maxLength,
				nullable,
				expected.dataType,
				expected.maxLength,
				upstreamModelMismatchIndexMigration,
			)
		}
	}
	return nil
}

func incompatibleConcurrentIndexError(expected expectedConcurrentIndex, field, got, want string) error {
	return fmt.Errorf(
		"existing index %s is incompatible with migration %s (%s=%q, want %q); refusing to mark migration applied; repair or remove the index explicitly",
		expected.indexName,
		expected.migration,
		field,
		got,
		want,
	)
}

var redundantIndexSQLParentheses = regexp.MustCompile(`\(\(([a-z_][a-z0-9_]*)\)\)`)
var indexSQLTypeCast = regexp.MustCompile(`::(?:pg_catalog\.)?(?:text|character varying|varchar)`)

// canonicalizeIndexSQL normalizes the stable semantic portions emitted by
// pg_get_expr/pg_get_indexdef across PostgreSQL versions (casts and redundant
// parentheses vary by server version). It intentionally does not attempt to
// parse arbitrary SQL; the expected expressions are fixed reviewed literals.
func canonicalizeIndexSQL(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, `"`, "")
	value = strings.ReplaceAll(value, "pg_catalog.", "")
	value = indexSQLTypeCast.ReplaceAllString(value, "")
	value = strings.ReplaceAll(value, "btrim(", "trim(")
	value = strings.Join(strings.Fields(value), "")
	value = strings.ReplaceAll(value, "trim(bothfrom", "trim(")
	for {
		next := redundantIndexSQLParentheses.ReplaceAllString(value, "($1)")
		if next == value {
			break
		}
		value = next
	}
	return stripEnclosingIndexSQLParentheses(value)
}

func stripEnclosingIndexSQLParentheses(value string) string {
	for len(value) >= 2 && value[0] == '(' && value[len(value)-1] == ')' {
		depth := 0
		encloses := true
		for index := 0; index < len(value); index++ {
			switch value[index] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 && index != len(value)-1 {
					encloses = false
				}
			}
			if depth < 0 {
				encloses = false
				break
			}
		}
		if !encloses || depth != 0 {
			break
		}
		value = value[1 : len(value)-1]
	}
	return value
}

func preparePaymentOrdersOutTradeNoUniqueMigration(ctx context.Context, db migrationQueryExecer) error {
	duplicates, err := findDuplicatePaymentOrderOutTradeNos(ctx, db)
	if err != nil {
		return fmt.Errorf("precheck duplicate out_trade_no: %w", err)
	}
	if len(duplicates) > 0 {
		return fmt.Errorf(
			"duplicate out_trade_no values block %s; remediate duplicates before retrying: %s",
			paymentOrdersOutTradeNoUniqueMigration,
			strings.Join(duplicates, ", "),
		)
	}

	return dropInvalidIndexIfPresent(ctx, db, paymentOrdersOutTradeNoUniqueIndex)
}

func dropInvalidIndexIfPresent(ctx context.Context, db migrationQueryExecer, indexName string) error {
	invalid, err := indexIsInvalid(ctx, db, indexName)
	if err != nil {
		return fmt.Errorf("check invalid index %s: %w", indexName, err)
	}
	if !invalid {
		return nil
	}

	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP INDEX CONCURRENTLY IF EXISTS %s", indexName)); err != nil {
		return fmt.Errorf("drop invalid index %s: %w", indexName, err)
	}
	return nil
}

func findDuplicatePaymentOrderOutTradeNos(ctx context.Context, db migrationQueryExecer) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT out_trade_no, COUNT(*) AS duplicate_count
		FROM payment_orders
		WHERE out_trade_no <> ''
		GROUP BY out_trade_no
		HAVING COUNT(*) > 1
		ORDER BY duplicate_count DESC, out_trade_no
		LIMIT 5
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	duplicates := make([]string, 0, 5)
	for rows.Next() {
		var outTradeNo string
		var duplicateCount int
		if err := rows.Scan(&outTradeNo, &duplicateCount); err != nil {
			return nil, err
		}
		duplicates = append(duplicates, fmt.Sprintf("%s (count=%d)", outTradeNo, duplicateCount))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return duplicates, nil
}

func indexIsInvalid(ctx context.Context, db migrationQueryExecer, indexName string) (bool, error) {
	var invalid bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_class idx
			JOIN pg_namespace ns ON ns.oid = idx.relnamespace
			JOIN pg_index i ON i.indexrelid = idx.oid
			WHERE ns.nspname = 'public'
			  AND idx.relname = $1
			  AND NOT i.indisvalid
		)
	`, indexName).Scan(&invalid)
	return invalid, err
}

// ensureAtlasBaselineAligned keeps the standalone compatibility helper aligned
// with applyMigrationsFS: only a persisted legacy origin may create a baseline.
func ensureAtlasBaselineAligned(ctx context.Context, db migrationQueryExecer, fsys fs.FS) error {
	files, err := collectValidatedMigrationFiles(fsys)
	if err != nil {
		return fmt.Errorf("validate migrations: %w", err)
	}

	origin, err := readSchemaMigrationOrigin(ctx, db)
	if err != nil {
		return fmt.Errorf("determine atlas baseline eligibility: %w", err)
	}
	if origin == schemaMigrationOriginFresh {
		return nil
	}
	return ensureAtlasBaselineAlignedWithFiles(ctx, db, files)
}

func ensureAtlasBaselineAlignedWithFiles(ctx context.Context, db migrationQueryExecer, files []validatedMigrationFile) error {
	hasAtlas, err := tableExists(ctx, db, "atlas_schema_revisions")
	if err != nil {
		return fmt.Errorf("check atlas_schema_revisions: %w", err)
	}
	if !hasAtlas {
		if _, err := db.ExecContext(ctx, atlasSchemaRevisionsTableDDL); err != nil {
			return fmt.Errorf("create atlas_schema_revisions: %w", err)
		}
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM atlas_schema_revisions").Scan(&count); err != nil {
		return fmt.Errorf("count atlas_schema_revisions: %w", err)
	}
	if count > 0 {
		return nil
	}

	version, description, hash := latestMigrationBaselineFromFiles(files)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO atlas_schema_revisions (version, description, type, applied, total, executed_at, execution_time, hash)
		VALUES ($1, $2, $3, 0, 0, NOW(), 0, $4)
	`, version, description, 1, hash); err != nil {
		return fmt.Errorf("insert atlas baseline: %w", err)
	}
	return nil
}

func tableExists(ctx context.Context, db migrationQueryExecer, tableName string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`, tableName).Scan(&exists)
	return exists, err
}

func latestMigrationBaseline(fsys fs.FS) (string, string, string, error) {
	files, err := collectValidatedMigrationFiles(fsys)
	if err != nil {
		return "", "", "", err
	}
	version, description, hash := latestMigrationBaselineFromFiles(files)
	return version, description, hash, nil
}

func latestMigrationBaselineFromFiles(files []validatedMigrationFile) (string, string, string) {
	if len(files) == 0 {
		return "baseline", "baseline", ""
	}
	latest := files[len(files)-1]
	name := latest.name
	version := strings.TrimSuffix(name, ".sql")
	return version, version, latest.checksum
}

func checksumSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func newMigrationChecksumCompatibilityRule(fileChecksum string, acceptedDBChecksums ...string) migrationChecksumCompatibilityRule {
	return migrationChecksumCompatibilityRule{
		fileChecksum:       fileChecksum,
		acceptedDBChecksum: checksumSet(acceptedDBChecksums...),
		acceptedChecksums:  checksumSet(append([]string{fileChecksum}, acceptedDBChecksums...)...),
	}
}

func isMigrationChecksumCompatible(name, dbChecksum, fileChecksum string) bool {
	rule, ok := migrationChecksumCompatibilityRules[name]
	if !ok {
		return false
	}
	_, dbOK := rule.acceptedChecksums[dbChecksum]
	if !dbOK {
		return false
	}
	_, fileOK := rule.acceptedChecksums[fileChecksum]
	return fileOK
}

func validateMigrationExecutionMode(name, content string) (bool, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	upperContent := strings.ToUpper(content)
	nonTx := strings.HasSuffix(normalizedName, nonTransactionalMigrationSuffix)

	if !nonTx {
		if strings.Contains(upperContent, "CONCURRENTLY") {
			return false, errors.New("CONCURRENTLY statements must be placed in *_notx.sql migrations")
		}
		return false, nil
	}

	if strings.Contains(upperContent, "BEGIN") || strings.Contains(upperContent, "COMMIT") || strings.Contains(upperContent, "ROLLBACK") {
		return false, errors.New("*_notx.sql must not contain transaction control statements (BEGIN/COMMIT/ROLLBACK)")
	}

	statements := splitSQLStatements(content)
	for _, stmt := range statements {
		normalizedStmt := strings.ToUpper(stripSQLLineComment(strings.TrimSpace(stmt)))
		if normalizedStmt == "" {
			continue
		}

		if strings.Contains(normalizedStmt, "CONCURRENTLY") {
			isCreateIndex := strings.Contains(normalizedStmt, "CREATE") && strings.Contains(normalizedStmt, "INDEX")
			isDropIndex := strings.Contains(normalizedStmt, "DROP") && strings.Contains(normalizedStmt, "INDEX")
			if !isCreateIndex && !isDropIndex {
				return false, errors.New("*_notx.sql currently only supports CREATE/DROP INDEX CONCURRENTLY statements")
			}
			if isCreateIndex && !strings.Contains(normalizedStmt, "IF NOT EXISTS") {
				return false, errors.New("CREATE INDEX CONCURRENTLY in *_notx.sql must include IF NOT EXISTS for idempotency")
			}
			if isDropIndex && !strings.Contains(normalizedStmt, "IF EXISTS") {
				return false, errors.New("DROP INDEX CONCURRENTLY in *_notx.sql must include IF EXISTS for idempotency")
			}
			continue
		}

		return false, errors.New("*_notx.sql must not mix non-CONCURRENTLY SQL statements")
	}

	return true, nil
}

func splitSQLStatements(content string) []string {
	parts := strings.Split(content, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func stripSQLLineComment(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// pgAdvisoryLock 获取 PostgreSQL Advisory Lock。
// Advisory Lock 是一种轻量级的锁机制，不与任何特定的数据库对象关联。
// 它非常适合用于应用层面的分布式锁场景，如迁移序列化。
func pgAdvisoryLock(ctx context.Context, db migrationQueryExecer) error {
	startedAt := time.Now()
	waited := false
	timedOut := false
	acquired := false
	defer func() {
		opsruntime.ObserveMigrationLock(time.Since(startedAt), waited, timedOut, acquired)
	}()

	ticker := time.NewTicker(migrationsLockRetryInterval)
	defer ticker.Stop()

	for {
		var locked bool
		if err := db.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", migrationsAdvisoryLockID).Scan(&locked); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				waited = true
				timedOut = true
			}
			return fmt.Errorf("acquire migrations lock: %w", err)
		}
		if locked {
			acquired = true
			return nil
		}
		waited = true
		select {
		case <-ctx.Done():
			timedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
			return fmt.Errorf("acquire migrations lock: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

// pgAdvisoryUnlock 释放 PostgreSQL Advisory Lock。
// 必须在获取锁后确保释放，否则会阻塞其他实例的迁移操作。
func pgAdvisoryUnlock(ctx context.Context, db migrationQueryExecer) error {
	var unlocked bool
	if err := db.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", migrationsAdvisoryLockID).Scan(&unlocked); err != nil {
		return fmt.Errorf("release migrations lock: %w", err)
	}
	if !unlocked {
		return errors.New("release migrations lock: current session did not hold the lock")
	}
	return nil
}

func discardSQLConn(conn *sql.Conn) {
	if conn == nil {
		return
	}
	// Returning driver.ErrBadConn tells database/sql to close this physical
	// connection instead of putting it back into the idle pool.
	_ = conn.Raw(func(any) error {
		return driver.ErrBadConn
	})
}
