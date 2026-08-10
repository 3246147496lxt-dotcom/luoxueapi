package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newTranscriptionAdmissionTestCache(t *testing.T) *gatewayCache {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return &gatewayCache{rdb: client}
}

func transcriptionAdmissionInput(userID int64, ipHash, keyHash, leaseID string) service.TranscriptionAdmissionCacheInput {
	return service.TranscriptionAdmissionCacheInput{
		UserID:                userID,
		IPHash:                ipHash,
		IdempotencyKeyHash:    keyHash,
		LeaseID:               leaseID,
		LeaseTTL:              2 * time.Minute,
		IdempotencyTTL:        2 * time.Minute,
		MaxConcurrentGlobal:   2,
		MaxConcurrentPerUser:  1,
		UserRequestsPerMinute: 6,
		IPRequestsPerMinute:   20,
	}
}

func TestGatewayCacheTranscriptionAdmissionIsAtomicAcrossDuplicateConcurrencyAndRelease(t *testing.T) {
	cache := newTranscriptionAdmissionTestCache(t)
	ctx := context.Background()
	first := transcriptionAdmissionInput(7, "ip-a", "key-a", "lease-a")

	decision, err := cache.AcquireTranscriptionAdmission(ctx, first)
	require.NoError(t, err)
	require.Equal(t, service.TranscriptionAdmissionAcquired, decision)

	duplicate := first
	duplicate.LeaseID = "lease-duplicate"
	decision, err = cache.AcquireTranscriptionAdmission(ctx, duplicate)
	require.NoError(t, err)
	require.Equal(t, service.TranscriptionAdmissionDuplicate, decision)

	busy := transcriptionAdmissionInput(7, "ip-a", "key-b", "lease-b")
	decision, err = cache.AcquireTranscriptionAdmission(ctx, busy)
	require.NoError(t, err)
	require.Equal(t, service.TranscriptionAdmissionBusy, decision)

	require.NoError(t, cache.ReleaseTranscriptionAdmission(ctx, 7, "lease-a"))
	decision, err = cache.AcquireTranscriptionAdmission(ctx, busy)
	require.NoError(t, err)
	require.Equal(t, service.TranscriptionAdmissionAcquired, decision)
}

func TestGatewayCacheTranscriptionAdmissionEnforcesUserAndIPRPM(t *testing.T) {
	cache := newTranscriptionAdmissionTestCache(t)
	ctx := context.Background()
	first := transcriptionAdmissionInput(7, "shared-ip", "key-a", "lease-a")
	first.UserRequestsPerMinute = 1
	first.IPRequestsPerMinute = 1
	decision, err := cache.AcquireTranscriptionAdmission(ctx, first)
	require.NoError(t, err)
	require.Equal(t, service.TranscriptionAdmissionAcquired, decision)
	require.NoError(t, cache.ReleaseTranscriptionAdmission(ctx, 7, "lease-a"))

	userLimited := transcriptionAdmissionInput(7, "other-ip", "key-b", "lease-b")
	userLimited.UserRequestsPerMinute = 1
	decision, err = cache.AcquireTranscriptionAdmission(ctx, userLimited)
	require.NoError(t, err)
	require.Equal(t, service.TranscriptionAdmissionRate, decision)

	ipLimited := transcriptionAdmissionInput(8, "shared-ip", "key-c", "lease-c")
	ipLimited.IPRequestsPerMinute = 1
	decision, err = cache.AcquireTranscriptionAdmission(ctx, ipLimited)
	require.NoError(t, err)
	require.Equal(t, service.TranscriptionAdmissionRate, decision)
}

