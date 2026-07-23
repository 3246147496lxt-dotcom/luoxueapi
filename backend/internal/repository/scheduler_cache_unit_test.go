//go:build unit

package repository

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newSchedulerCacheUnit(t *testing.T) *schedulerCache {
	cache, _ := newSchedulerCacheUnitWithRedis(t)
	return cache
}

func newSchedulerCacheUnitWithRedis(t *testing.T) (*schedulerCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache, ok := newSchedulerCacheWithChunkSizes(rdb, defaultSchedulerSnapshotMGetChunkSize, defaultSchedulerSnapshotWriteChunkSize).(*schedulerCache)
	require.True(t, ok)
	return cache, mr
}

func (c *schedulerCache) writeAccounts(ctx context.Context, accounts []service.Account) ([]service.Account, error) {
	generation, err := c.ensureSchedulerGeneration(ctx)
	if err != nil {
		return nil, err
	}
	return c.writeAccountsForGeneration(ctx, accounts, generation)
}

func (c *schedulerCache) updateAccountLastUsedCAS(ctx context.Context, accountID int64, candidate time.Time, initial any, generation string) error {
	write, needed, err := prepareSchedulerLastUsedWrite(accountID, candidate, initial)
	if err != nil {
		return err
	}
	if !needed {
		return nil
	}
	result, err := queueSchedulerLastUsedCAS(ctx, c.rdb, write, generation).Int64()
	if err != nil {
		return err
	}
	switch result {
	case 1:
		logSchedulerLastUsedEncodingCleanup(write)
		return nil
	case 0:
		return nil
	case -1:
		return c.retryAccountLastUsedCAS(ctx, accountID, candidate, generation)
	case -2:
		return service.ErrSchedulerBucketWriteFenced
	default:
		return fmt.Errorf("update scheduler account last_used CAS returned %d", result)
	}
}

func TestSchedulerCacheWriteAccountsSkipsUnencodableTimes(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	invalidTime := time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)

	cacheable, err := cache.writeAccounts(ctx, []service.Account{
		{ID: 111, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey},
		{ID: 112, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, ExpiresAt: &invalidTime},
	})
	require.NoError(t, err)
	require.Len(t, cacheable, 1)
	require.Equal(t, int64(111), cacheable[0].ID)

	cached, err := cache.GetAccount(ctx, 111)
	require.NoError(t, err)
	require.NotNil(t, cached)

	invalid, err := cache.GetAccount(ctx, 112)
	require.NoError(t, err)
	require.Nil(t, invalid)
}

func TestSchedulerCacheSetAccountClearsUnencodablePayload(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)

	account := service.Account{ID: 113, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	require.NoError(t, cache.SetAccount(ctx, &account))

	invalidTime := time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	account.ExpiresAt = &invalidTime
	require.NoError(t, cache.SetAccount(ctx, &account))

	cached, err := cache.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, cached)

	tombstoned, err := cache.rdb.SIsMember(ctx, schedulerAccountTombstoneSetKey, strconv.FormatInt(account.ID, 10)).Result()
	require.NoError(t, err)
	require.False(t, tombstoned, "an encoding failure is cache cleanup, not a durable account deletion")

	account.ExpiresAt = nil
	require.NoError(t, cache.SetAccount(ctx, &account))
	cached, err = cache.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, cached, "a later encodable projection must be allowed after payload-only cleanup")
}

func TestSchedulerCacheDeleteAccountAtomicallyTombstonesAndDeletesBothPayloads(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	account := service.Account{ID: 122, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	require.NoError(t, cache.SetAccount(ctx, &account))

	id := strconv.FormatInt(account.ID, 10)
	require.NoError(t, cache.DeleteAccount(ctx, account.ID))
	require.NoError(t, cache.DeleteAccount(ctx, account.ID), "delete must be idempotent")

	exists, err := cache.rdb.Exists(ctx, schedulerAccountKey(id), schedulerAccountMetaKey(id)).Result()
	require.NoError(t, err)
	require.Zero(t, exists, "durable deletion must remove full and metadata projections together")
	tombstoned, err := cache.rdb.SIsMember(ctx, schedulerAccountTombstoneSetKey, id).Result()
	require.NoError(t, err)
	require.True(t, tombstoned)
	count, err := cache.rdb.SCard(ctx, schedulerAccountTombstoneSetKey).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, count, "the single tombstone set keeps idempotent deletes compact")
	ttl, err := cache.rdb.TTL(ctx, schedulerAccountTombstoneSetKey).Result()
	require.NoError(t, err)
	require.Less(t, ttl, time.Duration(0), "deletion fences must not expire while a stale rebuild can still resume")
}

func TestSchedulerCacheFirstProjectionWriteCannotResurrectAccountDeletedAfterDatabaseRead(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	staleDatabaseRow := service.Account{
		ID:        123,
		Name:      "read before delete",
		Platform:  service.PlatformOpenAI,
		Type:      service.AccountTypeAPIKey,
		UpdatedAt: time.Date(2026, 7, 23, 7, 30, 0, 0, time.UTC),
	}

	// The rebuild already owns staleDatabaseRow, but the outbox deletion reaches
	// Redis before this account has ever been projected. Its initial MGET will
	// therefore observe a missing payload rather than an older CAS version.
	require.NoError(t, cache.DeleteAccount(ctx, staleDatabaseRow.ID))
	written, err := cache.writeAccounts(ctx, []service.Account{staleDatabaseRow})
	require.NoError(t, err)
	require.Empty(t, written)

	id := strconv.FormatInt(staleDatabaseRow.ID, 10)
	exists, err := cache.rdb.Exists(ctx, schedulerAccountKey(id), schedulerAccountMetaKey(id)).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}

func TestSchedulerCacheUpdateLastUsedClearsUnencodablePayload(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	account := service.Account{ID: 114, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	require.NoError(t, cache.SetAccount(ctx, &account))

	invalidTime := time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, cache.UpdateLastUsed(ctx, map[int64]time.Time{account.ID: invalidTime}))

	cached, err := cache.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, cached)
}

func TestSchedulerCacheUpdateLastUsedIsMonotonicAndUpdatesBothPayloads(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	base := time.Date(2026, 7, 23, 1, 0, 0, 0, time.UTC)
	newer := base.Add(2 * time.Minute)
	older := base.Add(time.Minute)
	account := service.Account{
		ID:         115,
		Platform:   service.PlatformOpenAI,
		Type:       service.AccountTypeAPIKey,
		LastUsedAt: &base,
	}
	require.NoError(t, cache.SetAccount(ctx, &account))

	require.NoError(t, cache.UpdateLastUsed(ctx, map[int64]time.Time{account.ID: newer}))
	require.NoError(t, cache.UpdateLastUsed(ctx, map[int64]time.Time{account.ID: older}))

	full, err := cache.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, full)
	require.NotNil(t, full.LastUsedAt)
	require.True(t, full.LastUsedAt.Equal(newer))
	metaRaw, err := cache.rdb.Get(ctx, schedulerAccountMetaKey(strconv.FormatInt(account.ID, 10))).Result()
	require.NoError(t, err)
	meta, err := decodeCachedAccount(metaRaw)
	require.NoError(t, err)
	require.NotNil(t, meta.LastUsedAt)
	require.True(t, meta.LastUsedAt.Equal(newer))
}

func TestSchedulerCacheUpdateLastUsedConcurrentFinalValueIsMaximum(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	account := service.Account{ID: 116, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	require.NoError(t, cache.SetAccount(ctx, &account))

	base := time.Date(2026, 7, 23, 2, 0, 0, 0, time.UTC)
	updates := make([]time.Time, 16)
	for i := range updates {
		updates[i] = base.Add(time.Duration(i) * time.Second)
	}
	errCh := make(chan error, len(updates))
	var wg sync.WaitGroup
	for _, updatedAt := range updates {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- cache.UpdateLastUsed(ctx, map[int64]time.Time{account.ID: updatedAt})
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	cached, err := cache.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, cached)
	require.NotNil(t, cached.LastUsedAt)
	require.True(t, cached.LastUsedAt.Equal(updates[len(updates)-1]))
}

type schedulerCachePipelineProbe struct {
	mu                sync.Mutex
	evalPipelineSizes []int
	directEvalCalls   int
	afterFirstMGet    func()
	mgetOnce          sync.Once
}

func (p *schedulerCachePipelineProbe) DialHook(next redis.DialHook) redis.DialHook { return next }

func (p *schedulerCachePipelineProbe) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		name := strings.ToLower(cmd.Name())
		if strings.HasPrefix(name, "eval") && isSchedulerLastUsedEval(cmd) {
			p.mu.Lock()
			p.directEvalCalls++
			p.mu.Unlock()
		}
		if name == "mget" && p.afterFirstMGet != nil {
			p.mgetOnce.Do(p.afterFirstMGet)
		}
		return err
	}
}

func (p *schedulerCachePipelineProbe) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, commands []redis.Cmder) error {
		evalCount := 0
		for _, command := range commands {
			if strings.HasPrefix(strings.ToLower(command.Name()), "eval") && isSchedulerLastUsedEval(command) {
				evalCount++
			}
		}
		if evalCount > 0 {
			p.mu.Lock()
			p.evalPipelineSizes = append(p.evalPipelineSizes, evalCount)
			p.mu.Unlock()
		}
		return next(ctx, commands)
	}
}

