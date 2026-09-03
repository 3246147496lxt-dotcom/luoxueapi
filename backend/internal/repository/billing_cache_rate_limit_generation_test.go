//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestBillingCacheGenerationKeysShareCanonicalHashTag(t *testing.T) {
	const (
		userID = int64(7000)
		keyID  = int64(8000)
	)

	// The legacy keys intentionally remain untagged for rolling compatibility;
	// only the canonical keys passed together to Lua need the shared tag.
	require.Equal(t, "billing:balance:7000", billingBalanceKey(userID))
	require.Equal(t, "billing:balance:{7000}", billingBalanceTaggedKey(userID))
	require.Equal(t, "billing:balance:generation:{7000}", billingBalanceGenerationKey(userID))
	require.Equal(t, "apikey:rate:8000", billingRateLimitKey(keyID))
	require.Equal(t, "apikey:rate:{8000}", billingRateLimitTaggedKey(keyID))
	require.Equal(t, "apikey:rate:generation:{8000}", billingRateLimitGenerationKey(keyID))
	require.Equal(t,
		redisClusterSlot(billingBalanceTaggedKey(userID)),
		redisClusterSlot(billingBalanceGenerationKey(userID)),
		"balance Lua KEYS must share a Redis Cluster slot",
	)
	require.Equal(t,
		redisClusterSlot(billingRateLimitTaggedKey(keyID)),
		redisClusterSlot(billingRateLimitGenerationKey(keyID)),
		"rate-limit Lua KEYS must share a Redis Cluster slot",
	)
}