func TestGatewayCacheTranscriptionAdmissionCountsDuplicateAndBusyAttemptsTowardRPM(t *testing.T) {
	t.Run("duplicate", func(t *testing.T) {
		cache := newTranscriptionAdmissionTestCache(t)
		ctx := context.Background()
		first := transcriptionAdmissionInput(7, "ip-a", "key-a", "lease-a")
		first.UserRequestsPerMinute = 2
		requireDecision(t, cache, ctx, first, service.TranscriptionAdmissionAcquired)

		duplicate := first
		duplicate.LeaseID = "lease-duplicate"
		requireDecision(t, cache, ctx, duplicate, service.TranscriptionAdmissionDuplicate)
		require.NoError(t, cache.ReleaseTranscriptionAdmission(ctx, 7, first.LeaseID))

		afterLimit := transcriptionAdmissionInput(7, "ip-b", "key-b", "lease-b")
		afterLimit.UserRequestsPerMinute = 2
		requireDecision(t, cache, ctx, afterLimit, service.TranscriptionAdmissionRate)
	})

	t.Run("busy", func(t *testing.T) {
		cache := newTranscriptionAdmissionTestCache(t)
		ctx := context.Background()
		first := transcriptionAdmissionInput(7, "ip-a", "key-a", "lease-a")
		first.UserRequestsPerMinute = 2
		requireDecision(t, cache, ctx, first, service.TranscriptionAdmissionAcquired)

		busy := transcriptionAdmissionInput(7, "ip-a", "key-b", "lease-b")
		busy.UserRequestsPerMinute = 2
		requireDecision(t, cache, ctx, busy, service.TranscriptionAdmissionBusy)
		require.NoError(t, cache.ReleaseTranscriptionAdmission(ctx, 7, first.LeaseID))

		afterLimit := transcriptionAdmissionInput(7, "ip-b", "key-c", "lease-c")
		afterLimit.UserRequestsPerMinute = 2
		requireDecision(t, cache, ctx, afterLimit, service.TranscriptionAdmissionRate)
	})
}

func TestGatewayCacheTranscriptionAdmissionShortLeaseDoesNotShrinkLongLeaseKeyTTL(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := &gatewayCache{rdb: client}
	ctx := context.Background()

	longLease := transcriptionAdmissionInput(7, "ip-a", "key-a", "lease-long")
	longLease.LeaseTTL = 10 * time.Minute
	longLease.IdempotencyTTL = 15 * time.Minute
	longLease.MaxConcurrentGlobal = 3
	requireDecision(t, cache, ctx, longLease, service.TranscriptionAdmissionAcquired)
	globalKey := transcriptionAdmissionPrefix + "concurrency:global"
	longTTL := mr.TTL(globalKey)
	require.GreaterOrEqual(t, longTTL, 10*time.Minute)

	shortLease := transcriptionAdmissionInput(8, "ip-b", "key-b", "lease-short")
	shortLease.LeaseTTL = time.Minute
	shortLease.IdempotencyTTL = 15 * time.Minute
	shortLease.MaxConcurrentGlobal = 3
	requireDecision(t, cache, ctx, shortLease, service.TranscriptionAdmissionAcquired)
	newTTL := mr.TTL(globalKey)
	require.GreaterOrEqual(t, newTTL, longTTL-50*time.Millisecond)
	require.Greater(t, newTTL, shortLease.LeaseTTL+time.Minute)
}

func TestGatewayCacheTranscriptionDailyAudioReservationIsAtomic(t *testing.T) {
	cache := newTranscriptionAdmissionTestCache(t)
	ctx := context.Background()

	allowed, err := cache.ReserveTranscriptionDailyAudio(ctx, 7, 119, 120)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = cache.ReserveTranscriptionDailyAudio(ctx, 7, 2, 120)
	require.NoError(t, err)
	require.False(t, allowed)
	allowed, err = cache.ReserveTranscriptionDailyAudio(ctx, 8, 120, 120)
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestGatewayCacheTranscriptionDailyAudioAppliesRaisedAndLoweredLimitsAtomically(t *testing.T) {
	cache := newTranscriptionAdmissionTestCache(t)
	ctx := context.Background()

	allowed, err := cache.ReserveTranscriptionDailyAudio(ctx, 7, 90, 100)
	require.NoError(t, err)
	require.True(t, allowed)

	// A rejected reservation must not increment the stored usage. Raising the
	// limit should therefore allow exactly the remaining 30 seconds.
	allowed, err = cache.ReserveTranscriptionDailyAudio(ctx, 7, 20, 100)
	require.NoError(t, err)
	require.False(t, allowed)
	allowed, err = cache.ReserveTranscriptionDailyAudio(ctx, 7, 30, 120)
	require.NoError(t, err)
	require.True(t, allowed)

	// Lowering the limit below today's already-reserved usage takes effect on the
	// next request and remains fail-closed without mutating that usage.
	allowed, err = cache.ReserveTranscriptionDailyAudio(ctx, 7, 1, 100)
	require.NoError(t, err)
	require.False(t, allowed)
	allowed, err = cache.ReserveTranscriptionDailyAudio(ctx, 7, 1, 121)
	require.NoError(t, err)
	require.True(t, allowed)
}

func requireDecision(
	t *testing.T,
	cache *gatewayCache,
	ctx context.Context,
	input service.TranscriptionAdmissionCacheInput,
	want service.TranscriptionAdmissionDecision,
) {
	t.Helper()
	decision, err := cache.AcquireTranscriptionAdmission(ctx, input)
	require.NoError(t, err)
	require.Equal(t, want, decision)
}