func isSchedulerLastUsedEval(command redis.Cmder) bool {
	args := command.Args()
	if len(args) < 6 || fmt.Sprint(args[2]) != "5" {
		return false
	}
	return strings.HasPrefix(fmt.Sprint(args[3]), schedulerAccountPrefix) &&
		strings.HasPrefix(fmt.Sprint(args[4]), schedulerAccountMetaPrefix) &&
		fmt.Sprint(args[5]) == schedulerAccountTombstoneSetKey
}

func TestSchedulerCacheUpdateLastUsedPipelinesFirstRoundInChunks(t *testing.T) {
	ctx := context.Background()
	cache, ok := newSchedulerCacheWithChunkSizes(
		redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()}),
		128,
		2,
	).(*schedulerCache)
	require.True(t, ok)
	t.Cleanup(func() { _ = cache.rdb.Close() })
	updates := make(map[int64]time.Time, 5)
	for id := int64(2101); id <= 2105; id++ {
		account := service.Account{ID: id, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
		require.NoError(t, cache.SetAccount(ctx, &account))
		updates[id] = time.Date(2026, 7, 23, 8, 0, int(id-2101), 0, time.UTC)
	}
	require.NoError(t, updateLastUsedCASScript.Load(ctx, cache.rdb).Err())
	probe := &schedulerCachePipelineProbe{}
	cache.rdb.AddHook(probe)

	require.NoError(t, cache.UpdateLastUsed(ctx, updates))
	probe.mu.Lock()
	defer probe.mu.Unlock()
	require.Equal(t, []int{2, 2, 1}, probe.evalPipelineSizes)
	require.Zero(t, probe.directEvalCalls, "conflict-free items must not fall back to per-account scripts")
}

func TestSchedulerCacheUpdateLastUsedRetriesOnlyConflictedItem(t *testing.T) {
	ctx := context.Background()
	cache, mr := newSchedulerCacheUnitWithRedis(t)
	secondary := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = secondary.Close() })
	base := time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC)
	for _, id := range []int64{2201, 2202} {
		account := service.Account{ID: id, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, LastUsedAt: ptrTime(base)}
		require.NoError(t, cache.SetAccount(ctx, &account))
	}
	require.NoError(t, updateLastUsedCASScript.Load(ctx, cache.rdb).Err())
	probe := &schedulerCachePipelineProbe{}
	probe.afterFirstMGet = func() {
		id := strconv.FormatInt(2201, 10)
		raw, err := secondary.Get(ctx, schedulerAccountKey(id)).Result()
		require.NoError(t, err)
		account, err := decodeCachedAccount(raw)
		require.NoError(t, err)
		account.LastUsedAt = ptrTime(base.Add(time.Minute))
		full, meta, err := marshalSchedulerCacheAccount(*account)
		require.NoError(t, err)
		require.NoError(t, secondary.MSet(ctx,
			schedulerAccountKey(id), full,
			schedulerAccountMetaKey(id), meta,
		).Err())
	}
	cache.rdb.AddHook(probe)
	candidate := base.Add(2 * time.Minute)
	require.NoError(t, cache.UpdateLastUsed(ctx, map[int64]time.Time{2201: candidate, 2202: candidate}))

	probe.mu.Lock()
	require.Equal(t, []int{2}, probe.evalPipelineSizes)
	require.Equal(t, 1, probe.directEvalCalls, "only the CAS conflict should be retried individually")
	probe.mu.Unlock()
	for _, id := range []int64{2201, 2202} {
		account, err := cache.GetAccount(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, account.LastUsedAt)
		require.True(t, account.LastUsedAt.Equal(candidate))
	}
}

func TestSchedulerCacheSetAccountPreservesNewerProjectionAndLastUsed(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	base := time.Date(2026, 7, 23, 3, 0, 0, 0, time.UTC)
	currentLastUsed := base.Add(3 * time.Minute)
	current := service.Account{
		ID:         117,
		Name:       "newer projection",
		Platform:   service.PlatformOpenAI,
		Type:       service.AccountTypeAPIKey,
		UpdatedAt:  base.Add(2 * time.Minute),
		LastUsedAt: ptrTime(base.Add(time.Minute)),
	}
	require.NoError(t, cache.SetAccount(ctx, &current))
	require.NoError(t, cache.UpdateLastUsed(ctx, map[int64]time.Time{current.ID: currentLastUsed}))

	stale := current
	stale.Name = "stale projection"
	stale.UpdatedAt = base
	stale.LastUsedAt = ptrTime(base.Add(2 * time.Minute))
	require.NoError(t, cache.SetAccount(ctx, &stale))

	assertSchedulerAccountProjection(t, ctx, cache, current.ID, "newer projection", current.UpdatedAt, currentLastUsed)
}

func TestSchedulerCacheRebuildPreservesNewerProjectionAndLastUsed(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	base := time.Date(2026, 7, 23, 4, 0, 0, 0, time.UTC)
	currentLastUsed := base.Add(4 * time.Minute)
	current := service.Account{
		ID:         118,
		Name:       "newer projection",
		Platform:   service.PlatformOpenAI,
		Type:       service.AccountTypeAPIKey,
		UpdatedAt:  base.Add(3 * time.Minute),
		LastUsedAt: ptrTime(base.Add(time.Minute)),
	}
	require.NoError(t, cache.SetAccount(ctx, &current))
	require.NoError(t, cache.UpdateLastUsed(ctx, map[int64]time.Time{current.ID: currentLastUsed}))

	stale := current
	stale.Name = "stale rebuild"
	stale.UpdatedAt = base
	stale.LastUsedAt = nil
	written, err := cache.writeAccounts(ctx, []service.Account{stale})
	require.NoError(t, err)
	require.Len(t, written, 1)
	require.Equal(t, "newer projection", written[0].Name)
	require.NotNil(t, written[0].LastUsedAt)
	require.True(t, written[0].LastUsedAt.Equal(currentLastUsed))

	assertSchedulerAccountProjection(t, ctx, cache, current.ID, "newer projection", current.UpdatedAt, currentLastUsed)
}

func TestSchedulerCacheProjectionCASRetriesWithoutRegressingLastUsed(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	base := time.Date(2026, 7, 23, 5, 0, 0, 0, time.UTC)
	account := service.Account{
		ID:         119,
		Name:       "account",
		Platform:   service.PlatformOpenAI,
		Type:       service.AccountTypeAPIKey,
		UpdatedAt:  base,
		LastUsedAt: ptrTime(base),
	}
	require.NoError(t, cache.SetAccount(ctx, &account))

	initial, err := cache.rdb.Get(ctx, schedulerAccountKey(strconv.FormatInt(account.ID, 10))).Result()
	require.NoError(t, err)
	staleWrite, err := prepareSchedulerAccountProjection(account, initial)
	require.NoError(t, err)
	generation, err := cache.rdb.Get(ctx, schedulerGenerationKey).Result()
	require.NoError(t, err)

	newerLastUsed := base.Add(time.Minute)
	require.NoError(t, cache.UpdateLastUsed(ctx, map[int64]time.Time{account.ID: newerLastUsed}))
	result, err := queueSchedulerAccountProjection(ctx, cache.rdb, staleWrite, "set", generation).Int64()
	require.NoError(t, err)
	require.EqualValues(t, -1, result, "the prepared stale write must lose its CAS")

	written, ok, err := cache.retrySchedulerAccountProjection(ctx, account, generation)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, written.LastUsedAt)
	require.True(t, written.LastUsedAt.Equal(newerLastUsed))
	assertSchedulerAccountProjection(t, ctx, cache, account.ID, "account", account.UpdatedAt, newerLastUsed)
}

func TestSchedulerCacheProjectionRetryDoesNotResurrectConcurrentDeletion(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	account := service.Account{
		ID:        120,
		Name:      "deleted",
		Platform:  service.PlatformOpenAI,
		Type:      service.AccountTypeAPIKey,
		UpdatedAt: time.Date(2026, 7, 23, 6, 0, 0, 0, time.UTC),
	}
	require.NoError(t, cache.SetAccount(ctx, &account))

	initial, err := cache.rdb.Get(ctx, schedulerAccountKey(strconv.FormatInt(account.ID, 10))).Result()
	require.NoError(t, err)
	staleWrite, err := prepareSchedulerAccountProjection(account, initial)
	require.NoError(t, err)
	generation, err := cache.rdb.Get(ctx, schedulerGenerationKey).Result()
	require.NoError(t, err)
	require.NoError(t, cache.DeleteAccount(ctx, account.ID))

	result, err := queueSchedulerAccountProjection(ctx, cache.rdb, staleWrite, "set", generation).Int64()
	require.NoError(t, err)
	require.EqualValues(t, 0, result, "the tombstone must reject a write prepared before deletion")
	_, ok, err := cache.retrySchedulerAccountProjection(ctx, account, generation)
	require.NoError(t, err)
	require.False(t, ok)

	full, err := cache.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, full)
	_, err = cache.rdb.Get(ctx, schedulerAccountMetaKey(strconv.FormatInt(account.ID, 10))).Result()
	require.ErrorIs(t, err, redis.Nil)
}