// redisClusterSlot is the Redis Cluster CRC16/XMODEM slot calculation kept
// local to this regression test so it does not depend on go-redis internals.
func redisClusterSlot(key string) int {
	start := -1
	for i := 0; i < len(key); i++ {
		if key[i] == '{' {
			start = i + 1
			break
		}
	}
	if start >= 0 {
		for i := start; i < len(key); i++ {
			if key[i] == '}' {
				if i > start {
					key = key[start:i]
				}
				break
			}
		}
	}
	var crc uint16
	for i := 0; i < len(key); i++ {
		crc ^= uint16(key[i]) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return int(crc % 16384)
}

func TestBillingCacheVersionedReadsLegacySnapshotBeforeFirstInvalidation(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()

	require.NoError(t, cache.rdb.Set(ctx, billingBalanceKey(7010), "12.5", time.Minute).Err())
	balance, generation, err := cache.GetUserBalanceWithGeneration(ctx, 7010)
	require.NoError(t, err)
	require.Equal(t, 12.5, balance)
	require.Zero(t, generation)

	require.NoError(t, cache.rdb.HSet(ctx, billingRateLimitKey(8010), map[string]any{
		rateLimitFieldUsage5h:  1.5,
		rateLimitFieldUsage1d:  2.5,
		rateLimitFieldUsage7d:  3.5,
		rateLimitFieldWindow5h: 100,
		rateLimitFieldWindow1d: 200,
		rateLimitFieldWindow7d: 300,
	}).Err())
	rateData, generation, err := cache.GetAPIKeyRateLimitWithGeneration(ctx, 8010)
	require.NoError(t, err)
	require.Equal(t, 1.5, rateData.Usage5h)
	require.Equal(t, int64(100), rateData.Window5h)
	require.Zero(t, generation)
}

func TestBillingCacheVersionedReadRejectsResurrectedLegacyAfterInvalidation(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()
	const userID int64 = 7011
	const keyID int64 = 8011

	require.NoError(t, cache.InvalidateUserBalance(ctx, userID))
	require.NoError(t, cache.rdb.Set(ctx, billingBalanceKey(userID), "99", time.Minute).Err())
	_, generation, err := cache.GetUserBalanceWithGeneration(ctx, userID)
	require.ErrorIs(t, err, service.ErrBillingCacheMiss)
	require.EqualValues(t, 1, generation)
	_, err = cache.GetUserBalance(ctx, userID)
	require.ErrorIs(t, err, service.ErrBillingCacheMiss)

	require.NoError(t, cache.InvalidateAPIKeyRateLimit(ctx, keyID))
	require.NoError(t, cache.rdb.HSet(ctx, billingRateLimitKey(keyID), rateLimitFieldUsage5h, 99).Err())
	_, generation, err = cache.GetAPIKeyRateLimitWithGeneration(ctx, keyID)
	require.ErrorIs(t, err, service.ErrBillingCacheMiss)
	require.EqualValues(t, 1, generation)
	_, err = cache.GetAPIKeyRateLimit(ctx, keyID)
	require.ErrorIs(t, err, redis.Nil)
}

// TestBillingCacheAPIKeyRateLimitGenerationFence reproduces the dangerous
// miss -> DB reload -> post-commit invalidation ordering. The stale snapshot
// must be rejected by the conditional write, while a snapshot captured after
// invalidation remains writable.
func TestBillingCacheAPIKeyRateLimitGenerationFence(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()
	const keyID int64 = 7001

	_, generation, err := cache.GetAPIKeyRateLimitWithGeneration(ctx, keyID)
	require.ErrorIs(t, err, service.ErrBillingCacheMiss)
	require.Zero(t, generation)

	stale := &service.APIKeyRateLimitCacheData{Usage5h: 1.25, Usage1d: 1.25, Usage7d: 1.25}
	// This is the authoritative post-commit eviction that races the DB read.
	require.NoError(t, cache.InvalidateAPIKeyRateLimit(ctx, keyID))

	// A reader that captured generation 0 before the eviction must not be able
	// to resurrect the pre-charge snapshot.
	require.ErrorIs(t, cache.SetAPIKeyRateLimitIfGeneration(ctx, keyID, stale, generation), service.ErrBillingCacheGenerationChanged)
	_, err = cache.GetAPIKeyRateLimit(ctx, keyID)
	require.ErrorIs(t, err, redis.Nil)

	_, generation, err = cache.GetAPIKeyRateLimitWithGeneration(ctx, keyID)
	require.ErrorIs(t, err, service.ErrBillingCacheMiss)
	require.EqualValues(t, 1, generation)

	fresh := &service.APIKeyRateLimitCacheData{Usage5h: 2.5, Usage1d: 2.5, Usage7d: 2.5}
	require.NoError(t, cache.SetAPIKeyRateLimitIfGeneration(ctx, keyID, fresh, generation))
	got, err := cache.GetAPIKeyRateLimit(ctx, keyID)
	require.NoError(t, err)
	require.Equal(t, fresh.Usage5h, got.Usage5h)
	require.Equal(t, fresh.Usage1d, got.Usage1d)
	require.Equal(t, fresh.Usage7d, got.Usage7d)
}

func TestBillingCacheBalanceGenerationFence(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()
	const userID int64 = 7005

	_, generation, err := cache.GetUserBalanceWithGeneration(ctx, userID)
	require.ErrorIs(t, err, service.ErrBillingCacheMiss)
	require.Zero(t, generation)

	// Eviction advances the durable token even though the balance key is absent.
	require.NoError(t, cache.InvalidateUserBalance(ctx, userID))
	// A DB read that began at generation 0 must not resurrect its old snapshot.
	require.ErrorIs(t,
		cache.SetUserBalanceIfGeneration(ctx, userID, 10, generation),
		service.ErrBillingCacheGenerationChanged,
	)
	_, err = cache.GetUserBalance(ctx, userID)
	require.ErrorIs(t, err, service.ErrBillingCacheMiss)

	_, generation, err = cache.GetUserBalanceWithGeneration(ctx, userID)
	require.ErrorIs(t, err, service.ErrBillingCacheMiss)
	require.EqualValues(t, 1, generation)
	require.NoError(t, cache.SetUserBalanceIfGeneration(ctx, userID, 7.5, generation))
	got, err := cache.GetUserBalance(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 7.5, got)
}

func TestBillingCacheAPIKeyRateLimitInvalidationKeepsGenerationAfterIdleMiss(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()
	const keyID int64 = 7002

	// Repeated invalidations must advance a durable token even when the hash is
	// already absent. This prevents a very old in-flight read from becoming
	// valid again after the cache has been idle and its snapshot TTL elapsed.
	require.NoError(t, cache.InvalidateAPIKeyRateLimit(ctx, keyID))
	require.NoError(t, cache.InvalidateAPIKeyRateLimit(ctx, keyID))
	_, generation, err := cache.GetAPIKeyRateLimitWithGeneration(ctx, keyID)
	require.ErrorIs(t, err, service.ErrBillingCacheMiss)
	require.EqualValues(t, 2, generation)
}

func TestBillingCacheLegacyRateLimitUpdateEvictsAuthoritativeSnapshot(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()
	const keyID int64 = 7003

	require.NoError(t, cache.SetAPIKeyRateLimit(ctx, keyID, &service.APIKeyRateLimitCacheData{
		Usage5h: 1,
		Usage1d: 1,
		Usage7d: 1,
	}))
	require.NoError(t, cache.UpdateAPIKeyRateLimitUsage(ctx, keyID, 0.5))

	_, err := cache.GetAPIKeyRateLimit(ctx, keyID)
	require.ErrorIs(t, err, redis.Nil)
	_, generation, err := cache.GetAPIKeyRateLimitWithGeneration(ctx, keyID)
	require.ErrorIs(t, err, service.ErrBillingCacheMiss)
	require.EqualValues(t, 1, generation)
}

func TestBillingCacheBalanceDeductAdvancesGeneration(t *testing.T) {
	cache, _ := newMiniRedisCache(t)
	ctx := context.Background()
	const userID int64 = 7004

	require.NoError(t, cache.SetUserBalance(ctx, userID, 10))
	_, generation, err := cache.GetUserBalanceWithGeneration(ctx, userID)
	require.NoError(t, err)
	require.Zero(t, generation)

	// DeductUserBalance is retained for legacy callers, but must advance the
	// same durable token as InvalidateUserBalance. Otherwise a different process
	// could refill a stale DB snapshot after this cache-side deduction.
	require.NoError(t, cache.DeductUserBalance(ctx, userID, 2.5))
	got, generation, err := cache.GetUserBalanceWithGeneration(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, 7.5, got)
	require.EqualValues(t, 1, generation)
}
