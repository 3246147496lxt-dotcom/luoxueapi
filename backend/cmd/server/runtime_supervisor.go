package main

import (
	"context"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
	schedulerapp "github.com/Wei-Shaw/sub2api/internal/modules/scheduler/application"
	applogger "github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/websearch"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// buildApplicationSupervisor is kept outside the generated Wire file so a
// regenerated injector and the checked-in injector share the same lifecycle
// ordering.
func buildApplicationSupervisor(
	pricing *service.PricingService,
	settingService *service.SettingService,
	routerSettingsRuntime *server.RouterSettingsRuntime,
	opsService *service.OpsService,
	idempotencyCoordinator *service.IdempotencyCoordinator,
	subscriptionService *service.SubscriptionService,
	oauth *service.OAuthService,
	openaiOAuth *service.OpenAIOAuthService,
	geminiOAuth *service.GeminiOAuthService,
	antigravityOAuth *service.AntigravityOAuthService,
	grokOAuth *service.GrokOAuthService,
	openAIGateway *service.OpenAIGatewayService,
	digestSessionStore *service.DigestSessionStore,
	tlsFingerprintProfiles *service.TLSFingerprintProfileService,
	errorPassthrough *service.ErrorPassthroughService,
	timingWheel *service.TimingWheelService,
	deferred *service.DeferredService,
	quotaFlusher *service.UserPlatformQuotaUsageFlusher,
	billingCache *service.BillingCacheService,
	auditLog *service.AuditLogService,
	opsSystemLogSink *service.OpsSystemLogSink,
	emailQueue *service.EmailQueueService,
	usageLogBatchRuntime repository.UsageLogBatchRuntime,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	batchImageWorker *service.BatchImageWorkerRuntime,
	schedulerSnapshot *service.SchedulerSnapshotService,
	schedulerModule *schedulerapp.Facade,
	apiKeyService *service.APIKeyService,
	concurrencyService *service.ConcurrencyService,
	userMessageQueue *service.UserMessageQueueService,
	contentModeration *service.ContentModerationService,
	batchImageCleanup *service.BatchImageCleanupService,
	idempotencyCleanup *service.IdempotencyCleanupService,
	chatAttemptRecovery *service.ChatAttemptService,
	dashboardAggregation *service.DashboardAggregationService,
	usageCleanup *service.UsageCleanupService,
	opsMetricsCollector *service.OpsMetricsCollector,
	opsAggregation *service.OpsAggregationService,
	opsAlertEvaluator *service.OpsAlertEvaluatorService,
	opsCleanup *service.OpsCleanupService,
	opsScheduledReport *service.OpsScheduledReportService,
	tokenRefresh *service.TokenRefreshService,
	accountExpiry *service.AccountExpiryService,
	proxyExpiry *service.ProxyExpiryService,
	subscriptionExpiry *service.SubscriptionExpiryService,
	proxyHealth *service.ProxyHealthService,
	scheduledTestRunner *service.ScheduledTestRunnerService,
	backupSvc *service.BackupService,
	paymentOrderExpiry *service.PaymentOrderExpiryService,
	channelMonitorRunner *service.ChannelMonitorRunner,
	upstreamBillingProbe *service.UpstreamBillingProbeService,
	redisClient *redis.Client,
	cfg *config.Config,
) *lifecycle.Supervisor {
	pricingStop := applicationStopWithinContext(pricing.Stop)
	settingsRuntimeStop := applicationStopWithinContext(settingService.StopRuntime)
	timingWheelStop := applicationStopWithinContext(timingWheel.Stop)
	opsSystemLogSinkStop := applicationStopWithinContext(func() {
		applogger.SetSink(nil)
		opsSystemLogSink.Stop()
	})
	schedulerSnapshotStop := applicationStopWithinContext(schedulerSnapshot.Stop)

	return lifecycle.NewSupervisor(
		// Supporting runtimes start first and therefore stop last.
		lifecycle.ComponentFuncs{
			ComponentName: "pricing",
			StartFunc: func(context.Context) error {
				if err := pricing.Initialize(); err != nil {
					log.Printf("[Lifecycle] pricing initialization degraded: %v", err)
				}
				return nil
			},
			StopFunc: pricingStop,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "settings-runtime",
			StartFunc: func(ctx context.Context) error {
				configureWebSearchManagerBuilder(settingService, redisClient)
				return settingService.StartRuntime(ctx)
			},
			StopFunc: settingsRuntimeStop,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "router-settings-runtime",
			StartFunc:     routerSettingsRuntime.Start,
			StopFunc:      routerSettingsRuntime.Stop,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "ops-runtime-log-config",
			StartFunc:     opsService.StartRuntime,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "default-idempotency-coordinator",
			StartFunc: func(context.Context) error {
				service.SetDefaultIdempotencyCoordinator(idempotencyCoordinator)
				return nil
			},
			StopFunc: func(context.Context) error {
				if service.DefaultIdempotencyCoordinator() == idempotencyCoordinator {
					service.SetDefaultIdempotencyCoordinator(nil)
				}
				return nil
			},
		},
		applicationVoidLifecycleComponent("oauth-claude", oauth.Start, oauth.Stop),
		applicationVoidLifecycleComponent("oauth-openai", openaiOAuth.Start, openaiOAuth.Stop),
		applicationVoidLifecycleComponent("oauth-gemini", geminiOAuth.Start, geminiOAuth.Stop),
		applicationVoidLifecycleComponent("oauth-antigravity", antigravityOAuth.Start, antigravityOAuth.Stop),
		applicationVoidLifecycleComponent("oauth-grok", grokOAuth.Start, grokOAuth.Stop),
		applicationVoidLifecycleComponent("digest-session-store", digestSessionStore.Start, digestSessionStore.Stop),
		lifecycle.ComponentFuncs{
			ComponentName: "tls-fingerprint-profile-cache",
			StartFunc:     tlsFingerprintProfiles.Start,
			StopFunc:      tlsFingerprintProfiles.Stop,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "error-passthrough-cache",
			StartFunc:     errorPassthrough.Start,
			StopFunc:      errorPassthrough.Stop,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "timing-wheel",
			StartFunc: func(context.Context) error {
				return timingWheel.StartWithError()
			},
			StopFunc: timingWheelStop,
		},

		// Flushers are stopped after request producers and queue consumers.
		applicationVoidLifecycleComponent("deferred-account-updates", deferred.Start, deferred.Stop),
		applicationVoidLifecycleComponent("platform-quota-flusher", quotaFlusher.Start, quotaFlusher.Stop),
		applicationVoidLifecycleComponent("billing-cache-writer", billingCache.Start, billingCache.Stop),
		applicationVoidLifecycleComponent("audit-log", auditLog.Start, auditLog.Stop),
		lifecycle.ComponentFuncs{
			ComponentName: "ops-system-log-sink",
			StartFunc: func(context.Context) error {
				opsSystemLogSink.Start()
				applogger.SetSink(opsSystemLogSink)
				return nil
			},
			StopFunc: opsSystemLogSinkStop,
		},

		// The repository sink stops after every consumer below has stopped
		// producing usage records, but before flushers and infrastructure.
		lifecycle.ComponentFuncs{
			ComponentName: "usage-log-batch-runtime",
			StopFunc:      usageLogBatchRuntime.Stop,
		},

		// Consumers are stopped before flushers and before the timing wheel.
		lifecycle.ComponentFuncs{
			ComponentName: "subscription-service",
			StartFunc: func(context.Context) error {
				subscriptionService.Start()
				return nil
			},
			StopFunc: subscriptionService.StopContext,
		},
		applicationVoidLifecycleComponent("openai-websocket-pool", nil, openAIGateway.CloseOpenAIWSPool),
		applicationVoidLifecycleComponent("email-queue", emailQueue.Start, emailQueue.Stop),
		applicationVoidLifecycleComponent("usage-record-worker-pool", usageRecordWorkerPool.Start, usageRecordWorkerPool.Stop),
		applicationVoidLifecycleComponent("batch-image-worker", batchImageWorker.Start, batchImageWorker.Stop),
		lifecycle.ComponentFuncs{
			ComponentName: "api-key-cache-subscriber",
			StartFunc: func(ctx context.Context) error {
				apiKeyService.StartAuthCacheInvalidationSubscriber(ctx)
				return nil
			},
			StopFunc: apiKeyService.StopAuthCacheInvalidationSubscriber,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "content-moderation",
			StartFunc: func(context.Context) error {
				contentModeration.Start()
				return nil
			},
			StopFunc: contentModeration.Stop,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "user-message-queue-cleanup",
			StartFunc: func(context.Context) error {
				if cfg != nil && cfg.Gateway.UserMessageQueue.CleanupIntervalSeconds > 0 {
					userMessageQueue.StartCleanupWorker(time.Duration(cfg.Gateway.UserMessageQueue.CleanupIntervalSeconds) * time.Second)
				}
				return nil
			},
			StopFunc: userMessageQueue.StopContext,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "concurrency-slot-cleanup",
			StartFunc: func(ctx context.Context) error {
				if err := concurrencyService.CleanupStaleProcessSlots(ctx); err != nil {
					return err
				}
				if cfg != nil {
					concurrencyService.StartSlotCleanupWorker(nil, cfg.Gateway.Scheduling.SlotCleanupInterval)
				}
				return nil
			},
			StopFunc: concurrencyService.StopSlotCleanupWorker,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "scheduler-snapshot",
			StartFunc: func(context.Context) error {
				schedulerSnapshot.Start()
				return nil
			},
			StopFunc: schedulerSnapshotStop,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "scheduler-shadow-comparison",
			StartFunc: func(ctx context.Context) error {
				if schedulerModule == nil {
					return nil
				}
				return schedulerModule.StartShadow(ctx)
			},
			StopFunc: func(ctx context.Context) error {
				if schedulerModule == nil {
					return nil
				}
				return schedulerModule.StopShadow(ctx)
			},
		},

		// Periodic producers start last, so shutdown disables them first.
		applicationVoidLifecycleComponent("batch-image-cleanup", batchImageCleanup.Start, batchImageCleanup.Stop),
		applicationVoidLifecycleComponent("idempotency-cleanup", idempotencyCleanup.Start, idempotencyCleanup.Stop),
		lifecycle.ComponentFuncs{
			ComponentName: "chat-attempt-recovery",
			StartFunc:     chatAttemptRecovery.StartRecovery,
			StopFunc:      chatAttemptRecovery.StopRecovery,
		},
		lifecycle.ComponentFuncs{
			ComponentName: "dashboard-aggregation",
			StartFunc: func(context.Context) error {
				dashboardAggregation.Start()
				return nil
			},
			StopFunc: dashboardAggregation.StopContext,
		},
		applicationVoidLifecycleComponent("usage-cleanup", usageCleanup.Start, usageCleanup.Stop),
		applicationVoidLifecycleComponent("ops-metrics", opsMetricsCollector.Start, opsMetricsCollector.Stop),
		applicationVoidLifecycleComponent("ops-aggregation", opsAggregation.Start, opsAggregation.Stop),
		applicationVoidLifecycleComponent("ops-alert-evaluator", opsAlertEvaluator.Start, opsAlertEvaluator.Stop),
		applicationVoidLifecycleComponent("ops-cleanup", opsCleanup.Start, opsCleanup.Stop),
		applicationVoidLifecycleComponent("ops-scheduled-report", opsScheduledReport.Start, opsScheduledReport.Stop),
		applicationVoidLifecycleComponent("token-refresh", tokenRefresh.Start, tokenRefresh.Stop),
		applicationVoidLifecycleComponent("account-expiry", accountExpiry.Start, accountExpiry.Stop),
		applicationVoidLifecycleComponent("proxy-expiry", proxyExpiry.Start, proxyExpiry.Stop),
		applicationVoidLifecycleComponent("subscription-expiry", subscriptionExpiry.Start, subscriptionExpiry.Stop),
		applicationVoidLifecycleComponent("proxy-health", proxyHealth.Start, proxyHealth.Stop),
		applicationVoidLifecycleComponent("scheduled-tests", scheduledTestRunner.Start, scheduledTestRunner.Stop),
		lifecycle.ComponentFuncs{
			ComponentName: "backup-scheduler",
			StartFunc: func(context.Context) error {
				backupSvc.Start()
				return nil
			},
			StopFunc: backupSvc.StopContext,
		},
		applicationVoidLifecycleComponent("payment-order-expiry", paymentOrderExpiry.Start, paymentOrderExpiry.Stop),
		applicationVoidLifecycleComponent("channel-monitor", channelMonitorRunner.Start, channelMonitorRunner.Stop),
		applicationVoidLifecycleComponent("upstream-billing-probe", upstreamBillingProbe.Start, upstreamBillingProbe.Stop),
	)
}

func applicationVoidLifecycleComponent(name string, start, stop func()) lifecycle.Component {
	stopWithinContext := applicationStopWithinContext(stop)
	return lifecycle.ComponentFuncs{
		ComponentName: name,
		StartFunc: func(context.Context) error {
			if start != nil {
				start()
			}
			return nil
		},
		StopFunc: func(ctx context.Context) error {
			return stopWithinContext(ctx)
		},
	}
}

func applicationStopWithinContext(stop func()) func(context.Context) error {
	if stop == nil {
		return func(context.Context) error { return nil }
	}

	var once sync.Once
	done := make(chan struct{})
	return func(ctx context.Context) error {
		once.Do(func() {
			go func() {
				defer close(done)
				stop()
			}()
		})
		select {
		case <-done:
			return nil
		default:
		}
		if ctx == nil {
			ctx = context.Background()
		}
		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func infrastructureCleanup(entClient *ent.Client, redisClient *redis.Client) func() {
	return func() {
		if redisClient != nil {
			if err := redisClient.Close(); err != nil {
				log.Printf("[Cleanup] Redis close failed: %v", err)
			}
		}
		if entClient != nil {
			if err := entClient.Close(); err != nil {
				log.Printf("[Cleanup] database close failed: %v", err)
			}
		}
	}
}

// These providers live in a regular source file because both the Wire
// injector and the checked-in generated injector call them.
func providePrivacyClientFactory() service.PrivacyClientFactory {
	return repository.CreatePrivacyReqClient
}

func provideServiceBuildInfo(buildInfo handler.BuildInfo) service.BuildInfo {
	return service.BuildInfo{
		Version:   buildInfo.Version,
		BuildType: buildInfo.BuildType,
	}
}

func provideCleanup(entClient *ent.Client, redisClient *redis.Client) func() {
	return infrastructureCleanup(entClient, redisClient)
}

func configureWebSearchManagerBuilder(settingService *service.SettingService, redisClient *redis.Client) {
	settingService.ConfigureWebSearchManagerBuilder(func(cfg *service.WebSearchEmulationConfig, proxyURLs map[int64]string) {
		if cfg == nil || !cfg.Enabled || len(cfg.Providers) == 0 {
			service.SetWebSearchManager(nil)
			return
		}
		configs := make([]websearch.ProviderConfig, 0, len(cfg.Providers))
		for _, provider := range cfg.Providers {
			if provider.APIKey == "" {
				continue
			}
			providerConfig := websearch.ProviderConfig{
				Type:       provider.Type,
				APIKey:     provider.APIKey,
				QuotaLimit: derefInt64(provider.QuotaLimit),
				ExpiresAt:  provider.ExpiresAt,
			}
			if provider.SubscribedAt != nil {
				providerConfig.SubscribedAt = provider.SubscribedAt
			}
			if provider.ProxyID != nil {
				providerConfig.ProxyID = *provider.ProxyID
				proxyURL, ok := proxyURLs[*provider.ProxyID]
				if !ok {
					slog.Warn("websearch: proxy not found for provider, skipping",
						"provider", provider.Type, "proxy_id", *provider.ProxyID)
					continue
				}
				providerConfig.ProxyURL = proxyURL
			}
			configs = append(configs, providerConfig)
		}
		service.SetWebSearchManager(websearch.NewManager(configs, redisClient))
	})
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