func TestSchedulerCacheUpdateLastUsedPreparedBeforeDeleteCannotRecreatePayloads(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	base := time.Date(2026, 7, 23, 6, 30, 0, 0, time.UTC)
	account := service.Account{
		ID:         124,
		Platform:   service.PlatformOpenAI,
		Type:       service.AccountTypeAPIKey,
		LastUsedAt: ptrTime(base),
	}
	require.NoError(t, cache.SetAccount(ctx, &account))

	id := strconv.FormatInt(account.ID, 10)
	preparedPayload, err := cache.rdb.Get(ctx, schedulerAccountKey(id)).Result()
	require.NoError(t, err)
	generation, err := cache.rdb.Get(ctx, schedulerGenerationKey).Result()
	require.NoError(t, err)
	require.NoError(t, cache.DeleteAccount(ctx, account.ID))
	require.NoError(t, cache.updateAccountLastUsedCAS(ctx, account.ID, base.Add(time.Minute), preparedPayload, generation))

	exists, err := cache.rdb.Exists(ctx, schedulerAccountKey(id), schedulerAccountMetaKey(id)).Result()
	require.NoError(t, err)
	require.Zero(t, exists, "last_used CAS must honor a deletion that won after its initial read")
	tombstoned, err := cache.rdb.SIsMember(ctx, schedulerAccountTombstoneSetKey, id).Result()
	require.NoError(t, err)
	require.True(t, tombstoned)
}

func TestSchedulerCacheGenerationFencesWritersAcrossRedisFlush(t *testing.T) {
	ctx := context.Background()
	cache, mr := newSchedulerCacheUnitWithRedis(t)
	bucket := service.SchedulerBucket{GroupID: 125, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	oldAccount := service.Account{ID: 1251, Name: "old", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	oldToken, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	oldVersion, err := cache.allocateSnapshotVersion(ctx, bucket, oldToken)
	require.NoError(t, err)
	require.NoError(t, cache.SetAccount(ctx, &oldAccount))
	oldPayload, err := cache.rdb.Get(ctx, schedulerAccountKey(strconv.FormatInt(oldAccount.ID, 10))).Result()
	require.NoError(t, err)
	oldProjection, err := prepareSchedulerAccountProjection(oldAccount, oldPayload)
	require.NoError(t, err)

	mr.FlushAll()
	newToken, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	require.NotEqual(t, oldToken.Generation, newToken.Generation)
	require.NotEqual(t, oldToken.Epoch, newToken.Epoch)
	newAccount := service.Account{ID: 1252, Name: "new", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	require.NoError(t, cache.SetSnapshot(ctx, bucket, newToken, []service.Account{newAccount}))
	newActive, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerActivePrefix, bucket)).Result()
	require.NoError(t, err)

	// Simulate an old writer that had already allocated and is resuming after the
	// flush. Generation checks reject both projection and activation writes.
	result, err := queueSchedulerAccountProjection(ctx, cache.rdb, oldProjection, "set", oldToken.Generation).Int64()
	require.NoError(t, err)
	require.EqualValues(t, -2, result)
	require.NoError(t, cache.rdb.ZAdd(ctx, schedulerSnapshotKey(bucket, oldVersion), redis.Z{Score: 0, Member: oldAccount.ID}).Err())
	err = cache.activateSnapshotVersion(ctx, bucket, oldToken, oldVersion)
	require.ErrorIs(t, err, service.ErrSchedulerBucketWriteFenced)
	activeAfterOldWriter, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerActivePrefix, bucket)).Result()
	require.NoError(t, err)
	require.Equal(t, newActive, activeAfterOldWriter)

	oldCached, err := cache.GetAccount(ctx, oldAccount.ID)
	require.NoError(t, err)
	require.Nil(t, oldCached)
	snapshot, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, snapshot, 1)
	require.Equal(t, newAccount.ID, snapshot[0].ID)
}

func TestSchedulerCacheEpochRoundTripsAtLuaSafeIntegerBoundary(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	bucket := service.SchedulerBucket{GroupID: 126, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeForced}
	require.NoError(t, cache.rdb.Set(ctx, schedulerGenerationKey, "exact-generation", 0).Err())
	require.NoError(t, cache.rdb.Set(ctx, schedulerBucketKey(schedulerEpochPrefix, bucket), strconv.FormatInt(schedulerMaxSafeLuaInteger, 10), 0).Err())

	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	require.Equal(t, schedulerMaxSafeLuaInteger, token.Epoch)
	require.Equal(t, "exact-generation", token.Generation)
	require.ErrorIs(t, cache.RetireBucket(ctx, bucket), service.ErrSchedulerBucketWriteFenced)
}

