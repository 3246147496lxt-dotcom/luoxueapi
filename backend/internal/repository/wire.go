package repository

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

// ProvideConcurrencyCache 创建并发控制缓存，从配置读取 TTL 参数
// 性能优化：TTL 可配置，支持长时间运行的 LLM 请求场景
func ProvideConcurrencyCache(rdb *redis.Client, cfg *config.Config) service.ConcurrencyCache {
	waitTTLSeconds := int(cfg.Gateway.Scheduling.StickySessionWaitTimeout.Seconds())
	if cfg.Gateway.Scheduling.FallbackWaitTimeout > cfg.Gateway.Scheduling.StickySessionWaitTimeout {
		waitTTLSeconds = int(cfg.Gateway.Scheduling.FallbackWaitTimeout.Seconds())
	}
	if waitTTLSeconds <= 0 {
		waitTTLSeconds = cfg.Gateway.ConcurrencySlotTTLMinutes * 60
	}
	return NewConcurrencyCache(rdb, cfg.Gateway.ConcurrencySlotTTLMinutes, waitTTLSeconds)
}

// ProvideGitHubReleaseClient 创建 GitHub Release 客户端
// 从配置中读取代理设置，支持国内服务器通过代理访问 GitHub
func ProvideGitHubReleaseClient(cfg *config.Config) service.GitHubReleaseClient {
	return NewGitHubReleaseClient(cfg.Update.ProxyURL, cfg.Security.ProxyFallback.AllowDirectOnError)
}

// ProvidePricingRemoteClient 创建定价数据远程客户端
// 从配置中读取代理设置，支持国内服务器通过代理访问 GitHub 上的定价数据
func ProvidePricingRemoteClient(cfg *config.Config) service.PricingRemoteClient {
	return NewPricingRemoteClient(cfg.Update.ProxyURL, cfg.Security.ProxyFallback.AllowDirectOnError)
}

// ProvideSessionLimitCache 创建会话限制缓存
// 用于 Anthropic OAuth/SetupToken 账号的并发会话数量控制
func ProvideSessionLimitCache(rdb *redis.Client, cfg *config.Config) service.SessionLimitCache {
	defaultIdleTimeoutMinutes := 5 // 默认 5 分钟空闲超时
	if cfg != nil && cfg.Gateway.SessionIdleTimeoutMinutes > 0 {
		defaultIdleTimeoutMinutes = cfg.Gateway.SessionIdleTimeoutMinutes
	}
	return NewSessionLimitCache(rdb, defaultIdleTimeoutMinutes)
}

// ProvideSchedulerCache 创建调度快照缓存，并注入快照分块参数。
func ProvideSchedulerCache(rdb *redis.Client, cfg *config.Config) service.SchedulerCache {
	mgetChunkSize := defaultSchedulerSnapshotMGetChunkSize
	writeChunkSize := defaultSchedulerSnapshotWriteChunkSize
	if cfg != nil {
		if cfg.Gateway.Scheduling.SnapshotMGetChunkSize > 0 {
			mgetChunkSize = cfg.Gateway.Scheduling.SnapshotMGetChunkSize
		}
		if cfg.Gateway.Scheduling.SnapshotWriteChunkSize > 0 {
			writeChunkSize = cfg.Gateway.Scheduling.SnapshotWriteChunkSize
		}
	}
	return newSchedulerCacheWithChunkSizes(rdb, mgetChunkSize, writeChunkSize)
}

// ProvideAccountProjectionWriter narrows SchedulerCache to the only cache
// writes account persistence may perform. These writes are best-effort after
// commit; scheduler_outbox is the durable publication path.
func ProvideAccountProjectionWriter(cache service.SchedulerCache) service.AccountProjectionWriter {
	return cache
}

