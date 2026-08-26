package service

import (
	"context"
	"strconv"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// deferredSubscriptionCacheInvalidations carries cache work across a caller
// owned database transaction.  Subscription rows must be committed before a
// cache miss can safely reload them; invalidating while the transaction is
// still open can race a reader into the old snapshot and leave that value
// cached after commit.
type deferredSubscriptionCacheInvalidations struct {
	mu    sync.Mutex
	items map[string]deferredSubscriptionCacheInvalidation
}

type deferredSubscriptionCacheInvalidation struct {
	service *SubscriptionService
	userID  int64
	groupID int64
}

type deferredSubscriptionCacheInvalidationsContextKey struct{}

func withDeferredSubscriptionCacheInvalidations(ctx context.Context) (context.Context, *deferredSubscriptionCacheInvalidations) {
	if ctx == nil {
		ctx = context.Background()
	}
	if existing, ok := ctx.Value(deferredSubscriptionCacheInvalidationsContextKey{}).(*deferredSubscriptionCacheInvalidations); ok && existing != nil {
		return ctx, existing
	}
	collector := &deferredSubscriptionCacheInvalidations{
		items: make(map[string]deferredSubscriptionCacheInvalidation),
	}
	return context.WithValue(ctx, deferredSubscriptionCacheInvalidationsContextKey{}, collector), collector
}

func enqueueDeferredSubscriptionCacheInvalidation(ctx context.Context, service *SubscriptionService, userID, groupID int64) bool {
	if service == nil || userID <= 0 || groupID <= 0 || ctx == nil {
		return false
	}
	collector, ok := ctx.Value(deferredSubscriptionCacheInvalidationsContextKey{}).(*deferredSubscriptionCacheInvalidations)
	if !ok || collector == nil {
		return false
	}
	key := strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(groupID, 10)
	collector.mu.Lock()
	collector.items[key] = deferredSubscriptionCacheInvalidation{service: service, userID: userID, groupID: groupID}
	collector.mu.Unlock()
	return true
}

func flushDeferredSubscriptionCacheInvalidations(collector *deferredSubscriptionCacheInvalidations) {
	if collector == nil {
		return
	}
	collector.mu.Lock()
	items := make([]deferredSubscriptionCacheInvalidation, 0, len(collector.items))
	for _, item := range collector.items {
		items = append(items, item)
	}
	collector.items = make(map[string]deferredSubscriptionCacheInvalidation)
	collector.mu.Unlock()

	for _, item := range items {
		if item.service == nil {
			continue
		}
		if err := item.service.invalidateSubscriptionCaches(item.userID, item.groupID); err != nil {
			// Cache invalidation is best effort.  The database commit has already
			// succeeded, and the normal TTL/pubsub paths provide eventual repair.
			logger.LegacyPrintf("service.subscription", "[Subscription] deferred cache invalidation failed user_id=%d group_id=%d: %v", item.userID, item.groupID, err)
		}
	}
}