func TestSchedulerCacheAuthoritativeResetRotatesAndClearsNamespace(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	bucket := service.SchedulerBucket{GroupID: 127, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	live := service.Account{ID: 1271, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	deleted := service.Account{ID: 1272, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	require.NoError(t, cache.SetAccount(ctx, &live))
	require.NoError(t, cache.DeleteAccount(ctx, deleted.ID))
	oldToken, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	require.NoError(t, cache.SetSnapshot(ctx, bucket, oldToken, []service.Account{live}))
	require.NoError(t, cache.SetOutboxWatermark(ctx, 99))

	require.NoError(t, cache.ResetForAuthoritativeRebuild(ctx))
	newGeneration, err := cache.rdb.Get(ctx, schedulerGenerationKey).Result()
	require.NoError(t, err)
	require.NotEqual(t, oldToken.Generation, newGeneration)
	exists, err := cache.rdb.Exists(
		ctx,
		schedulerResetKey,
		schedulerResetLeaseKey,
		schedulerAccountKey(strconv.FormatInt(live.ID, 10)),
		schedulerAccountMetaKey(strconv.FormatInt(live.ID, 10)),
		schedulerAccountTombstoneSetKey,
		schedulerBucketKey(schedulerActivePrefix, bucket),
		schedulerOutboxWatermarkKey,
	).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
	require.ErrorIs(t, cache.SetSnapshot(ctx, bucket, oldToken, []service.Account{live}), service.ErrSchedulerBucketWriteFenced)

	// The restored PostgreSQL authority may legitimately contain an ID that was
	// tombstoned by the newer database, so reset removes the old tombstone set.
	require.NoError(t, cache.SetAccount(ctx, &deleted))
	restored, err := cache.GetAccount(ctx, deleted.ID)
	require.NoError(t, err)
	require.NotNil(t, restored)
}

func TestSchedulerCacheResetBarrierFailsClosed(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	account := service.Account{ID: 1281, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	bucket := service.SchedulerBucket{GroupID: 128, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	rebuildToken, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	require.NoError(t, cache.SetSnapshot(ctx, bucket, rebuildToken, []service.Account{account}))
	require.NoError(t, cache.SetOutboxWatermark(ctx, 12))
	require.NoError(t, cache.rdb.Set(ctx, schedulerGenerationKey, "reset-generation", 0).Err())
	require.NoError(t, cache.rdb.Set(ctx, schedulerResetKey, "owner", 0).Err())

	err = cache.SetAccount(ctx, &account)
	require.ErrorIs(t, err, service.ErrSchedulerCacheResetInProgress)
	_, err = cache.CaptureBucketWriteToken(ctx, bucket)
	require.ErrorIs(t, err, service.ErrSchedulerBucketWriteFenced)
	_, err = cache.GetAccount(ctx, account.ID)
	require.ErrorIs(t, err, service.ErrSchedulerCacheResetInProgress)
	_, _, err = cache.GetSnapshot(ctx, bucket)
	require.ErrorIs(t, err, service.ErrSchedulerCacheResetInProgress)
	_, err = cache.ListBuckets(ctx)
	require.ErrorIs(t, err, service.ErrSchedulerCacheResetInProgress)
	_, err = cache.GetOutboxWatermark(ctx)
	require.ErrorIs(t, err, service.ErrSchedulerCacheResetInProgress)
}

func TestSchedulerCacheInterruptedAuthoritativeResetCanBeTakenOver(t *testing.T) {
	ctx := context.Background()
	cache, mr := newSchedulerCacheUnitWithRedis(t)
	account := service.Account{ID: 1291, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	require.NoError(t, cache.SetAccount(ctx, &account))

	started, err := beginSchedulerAuthoritativeResetScript.Run(
		ctx,
		cache.rdb,
		[]string{schedulerGenerationKey, schedulerResetKey, schedulerResetLeaseKey},
		"interrupted-owner",
		"interrupted-generation",
		schedulerAuthoritativeResetLeaseTTL.Milliseconds(),
	).Int64()
	require.NoError(t, err)
	require.EqualValues(t, 1, started)

	mr.FastForward(schedulerAuthoritativeResetLeaseTTL + time.Second)
	exists, err := cache.rdb.Exists(ctx, schedulerResetKey, schedulerResetLeaseKey).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, exists, "durable reset marker must outlive the expired owner lease")
	_, err = cache.GetAccount(ctx, account.ID)
	require.ErrorIs(t, err, service.ErrSchedulerCacheResetInProgress)

	recovered, err := cache.RecoverInterruptedAuthoritativeReset(ctx)
	require.NoError(t, err)
	require.True(t, recovered)
	exists, err = cache.rdb.Exists(ctx, schedulerResetKey, schedulerResetLeaseKey).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
	accountExists, err := cache.rdb.Exists(ctx, schedulerAccountKey(strconv.FormatInt(account.ID, 10))).Result()
	require.NoError(t, err)
	require.Zero(t, accountExists)
}

func TestSchedulerCacheAuthoritativeResetRecoveryIsNoopWithoutMarker(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	generation, err := cache.ensureSchedulerGeneration(ctx)
	require.NoError(t, err)

	recovered, err := cache.RecoverInterruptedAuthoritativeReset(ctx)
	require.NoError(t, err)
	require.False(t, recovered)
	current, err := cache.rdb.Get(ctx, schedulerGenerationKey).Result()
	require.NoError(t, err)
	require.Equal(t, generation, current)
}

func TestSchedulerCacheAuthoritativeResetRecoveryWaitsForLiveOwner(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	started, err := beginSchedulerAuthoritativeResetScript.Run(
		ctx,
		cache.rdb,
		[]string{schedulerGenerationKey, schedulerResetKey, schedulerResetLeaseKey},
		"live-owner",
		"live-generation",
		schedulerAuthoritativeResetLeaseTTL.Milliseconds(),
	).Int64()
	require.NoError(t, err)
	require.EqualValues(t, 1, started)

	recovered, err := cache.RecoverInterruptedAuthoritativeReset(ctx)
	require.ErrorIs(t, err, service.ErrSchedulerCacheResetInProgress)
	require.False(t, recovered)
}

func TestSchedulerCacheResetTakeoverFencesOldOwnerDeleteChunks(t *testing.T) {
	ctx := context.Background()
	cache, mr := newSchedulerCacheUnitWithRedis(t)
	leaseTTLMillis := schedulerAuthoritativeResetLeaseTTL.Milliseconds()
	start := func(owner, generation string) int64 {
		started, err := beginSchedulerAuthoritativeResetScript.Run(
			ctx,
			cache.rdb,
			[]string{schedulerGenerationKey, schedulerResetKey, schedulerResetLeaseKey},
			owner,
			generation,
			leaseTTLMillis,
		).Int64()
		require.NoError(t, err)
		return started
	}

	require.EqualValues(t, 1, start("old-owner", "old-generation"))
	mr.FastForward(schedulerAuthoritativeResetLeaseTTL + time.Second)
	require.EqualValues(t, 1, start("new-owner", "new-generation"))

	victimKey := schedulerAccountKey("1292")
	require.NoError(t, cache.rdb.Set(ctx, victimKey, "new-owner-data", 0).Err())
	err := cache.deleteAuthoritativeResetKeys(ctx, "old-owner", "old-generation", []string{victimKey})
	require.ErrorContains(t, err, "ownership was lost")
	value, err := cache.rdb.Get(ctx, victimKey).Result()
	require.NoError(t, err)
	require.Equal(t, "new-owner-data", value)

	require.NoError(t, cache.deleteAuthoritativeResetKeys(ctx, "new-owner", "new-generation", []string{victimKey}))
	finished, err := finishSchedulerAuthoritativeResetScript.Run(
		ctx,
		cache.rdb,
		[]string{schedulerGenerationKey, schedulerResetKey, schedulerResetLeaseKey},
		"new-owner",
		"new-generation",
	).Int64()
	require.NoError(t, err)
	require.EqualValues(t, 1, finished)
}

func TestSchedulerCacheUnencodableStaleProjectionDoesNotDeleteNewerValue(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	base := time.Date(2026, 7, 23, 7, 0, 0, 0, time.UTC)
	current := service.Account{
		ID:        121,
		Name:      "newer",
		Platform:  service.PlatformOpenAI,
		Type:      service.AccountTypeAPIKey,
		UpdatedAt: base.Add(time.Minute),
	}
	require.NoError(t, cache.SetAccount(ctx, &current))

	invalidTime := time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	staleInvalid := current
	staleInvalid.Name = "stale invalid"
	staleInvalid.UpdatedAt = base
	staleInvalid.ExpiresAt = &invalidTime
	require.NoError(t, cache.SetAccount(ctx, &staleInvalid))

	cached, err := cache.GetAccount(ctx, current.ID)
	require.NoError(t, err)
	require.NotNil(t, cached)
	require.Equal(t, "newer", cached.Name)
}

func assertSchedulerAccountProjection(
	t *testing.T,
	ctx context.Context,
	cache *schedulerCache,
	accountID int64,
	wantName string,
	wantUpdatedAt time.Time,
	wantLastUsed time.Time,
) {
	t.Helper()
	id := strconv.FormatInt(accountID, 10)
	fullRaw, err := cache.rdb.Get(ctx, schedulerAccountKey(id)).Result()
	require.NoError(t, err)
	metaRaw, err := cache.rdb.Get(ctx, schedulerAccountMetaKey(id)).Result()
	require.NoError(t, err)
	full, err := decodeCachedAccount(fullRaw)
	require.NoError(t, err)
	meta, err := decodeCachedAccount(metaRaw)
	require.NoError(t, err)
	require.Equal(t, wantName, full.Name)
	require.True(t, full.UpdatedAt.Equal(wantUpdatedAt))
	require.NotNil(t, full.LastUsedAt)
	require.NotNil(t, meta.LastUsedAt)
	require.True(t, full.LastUsedAt.Equal(wantLastUsed))
	require.True(t, meta.LastUsedAt.Equal(wantLastUsed))
}

func TestSchedulerCacheOutboxWatermarkNeverRegresses(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)

	require.NoError(t, cache.SetOutboxWatermark(ctx, 300))
	require.NoError(t, cache.SetOutboxWatermark(ctx, 200))
	watermark, err := cache.GetOutboxWatermark(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(300), watermark)
}

func TestSchedulerCacheBucketRebuildLeaseStaleOwnerCannotDeleteSuccessor(t *testing.T) {
	ctx := context.Background()
	cache, mr := newSchedulerCacheUnitWithRedis(t)
	bucket := service.SchedulerBucket{GroupID: 25, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)

	first, acquired, err := cache.TryAcquireBucketRebuildLease(ctx, bucket, token, time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
	require.True(t, first.ValidFor(bucket))
	mr.FastForward(time.Minute + time.Second)

	second, acquired, err := cache.TryAcquireBucketRebuildLease(ctx, bucket, token, time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotEqual(t, first.OwnerToken, second.OwnerToken)
	require.ErrorIs(t, cache.ReleaseBucketRebuildLease(ctx, first), service.ErrSchedulerBucketRebuildLeaseLost)
	owner, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerLockPrefix, bucket)).Result()
	require.NoError(t, err)
	require.Equal(t, second.OwnerToken, owner)
	require.NoError(t, cache.ReleaseBucketRebuildLease(ctx, second))
}

func TestSchedulerCacheBucketRebuildLeaseRejectsResetAndOldGeneration(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	bucket := service.SchedulerBucket{GroupID: 26, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)

	require.NoError(t, cache.rdb.Set(ctx, schedulerResetKey, "reset-owner", 0).Err())
	_, acquired, err := cache.TryAcquireBucketRebuildLease(ctx, bucket, token, time.Minute)
	require.ErrorIs(t, err, service.ErrSchedulerBucketWriteFenced)
	require.False(t, acquired)
	require.NoError(t, cache.rdb.Del(ctx, schedulerResetKey).Err())

	require.NoError(t, cache.rdb.Set(ctx, schedulerGenerationKey, "next-generation", 0).Err())
	_, acquired, err = cache.TryAcquireBucketRebuildLease(ctx, bucket, token, time.Minute)
	require.ErrorIs(t, err, service.ErrSchedulerBucketWriteFenced)
	require.False(t, acquired)
	lockExists, err := cache.rdb.Exists(ctx, schedulerBucketKey(schedulerLockPrefix, bucket)).Result()
	require.NoError(t, err)
	require.Zero(t, lockExists)
}

func TestSchedulerCacheSnapshotStagingExpiresUntilActivation(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	bucket := service.SchedulerBucket{GroupID: 27, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	version, err := cache.allocateSnapshotVersion(ctx, bucket, token)
	require.NoError(t, err)
	require.NoError(t, cache.writeSnapshotAccountIDs(ctx, bucket, version, []int64{271, 272}))

	snapshotKey := schedulerSnapshotKey(bucket, version)
	ttl, err := cache.rdb.TTL(ctx, snapshotKey).Result()
	require.NoError(t, err)
	require.Greater(t, ttl, time.Duration(0))
	require.LessOrEqual(t, ttl, schedulerSnapshotStagingTTL)
	require.NoError(t, cache.activateSnapshotVersion(ctx, bucket, token, version))
	ttl, err = cache.rdb.TTL(ctx, snapshotKey).Result()
	require.NoError(t, err)
	require.Equal(t, time.Duration(-1), ttl)
}

func TestSchedulerCacheSnapshotStagingWriteIsGenerationFenced(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	bucket := service.SchedulerBucket{GroupID: 28, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	version, err := cache.allocateSnapshotVersion(ctx, bucket, token)
	require.NoError(t, err)
	require.NoError(t, cache.rdb.Set(ctx, schedulerResetKey, "reset-owner", 0).Err())

	err = cache.writeSnapshotAccountIDs(ctx, bucket, version, []int64{281})
	require.ErrorIs(t, err, service.ErrSchedulerBucketWriteFenced)
	exists, err := cache.rdb.Exists(ctx, schedulerSnapshotKey(bucket, version)).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}

func TestSchedulerCacheSnapshotAccountIDReusePreservesPayloadAndMembers(t *testing.T) {
	ctx := context.Background()
	cache, _ := newSchedulerCacheUnitWithRedis(t)
	invalidTime := time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	validOne := service.Account{
		ID:          701,
		Name:        "first",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Credentials: map[string]any{"model_mapping": map[string]any{"z": "last", "a": "first"}},
		Extra:       map[string]any{"mixed_scheduling": true},
		GroupIDs:    []int64{17},
	}
	validTwo := service.Account{ID: 702, Name: "second", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	invalid := service.Account{ID: 799, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, ExpiresAt: &invalidTime}
	accounts := []service.Account{validOne, invalid, validTwo, validOne}

	single := service.SchedulerBucket{GroupID: 17, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	singleToken, err := cache.CaptureBucketWriteToken(ctx, single)
	require.NoError(t, err)
	accountIDs, err := cache.SetSnapshotAndReturnAccountIDs(ctx, single, singleToken, accounts)
	require.NoError(t, err)
	require.Equal(t, []int64{701, 702, 701}, accountIDs, "应保留可编码账号的原顺序和重复项")

	wantFull, err := json.Marshal(validOne)
	require.NoError(t, err)
	wantMeta, err := json.Marshal(buildSchedulerMetadataAccount(validOne))
	require.NoError(t, err)
	fullBefore, err := cache.rdb.Get(ctx, schedulerAccountKey("701")).Bytes()
	require.NoError(t, err)
	metaBefore, err := cache.rdb.Get(ctx, schedulerAccountMetaKey("701")).Bytes()
	require.NoError(t, err)
	require.Equal(t, wantFull, fullBefore)
	require.Equal(t, wantMeta, metaBefore)

	forced := service.SchedulerBucket{GroupID: 17, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeForced}
	forcedToken, err := cache.CaptureBucketWriteToken(ctx, forced)
	require.NoError(t, err)
	require.NoError(t, cache.SetSnapshotByAccountIDs(ctx, forced, forcedToken, accountIDs))

	fullAfter, err := cache.rdb.Get(ctx, schedulerAccountKey("701")).Bytes()
	require.NoError(t, err)
	metaAfter, err := cache.rdb.Get(ctx, schedulerAccountMetaKey("701")).Bytes()
	require.NoError(t, err)
	require.Equal(t, fullBefore, fullAfter, "ID-only 路径不得重写完整账号键")
	require.Equal(t, metaBefore, metaAfter, "ID-only 路径不得重写调度元数据键")

	for _, bucket := range []service.SchedulerBucket{single, forced} {
		version, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerActivePrefix, bucket)).Result()
		require.NoError(t, err)
		members, err := cache.rdb.ZRange(ctx, schedulerSnapshotKey(bucket, version), 0, -1).Result()
		require.NoError(t, err)
		require.Equal(t, []string{"702", "701"}, members, bucket.String())
	}
	missing, err := cache.GetAccount(ctx, invalid.ID)
	require.NoError(t, err)
	require.Nil(t, missing)
}

func TestSchedulerCacheSnapshotAccountIDReuseKeepsEmptySnapshotSemantics(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	invalidTime := time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	accounts := []service.Account{{ID: 811, Platform: service.PlatformOpenAI, ExpiresAt: &invalidTime}}

	single := service.SchedulerBucket{GroupID: 18, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	singleToken, err := cache.CaptureBucketWriteToken(ctx, single)
	require.NoError(t, err)
	accountIDs, err := cache.SetSnapshotAndReturnAccountIDs(ctx, single, singleToken, accounts)
	require.NoError(t, err)
	require.Empty(t, accountIDs)

	forced := service.SchedulerBucket{GroupID: 18, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeForced}
	forcedToken, err := cache.CaptureBucketWriteToken(ctx, forced)
	require.NoError(t, err)
	require.NoError(t, cache.SetSnapshotByAccountIDs(ctx, forced, forcedToken, accountIDs))

	for _, bucket := range []service.SchedulerBucket{single, forced} {
		ready, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerReadyPrefix, bucket)).Result()
		require.NoError(t, err)
		require.Equal(t, "1", ready)
		snapshot, hit, err := cache.GetSnapshot(ctx, bucket)
		require.NoError(t, err)
		require.False(t, hit, bucket.String())
		require.Nil(t, snapshot)
	}
}

func TestSchedulerCacheSetSnapshotByAccountIDsKeepsFencing(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	bucket := service.SchedulerBucket{GroupID: 19, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeForced}

	err := cache.SetSnapshotByAccountIDs(ctx, bucket, service.SchedulerBucketWriteToken{}, []int64{901})
	require.ErrorIs(t, err, service.ErrSchedulerBucketWriteFenced)
	_, err = cache.rdb.Get(ctx, schedulerBucketKey(schedulerVersionPrefix, bucket)).Result()
	require.ErrorIs(t, err, redis.Nil)

	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	require.NoError(t, cache.RetireBucket(ctx, bucket))
	err = cache.SetSnapshotByAccountIDs(ctx, bucket, token, []int64{901})
	require.ErrorIs(t, err, service.ErrSchedulerBucketRetired)
}

func TestSchedulerCacheSetSnapshotByAccountIDsDoesNotResurrectDeletedAccount(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	account := service.Account{ID: 902, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth}
	single := service.SchedulerBucket{GroupID: 20, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	singleToken, err := cache.CaptureBucketWriteToken(ctx, single)
	require.NoError(t, err)
	accountIDs, err := cache.SetSnapshotAndReturnAccountIDs(ctx, single, singleToken, []service.Account{account})
	require.NoError(t, err)
	require.Equal(t, []int64{account.ID}, accountIDs)
	require.NoError(t, cache.DeleteAccount(ctx, account.ID))

	forced := service.SchedulerBucket{GroupID: 20, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeForced}
	forcedToken, err := cache.CaptureBucketWriteToken(ctx, forced)
	require.NoError(t, err)
	require.NoError(t, cache.SetSnapshotByAccountIDs(ctx, forced, forcedToken, accountIDs))

	full, err := cache.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, full, "ID-only 发布不得复活已删除的完整账号键")
	snapshot, hit, err := cache.GetSnapshot(ctx, forced)
	require.NoError(t, err)
	require.False(t, hit, "元数据缺失时必须安全回源，而不是返回残缺快照")
	require.Nil(t, snapshot)
}

func TestSchedulerCacheSnapshotBuildOmitsTombstonedAccountFromMembers(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	deleted := service.Account{ID: 903, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth}
	live := service.Account{ID: 904, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth}
	require.NoError(t, cache.DeleteAccount(ctx, deleted.ID))

	bucket := service.SchedulerBucket{GroupID: 21, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	accountIDs, err := cache.SetSnapshotAndReturnAccountIDs(ctx, bucket, token, []service.Account{deleted, live})
	require.NoError(t, err)
	require.Equal(t, []int64{live.ID}, accountIDs)

	activeVersion, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerActivePrefix, bucket)).Result()
	require.NoError(t, err)
	members, err := cache.rdb.ZRange(ctx, schedulerSnapshotKey(bucket, activeVersion), 0, -1).Result()
	require.NoError(t, err)
	require.Equal(t, []string{strconv.FormatInt(live.ID, 10)}, members)

	snapshot, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, snapshot, 1)
	require.Equal(t, live.ID, snapshot[0].ID)

	deletedID := strconv.FormatInt(deleted.ID, 10)
	exists, err := cache.rdb.Exists(ctx, schedulerAccountKey(deletedID), schedulerAccountMetaKey(deletedID)).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}

func TestMarshalSchedulerCacheAccountKeepsEncodingJSONWireFormat(t *testing.T) {
	cases := []struct {
		name    string
		account service.Account
	}{
		{name: "nil collections", account: service.Account{ID: 801}},
		{name: "empty collections", account: service.Account{
			ID:          802,
			Credentials: map[string]any{},
			Extra:       map[string]any{},
			GroupIDs:    []int64{},
			Groups:      []*service.Group{},
		}},
		{name: "nested maps and escaping", account: service.Account{
			ID:          803,
			Credentials: map[string]any{"model_mapping": map[string]any{"z": "<last>", "a": "&first"}},
			Extra:       map[string]any{"mixed_scheduling": true},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			full, meta, err := marshalSchedulerCacheAccount(tc.account)
			require.NoError(t, err)
			wantFull, err := json.Marshal(tc.account)
			require.NoError(t, err)
			wantMeta, err := json.Marshal(buildSchedulerMetadataAccount(tc.account))
			require.NoError(t, err)
			require.Equal(t, wantFull, full)
			require.Equal(t, wantMeta, meta)
		})
	}
}

func TestBuildSchedulerMetadataAccount_KeepsOpenAIWSFlags(t *testing.T) {
	account := service.Account{
		ID:       42,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Extra: map[string]any{
			"openai_oauth_responses_websockets_v2_enabled": true,
			"openai_oauth_responses_websockets_v2_mode":    service.OpenAIWSIngressModePassthrough,
			"openai_ws_force_http":                         true,
			"openai_responses_mode":                        "force_chat_completions",
			"openai_responses_supported":                   false,
			"mixed_scheduling":                             true,
			"unused_large_field":                           "drop-me",
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.Equal(t, true, got.Extra["openai_oauth_responses_websockets_v2_enabled"])
	require.Equal(t, service.OpenAIWSIngressModePassthrough, got.Extra["openai_oauth_responses_websockets_v2_mode"])
	require.Equal(t, true, got.Extra["openai_ws_force_http"])
	require.Equal(t, "force_chat_completions", got.Extra["openai_responses_mode"])
	require.Equal(t, false, got.Extra["openai_responses_supported"])
	require.Equal(t, true, got.Extra["mixed_scheduling"])
	require.Nil(t, got.Extra["unused_large_field"])
}

func TestBuildSchedulerMetadataAccount_KeepsSlimGroupMembership(t *testing.T) {
	account := service.Account{
		ID:       42,
		Platform: service.PlatformAnthropic,
		GroupIDs: []int64{7, 9, 7, 0},
		AccountGroups: []service.AccountGroup{
			{
				AccountID: 42,
				GroupID:   7,
				Priority:  2,
				Account:   &service.Account{ID: 42, Name: "drop-from-metadata"},
				Group:     &service.Group{ID: 7, Name: "drop-from-metadata"},
			},
			{
				AccountID: 42,
				GroupID:   11,
				Priority:  3,
				Group:     &service.Group{ID: 11, Name: "drop-from-metadata"},
			},
			{
				AccountID: 42,
				GroupID:   0,
				Priority:  4,
			},
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.Equal(t, []int64{7, 9, 11}, got.GroupIDs)
	require.Len(t, got.AccountGroups, 2)
	require.Equal(t, int64(42), got.AccountGroups[0].AccountID)
	require.Equal(t, int64(7), got.AccountGroups[0].GroupID)
	require.Equal(t, 2, got.AccountGroups[0].Priority)
	require.Nil(t, got.AccountGroups[0].Account)
	require.Nil(t, got.AccountGroups[0].Group)
	require.Equal(t, int64(11), got.AccountGroups[1].GroupID)
	require.Nil(t, got.Groups)
}

func TestBuildSchedulerMetadataAccount_KeepsQuotaAutoPauseFields(t *testing.T) {
	account := service.Account{
		ID: 88,
		Extra: map[string]any{
			"codex_5h_used_percent":        12.34,
			"codex_7d_used_percent":        56.78,
			"codex_5h_reset_at":            "2026-05-29T10:00:00Z",
			"codex_7d_reset_at":            "2026-06-01T10:00:00Z",
			"codex_5h_reset_after_seconds": 300,
			"codex_7d_reset_after_seconds": 600,
			"codex_usage_updated_at":       "2026-05-29T09:00:00Z",
			"auto_pause_5h_threshold":      0.95,
			"auto_pause_7d_threshold":      0.96,
			"auto_pause_5h_disabled":       true,
			"auto_pause_7d_disabled":       false,
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.Equal(t, 12.34, got.Extra["codex_5h_used_percent"])
	require.Equal(t, 56.78, got.Extra["codex_7d_used_percent"])
	require.Equal(t, "2026-05-29T10:00:00Z", got.Extra["codex_5h_reset_at"])
	require.Equal(t, "2026-06-01T10:00:00Z", got.Extra["codex_7d_reset_at"])
	require.Equal(t, 300, got.Extra["codex_5h_reset_after_seconds"])
	require.Equal(t, 600, got.Extra["codex_7d_reset_after_seconds"])
	require.Equal(t, "2026-05-29T09:00:00Z", got.Extra["codex_usage_updated_at"])
	require.Equal(t, 0.95, got.Extra["auto_pause_5h_threshold"])
	require.Equal(t, 0.96, got.Extra["auto_pause_7d_threshold"])
	require.Equal(t, true, got.Extra["auto_pause_5h_disabled"])
	require.Equal(t, false, got.Extra["auto_pause_7d_disabled"])
}

func TestBuildSchedulerMetadataAccount_KeepsModelRateLimits(t *testing.T) {
	account := service.Account{
		ID:       90,
		Platform: service.PlatformAntigravity,
		Extra: map[string]any{
			"model_rate_limits": map[string]any{
				"gemini-3-flash": map[string]any{
					"rate_limit_reset_at": "2026-05-30T10:10:00Z",
				},
				"antigravity:gemini": map[string]any{
					"rate_limit_reset_at": "2026-05-30T10:10:00Z",
				},
			},
			"unused_large_field": "drop-me",
		},
	}

	got := buildSchedulerMetadataAccount(account)

	limits, ok := got.Extra["model_rate_limits"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, limits, "gemini-3-flash")
	require.Contains(t, limits, "antigravity:gemini")
	require.Nil(t, got.Extra["unused_large_field"])
}

func TestBuildSchedulerMetadataAccount_KeepsSparkShadowRoutingIdentity(t *testing.T) {
	parentID := int64(100)
	account := service.Account{
		ID:              200,
		Platform:        service.PlatformOpenAI,
		Type:            service.AccountTypeOAuth,
		ParentAccountID: &parentID,
		QuotaDimension:  service.QuotaDimensionSpark,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"gpt-5.3-codex-spark": "gpt-5.3-codex-spark",
			},
			"compact_model_mapping": map[string]any{
				"gpt-5.4": "gpt-5.4-openai-compact",
			},
			"access_token": "drop-me",
		},
	}

	got := buildSchedulerMetadataAccount(account)

	require.NotNil(t, got.ParentAccountID)
	require.Equal(t, parentID, *got.ParentAccountID)
	require.Equal(t, service.QuotaDimensionSpark, got.QuotaDimension)
	require.Equal(t, map[string]any{"gpt-5.3-codex-spark": "gpt-5.3-codex-spark"}, got.Credentials["model_mapping"])
	require.Equal(t, map[string]any{"gpt-5.4": "gpt-5.4-openai-compact"}, got.Credentials["compact_model_mapping"])
	require.Nil(t, got.Credentials["access_token"])
}

func TestSchedulerCacheBucketRetirementFencesWritersAndReopen(t *testing.T) {
	ctx := context.Background()
	cache, mr := newSchedulerCacheUnitWithRedis(t)
	bucket := service.SchedulerBucket{GroupID: 41, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	otherBucket := service.SchedulerBucket{GroupID: 42, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	account := service.Account{ID: 4101, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}

	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	require.True(t, token.ValidFor(bucket))
	require.NoError(t, cache.SetSnapshot(ctx, bucket, token, []service.Account{account}))

	// A token is bound to the full bucket identity, not just an epoch number.
	err = cache.SetSnapshot(ctx, otherBucket, token, []service.Account{account})
	require.ErrorIs(t, err, service.ErrSchedulerBucketWriteFenced)
	_, err = cache.rdb.Get(ctx, schedulerBucketKey(schedulerVersionPrefix, otherBucket)).Result()
	require.ErrorIs(t, err, redis.Nil)
	otherAccount := service.Account{ID: 4201, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}
	otherToken, err := cache.CaptureBucketWriteToken(ctx, otherBucket)
	require.NoError(t, err)
	require.NoError(t, cache.SetSnapshot(ctx, otherBucket, otherToken, []service.Account{otherAccount}))
	otherEpoch := otherToken.Epoch

	activeVersion, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerActivePrefix, bucket)).Result()
	require.NoError(t, err)
	require.NoError(t, cache.RetireBucket(ctx, bucket))
	retiredEpoch, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerEpochPrefix, bucket)).Int64()
	require.NoError(t, err)
	require.Greater(t, retiredEpoch, token.Epoch)

	// Retirement is idempotent and does not advance the epoch again.
	require.NoError(t, cache.RetireBucket(ctx, bucket))
	retiredEpochAgain, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerEpochPrefix, bucket)).Int64()
	require.NoError(t, err)
	require.Equal(t, retiredEpoch, retiredEpochAgain)

	// New readers miss because ready/active were removed atomically. A reader that
	// captured activeVersion before retirement may still finish against that version.
	_, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.False(t, hit)
	ids, err := cache.rdb.ZRange(ctx, schedulerSnapshotKey(bucket, activeVersion), 0, -1).Result()
	require.NoError(t, err)
	require.Equal(t, []string{"4101"}, ids)
	ttl, err := cache.rdb.TTL(ctx, schedulerSnapshotKey(bucket, activeVersion)).Result()
	require.NoError(t, err)
	require.Positive(t, ttl)
	require.LessOrEqual(t, ttl, time.Duration(snapshotGraceTTLSeconds)*time.Second)

	buckets, err := cache.ListBuckets(ctx)
	require.NoError(t, err)
	require.NotContains(t, buckets, bucket)
	require.Contains(t, buckets, otherBucket)
	otherSnapshot, otherHit, err := cache.GetSnapshot(ctx, otherBucket)
	require.NoError(t, err)
	require.True(t, otherHit)
	require.Len(t, otherSnapshot, 1)
	require.Equal(t, otherAccount.ID, otherSnapshot[0].ID)
	otherEpochAfter, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerEpochPrefix, otherBucket)).Int64()
	require.NoError(t, err)
	require.Equal(t, otherEpoch, otherEpochAfter)

	_, err = cache.CaptureBucketWriteToken(ctx, bucket)
	require.ErrorIs(t, err, service.ErrSchedulerBucketRetired)
	versionBeforeRejectedWrite, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerVersionPrefix, bucket)).Int64()
	require.NoError(t, err)
	err = cache.SetSnapshot(ctx, bucket, token, []service.Account{account})
	require.ErrorIs(t, err, service.ErrSchedulerBucketRetired)
	versionAfterRejectedWrite, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerVersionPrefix, bucket)).Int64()
	require.NoError(t, err)
	require.Equal(t, versionBeforeRejectedWrite, versionAfterRejectedWrite, "fenced writers must not allocate a new version")
	retired, err := cache.rdb.Exists(ctx, schedulerBucketKey(schedulerRetiredPrefix, bucket)).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, retired, "ordinary writers must never clear the tombstone")
	mr.FastForward(time.Duration(snapshotGraceTTLSeconds+1) * time.Second)
	exists, err := cache.rdb.Exists(ctx, schedulerSnapshotKey(bucket, activeVersion)).Result()
	require.NoError(t, err)
	require.Zero(t, exists, "retired active snapshot must expire after the in-flight grace period")

	newToken, err := cache.ReopenBucket(ctx, bucket)
	require.NoError(t, err)
	require.True(t, newToken.ValidFor(bucket))
	require.Equal(t, retiredEpoch, newToken.Epoch)
	reopenedAgain, err := cache.ReopenBucket(ctx, bucket)
	require.NoError(t, err)
	require.Equal(t, newToken, reopenedAgain, "reopen must be idempotent within one retirement generation")
	err = cache.SetSnapshot(ctx, bucket, token, []service.Account{account})
	require.ErrorIs(t, err, service.ErrSchedulerBucketWriteFenced)
	require.NoError(t, cache.SetSnapshot(ctx, bucket, newToken, []service.Account{account}))
	reopenedWhileOpen, err := cache.ReopenBucket(ctx, bucket)
	require.NoError(t, err)
	require.Equal(t, newToken, reopenedWhileOpen)

	snapshot, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.Len(t, snapshot, 1)
	require.Equal(t, account.ID, snapshot[0].ID)
}