// ProviderSet is the Wire provider set for all repositories
var ProviderSet = wire.NewSet(
	NewUserRepository,
	NewAPIKeyRepository,
	NewChatPrincipalRepository,
	NewDesktopRepository,
	NewDesktopCleanupRepository,
	NewDesktopPairingStore,
	NewDesktopRouteCatalog,
	NewDesktopKeyInvalidator,
	NewGroupRepository,
	NewAccountRepository,
	NewAdminAccountRepository,
	NewScheduledTestPlanRepository,   // 定时测试计划仓储
	NewScheduledTestResultRepository, // 定时测试结果仓储
	NewProxyRepository,
	NewRedeemCodeRepository,
	NewPromoCodeRepository,
	NewAnnouncementRepository,
	NewAnnouncementReadRepository,
	NewUsageLogRepository,
	ProvideUsageLogBatchRuntime,
	NewUsageBillingRepository,
	NewBillingReceiptRepository,
	NewChatAttemptRepository,
	NewChatHistoryRepository,
	NewAdminChatHistoryRepository,
	NewBatchImageRepository,
	NewIdempotencyRepository,
	NewUsageCleanupRepository,
	NewDashboardAggregationRepository,
	NewSettingRepository,
	NewOpsRepository,
	NewAuditLogRepository,
	NewUserSubscriptionRepository,
	NewUserAttributeDefinitionRepository,
	NewUserAttributeValueRepository,
	NewUserGroupRateRepository,
	NewErrorPassthroughRepository,
	NewTLSFingerprintProfileRepository,
	NewChannelRepository,
	NewModelCatalogRepository,
	NewDocumentationRepository,
	NewChannelMonitorRepository,
	NewChannelMonitorRequestTemplateRepository,
	NewContentModerationRepository,
	NewAffiliateRepository,
	NewUserPlatformQuotaRepository,     // T14: user × platform quota
	NewUserPlatformQuotaServiceAdapter, // T14: adapter → service.UserPlatformQuotaRepository

	// Cache implementations
	NewGatewayCache,
	NewBillingCache,
	NewAPIKeyCache,
	NewTempUnschedCache,
	NewTimeoutCounterCache,
	NewOpenAI403CounterCache,
	NewInternal500CounterCache,
	ProvideConcurrencyCache,
	ProvideSessionLimitCache,
	NewRPMCache,
	NewUserRPMCache,
	NewUserMsgQueueCache,
	NewDashboardCache,
	NewEmailCache,
	NewIdentityCache,
	NewRedeemCache,
	NewUpdateCache,
	NewGeminiTokenCache,
	NewImageTaskStore,
	NewBatchImageQueue,
	NewBatchImageDownloadLimiter,
	NewLeaderLockCache,
	ProvideSchedulerCache,
	ProvideAccountProjectionWriter,
	NewSchedulerOutboxRepository,
	NewProxyLatencyCache,
	NewTotpCache,
	NewRefreshTokenCache,
	NewErrorPassthroughCache,
	NewTLSFingerprintProfileCache,
	NewContentModerationHashCache,

	// Encryptors
	NewAESEncryptor,

	// Backup infrastructure
	NewPgDumper,
	NewS3BackupStoreFactory,

	// Image storage (async image task result offload)
	ProvideImageStorage,

	// HTTP service ports (DI Strategy A: return interface directly)
	NewTurnstileVerifier,
	ProvidePricingRemoteClient,
	ProvideGitHubReleaseClient,
	NewProxyExitInfoProber,
	NewClaudeUsageFetcher,
	NewClaudeOAuthClient,
	NewHTTPUpstream,
	NewOpenAIOAuthClient,
	NewGrokOAuthClient,
	NewGeminiOAuthClient,
	NewGeminiCliCodeAssistClient,
	NewGeminiDriveClient,
)

// ProvideImageStorage 提供异步图片任务结果转存所用的对象存储实现。
// 仅当开关打开且 S3 凭证齐全时返回具体实现，否则返回 nil（功能整体禁用）。
func ProvideImageStorage(cfg *config.Config) (service.ImageStorage, error) {
	if !cfg.ImageStorage.Active() {
		return nil, nil
	}
	store, err := NewS3ImageStorage(context.Background(), &cfg.ImageStorage)
	if err != nil {
		return nil, err
	}
	return store, nil
}
