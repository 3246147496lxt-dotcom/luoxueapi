package service

import "github.com/Wei-Shaw/sub2api/internal/config"

func newStartedBillingCacheServiceForTest(
	cache BillingCache,
	userRepo UserRepository,
	subRepo UserSubscriptionRepository,
	apiKeyRepo APIKeyRepository,
	userRPMCache UserRPMCache,
	userGroupRateRepo UserGroupRateRepository,
	cfg *config.Config,
	userPlatformQuotaRepo UserPlatformQuotaRepository,
) *BillingCacheService {
	svc := NewBillingCacheService(cache, userRepo, subRepo, apiKeyRepo, userRPMCache, userGroupRateRepo, cfg, userPlatformQuotaRepo)
	svc.Start()
	return svc
}