func TestSchedulerCacheActivationIsFencedAfterRetire(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	bucket := service.SchedulerBucket{GroupID: 51, Platform: service.PlatformAnthropic, Mode: service.SchedulerModeMixed}
	account := service.Account{ID: 5101, Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey}

	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	version, err := cache.allocateSnapshotVersion(ctx, bucket, token)
	require.NoError(t, err)
	require.NoError(t, cache.writeSnapshotVersion(ctx, bucket, version, []service.Account{account}))

	// Deterministic race C: retirement and authoritative reopen both happen after
	// INCR/write but before the old writer activates.
	require.NoError(t, cache.RetireBucket(ctx, bucket))
	_, err = cache.ReopenBucket(ctx, bucket)
	require.NoError(t, err)
	err = cache.activateSnapshotVersion(ctx, bucket, token, version)
	require.ErrorIs(t, err, service.ErrSchedulerBucketWriteFenced)

	exists, err := cache.rdb.Exists(ctx, schedulerSnapshotKey(bucket, version)).Result()
	require.NoError(t, err)
	require.Zero(t, exists, "fenced activation must delete its unpublished snapshot")
	exists, err = cache.rdb.Exists(
		ctx,
		schedulerBucketKey(schedulerReadyPrefix, bucket),
		schedulerBucketKey(schedulerActivePrefix, bucket),
	).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
	buckets, err := cache.ListBuckets(ctx)
	require.NoError(t, err)
	require.NotContains(t, buckets, bucket)
}

