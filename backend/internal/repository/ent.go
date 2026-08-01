// Package repository 提供应用程序的基础设施层组件。
// 包括数据库连接初始化、ORM 客户端管理、Redis 连接、数据库迁移等核心功能。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/migrations"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/lib/pq"
)

const ConfiguredDatabaseIdentityContract = "sub2api-database-identity/v2"

// ConfiguredDatabaseIdentity is the non-secret identity used to prove that
// preflight, backup, and migrate-only all target the same PostgreSQL database.
type ConfiguredDatabaseIdentity struct {
	Contract         string `json:"contract"`
	Database         string `json:"database"`
	SystemIdentifier string `json:"system_identifier"`
	InRecovery       bool   `json:"in_recovery"`
}

// InitEnt 初始化 Ent ORM 客户端并返回客户端实例和底层的 *sql.DB。
//
// 该函数执行以下操作：
//  1. 初始化全局时区设置，确保时间处理一致性
//  2. 建立 PostgreSQL 数据库连接
//  3. 自动执行普通数据库迁移；已有库若存在 maintenance-only migration 则拒绝启动
//  4. 创建并返回 Ent 客户端实例
//
// 重要提示：调用者必须负责关闭返回的 ent.Client（关闭时会自动关闭底层的 driver/db）。
//
// 参数：
//   - cfg: 应用程序配置，包含数据库连接信息和时区设置
//
// 返回：
//   - *ent.Client: Ent ORM 客户端，用于执行数据库操作
//   - *sql.DB: 底层的 SQL 数据库连接，可用于直接执行原生 SQL
//   - error: 初始化过程中的错误
func InitEnt(cfg *config.Config) (*ent.Client, *sql.DB, error) {
	drv, err := openConfiguredEntDriver(cfg)
	if err != nil {
		return nil, nil, err
	}

	// 确保数据库 schema 已准备就绪。
	// SQL 迁移文件是 schema 的权威来源（source of truth）。
	// 这种方式比 Ent 的自动迁移更可控，支持复杂的迁移场景。
	migrationCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := ApplyMigrations(migrationCtx, drv.DB()); err != nil {
		_ = drv.Close() // 迁移失败时关闭驱动，避免资源泄露
		return nil, nil, err
	}

	// 创建 Ent 客户端，绑定到已配置的数据库驱动。
	client := ent.NewClient(ent.Driver(drv))

	// 启动阶段：从配置或数据库中确保系统密钥可用。
	if err := ensureBootstrapSecrets(migrationCtx, client, cfg); err != nil {
		_ = client.Close()
		return nil, nil, err
	}

	// 在密钥补齐后执行完整配置校验，避免空 jwt.secret 导致服务运行时失败。
	if err := cfg.Validate(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("validate config after secret bootstrap: %w", err)
	}

	// SIMPLE 模式：启动时补齐各平台默认分组。
	// - anthropic/openai/gemini: 确保存在 <platform>-default
	// - antigravity: 仅要求存在 >=2 个未软删除分组（用于 claude/gemini 混合调度场景）
	if cfg.RunMode == config.RunModeSimple {
		seedCtx, seedCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer seedCancel()
		if err := ensureSimpleModeDefaultGroups(seedCtx, client); err != nil {
			_ = client.Close()
			return nil, nil, err
		}
		if err := ensureSimpleModeAdminConcurrency(seedCtx, client); err != nil {
			_ = client.Close()
			return nil, nil, err
		}
	}

	return client, drv.DB(), nil
}

// ApplyConfiguredMigrations applies the embedded SQL migrations and exits
// without performing any other application bootstrap writes. It is intended
// for maintenance-window releases where HTTP handlers, workers, Redis-backed
// flushers, secret initialization, and simple-mode seed data must remain off.
func ApplyConfiguredMigrations(
	ctx context.Context,
	cfg *config.Config,
	expectedIdentity ConfiguredDatabaseIdentity,
) (retErr error) {
	if ctx == nil {
		return fmt.Errorf("apply configured migrations: context is nil")
	}
	if err := ValidateExpectedDatabaseIdentity(expectedIdentity); err != nil {
		return fmt.Errorf("apply configured migrations: %w", err)
	}

	drv, err := openConfiguredEntDriver(cfg)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, drv.Close())
	}()

	if err := applyMigrationsFSWithExpectedDatabaseIdentity(
		ctx,
		drv.DB(),
		migrations.FS,
		&expectedIdentity,
	); err != nil {
		return fmt.Errorf("apply embedded migrations: %w", err)
	}
	return nil
}

