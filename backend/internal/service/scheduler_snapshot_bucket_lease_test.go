package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type ownerTokenBucketLeaseTestCache struct {
	SchedulerCache
	mu                 sync.Mutex
	lease              SchedulerBucketRebuildLease
	acquireToken       SchedulerBucketWriteToken
	acquireTTL         time.Duration
	releaseHasDeadline bool
	releaseCtxErr      error
}

func (c *ownerTokenBucketLeaseTestCache) TryAcquireBucketRebuildLease(
	_ context.Context,
	bucket SchedulerBucket,
	token SchedulerBucketWriteToken,
	ttl time.Duration,
) (SchedulerBucketRebuildLease, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.acquireTTL = ttl
	c.acquireToken = token
	c.lease = SchedulerBucketRebuildLease{Bucket: bucket, OwnerToken: "owner-token"}
	return c.lease, true, nil
}

func (c *ownerTokenBucketLeaseTestCache) ReleaseBucketRebuildLease(ctx context.Context, lease SchedulerBucketRebuildLease) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, c.releaseHasDeadline = ctx.Deadline()
	c.releaseCtxErr = ctx.Err()
	if lease != c.lease {
		return ErrSchedulerBucketRebuildLeaseLost
	}
	return nil
}

func TestSchedulerSnapshotServiceUsesOwnerTokenBucketLeaseCapability(t *testing.T) {
	cache := &ownerTokenBucketLeaseTestCache{}
	svc := &SchedulerSnapshotService{cache: cache}
	bucket := SchedulerBucket{GroupID: 42, Platform: PlatformOpenAI, Mode: SchedulerModeSingle}
	token := SchedulerBucketWriteToken{Bucket: bucket, Generation: "generation", Epoch: 1}
	parentCtx, cancel := context.WithCancel(context.Background())

	release, acquired, err := svc.tryAcquireBucketRebuildLease(parentCtx, bucket, token)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, release)
	cancel()
	require.NoError(t, release())

	cache.mu.Lock()
	defer cache.mu.Unlock()
	require.Equal(t, token, cache.acquireToken)
	require.GreaterOrEqual(t, cache.acquireTTL, 2*schedulerBucketRebuildTimeout)
	require.True(t, cache.releaseHasDeadline)
	require.NoError(t, cache.releaseCtxErr, "release must use a fresh context after parent cancellation")
}