func TestSchedulerCacheConcurrentReopenReturnsSameToken(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	bucket := service.SchedulerBucket{GroupID: 53, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeForced}
	account := service.Account{ID: 5301, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey}

	oldToken, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	require.NoError(t, cache.RetireBucket(ctx, bucket))

	type reopenResult struct {
		token service.SchedulerBucketWriteToken
		err   error
	}
	start := make(chan struct{})
	results := make(chan reopenResult, 2)
	for range 2 {
		go func() {
			<-start
			token, err := cache.ReopenBucket(ctx, bucket)
			results <- reopenResult{token: token, err: err}
		}()
	}
	close(start)
	first := <-results
	second := <-results
	require.NoError(t, first.err)
	require.NoError(t, second.err)
	require.Equal(t, first.token, second.token)
	require.Greater(t, first.token.Epoch, oldToken.Epoch)

	require.ErrorIs(t, cache.SetSnapshot(ctx, bucket, oldToken, []service.Account{account}), service.ErrSchedulerBucketWriteFenced)
	require.NoError(t, cache.SetSnapshot(ctx, bucket, first.token, []service.Account{account}))
}

func TestSchedulerCacheReopenExpiresPreviousActiveSnapshot(t *testing.T) {
	ctx := context.Background()
	cache, mr := newSchedulerCacheUnitWithRedis(t)
	bucket := service.SchedulerBucket{GroupID: 52, Platform: service.PlatformGemini, Mode: service.SchedulerModeForced}
	account := service.Account{ID: 5201, Platform: service.PlatformGemini, Type: service.AccountTypeAPIKey}

	oldToken, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	require.NoError(t, cache.SetSnapshot(ctx, bucket, oldToken, []service.Account{account}))
	oldVersion, err := cache.rdb.Get(ctx, schedulerBucketKey(schedulerActivePrefix, bucket)).Result()
	require.NoError(t, err)
	retiredEpoch := oldToken.Epoch + 1
	require.NoError(t, cache.rdb.Set(ctx, schedulerBucketKey(schedulerEpochPrefix, bucket), retiredEpoch, 0).Err())
	require.NoError(t, cache.rdb.Set(ctx, schedulerBucketKey(schedulerRetiredPrefix, bucket), retiredEpoch, 0).Err())

	newToken, err := cache.ReopenBucket(ctx, bucket)
	require.NoError(t, err)
	require.Equal(t, retiredEpoch, newToken.Epoch)
	_, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.False(t, hit)
	ttl, err := cache.rdb.TTL(ctx, schedulerSnapshotKey(bucket, oldVersion)).Result()
	require.NoError(t, err)
	require.Positive(t, ttl)
	require.LessOrEqual(t, ttl, time.Duration(snapshotGraceTTLSeconds)*time.Second)

	require.ErrorIs(t, cache.SetSnapshot(ctx, bucket, oldToken, []service.Account{account}), service.ErrSchedulerBucketWriteFenced)
	mr.FastForward(time.Duration(snapshotGraceTTLSeconds+1) * time.Second)
	exists, err := cache.rdb.Exists(ctx, schedulerSnapshotKey(bucket, oldVersion)).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
	require.NoError(t, cache.SetSnapshot(ctx, bucket, newToken, []service.Account{account}))
}