// ReadConfiguredDatabaseIdentity loads the same database configuration as the
// migration entrypoint and performs one read-only identity query. It does not
// run migrations or initialize any application component.
func ReadConfiguredDatabaseIdentity(
	ctx context.Context,
	cfg *config.Config,
) (identity ConfiguredDatabaseIdentity, retErr error) {
	if ctx == nil {
		return identity, fmt.Errorf("read configured database identity: context is nil")
	}

	drv, err := openConfiguredEntDriver(cfg)
	if err != nil {
		return identity, fmt.Errorf("open database identity connection: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, drv.Close())
	}()

	return queryDatabaseIdentity(ctx, drv.DB())
}

func queryDatabaseIdentity(
	ctx context.Context,
	db migrationQueryExecer,
) (ConfiguredDatabaseIdentity, error) {
	identity := ConfiguredDatabaseIdentity{Contract: ConfiguredDatabaseIdentityContract}
	if err := db.QueryRowContext(ctx, `
		SELECT current_database(), system_identifier::text, pg_is_in_recovery()
		FROM pg_control_system()
	`).Scan(
		&identity.Database,
		&identity.SystemIdentifier,
		&identity.InRecovery,
	); err != nil {
		return ConfiguredDatabaseIdentity{}, fmt.Errorf("query database identity: %w", err)
	}
	identity.Database = strings.TrimSpace(identity.Database)
	identity.SystemIdentifier = strings.TrimSpace(identity.SystemIdentifier)
	if identity.Database == "" || identity.SystemIdentifier == "" {
		return ConfiguredDatabaseIdentity{}, fmt.Errorf("query database identity: empty identity field")
	}
	return identity, nil
}

// ValidateExpectedDatabaseIdentity validates operator-supplied identity
// evidence before a maintenance migration opens its database connection.
func ValidateExpectedDatabaseIdentity(identity ConfiguredDatabaseIdentity) error {
	if identity.Contract != ConfiguredDatabaseIdentityContract {
		return fmt.Errorf(
			"expected database identity contract must be %q",
			ConfiguredDatabaseIdentityContract,
		)
	}
	if identity.Database == "" || identity.Database != strings.TrimSpace(identity.Database) {
		return fmt.Errorf("expected database identity has invalid database name")
	}
	if identity.SystemIdentifier == "" ||
		identity.SystemIdentifier != strings.TrimSpace(identity.SystemIdentifier) {
		return fmt.Errorf("expected database identity has invalid system identifier")
	}
	for _, ch := range identity.SystemIdentifier {
		if ch < '0' || ch > '9' {
			return fmt.Errorf("expected database identity system identifier must be decimal")
		}
	}
	if identity.InRecovery {
		return fmt.Errorf("expected database identity must identify a writable primary")
	}
	return nil
}

func verifyExpectedDatabaseIdentity(
	ctx context.Context,
	db migrationQueryExecer,
	expected ConfiguredDatabaseIdentity,
) error {
	if err := ValidateExpectedDatabaseIdentity(expected); err != nil {
		return err
	}
	actual, err := queryDatabaseIdentity(ctx, db)
	if err != nil {
		return err
	}
	if actual.InRecovery {
		return fmt.Errorf("configured migration database is in recovery; refusing standby target")
	}
	if actual.Database != expected.Database ||
		actual.SystemIdentifier != expected.SystemIdentifier {
		return fmt.Errorf(
			"configured migration database identity mismatch (expected database=%q system_identifier=%q, got database=%q system_identifier=%q)",
			expected.Database,
			expected.SystemIdentifier,
			actual.Database,
			actual.SystemIdentifier,
		)
	}
	return nil
}

func openConfiguredEntDriver(cfg *config.Config) (*entsql.Driver, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database config is nil")
	}

	// 优先初始化时区设置，确保所有时间操作使用统一的时区。
	// 这对于跨时区部署和日志时间戳的一致性至关重要。
	if err := timezone.Init(cfg.Timezone); err != nil {
		return nil, err
	}

	// 构建包含时区信息的数据库连接字符串 (DSN)。
	// 时区信息会传递给 PostgreSQL，确保数据库层面的时间处理正确。
	dsn := cfg.Database.DSNWithTimezone(cfg.Timezone)

	// 使用 Ent 的 SQL 驱动打开 PostgreSQL 连接。
	// dialect.Postgres 指定使用 PostgreSQL 方言进行 SQL 生成。
	var drv *entsql.Driver
	if cfg.Server.EnableServerTiming {
		connector, err := pq.NewConnector(dsn)
		if err != nil {
			return nil, err
		}
		drv = entsql.OpenDB(dialect.Postgres, sql.OpenDB(newServerTimingConnector(connector)))
	} else {
		var err error
		drv, err = entsql.Open(dialect.Postgres, dsn)
		if err != nil {
			return nil, err
		}
	}
	applyDBPoolSettings(drv.DB(), cfg)
	return drv, nil
}