func TestSchedulerCacheGroupLifecycleLeaseConcurrentAcquireSingleOwner(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	const groupID int64 = 71

	type result struct {
		lease    service.SchedulerGroupLifecycleLease
		acquired bool
		err      error
	}
	start := make(chan struct{})
	results := make(chan result, 32)
	for range 32 {
		go func() {
			<-start
			lease, acquired, err := cache.TryAcquireGroupLifecycleLease(ctx, groupID, time.Minute)
			results <- result{lease: lease, acquired: acquired, err: err}
		}()
	}
	close(start)

	var owner service.SchedulerGroupLifecycleLease
	acquiredCount := 0
	for range 32 {
		got := <-results
		require.NoError(t, got.err)
		if got.acquired {
			acquiredCount++
			owner = got.lease
			require.True(t, got.lease.ValidFor(groupID))
		} else {
			require.Equal(t, service.SchedulerGroupLifecycleLease{}, got.lease)
		}
	}
	require.Equal(t, 1, acquiredCount)
	require.Len(t, owner.OwnerToken, schedulerGroupLifecycleOwnerTokenBytes*2)
	require.Equal(t, strings.ToLower(owner.OwnerToken), owner.OwnerToken)
	decodedOwner, err := hex.DecodeString(owner.OwnerToken)
	require.NoError(t, err)
	require.Len(t, decodedOwner, schedulerGroupLifecycleOwnerTokenBytes)

	require.NoError(t, cache.ReleaseGroupLifecycleLease(ctx, owner))
	next, acquired, err := cache.TryAcquireGroupLifecycleLease(ctx, groupID, time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
	require.True(t, next.ValidFor(groupID))
	require.NotEqual(t, owner.OwnerToken, next.OwnerToken)
	require.NoError(t, cache.ReleaseGroupLifecycleLease(ctx, next))
}

func TestSchedulerCacheGroupLifecycleLeaseStaleReleaseCannotDeleteSuccessor(t *testing.T) {
	ctx := context.Background()
	cache, mr := newSchedulerCacheUnitWithRedis(t)
	const groupID int64 = 72
	const ttl = time.Minute

	first, acquired, err := cache.TryAcquireGroupLifecycleLease(ctx, groupID, ttl)
	require.NoError(t, err)
	require.True(t, acquired)

	mr.FastForward(ttl + time.Second)
	second, acquired, err := cache.TryAcquireGroupLifecycleLease(ctx, groupID, ttl)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotEqual(t, first.OwnerToken, second.OwnerToken)

	require.ErrorIs(t, cache.ReleaseGroupLifecycleLease(ctx, first), service.ErrSchedulerGroupLifecycleLeaseLost)
	owner, err := cache.rdb.Get(ctx, schedulerGroupLifecycleLockKey(groupID)).Result()
	require.NoError(t, err)
	require.Equal(t, second.OwnerToken, owner)

	_, acquired, err = cache.TryAcquireGroupLifecycleLease(ctx, groupID, ttl)
	require.NoError(t, err)
	require.False(t, acquired)
	require.NoError(t, cache.ReleaseGroupLifecycleLease(ctx, second))
	require.ErrorIs(t, cache.ReleaseGroupLifecycleLease(ctx, second), service.ErrSchedulerGroupLifecycleLeaseLost)
}

func TestSchedulerCacheGroupLifecycleLeaseExpiredReleaseIsLost(t *testing.T) {
	ctx := context.Background()
	cache, mr := newSchedulerCacheUnitWithRedis(t)
	const groupID int64 = 73
	const ttl = time.Minute

	lease, acquired, err := cache.TryAcquireGroupLifecycleLease(ctx, groupID, ttl)
	require.NoError(t, err)
	require.True(t, acquired)
	mr.FastForward(ttl + time.Second)

	require.ErrorIs(t, cache.ReleaseGroupLifecycleLease(ctx, lease), service.ErrSchedulerGroupLifecycleLeaseLost)
}

func TestSchedulerCacheGroupLifecycleLeaseWrongOwnerAndCrossGroupAreLost(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	const firstGroupID int64 = 74
	const secondGroupID int64 = 75

	first, acquired, err := cache.TryAcquireGroupLifecycleLease(ctx, firstGroupID, time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
	second, acquired, err := cache.TryAcquireGroupLifecycleLease(ctx, secondGroupID, time.Minute)
	require.NoError(t, err)
	require.True(t, acquired, "different groups must acquire independently")
	require.NotEqual(t, first.OwnerToken, second.OwnerToken)

	wrongOwner := first
	wrongOwner.OwnerToken = strings.Repeat("0", schedulerGroupLifecycleOwnerTokenBytes*2)
	if wrongOwner.OwnerToken == first.OwnerToken {
		wrongOwner.OwnerToken = strings.Repeat("1", schedulerGroupLifecycleOwnerTokenBytes*2)
	}
	require.ErrorIs(t, cache.ReleaseGroupLifecycleLease(ctx, wrongOwner), service.ErrSchedulerGroupLifecycleLeaseLost)

	crossGroup := first
	crossGroup.GroupID = secondGroupID
	require.ErrorIs(t, cache.ReleaseGroupLifecycleLease(ctx, crossGroup), service.ErrSchedulerGroupLifecycleLeaseLost)

	firstOwner, err := cache.rdb.Get(ctx, schedulerGroupLifecycleLockKey(firstGroupID)).Result()
	require.NoError(t, err)
	require.Equal(t, first.OwnerToken, firstOwner)
	secondOwner, err := cache.rdb.Get(ctx, schedulerGroupLifecycleLockKey(secondGroupID)).Result()
	require.NoError(t, err)
	require.Equal(t, second.OwnerToken, secondOwner)

	require.NoError(t, cache.ReleaseGroupLifecycleLease(ctx, first))
	require.NoError(t, cache.ReleaseGroupLifecycleLease(ctx, second))
}

func TestSchedulerCacheGroupLifecycleLeaseCanceledContextFailsClosed(t *testing.T) {
	cache := newSchedulerCacheUnit(t)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	lease, acquired, err := cache.TryAcquireGroupLifecycleLease(canceledCtx, 76, time.Minute)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, acquired)
	require.Equal(t, service.SchedulerGroupLifecycleLease{}, lease)

	ctx := context.Background()
	lease, acquired, err = cache.TryAcquireGroupLifecycleLease(ctx, 76, time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
	require.ErrorIs(t, cache.ReleaseGroupLifecycleLease(canceledCtx, lease), context.Canceled)
	owner, err := cache.rdb.Get(ctx, schedulerGroupLifecycleLockKey(lease.GroupID)).Result()
	require.NoError(t, err)
	require.Equal(t, lease.OwnerToken, owner)
	require.NoError(t, cache.ReleaseGroupLifecycleLease(ctx, lease))
}

func TestSchedulerCacheGroupLifecycleLeaseRejectsInvalidInput(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)

	lease, acquired, err := cache.TryAcquireGroupLifecycleLease(ctx, 0, time.Minute)
	require.ErrorIs(t, err, service.ErrSchedulerGroupLifecycleLeaseInvalid)
	require.False(t, acquired)
	require.Equal(t, service.SchedulerGroupLifecycleLease{}, lease)

	lease, acquired, err = cache.TryAcquireGroupLifecycleLease(ctx, 73, 0)
	require.ErrorIs(t, err, service.ErrSchedulerGroupLifecycleLeaseInvalid)
	require.False(t, acquired)
	require.Equal(t, service.SchedulerGroupLifecycleLease{}, lease)

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, cache.ReleaseGroupLifecycleLease(canceledCtx, service.SchedulerGroupLifecycleLease{}), service.ErrSchedulerGroupLifecycleLeaseInvalid)
	keys, err := cache.rdb.DBSize(ctx).Result()
	require.NoError(t, err)
	require.Zero(t, keys)
}

var schedulerCachePayloadBenchmarkSink int

func BenchmarkSchedulerCacheAccountPayloadReuse(b *testing.B) {
	for _, size := range []int{1, 100, 10_000} {
		accounts := schedulerCacheBenchmarkAccounts(size)
		b.Run(fmt.Sprintf("pair_baseline_%d_accounts", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				first, err := benchmarkSchedulerLegacySnapshotPayload(accounts)
				if err != nil {
					b.Fatal(err)
				}
				second, err := benchmarkSchedulerLegacySnapshotPayload(accounts)
				if err != nil {
					b.Fatal(err)
				}
				schedulerCachePayloadBenchmarkSink = first + second
			}
		})
		b.Run(fmt.Sprintf("pair_reuse_%d_accounts", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ids, total, err := benchmarkSchedulerReusableSnapshotPayload(accounts)
				if err != nil {
					b.Fatal(err)
				}
				// 第二个桶仍构造成员，只跳过账号 JSON 与全局账号键。
				total += len(schedulerSnapshotMembers(ids))
				schedulerCachePayloadBenchmarkSink = total
			}
		})
		b.Run(fmt.Sprintf("first_baseline_%d_accounts", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				total, err := benchmarkSchedulerLegacySnapshotPayload(accounts)
				if err != nil {
					b.Fatal(err)
				}
				schedulerCachePayloadBenchmarkSink = total
			}
		})
		b.Run(fmt.Sprintf("first_reuse_%d_accounts", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ids, total, err := benchmarkSchedulerReusableSnapshotPayload(accounts)
				if err != nil {
					b.Fatal(err)
				}
				total += len(ids)
				schedulerCachePayloadBenchmarkSink = total
			}
		})
	}
}

func benchmarkSchedulerLegacySnapshotPayload(accounts []service.Account) (int, error) {
	cacheable := make([]service.Account, 0, len(accounts))
	total := 0
	for _, account := range accounts {
		full, meta, err := marshalSchedulerCacheAccount(account)
		if err != nil {
			continue
		}
		total += len(full) + len(meta)
		cacheable = append(cacheable, account)
	}
	members := make([]redis.Z, 0, len(cacheable))
	for idx, account := range cacheable {
		members = append(members, redis.Z{Score: float64(idx), Member: strconv.FormatInt(account.ID, 10)})
	}
	return total + len(members), nil
}

func benchmarkSchedulerReusableSnapshotPayload(accounts []service.Account) ([]int64, int, error) {
	accountIDs := make([]int64, 0, len(accounts))
	total := 0
	for _, account := range accounts {
		full, meta, err := marshalSchedulerCacheAccount(account)
		if err != nil {
			continue
		}
		total += len(full) + len(meta)
		accountIDs = append(accountIDs, account.ID)
	}
	total += len(schedulerSnapshotMembers(accountIDs))
	return accountIDs, total, nil
}

func schedulerCacheBenchmarkAccounts(size int) []service.Account {
	largeValue := strings.Repeat("x", 4096)
	credentials := map[string]any{
		"api_key":       "benchmark-key",
		"model_mapping": map[string]any{"z-model": "z-target", "a-model": "a-target"},
		"large_value":   largeValue,
	}
	extra := map[string]any{
		"mixed_scheduling": true,
		"model_rate_limits": map[string]any{
			"z-model": map[string]any{"rate_limit_reset_at": "2026-07-16T00:00:00Z"},
			"a-model": map[string]any{"rate_limit_reset_at": "2026-07-16T00:00:00Z"},
		},
		"large_value": largeValue,
	}
	accounts := make([]service.Account, size)
	for i := range accounts {
		id := int64(i + 1)
		accounts[i] = service.Account{
			ID:          id,
			Name:        "benchmark-account",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: credentials,
			Extra:       extra,
			GroupIDs:    []int64{7, 9},
			AccountGroups: []service.AccountGroup{
				{AccountID: id, GroupID: 7, Priority: 1},
				{AccountID: id, GroupID: 9, Priority: 2},
			},
		}
	}
	return accounts
}
