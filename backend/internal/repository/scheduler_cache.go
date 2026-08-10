package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	schedulerBucketSetKey       = "sched:buckets"
	schedulerOutboxWatermarkKey = "sched:outbox:watermark"
	schedulerGenerationKey      = "sched:generation"
	schedulerResetKey           = "sched:reset"
	schedulerResetLeaseKey      = "sched:reset:lease"
	// schedulerAccountTombstoneSetKey is a persistent deletion fence. Account
	// IDs come from a PostgreSQL sequence and must never be reused; keeping all
	// deleted IDs in one Redis set is cheaper than one permanent key per ID and
	// gives operators one explicit key to preserve during cache maintenance.
	schedulerAccountTombstoneSetKey = "sched:account:tombstones"
	schedulerAccountPrefix          = "sched:acc:"
	schedulerAccountMetaPrefix      = "sched:meta:"
	schedulerActivePrefix           = "sched:active:"
	schedulerReadyPrefix            = "sched:ready:"
	schedulerVersionPrefix          = "sched:ver:"
	schedulerEpochPrefix            = "sched:epoch:"
	schedulerRetiredPrefix          = "sched:retired:"
	schedulerSnapshotPrefix         = "sched:"
	schedulerLockPrefix             = "sched:lock:"

	defaultSchedulerSnapshotMGetChunkSize  = 128
	defaultSchedulerSnapshotWriteChunkSize = 256
	schedulerLastUsedCASMaxRetries         = 32
	schedulerAuthoritativeResetLeaseTTL    = 30 * time.Second
	schedulerSnapshotStagingTTL            = 5 * time.Minute
	// Redis Lua 5.1 represents numbers as IEEE-754 doubles. Random bucket
	// epochs stay below 2^53 so incrementing/comparing retirement epochs is
	// exact in Lua and round-trips through decimal strings without truncation.
	schedulerMaxSafeLuaInteger = int64(1<<53) - 1
	// Starting in the lower half of the exact range leaves effectively
	// unbounded room for monotonic retire/reopen cycles without approaching the
	// Lua integer precision boundary.
	schedulerMaxInitialEpoch = int64(1 << 52)

	// snapshotGraceTTLSeconds 旧快照过期的宽限期（秒）。
	// 替代立即 DEL，让正在读取旧版本的 reader 有足够时间完成 ZRANGE。
	snapshotGraceTTLSeconds = 60
)

const (
	schedulerGroupLifecycleLockPrefix      = "sched:group:lifecycle-lock:"
	schedulerGroupLifecycleOwnerTokenBytes = 16
)

var (
	// epoch 标识 bucket writer 的代际，retired key 是持久退休标记。
	// Capture、allocate、activate 都在 Lua 内同时校验两者：-1 表示已退休，-2 表示 epoch 无效或与 token 代际不匹配；
	// allocate 与 activate 的双重校验可拦截快照写入期间发生的 Retire。
	// Retire 仅在首次退休时推进 epoch，Reopen 只清除标记并沿用该代际，因此重复调用保持幂等。
	ensureSchedulerGenerationScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[2]) == 1 then
    return {0, ''}
end
local current = redis.call('GET', KEYS[1])
if current == false then
    current = ARGV[1]
    redis.call('SET', KEYS[1], current)
end
return {1, current}
`)

	captureBucketWriteTokenScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[4]) == 1 then
    return {-2, '', ''}
end
local generation = redis.call('GET', KEYS[3])
if generation == false then
    generation = ARGV[2]
    redis.call('SET', KEYS[3], generation)
end
if redis.call('EXISTS', KEYS[2]) == 1 then
    return {-1, '', ''}
end
local currentEpoch = redis.call('GET', KEYS[1])
if currentEpoch == false then
    currentEpoch = ARGV[1]
    redis.call('SET', KEYS[1], currentEpoch)
end
local parsedEpoch = tonumber(currentEpoch)
local maxEpoch = tonumber(ARGV[3])
if parsedEpoch == nil or maxEpoch == nil or parsedEpoch < 1 or parsedEpoch > maxEpoch or parsedEpoch ~= math.floor(parsedEpoch) then
    return {-2, '', ''}
end
local canonicalEpoch = string.format('%.0f', parsedEpoch)
if canonicalEpoch ~= currentEpoch then
    redis.call('SET', KEYS[1], canonicalEpoch)
end
return {1, generation, canonicalEpoch}
`)

	allocateSnapshotVersionScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[5]) == 1 or redis.call('GET', KEYS[4]) ~= ARGV[2] then
    return -2
end
if redis.call('EXISTS', KEYS[2]) == 1 then
    return -1
end
local currentEpoch = redis.call('GET', KEYS[1])
if currentEpoch == false or currentEpoch ~= ARGV[1] then
    return -2
end
return redis.call('INCR', KEYS[3])
`)

	retireBucketScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[7]) == 1 then
    return {-2, ''}
end
local generation = redis.call('GET', KEYS[6])
if generation == false then
    generation = ARGV[5]
    redis.call('SET', KEYS[6], generation)
end
local retired = redis.call('GET', KEYS[2])
local currentEpochRaw = redis.call('GET', KEYS[1])
local currentEpoch = tonumber(currentEpochRaw)
local maxEpoch = tonumber(ARGV[6])
local function validEpoch(epoch)
    return epoch ~= nil and maxEpoch ~= nil and epoch >= 1 and epoch <= maxEpoch and epoch == math.floor(epoch)
end
local retiredEpoch = nil
if retired ~= false then
    retiredEpoch = tonumber(retired)
    if not validEpoch(retiredEpoch) then
        return {-2, ''}
    end
end

if retired == false then
    if currentEpochRaw == false then
        currentEpoch = tonumber(ARGV[4])
    else
        if not validEpoch(currentEpoch) or currentEpoch >= maxEpoch then
            return {-2, ''}
        end
        currentEpoch = currentEpoch + 1
    end
    if not validEpoch(currentEpoch) then
        return {-2, ''}
    end
    local canonicalEpoch = string.format('%.0f', currentEpoch)
    redis.call('SET', KEYS[1], canonicalEpoch)
    redis.call('SET', KEYS[2], canonicalEpoch)
elseif not validEpoch(currentEpoch) then
    currentEpoch = retiredEpoch
    local canonicalEpoch = string.format('%.0f', currentEpoch)
    redis.call('SET', KEYS[1], canonicalEpoch)
    redis.call('SET', KEYS[2], canonicalEpoch)
else
    redis.call('SET', KEYS[1], string.format('%.0f', currentEpoch))
    redis.call('SET', KEYS[2], string.format('%.0f', retiredEpoch))
end

redis.call('SREM', KEYS[3], ARGV[1])
local currentActive = redis.call('GET', KEYS[5])
if currentActive ~= false then
    redis.call('EXPIRE', ARGV[2] .. currentActive, tonumber(ARGV[3]))
end
redis.call('DEL', KEYS[4], KEYS[5])
return {1, string.format('%.0f', currentEpoch)}
`)

	reopenBucketScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[7]) == 1 then
    return {-2, '', ''}
end
local generation = redis.call('GET', KEYS[6])
if generation == false then
    generation = ARGV[5]
    redis.call('SET', KEYS[6], generation)
end
local currentEpochRaw = redis.call('GET', KEYS[1])
local currentEpoch = tonumber(currentEpochRaw)
local retiredEpochRaw = redis.call('GET', KEYS[2])
local maxEpoch = tonumber(ARGV[6])
local function validEpoch(epoch)
    return epoch ~= nil and maxEpoch ~= nil and epoch >= 1 and epoch <= maxEpoch and epoch == math.floor(epoch)
end

if retiredEpochRaw == false then
    if currentEpochRaw == false then
        currentEpochRaw = ARGV[4]
        currentEpoch = tonumber(currentEpochRaw)
        if not validEpoch(currentEpoch) then
            return {-2, '', ''}
        end
        redis.call('SET', KEYS[1], currentEpochRaw)
    end
    if not validEpoch(currentEpoch) then
        return {-2, '', ''}
    end
    local canonicalEpoch = string.format('%.0f', currentEpoch)
    redis.call('SET', KEYS[1], canonicalEpoch)
    return {1, generation, canonicalEpoch}
end

local retiredEpoch = tonumber(retiredEpochRaw)
if not validEpoch(retiredEpoch) then
    return {-2, '', ''}
end
if not validEpoch(currentEpoch) or currentEpoch < retiredEpoch then
    currentEpoch = retiredEpoch
end

local canonicalEpoch = string.format('%.0f', currentEpoch)
redis.call('SET', KEYS[1], canonicalEpoch)
redis.call('DEL', KEYS[2])
redis.call('SREM', KEYS[3], ARGV[1])
local currentActive = redis.call('GET', KEYS[5])
if currentActive ~= false then
    redis.call('EXPIRE', ARGV[2] .. currentActive, tonumber(ARGV[3]))
end
redis.call('DEL', KEYS[4], KEYS[5])
return {1, generation, canonicalEpoch}
`)

	// 释放租约必须先比较所有者令牌再删除，过期持有者的延迟释放不能误删继任租约。
	releaseGroupLifecycleLeaseScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('DEL', KEYS[1])
end
return 0
`)

	// Bucket rebuild leases use the same owner-token compare-delete protocol as
	// group lifecycle leases, but retain a separate script name and key space.
	releaseBucketRebuildLeaseScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('DEL', KEYS[1])
end
return 0
`)

	// A rebuild lease is part of the writer fence: an old-generation task must
	// not recreate a lock after an authoritative reset scan has passed its key.
	acquireBucketRebuildLeaseScript = redis.NewScript(`
local ttl = tonumber(ARGV[3])
if ttl == nil or ttl < 1 then
    return redis.error_reply('invalid scheduler bucket rebuild lease ttl')
end
if redis.call('EXISTS', KEYS[3]) == 1 or redis.call('GET', KEYS[2]) ~= ARGV[2] then
    return -1
end
if redis.call('SET', KEYS[1], ARGV[1], 'PX', ttl, 'NX') then
    return 1
end
return 0
`)

	// Every staging chunk is generation fenced and receives a short TTL in the
	// same command. Activation removes the TTL only after the final CAS succeeds.
	writeSnapshotMembersScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[3]) == 1 or redis.call('GET', KEYS[2]) ~= ARGV[1] then
    redis.call('DEL', KEYS[1])
    return -2
end
local ttl = tonumber(ARGV[2])
if ttl == nil or ttl < 1 then
    return redis.error_reply('invalid scheduler snapshot staging ttl')
end
if #ARGV > 2 then
    redis.call('ZADD', KEYS[1], unpack(ARGV, 3))
    redis.call('PEXPIRE', KEYS[1], ttl)
end
return 1
`)

	advanceOutboxWatermarkScript = redis.NewScript(`
local candidate = tonumber(ARGV[1])
if candidate == nil then
    return redis.error_reply('invalid scheduler outbox watermark candidate')
end
local currentRaw = redis.call('GET', KEYS[1])
if currentRaw ~= false then
    local current = tonumber(currentRaw)
    if current == nil then
        return redis.error_reply('invalid current scheduler outbox watermark')
    end
    if current >= candidate then
        return current
    end
end
redis.call('SET', KEYS[1], ARGV[1])
return candidate
`)

	// The full account payload is the CAS version. A successful write or safe
	// deletion updates both the full and metadata keys in one Redis operation.
	updateLastUsedCASScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[5]) == 1 or redis.call('GET', KEYS[4]) ~= ARGV[6] then
    return -2
end
if redis.call('SISMEMBER', KEYS[3], ARGV[5]) == 1 then
    return 0
end
local current = redis.call('GET', KEYS[1])
if current == false then
    return 0
end
if current ~= ARGV[1] then
    return -1
end
if ARGV[2] == 'delete' then
    redis.call('DEL', KEYS[1], KEYS[2])
    return 1
end
redis.call('SET', KEYS[1], ARGV[3])
redis.call('SET', KEYS[2], ARGV[4])
return 1
`)

	// Account projections use the full payload as their optimistic version. The
	// missing sentinel lets initial creates and orphan-metadata cleanup use the
	// same atomic operation as updates. Both cache representations are always
	// changed together, so readers can never observe a split projection written
	// by this implementation.
	writeAccountProjectionCASScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[5]) == 1 or redis.call('GET', KEYS[4]) ~= ARGV[7] then
    return -2
end
if redis.call('SISMEMBER', KEYS[3], ARGV[6]) == 1 then
    return 0
end
local current = redis.call('GET', KEYS[1])
if ARGV[1] == 'missing' then
    if current ~= false then
        return -1
    end
elseif current == false or current ~= ARGV[2] then
    return -1
end

if ARGV[3] == 'delete' then
    redis.call('DEL', KEYS[1], KEYS[2])
    return 1
end

redis.call('SET', KEYS[1], ARGV[4])
redis.call('SET', KEYS[2], ARGV[5])
return 1
`)

	// A durable account deletion must win atomically over both cache payloads.
	// The tombstone intentionally has no TTL: stale rebuilds may resume after an
	// arbitrary delay, and PostgreSQL account IDs are never reused.
	deleteAccountProjectionScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[5]) == 1 or redis.call('GET', KEYS[4]) ~= ARGV[2] then
    return -2
end
redis.call('SADD', KEYS[3], ARGV[1])
redis.call('DEL', KEYS[1], KEYS[2])
return 1
`)

	// The reset marker is durable so readers remain fail-closed after an owner
	// crashes. A separate expiring lease permits takeover; every destructive
	// chunk verifies both owner keys and the rotated generation before deleting.
	beginSchedulerAuthoritativeResetScript = redis.NewScript(`
local ttl = tonumber(ARGV[3])
if ttl == nil or ttl < 1 then
    return redis.error_reply('invalid scheduler authoritative reset lease ttl')
end
if ARGV[4] == '1' and redis.call('EXISTS', KEYS[2]) == 0 then
    return 2
end
if redis.call('EXISTS', KEYS[3]) == 1 then
    return 0
end
redis.call('SET', KEYS[2], ARGV[1])
redis.call('SET', KEYS[1], ARGV[2])
redis.call('SET', KEYS[3], ARGV[1], 'PX', ttl)
return 1
`)

	deleteSchedulerAuthoritativeResetKeysScript = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[1]
    or redis.call('GET', KEYS[3]) ~= ARGV[1]
    or redis.call('GET', KEYS[1]) ~= ARGV[2] then
    return 0
end
local ttl = tonumber(ARGV[3])
if ttl == nil or ttl < 1 then
    return redis.error_reply('invalid scheduler authoritative reset lease ttl')
end
redis.call('PEXPIRE', KEYS[3], ttl)
if #KEYS > 3 then
    redis.call('DEL', unpack(KEYS, 4))
end
return 1
`)

	finishSchedulerAuthoritativeResetScript = redis.NewScript(`
if redis.call('GET', KEYS[2]) ~= ARGV[1]
    or redis.call('GET', KEYS[3]) ~= ARGV[1]
    or redis.call('GET', KEYS[1]) ~= ARGV[2] then
    return 0
end
redis.call('DEL', KEYS[2], KEYS[3])
return 1
`)

	// activateSnapshotScript 原子 CAS 切换快照版本。
	// 仅当新版本号 >= 当前激活版本时才切换，防止并发写入导致版本回滚。
	// 旧快照使用 EXPIRE 设置宽限期而非立即 DEL，避免与 reader 竞态。
	//
	// KEYS[1] = activeKey     (sched:active:{bucket})
	// KEYS[2] = readyKey      (sched:ready:{bucket})
	// KEYS[3] = bucketSetKey  (sched:buckets)
	// KEYS[4] = snapshotKey   (新写入的快照 key)
	// KEYS[5] = epochKey
	// KEYS[6] = retiredKey
	// KEYS[7] = generationKey
	// KEYS[8] = resetKey
	// ARGV[1] = 新版本号字符串
	// ARGV[2] = bucket 字符串 (用于 SADD)
	// ARGV[3] = 快照 key 前缀 (用于构造旧快照 key)
	// ARGV[4] = 宽限期 TTL 秒数
	// ARGV[5] = writer epoch
	// ARGV[6] = writer generation
	// ARGV[7] = per-generation sequence
	//
	// 返回 1 = 已激活, 0 = 版本过旧未激活
	activateSnapshotScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[8]) == 1 or redis.call('GET', KEYS[7]) ~= ARGV[6] then
    redis.call('DEL', KEYS[4])
    return -2
end
if redis.call('EXISTS', KEYS[6]) == 1 then
    redis.call('DEL', KEYS[4])
    return -1
end

local currentEpoch = redis.call('GET', KEYS[5])
if currentEpoch == false or currentEpoch ~= ARGV[5] then
    redis.call('DEL', KEYS[4])
    return -2
end

local currentActive = redis.call('GET', KEYS[1])
local newSequence = tonumber(ARGV[7])
if newSequence == nil then
    redis.call('DEL', KEYS[4])
    return -2
end

if currentActive ~= false then
	local activeGeneration, activeSequenceRaw = string.match(currentActive, '^([^%.]+)%.([0-9]+)$')
	local activeSequence = tonumber(activeSequenceRaw)
	if activeGeneration == ARGV[6] and activeSequence and newSequence < activeSequence then
		redis.call('DEL', KEYS[4])
		return 0
	end
	local legacySequence = tonumber(currentActive)
	if legacySequence and newSequence < legacySequence then
		redis.call('DEL', KEYS[4])
		return 0
	end
end

redis.call('SET', KEYS[1], ARGV[1])
redis.call('SET', KEYS[2], '1')
redis.call('SADD', KEYS[3], ARGV[2])
redis.call('PERSIST', KEYS[4])

if currentActive ~= false and currentActive ~= ARGV[1] then
	redis.call('EXPIRE', ARGV[3] .. currentActive, tonumber(ARGV[4]))
end

return 1
`)
)

type schedulerCache struct {
	rdb            *redis.Client
	mgetChunkSize  int
	writeChunkSize int
}

type schedulerCacheReadFence struct {
	generation    string
	hasGeneration bool
}

func NewSchedulerCache(rdb *redis.Client) service.SchedulerCache {
	return newSchedulerCacheWithChunkSizes(rdb, defaultSchedulerSnapshotMGetChunkSize, defaultSchedulerSnapshotWriteChunkSize)
}

func newSchedulerCacheWithChunkSizes(rdb *redis.Client, mgetChunkSize, writeChunkSize int) service.SchedulerCache {
	if mgetChunkSize <= 0 {
		mgetChunkSize = defaultSchedulerSnapshotMGetChunkSize
	}
	if writeChunkSize <= 0 {
		writeChunkSize = defaultSchedulerSnapshotWriteChunkSize
	}
	return &schedulerCache{
		rdb:            rdb,
		mgetChunkSize:  mgetChunkSize,
		writeChunkSize: writeChunkSize,
	}
}

func (c *schedulerCache) captureReadFence(ctx context.Context) (schedulerCacheReadFence, error) {
	values, err := c.rdb.MGet(ctx, schedulerResetKey, schedulerGenerationKey).Result()
	if err != nil {
		return schedulerCacheReadFence{}, err
	}
	return schedulerReadFenceFromValues(values)
}

func schedulerReadFenceFromValues(values []any) (schedulerCacheReadFence, error) {
	if len(values) != 2 {
		return schedulerCacheReadFence{}, fmt.Errorf("capture scheduler cache read fence returned %d fields", len(values))
	}
	if values[0] != nil {
		return schedulerCacheReadFence{}, service.ErrSchedulerCacheResetInProgress
	}
	if values[1] == nil {
		return schedulerCacheReadFence{}, nil
	}
	generation, err := schedulerLuaString(values[1])
	if err != nil {
		return schedulerCacheReadFence{}, err
	}
	if generation == "" {
		return schedulerCacheReadFence{}, errors.New("scheduler cache generation is empty")
	}
	return schedulerCacheReadFence{generation: generation, hasGeneration: true}, nil
}

func (c *schedulerCache) validateReadFence(ctx context.Context, fence schedulerCacheReadFence) error {
	current, err := c.captureReadFence(ctx)
	if err != nil {
		return err
	}
	if current.hasGeneration != fence.hasGeneration || current.generation != fence.generation {
		return service.ErrSchedulerBucketWriteFenced
	}
	return nil
}

func (c *schedulerCache) GetSnapshot(ctx context.Context, bucket service.SchedulerBucket) ([]*service.Account, bool, error) {
	readyKey := schedulerBucketKey(schedulerReadyPrefix, bucket)
	activeKey := schedulerBucketKey(schedulerActivePrefix, bucket)
	state, err := c.rdb.MGet(ctx, schedulerResetKey, schedulerGenerationKey, readyKey, activeKey).Result()
	if err != nil {
		return nil, false, err
	}
	if len(state) != 4 {
		return nil, false, fmt.Errorf("read scheduler snapshot state returned %d fields", len(state))
	}
	readFence, err := schedulerReadFenceFromValues(state[:2])
	if err != nil {
		return nil, false, err
	}
	if state[2] == nil {
		return nil, false, nil
	}
	readyVal, err := schedulerLuaString(state[2])
	if err != nil {
		return nil, false, err
	}
	if readyVal != "1" {
		return nil, false, nil
	}
	if state[3] == nil {
		return nil, false, nil
	}
	activeVal, err := schedulerLuaString(state[3])
	if err != nil {
		return nil, false, err
	}
	if readFence.hasGeneration {
		activeGeneration, err := schedulerSnapshotVersionGeneration(activeVal)
		if err != nil || activeGeneration != readFence.generation {
			return nil, false, nil
		}
	}

	snapshotKey := schedulerSnapshotKey(bucket, activeVal)
	ids, err := c.rdb.ZRange(ctx, snapshotKey, 0, -1).Result()
	if err != nil {
		return nil, false, err
	}
	if len(ids) == 0 {
		// 空快照视为缓存未命中，触发数据库回退查询
		// 这解决了新分组创建后立即绑定账号时的竞态条件问题
		return nil, false, nil
	}

	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, schedulerAccountMetaKey(id))
	}
	values, err := c.mgetChunked(ctx, keys)
	if err != nil {
		return nil, false, err
	}

	accounts := make([]*service.Account, 0, len(values))
	for _, val := range values {
		if val == nil {
			return nil, false, nil
		}
		account, err := decodeCachedAccount(val)
		if err != nil {
			return nil, false, err
		}
		accounts = append(accounts, account)
	}

	if err := c.validateReadFence(ctx, readFence); err != nil {
		return nil, false, err
	}
	return accounts, true, nil
}

func (c *schedulerCache) CaptureBucketWriteToken(ctx context.Context, bucket service.SchedulerBucket) (service.SchedulerBucketWriteToken, error) {
	candidateEpoch, err := newSchedulerEpoch()
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	candidateGeneration, err := newSchedulerGeneration()
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	result, err := captureBucketWriteTokenScript.Run(ctx, c.rdb, []string{
		schedulerBucketKey(schedulerEpochPrefix, bucket),
		schedulerBucketKey(schedulerRetiredPrefix, bucket),
		schedulerGenerationKey,
		schedulerResetKey,
	}, strconv.FormatInt(candidateEpoch, 10), candidateGeneration, strconv.FormatInt(schedulerMaxSafeLuaInteger, 10)).Slice()
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	token, err := schedulerBucketWriteTokenResult(result, bucket)
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	return token, nil
}

func (c *schedulerCache) RetireBucket(ctx context.Context, bucket service.SchedulerBucket) error {
	candidateEpoch, err := newSchedulerEpoch()
	if err != nil {
		return err
	}
	candidateGeneration, err := newSchedulerGeneration()
	if err != nil {
		return err
	}
	snapshotKeyPrefix := fmt.Sprintf("%s%d:%s:%s:v", schedulerSnapshotPrefix, bucket.GroupID, bucket.Platform, bucket.Mode)
	result, err := retireBucketScript.Run(ctx, c.rdb, []string{
		schedulerBucketKey(schedulerEpochPrefix, bucket),
		schedulerBucketKey(schedulerRetiredPrefix, bucket),
		schedulerBucketSetKey,
		schedulerBucketKey(schedulerReadyPrefix, bucket),
		schedulerBucketKey(schedulerActivePrefix, bucket),
		schedulerGenerationKey,
		schedulerResetKey,
	}, bucket.String(), snapshotKeyPrefix, snapshotGraceTTLSeconds, strconv.FormatInt(candidateEpoch, 10), candidateGeneration, strconv.FormatInt(schedulerMaxSafeLuaInteger, 10)).Slice()
	if err != nil {
		return err
	}
	status, _, err := schedulerEpochResult(result)
	if err != nil {
		return fmt.Errorf("retire scheduler bucket %s: %w", bucket.String(), err)
	}
	if status == -2 {
		return fmt.Errorf("%w: bucket=%s", service.ErrSchedulerBucketWriteFenced, bucket.String())
	}
	if status != 1 {
		return fmt.Errorf("retire scheduler bucket %s returned status %d", bucket.String(), status)
	}
	return nil
}

func (c *schedulerCache) ReopenBucket(ctx context.Context, bucket service.SchedulerBucket) (service.SchedulerBucketWriteToken, error) {
	candidateEpoch, err := newSchedulerEpoch()
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	candidateGeneration, err := newSchedulerGeneration()
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	snapshotKeyPrefix := fmt.Sprintf("%s%d:%s:%s:v", schedulerSnapshotPrefix, bucket.GroupID, bucket.Platform, bucket.Mode)
	result, err := reopenBucketScript.Run(ctx, c.rdb, []string{
		schedulerBucketKey(schedulerEpochPrefix, bucket),
		schedulerBucketKey(schedulerRetiredPrefix, bucket),
		schedulerBucketSetKey,
		schedulerBucketKey(schedulerReadyPrefix, bucket),
		schedulerBucketKey(schedulerActivePrefix, bucket),
		schedulerGenerationKey,
		schedulerResetKey,
	}, bucket.String(), snapshotKeyPrefix, snapshotGraceTTLSeconds, strconv.FormatInt(candidateEpoch, 10), candidateGeneration, strconv.FormatInt(schedulerMaxSafeLuaInteger, 10)).Slice()
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	token, err := schedulerBucketWriteTokenResult(result, bucket)
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	return token, nil
}

func (c *schedulerCache) TryAcquireGroupLifecycleLease(ctx context.Context, groupID int64, ttl time.Duration) (service.SchedulerGroupLifecycleLease, bool, error) {
	if groupID <= 0 {
		return service.SchedulerGroupLifecycleLease{}, false, fmt.Errorf("%w: group id must be positive", service.ErrSchedulerGroupLifecycleLeaseInvalid)
	}
	if ttl <= 0 {
		return service.SchedulerGroupLifecycleLease{}, false, fmt.Errorf("%w: ttl must be positive", service.ErrSchedulerGroupLifecycleLeaseInvalid)
	}
	ownerToken, err := newSchedulerGroupLifecycleOwnerToken()
	if err != nil {
		return service.SchedulerGroupLifecycleLease{}, false, err
	}
	acquired, err := c.rdb.SetNX(ctx, schedulerGroupLifecycleLockKey(groupID), ownerToken, ttl).Result()
	if err != nil {
		return service.SchedulerGroupLifecycleLease{}, false, err
	}
	if !acquired {
		return service.SchedulerGroupLifecycleLease{}, false, nil
	}
	return service.SchedulerGroupLifecycleLease{GroupID: groupID, OwnerToken: ownerToken}, true, nil
}

func (c *schedulerCache) ReleaseGroupLifecycleLease(ctx context.Context, lease service.SchedulerGroupLifecycleLease) error {
	if !lease.ValidFor(lease.GroupID) {
		return service.ErrSchedulerGroupLifecycleLeaseInvalid
	}
	result, err := releaseGroupLifecycleLeaseScript.Run(
		ctx,
		c.rdb,
		[]string{schedulerGroupLifecycleLockKey(lease.GroupID)},
		lease.OwnerToken,
	).Int64()
	if err != nil {
		return err
	}
	if result == 0 {
		return fmt.Errorf("%w: group=%d", service.ErrSchedulerGroupLifecycleLeaseLost, lease.GroupID)
	}
	if result != 1 {
		return fmt.Errorf("release scheduler group lifecycle lease returned %d", result)
	}
	return nil
}

func newSchedulerGroupLifecycleOwnerToken() (string, error) {
	raw := make([]byte, schedulerGroupLifecycleOwnerTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate scheduler group lifecycle owner token: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

func newSchedulerGeneration() (string, error) {
	return newSchedulerGroupLifecycleOwnerToken()
}

func newSchedulerEpoch() (int64, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(schedulerMaxInitialEpoch))
	if err != nil {
		return 0, fmt.Errorf("generate scheduler bucket epoch: %w", err)
	}
	return value.Int64() + 1, nil
}

func (c *schedulerCache) ensureSchedulerGeneration(ctx context.Context) (string, error) {
	candidate, err := newSchedulerGeneration()
	if err != nil {
		return "", err
	}
	result, err := ensureSchedulerGenerationScript.Run(
		ctx,
		c.rdb,
		[]string{schedulerGenerationKey, schedulerResetKey},
		candidate,
	).Slice()
	if err != nil {
		return "", err
	}
	if len(result) != 2 {
		return "", fmt.Errorf("ensure scheduler generation returned %d fields", len(result))
	}
	status, err := schedulerLuaInt64(result[0])
	if err != nil {
		return "", err
	}
	if status == 0 {
		return "", service.ErrSchedulerCacheResetInProgress
	}
	if status != 1 {
		return "", fmt.Errorf("ensure scheduler generation returned status %d", status)
	}
	generation, err := schedulerLuaString(result[1])
	if err != nil {
		return "", err
	}
	if generation == "" {
		return "", errors.New("ensure scheduler generation returned an empty generation")
	}
	return generation, nil
}

func (c *schedulerCache) ResetForAuthoritativeRebuild(ctx context.Context) error {
	_, err := c.resetForAuthoritativeRebuild(ctx, false)
	return err
}

func (c *schedulerCache) RecoverInterruptedAuthoritativeReset(ctx context.Context) (bool, error) {
	return c.resetForAuthoritativeRebuild(ctx, true)
}

func (c *schedulerCache) resetForAuthoritativeRebuild(ctx context.Context, requirePending bool) (bool, error) {
	if c == nil || c.rdb == nil {
		return false, service.ErrSchedulerCacheNotReady
	}
	ownerToken, err := newSchedulerGeneration()
	if err != nil {
		return false, err
	}
	generation, err := newSchedulerGeneration()
	if err != nil {
		return false, err
	}
	requirePendingArg := 0
	if requirePending {
		requirePendingArg = 1
	}
	started, err := beginSchedulerAuthoritativeResetScript.Run(
		ctx,
		c.rdb,
		[]string{schedulerGenerationKey, schedulerResetKey, schedulerResetLeaseKey},
		ownerToken,
		generation,
		schedulerAuthoritativeResetLeaseTTL.Milliseconds(),
		requirePendingArg,
	).Int64()
	if err != nil {
		return false, err
	}
	switch started {
	case 0:
		return false, service.ErrSchedulerCacheResetInProgress
	case 2:
		return false, nil
	case 1:
	default:
		return false, fmt.Errorf("begin scheduler authoritative reset returned %d", started)
	}

	for {
		deletedAny := false
		var cursor uint64
		for {
			keys, nextCursor, err := c.rdb.Scan(ctx, cursor, schedulerSnapshotPrefix+"*", 512).Result()
			if err != nil {
				return false, fmt.Errorf("scan scheduler cache for authoritative reset: %w", err)
			}
			deleteKeys := make([]string, 0, len(keys))
			for _, key := range keys {
				if key == schedulerGenerationKey || key == schedulerResetKey || key == schedulerResetLeaseKey {
					continue
				}
				deleteKeys = append(deleteKeys, key)
			}
			if len(deleteKeys) > 0 {
				deletedAny = true
			}
			if err := c.deleteAuthoritativeResetKeys(ctx, ownerToken, generation, deleteKeys); err != nil {
				return false, err
			}
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
		// SCAN may skip keys while the same pass deletes the hash-table entries it
		// is walking. Once a complete pass finds nothing, the reset marker prevents
		// generation-aware writers from repopulating the namespace.
		if !deletedAny {
			break
		}
	}

	finished, err := finishSchedulerAuthoritativeResetScript.Run(
		ctx,
		c.rdb,
		[]string{schedulerGenerationKey, schedulerResetKey, schedulerResetLeaseKey},
		ownerToken,
		generation,
	).Int64()
	if err != nil {
		return false, err
	}
	if finished != 1 {
		return false, errors.New("scheduler authoritative reset ownership was lost")
	}
	return true, nil
}

func (c *schedulerCache) deleteAuthoritativeResetKeys(ctx context.Context, ownerToken, generation string, deleteKeys []string) error {
	keys := make([]string, 0, len(deleteKeys)+3)
	keys = append(keys, schedulerGenerationKey, schedulerResetKey, schedulerResetLeaseKey)
	keys = append(keys, deleteKeys...)
	deleted, err := deleteSchedulerAuthoritativeResetKeysScript.Run(
		ctx,
		c.rdb,
		keys,
		ownerToken,
		generation,
		schedulerAuthoritativeResetLeaseTTL.Milliseconds(),
	).Int64()
	if err != nil {
		return fmt.Errorf("delete scheduler cache during authoritative reset: %w", err)
	}
	if deleted != 1 {
		return errors.New("scheduler authoritative reset ownership was lost")
	}
	return nil
}

func (c *schedulerCache) SetSnapshot(ctx context.Context, bucket service.SchedulerBucket, token service.SchedulerBucketWriteToken, accounts []service.Account) error {
	if !token.ValidFor(bucket) || token.Generation == "" {
		return fmt.Errorf("%w: bucket=%s", service.ErrSchedulerBucketWriteFenced, bucket.String())
	}
	// 分配版本与激活指针是两个 fencing 边界；中间写入的数据只有通过第二次校验才能发布。
	version, err := c.allocateSnapshotVersion(ctx, bucket, token)
	if err != nil {
		return err
	}
	if err := c.writeSnapshotVersion(ctx, bucket, version, accounts); err != nil {
		return err
	}
	return c.activateSnapshotVersion(ctx, bucket, token, version)
}

// SetSnapshotAndReturnAccountIDs 完整发布快照，并返回 writeAccounts 实际接受的有序账号 ID。
// 该可选能力只供同一重建批次复用，返回前仍会完成版本激活与 fencing 校验。
func (c *schedulerCache) SetSnapshotAndReturnAccountIDs(ctx context.Context, bucket service.SchedulerBucket, token service.SchedulerBucketWriteToken, accounts []service.Account) ([]int64, error) {
	if !token.ValidFor(bucket) || token.Generation == "" {
		return nil, fmt.Errorf("%w: bucket=%s", service.ErrSchedulerBucketWriteFenced, bucket.String())
	}
	// 分配版本与激活指针是两个 fencing 边界；中间写入的数据只有通过第二次校验才能发布。
	version, err := c.allocateSnapshotVersion(ctx, bucket, token)
	if err != nil {
		return nil, err
	}
	accountIDs, err := c.writeSnapshotVersionAndReturnAccountIDs(ctx, bucket, version, accounts)
	if err != nil {
		return nil, err
	}
	if err := c.activateSnapshotVersion(ctx, bucket, token, version); err != nil {
		return nil, err
	}
	return accountIDs, nil
}

// SetSnapshotByAccountIDs 复用同批次首次完整写入后得到的账号成员。
// 每个桶仍独立分配版本、写入有序集合并执行激活 fencing，只省略重复的账号 JSON 与全局键写入。
func (c *schedulerCache) SetSnapshotByAccountIDs(ctx context.Context, bucket service.SchedulerBucket, token service.SchedulerBucketWriteToken, accountIDs []int64) error {
	if !token.ValidFor(bucket) || token.Generation == "" {
		return fmt.Errorf("%w: bucket=%s", service.ErrSchedulerBucketWriteFenced, bucket.String())
	}
	version, err := c.allocateSnapshotVersion(ctx, bucket, token)
	if err != nil {
		return err
	}
	if err := c.writeSnapshotAccountIDs(ctx, bucket, version, accountIDs); err != nil {
		return err
	}
	return c.activateSnapshotVersion(ctx, bucket, token, version)
}

func (c *schedulerCache) allocateSnapshotVersion(ctx context.Context, bucket service.SchedulerBucket, token service.SchedulerBucketWriteToken) (string, error) {
	result, err := allocateSnapshotVersionScript.Run(ctx, c.rdb, []string{
		schedulerBucketKey(schedulerEpochPrefix, bucket),
		schedulerBucketKey(schedulerRetiredPrefix, bucket),
		schedulerBucketKey(schedulerVersionPrefix, bucket),
		schedulerGenerationKey,
		schedulerResetKey,
	}, strconv.FormatInt(token.Epoch, 10), token.Generation).Int64()
	if err != nil {
		return "", err
	}
	if err := schedulerBucketWriteResultError(result, bucket); err != nil {
		return "", err
	}
	return token.Generation + "." + strconv.FormatInt(result, 10), nil
}

func (c *schedulerCache) writeSnapshotVersion(ctx context.Context, bucket service.SchedulerBucket, version string, accounts []service.Account) error {
	generation, err := schedulerSnapshotVersionGeneration(version)
	if err != nil {
		return err
	}
	cacheableAccounts, err := c.writeAccountsForGeneration(ctx, accounts, generation)
	if err != nil {
		return err
	}
	return c.writeSnapshotAccounts(ctx, bucket, version, cacheableAccounts)
}

func (c *schedulerCache) writeSnapshotVersionAndReturnAccountIDs(ctx context.Context, bucket service.SchedulerBucket, version string, accounts []service.Account) ([]int64, error) {
	generation, err := schedulerSnapshotVersionGeneration(version)
	if err != nil {
		return nil, err
	}
	accountIDs, err := c.writeAccountIDsForGeneration(ctx, accounts, generation)
	if err != nil {
		return nil, err
	}
	if err := c.writeSnapshotAccountIDs(ctx, bucket, version, accountIDs); err != nil {
		return nil, err
	}
	return accountIDs, nil
}

func (c *schedulerCache) writeSnapshotAccounts(ctx context.Context, bucket service.SchedulerBucket, version string, accounts []service.Account) error {
	if len(accounts) == 0 {
		return nil
	}
	members := make([]redis.Z, 0, len(accounts))
	for idx, account := range accounts {
		members = append(members, redis.Z{
			Score:  float64(idx),
			Member: strconv.FormatInt(account.ID, 10),
		})
	}
	return c.writeSnapshotMembers(ctx, bucket, version, members)
}

func (c *schedulerCache) writeSnapshotAccountIDs(ctx context.Context, bucket service.SchedulerBucket, version string, accountIDs []int64) error {
	members := schedulerSnapshotMembers(accountIDs)
	return c.writeSnapshotMembers(ctx, bucket, version, members)
}

func schedulerSnapshotMembers(accountIDs []int64) []redis.Z {
	if len(accountIDs) == 0 {
		return nil
	}
	// 使用序号作为 score，保持数据库返回的排序语义；重复 ID 继续交由 Redis ZADD
	// 按最后一个 score 覆盖，与直接从账号切片构造成员时的行为一致。
	members := make([]redis.Z, 0, len(accountIDs))
	for idx, accountID := range accountIDs {
		members = append(members, redis.Z{
			Score:  float64(idx),
			Member: strconv.FormatInt(accountID, 10),
		})
	}
	return members
}

func (c *schedulerCache) writeSnapshotMembers(ctx context.Context, bucket service.SchedulerBucket, version string, members []redis.Z) error {
	if len(members) == 0 {
		return nil
	}
	generation, err := schedulerSnapshotVersionGeneration(version)
	if err != nil {
		return err
	}
	snapshotKey := schedulerSnapshotKey(bucket, version)
	for start := 0; start < len(members); start += c.writeChunkSize {
		end := start + c.writeChunkSize
		if end > len(members) {
			end = len(members)
		}
		args := make([]any, 0, 2+(end-start)*2)
		args = append(args, generation, schedulerSnapshotStagingTTL.Milliseconds())
		for _, member := range members[start:end] {
			args = append(args, member.Score, member.Member)
		}
		result, err := writeSnapshotMembersScript.Run(ctx, c.rdb, []string{
			snapshotKey,
			schedulerGenerationKey,
			schedulerResetKey,
		}, args...).Int64()
		if err != nil {
			return err
		}
		if err := schedulerBucketWriteResultError(result, bucket); err != nil {
			return err
		}
	}
	return nil
}

func (c *schedulerCache) activateSnapshotVersion(ctx context.Context, bucket service.SchedulerBucket, token service.SchedulerBucketWriteToken, version string) error {
	sequence, err := schedulerSnapshotVersionSequence(version, token.Generation)
	if err != nil {
		return err
	}
	snapshotKey := schedulerSnapshotKey(bucket, version)
	// Phase 2: 原子 CAS 切换版本，同时再次校验退休状态与 writer epoch。
	// Lua 脚本保证：仅当新版本 >= 当前激活版本时才切换 active 指针，
	// 防止并发写入导致版本回滚。
	// 旧快照使用 EXPIRE 宽限期而非立即 DEL，避免 reader 竞态。
	activeKey := schedulerBucketKey(schedulerActivePrefix, bucket)
	readyKey := schedulerBucketKey(schedulerReadyPrefix, bucket)
	snapshotKeyPrefix := fmt.Sprintf("%s%d:%s:%s:v", schedulerSnapshotPrefix, bucket.GroupID, bucket.Platform, bucket.Mode)

	keys := []string{
		activeKey,
		readyKey,
		schedulerBucketSetKey,
		snapshotKey,
		schedulerBucketKey(schedulerEpochPrefix, bucket),
		schedulerBucketKey(schedulerRetiredPrefix, bucket),
		schedulerGenerationKey,
		schedulerResetKey,
	}
	args := []any{version, bucket.String(), snapshotKeyPrefix, snapshotGraceTTLSeconds, strconv.FormatInt(token.Epoch, 10), token.Generation, sequence}

	result, err := activateSnapshotScript.Run(ctx, c.rdb, keys, args...).Int64()
	if err != nil {
		return err
	}
	return schedulerBucketWriteResultError(result, bucket)
}

func schedulerBucketWriteResultError(result int64, bucket service.SchedulerBucket) error {
	switch result {
	case -1:
		return fmt.Errorf("%w: bucket=%s", service.ErrSchedulerBucketRetired, bucket.String())
	case -2:
		return fmt.Errorf("%w: bucket=%s", service.ErrSchedulerBucketWriteFenced, bucket.String())
	default:
		return nil
	}
}

func schedulerBucketWriteTokenResult(result []any, bucket service.SchedulerBucket) (service.SchedulerBucketWriteToken, error) {
	if len(result) != 3 {
		return service.SchedulerBucketWriteToken{}, fmt.Errorf("scheduler bucket token returned %d fields", len(result))
	}
	status, err := schedulerLuaInt64(result[0])
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	if err := schedulerBucketWriteResultError(status, bucket); err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	if status != 1 {
		return service.SchedulerBucketWriteToken{}, fmt.Errorf("scheduler bucket token returned status %d", status)
	}
	generation, err := schedulerLuaString(result[1])
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	epochRaw, err := schedulerLuaString(result[2])
	if err != nil {
		return service.SchedulerBucketWriteToken{}, err
	}
	epoch, err := strconv.ParseInt(epochRaw, 10, 64)
	if err != nil || epoch <= 0 || epoch > schedulerMaxSafeLuaInteger {
		return service.SchedulerBucketWriteToken{}, fmt.Errorf("invalid scheduler bucket epoch %q", epochRaw)
	}
	if generation == "" {
		return service.SchedulerBucketWriteToken{}, errors.New("scheduler bucket token returned an empty generation")
	}
	return service.SchedulerBucketWriteToken{Bucket: bucket, Generation: generation, Epoch: epoch}, nil
}

func schedulerEpochResult(result []any) (int64, int64, error) {
	if len(result) != 2 {
		return 0, 0, fmt.Errorf("scheduler epoch result returned %d fields", len(result))
	}
	status, err := schedulerLuaInt64(result[0])
	if err != nil {
		return 0, 0, err
	}
	if status != 1 {
		return status, 0, nil
	}
	epochRaw, err := schedulerLuaString(result[1])
	if err != nil {
		return 0, 0, err
	}
	epoch, err := strconv.ParseInt(epochRaw, 10, 64)
	if err != nil || epoch <= 0 || epoch > schedulerMaxSafeLuaInteger {
		return 0, 0, fmt.Errorf("invalid scheduler bucket epoch %q", epochRaw)
	}
	return status, epoch, nil
}

func schedulerLuaInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse scheduler Lua integer %q: %w", typed, err)
		}
		return parsed, nil
	case []byte:
		return schedulerLuaInt64(string(typed))
	default:
		return 0, fmt.Errorf("unexpected scheduler Lua integer type %T", value)
	}
}

func schedulerLuaString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case []byte:
		return string(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	default:
		return "", fmt.Errorf("unexpected scheduler Lua string type %T", value)
	}
}

func schedulerSnapshotVersionGeneration(version string) (string, error) {
	dot := strings.LastIndexByte(version, '.')
	if dot <= 0 || dot == len(version)-1 {
		return "", fmt.Errorf("invalid scheduler snapshot version %q", version)
	}
	return version[:dot], nil
}

func schedulerSnapshotVersionSequence(version, generation string) (int64, error) {
	versionGeneration, err := schedulerSnapshotVersionGeneration(version)
	if err != nil {
		return 0, err
	}
	if versionGeneration != generation {
		return 0, service.ErrSchedulerBucketWriteFenced
	}
	sequenceRaw := version[len(generation)+1:]
	sequence, err := strconv.ParseInt(sequenceRaw, 10, 64)
	if err != nil || sequence <= 0 {
		return 0, fmt.Errorf("invalid scheduler snapshot sequence %q", sequenceRaw)
	}
	return sequence, nil
}

func (c *schedulerCache) GetAccount(ctx context.Context, accountID int64) (*service.Account, error) {
	key := schedulerAccountKey(strconv.FormatInt(accountID, 10))
	state, err := c.rdb.MGet(ctx, schedulerResetKey, schedulerGenerationKey, key).Result()
	if err != nil {
		return nil, err
	}
	if len(state) != 3 {
		return nil, fmt.Errorf("read scheduler account state returned %d fields", len(state))
	}
	readFence, err := schedulerReadFenceFromValues(state[:2])
	if err != nil {
		return nil, err
	}
	if state[2] == nil {
		return nil, nil
	}
	account, err := decodeCachedAccount(state[2])
	if err != nil {
		return nil, err
	}
	if err := c.validateReadFence(ctx, readFence); err != nil {
		return nil, err
	}
	return account, nil
}

func (c *schedulerCache) SetAccount(ctx context.Context, account *service.Account) error {
	if account == nil || account.ID <= 0 {
		return nil
	}
	generation, err := c.ensureSchedulerGeneration(ctx)
	if err != nil {
		return err
	}
	if _, _, err := marshalSchedulerCacheAccount(*account); err != nil {
		slog.Warn("scheduler cache removes account with unencodable payload",
			"account_id", account.ID,
			"error", err,
		)
		return c.deleteUnencodableAccountProjection(ctx, *account, generation)
	}
	// An encodable projection may still be omitted if its CAS observes a
	// concurrent account deletion. That is a successful deletion-wins outcome,
	// not a reason to enter the unencodable deletion path again.
	_, err = c.writeAccountsForGeneration(ctx, []service.Account{*account}, generation)
	return err
}

func (c *schedulerCache) DeleteAccount(ctx context.Context, accountID int64) error {
	if accountID <= 0 {
		return nil
	}
	generation, err := c.ensureSchedulerGeneration(ctx)
	if err != nil {
		return err
	}
	id := strconv.FormatInt(accountID, 10)
	result, err := deleteAccountProjectionScript.Run(
		ctx,
		c.rdb,
		[]string{schedulerAccountKey(id), schedulerAccountMetaKey(id), schedulerAccountTombstoneSetKey, schedulerGenerationKey, schedulerResetKey},
		id,
		generation,
	).Int64()
	if err != nil {
		return err
	}
	if result == -2 {
		return service.ErrSchedulerBucketWriteFenced
	}
	if result != 1 {
		return fmt.Errorf("delete scheduler account projection returned %d", result)
	}
	return nil
}

func (c *schedulerCache) UpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	if len(updates) == 0 {
		return nil
	}
	generation, err := c.ensureSchedulerGeneration(ctx)
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(updates))
	ids := make([]int64, 0, len(updates))
	for id := range updates {
		keys = append(keys, schedulerAccountKey(strconv.FormatInt(id, 10)))
		ids = append(ids, id)
	}

	values, err := c.mgetChunked(ctx, keys)
	if err != nil {
		return err
	}

	writes := make([]schedulerLastUsedWrite, 0, len(values))
	for i, val := range values {
		if val == nil {
			continue
		}
		write, needed, err := prepareSchedulerLastUsedWrite(ids[i], updates[ids[i]], val)
		if err != nil {
			return err
		}
		if needed {
			writes = append(writes, write)
		}
	}

	chunkSize := c.writeChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultSchedulerSnapshotWriteChunkSize
	}
	for start := 0; start < len(writes); start += chunkSize {
		end := start + chunkSize
		if end > len(writes) {
			end = len(writes)
		}
		pipe := c.rdb.Pipeline()
		commands := make([]*redis.Cmd, 0, end-start)
		for idx := start; idx < end; idx++ {
			commands = append(commands, queueSchedulerLastUsedCAS(ctx, pipe, writes[idx], generation))
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
		for idx, command := range commands {
			write := writes[start+idx]
			result, err := command.Int64()
			if err != nil {
				return err
			}
			switch result {
			case 1:
				logSchedulerLastUsedEncodingCleanup(write)
			case 0:
				// Missing and tombstoned accounts are both deletion-wins no-ops.
			case -1:
				if err := c.retryAccountLastUsedCAS(ctx, write.accountID, write.candidate, generation); err != nil {
					return err
				}
			case -2:
				return service.ErrSchedulerBucketWriteFenced
			default:
				return fmt.Errorf("update scheduler account last_used CAS returned %d", result)
			}
		}
	}
	return nil
}

type schedulerLastUsedWrite struct {
	accountID      int64
	candidate      time.Time
	currentPayload string
	mode           string
	fullPayload    []byte
	metaPayload    []byte
	marshalError   error
}

func prepareSchedulerLastUsedWrite(accountID int64, candidate time.Time, current any) (schedulerLastUsedWrite, bool, error) {
	currentPayload, err := schedulerAccountPayloadString(current)
	if err != nil {
		return schedulerLastUsedWrite{}, false, err
	}
	account, err := decodeCachedAccount(currentPayload)
	if err != nil {
		return schedulerLastUsedWrite{}, false, err
	}
	if account.LastUsedAt != nil && !candidate.After(*account.LastUsedAt) {
		return schedulerLastUsedWrite{}, false, nil
	}
	account.LastUsedAt = ptrTime(candidate)
	fullPayload, metaPayload, marshalErr := marshalSchedulerCacheAccount(*account)
	mode := "set"
	if marshalErr != nil {
		mode = "delete"
		fullPayload = nil
		metaPayload = nil
	}
	return schedulerLastUsedWrite{
		accountID:      accountID,
		candidate:      candidate,
		currentPayload: currentPayload,
		mode:           mode,
		fullPayload:    fullPayload,
		metaPayload:    metaPayload,
		marshalError:   marshalErr,
	}, true, nil
}

func queueSchedulerLastUsedCAS(ctx context.Context, scripter redis.Scripter, write schedulerLastUsedWrite, generation string) *redis.Cmd {
	id := strconv.FormatInt(write.accountID, 10)
	return updateLastUsedCASScript.Eval(
		ctx,
		scripter,
		[]string{schedulerAccountKey(id), schedulerAccountMetaKey(id), schedulerAccountTombstoneSetKey, schedulerGenerationKey, schedulerResetKey},
		write.currentPayload,
		write.mode,
		write.fullPayload,
		write.metaPayload,
		id,
		generation,
	)
}

func (c *schedulerCache) retryAccountLastUsedCAS(ctx context.Context, accountID int64, candidate time.Time, generation string) error {
	fullKey := schedulerAccountKey(strconv.FormatInt(accountID, 10))
	for attempt := 0; attempt < schedulerLastUsedCASMaxRetries; attempt++ {
		current, err := c.rdb.Get(ctx, fullKey).Result()
		if err == redis.Nil {
			return nil
		}
		if err != nil {
			return err
		}
		write, needed, err := prepareSchedulerLastUsedWrite(accountID, candidate, current)
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
			continue
		case -2:
			return service.ErrSchedulerBucketWriteFenced
		default:
			return fmt.Errorf("retry scheduler account last_used CAS returned %d", result)
		}
	}
	return fmt.Errorf("update scheduler account last_used CAS exhausted retries: account=%d", accountID)
}

func logSchedulerLastUsedEncodingCleanup(write schedulerLastUsedWrite) {
	if write.marshalError == nil {
		return
	}
	slog.Warn("scheduler cache removes account with unencodable payload",
		"account_id", write.accountID,
		"error", write.marshalError,
	)
}

func schedulerAccountPayloadString(value any) (string, error) {
	switch payload := value.(type) {
	case string:
		return payload, nil
	case []byte:
		return string(payload), nil
	default:
		return "", fmt.Errorf("unexpected account cache type: %T", value)
	}
}

func (c *schedulerCache) TryAcquireBucketRebuildLease(ctx context.Context, bucket service.SchedulerBucket, token service.SchedulerBucketWriteToken, ttl time.Duration) (service.SchedulerBucketRebuildLease, bool, error) {
	if !token.ValidFor(bucket) || token.Generation == "" {
		return service.SchedulerBucketRebuildLease{}, false, fmt.Errorf("%w: bucket=%s", service.ErrSchedulerBucketWriteFenced, bucket.String())
	}
	if ttl <= 0 {
		return service.SchedulerBucketRebuildLease{}, false, fmt.Errorf("%w: ttl must be positive", service.ErrSchedulerBucketRebuildLeaseInvalid)
	}
	ownerToken, err := newSchedulerGroupLifecycleOwnerToken()
	if err != nil {
		return service.SchedulerBucketRebuildLease{}, false, err
	}
	result, err := acquireBucketRebuildLeaseScript.Run(
		ctx,
		c.rdb,
		[]string{
			schedulerBucketKey(schedulerLockPrefix, bucket),
			schedulerGenerationKey,
			schedulerResetKey,
		},
		ownerToken,
		token.Generation,
		ttl.Milliseconds(),
	).Int64()
	if err != nil {
		return service.SchedulerBucketRebuildLease{}, false, err
	}
	if result == -1 {
		return service.SchedulerBucketRebuildLease{}, false, fmt.Errorf("%w: bucket=%s", service.ErrSchedulerBucketWriteFenced, bucket.String())
	}
	if result == 0 {
		return service.SchedulerBucketRebuildLease{}, false, nil
	}
	if result != 1 {
		return service.SchedulerBucketRebuildLease{}, false, fmt.Errorf("acquire scheduler bucket rebuild lease returned %d", result)
	}
	return service.SchedulerBucketRebuildLease{Bucket: bucket, OwnerToken: ownerToken}, true, nil
}

func (c *schedulerCache) ReleaseBucketRebuildLease(ctx context.Context, lease service.SchedulerBucketRebuildLease) error {
	if !lease.ValidFor(lease.Bucket) {
		return service.ErrSchedulerBucketRebuildLeaseInvalid
	}
	result, err := releaseBucketRebuildLeaseScript.Run(
		ctx,
		c.rdb,
		[]string{schedulerBucketKey(schedulerLockPrefix, lease.Bucket)},
		lease.OwnerToken,
	).Int64()
	if err != nil {
		return err
	}
	if result == 0 {
		return fmt.Errorf("%w: bucket=%s", service.ErrSchedulerBucketRebuildLeaseLost, lease.Bucket.String())
	}
	if result != 1 {
		return fmt.Errorf("release scheduler bucket rebuild lease returned %d", result)
	}
	return nil
}

func (c *schedulerCache) TryLockBucket(ctx context.Context, bucket service.SchedulerBucket, ttl time.Duration) (bool, error) {
	key := schedulerBucketKey(schedulerLockPrefix, bucket)
	return c.rdb.SetNX(ctx, key, time.Now().UnixNano(), ttl).Result()
}

func (c *schedulerCache) UnlockBucket(ctx context.Context, bucket service.SchedulerBucket) error {
	key := schedulerBucketKey(schedulerLockPrefix, bucket)
	return c.rdb.Del(ctx, key).Err()
}

func (c *schedulerCache) ListBuckets(ctx context.Context) ([]service.SchedulerBucket, error) {
	readFence, err := c.captureReadFence(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := c.rdb.SMembers(ctx, schedulerBucketSetKey).Result()
	if err != nil {
		return nil, err
	}
	out := make([]service.SchedulerBucket, 0, len(raw))
	for _, entry := range raw {
		bucket, ok := service.ParseSchedulerBucket(entry)
		if !ok {
			continue
		}
		out = append(out, bucket)
	}
	if err := c.validateReadFence(ctx, readFence); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *schedulerCache) GetOutboxWatermark(ctx context.Context) (int64, error) {
	state, err := c.rdb.MGet(ctx, schedulerResetKey, schedulerGenerationKey, schedulerOutboxWatermarkKey).Result()
	if err != nil {
		return 0, err
	}
	if len(state) != 3 {
		return 0, fmt.Errorf("read scheduler outbox watermark state returned %d fields", len(state))
	}
	readFence, err := schedulerReadFenceFromValues(state[:2])
	if err != nil {
		return 0, err
	}
	if state[2] == nil {
		if err := c.validateReadFence(ctx, readFence); err != nil {
			return 0, err
		}
		return 0, nil
	}
	val, err := schedulerLuaString(state[2])
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}
	if err := c.validateReadFence(ctx, readFence); err != nil {
		return 0, err
	}
	return id, nil
}

func (c *schedulerCache) SetOutboxWatermark(ctx context.Context, id int64) error {
	return advanceOutboxWatermarkScript.Run(
		ctx,
		c.rdb,
		[]string{schedulerOutboxWatermarkKey},
		strconv.FormatInt(id, 10),
	).Err()
}

func schedulerBucketKey(prefix string, bucket service.SchedulerBucket) string {
	return fmt.Sprintf("%s%d:%s:%s", prefix, bucket.GroupID, bucket.Platform, bucket.Mode)
}

func schedulerGroupLifecycleLockKey(groupID int64) string {
	return schedulerGroupLifecycleLockPrefix + strconv.FormatInt(groupID, 10)
}

func schedulerSnapshotKey(bucket service.SchedulerBucket, version string) string {
	return fmt.Sprintf("%s%d:%s:%s:v%s", schedulerSnapshotPrefix, bucket.GroupID, bucket.Platform, bucket.Mode, version)
}

func schedulerAccountKey(id string) string {
	return schedulerAccountPrefix + id
}

func schedulerAccountMetaKey(id string) string {
	return schedulerAccountMetaPrefix + id
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func decodeCachedAccount(val any) (*service.Account, error) {
	var payload []byte
	switch raw := val.(type) {
	case string:
		payload = []byte(raw)
	case []byte:
		payload = raw
	default:
		return nil, fmt.Errorf("unexpected account cache type: %T", val)
	}
	var account service.Account
	if err := json.Unmarshal(payload, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

func (c *schedulerCache) writeAccountsForGeneration(ctx context.Context, accounts []service.Account, generation string) ([]service.Account, error) {
	cacheableAccounts, _, err := c.writeAccountPayloads(ctx, accounts, false, generation)
	return cacheableAccounts, err
}

func (c *schedulerCache) writeAccountIDsForGeneration(ctx context.Context, accounts []service.Account, generation string) ([]int64, error) {
	_, accountIDs, err := c.writeAccountPayloads(ctx, accounts, true, generation)
	return accountIDs, err
}

func (c *schedulerCache) writeAccountPayloads(ctx context.Context, accounts []service.Account, collectIDs bool, generation string) ([]service.Account, []int64, error) {
	if len(accounts) == 0 {
		return nil, nil, nil
	}
	if generation == "" {
		return nil, nil, service.ErrSchedulerBucketWriteFenced
	}

	cacheableCandidates := make([]service.Account, 0, len(accounts))
	keys := make([]string, 0, len(accounts))
	for _, account := range accounts {
		if _, _, err := marshalSchedulerCacheAccount(account); err != nil {
			slog.Warn("scheduler cache skips account with unencodable payload",
				"account_id", account.ID,
				"error", err,
			)
			continue
		}
		cacheableCandidates = append(cacheableCandidates, account)
		keys = append(keys, schedulerAccountKey(strconv.FormatInt(account.ID, 10)))
	}
	if len(cacheableCandidates) == 0 {
		return nil, nil, nil
	}

	currentPayloads, err := c.mgetChunked(ctx, keys)
	if err != nil {
		return nil, nil, err
	}

	var cacheableAccounts []service.Account
	var accountIDs []int64
	if collectIDs {
		accountIDs = make([]int64, 0, len(cacheableCandidates))
	} else {
		cacheableAccounts = make([]service.Account, 0, len(cacheableCandidates))
	}
	appendWritten := func(account service.Account) {
		if collectIDs {
			accountIDs = append(accountIDs, account.ID)
		} else {
			cacheableAccounts = append(cacheableAccounts, account)
		}
	}

	chunkSize := c.writeChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultSchedulerSnapshotWriteChunkSize
	}
	for start := 0; start < len(cacheableCandidates); start += chunkSize {
		end := start + chunkSize
		if end > len(cacheableCandidates) {
			end = len(cacheableCandidates)
		}

		pipe := c.rdb.Pipeline()
		writes := make([]schedulerAccountProjectionWrite, 0, end-start)
		commands := make([]*redis.Cmd, 0, end-start)
		for idx := start; idx < end; idx++ {
			write, err := prepareSchedulerAccountProjection(cacheableCandidates[idx], currentPayloads[idx])
			if err != nil {
				return nil, nil, err
			}
			writes = append(writes, write)
			commands = append(commands, queueSchedulerAccountProjection(ctx, pipe, write, "set", generation))
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return nil, nil, err
		}

		for idx, command := range commands {
			result, err := command.Int64()
			if err != nil {
				return nil, nil, err
			}
			switch result {
			case 1:
				appendWritten(writes[idx].account)
			case 0:
				// A durable account deletion won. Omitting the account here
				// also keeps it out of the snapshot member set being built.
			case -1:
				written, ok, err := c.retrySchedulerAccountProjection(ctx, cacheableCandidates[start+idx], generation)
				if err != nil {
					return nil, nil, err
				}
				if ok {
					appendWritten(written)
				}
			case -2:
				return nil, nil, service.ErrSchedulerBucketWriteFenced
			default:
				return nil, nil, fmt.Errorf("write scheduler account projection CAS returned %d", result)
			}
		}
	}
	return cacheableAccounts, accountIDs, nil
}

type schedulerAccountProjectionWrite struct {
	account         service.Account
	expectedPayload string
	expectedMissing bool
	fullPayload     []byte
	metaPayload     []byte
}

func prepareSchedulerAccountProjection(incoming service.Account, current any) (schedulerAccountProjectionWrite, error) {
	write := schedulerAccountProjectionWrite{
		account:         incoming,
		expectedMissing: current == nil,
	}
	if current != nil {
		currentPayload, err := schedulerAccountPayloadString(current)
		if err != nil {
			return schedulerAccountProjectionWrite{}, err
		}
		write.expectedPayload = currentPayload
		cached, err := decodeCachedAccount(currentPayload)
		if err == nil {
			write.account = mergeSchedulerAccountProjection(incoming, *cached)
		}
	}

	fullPayload, metaPayload, err := marshalSchedulerCacheAccount(write.account)
	if err != nil {
		return schedulerAccountProjectionWrite{}, err
	}
	write.fullPayload = fullPayload
	write.metaPayload = metaPayload
	return write, nil
}

func mergeSchedulerAccountProjection(incoming, cached service.Account) service.Account {
	merged := incoming
	if cached.UpdatedAt.After(incoming.UpdatedAt) {
		merged = cached
	}
	if cached.LastUsedAt != nil && (merged.LastUsedAt == nil || cached.LastUsedAt.After(*merged.LastUsedAt)) {
		merged.LastUsedAt = ptrTime(*cached.LastUsedAt)
	}
	if incoming.LastUsedAt != nil && (merged.LastUsedAt == nil || incoming.LastUsedAt.After(*merged.LastUsedAt)) {
		merged.LastUsedAt = ptrTime(*incoming.LastUsedAt)
	}
	return merged
}

func queueSchedulerAccountProjection(ctx context.Context, scripter redis.Scripter, write schedulerAccountProjectionWrite, mode, generation string) *redis.Cmd {
	expectedState := "present"
	if write.expectedMissing {
		expectedState = "missing"
	}
	id := strconv.FormatInt(write.account.ID, 10)
	return writeAccountProjectionCASScript.Eval(
		ctx,
		scripter,
		[]string{schedulerAccountKey(id), schedulerAccountMetaKey(id), schedulerAccountTombstoneSetKey, schedulerGenerationKey, schedulerResetKey},
		expectedState,
		write.expectedPayload,
		mode,
		write.fullPayload,
		write.metaPayload,
		id,
		generation,
	)
}

func (c *schedulerCache) retrySchedulerAccountProjection(ctx context.Context, incoming service.Account, generation string) (service.Account, bool, error) {
	id := strconv.FormatInt(incoming.ID, 10)
	for attempt := 0; attempt < schedulerLastUsedCASMaxRetries; attempt++ {
		current, err := c.rdb.Get(ctx, schedulerAccountKey(id)).Result()
		if err == redis.Nil {
			// A concurrent account deletion wins over a projection that started
			// from an older value. Returning no member also prevents a freshly
			// published snapshot from pointing at a missing metadata key.
			return service.Account{}, false, nil
		}
		if err != nil {
			return service.Account{}, false, err
		}
		write, err := prepareSchedulerAccountProjection(incoming, current)
		if err != nil {
			return service.Account{}, false, err
		}
		result, err := queueSchedulerAccountProjection(ctx, c.rdb, write, "set", generation).Int64()
		if err != nil {
			return service.Account{}, false, err
		}
		switch result {
		case 1:
			return write.account, true, nil
		case 0:
			return service.Account{}, false, nil
		case -1:
			continue
		case -2:
			return service.Account{}, false, service.ErrSchedulerBucketWriteFenced
		default:
			return service.Account{}, false, fmt.Errorf("retry scheduler account projection CAS returned %d", result)
		}
	}
	return service.Account{}, false, fmt.Errorf("write scheduler account projection CAS exhausted retries: account=%d", incoming.ID)
}

func (c *schedulerCache) deleteUnencodableAccountProjection(ctx context.Context, incoming service.Account, generation string) error {
	id := strconv.FormatInt(incoming.ID, 10)
	for attempt := 0; attempt < schedulerLastUsedCASMaxRetries; attempt++ {
		current, err := c.rdb.Get(ctx, schedulerAccountKey(id)).Result()
		if err == redis.Nil {
			write := schedulerAccountProjectionWrite{account: incoming, expectedMissing: true}
			result, err := queueSchedulerAccountProjection(ctx, c.rdb, write, "delete", generation).Int64()
			if err != nil {
				return err
			}
			if result == 1 {
				return nil
			}
			if result == 0 {
				// A durable account deletion won the race. Unencodable
				// cleanup never creates or clears that tombstone.
				return nil
			}
			if result == -1 {
				continue
			}
			if result == -2 {
				return service.ErrSchedulerBucketWriteFenced
			}
			return fmt.Errorf("delete missing scheduler account projection CAS returned %d", result)
		}
		if err != nil {
			return err
		}

		cached, decodeErr := decodeCachedAccount(current)
		if decodeErr == nil && cached.UpdatedAt.After(incoming.UpdatedAt) {
			return nil
		}
		write := schedulerAccountProjectionWrite{
			account:         incoming,
			expectedPayload: current,
		}
		result, err := queueSchedulerAccountProjection(ctx, c.rdb, write, "delete", generation).Int64()
		if err != nil {
			return err
		}
		switch result {
		case 1:
			return nil
		case 0:
			// A durable account deletion won the race. Unencodable
			// cleanup never creates or clears that tombstone.
			return nil
		case -1:
			continue
		case -2:
			return service.ErrSchedulerBucketWriteFenced
		default:
			return fmt.Errorf("delete scheduler account projection CAS returned %d", result)
		}
	}
	return fmt.Errorf("delete scheduler account projection CAS exhausted retries: account=%d", incoming.ID)
}

func marshalSchedulerCacheAccount(account service.Account) ([]byte, []byte, error) {
	fullPayload, err := json.Marshal(account)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal account: %w", err)
	}
	metaPayload, err := json.Marshal(buildSchedulerMetadataAccount(account))
	if err != nil {
		return nil, nil, fmt.Errorf("marshal account metadata: %w", err)
	}
	return fullPayload, metaPayload, nil
}

func (c *schedulerCache) mgetChunked(ctx context.Context, keys []string) ([]any, error) {
	if len(keys) == 0 {
		return []any{}, nil
	}

	out := make([]any, 0, len(keys))
	chunkSize := c.mgetChunkSize
	if chunkSize <= 0 {
		chunkSize = defaultSchedulerSnapshotMGetChunkSize
	}
	for start := 0; start < len(keys); start += chunkSize {
		end := start + chunkSize
		if end > len(keys) {
			end = len(keys)
		}
		part, err := c.rdb.MGet(ctx, keys[start:end]...).Result()
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

func buildSchedulerMetadataAccount(account service.Account) service.Account {
	return service.Account{
		ID:                      account.ID,
		Name:                    account.Name,
		Platform:                account.Platform,
		Type:                    account.Type,
		Concurrency:             account.Concurrency,
		LoadFactor:              account.LoadFactor,
		Priority:                account.Priority,
		RateMultiplier:          account.RateMultiplier,
		Status:                  account.Status,
		LastUsedAt:              account.LastUsedAt,
		ExpiresAt:               account.ExpiresAt,
		AutoPauseOnExpired:      account.AutoPauseOnExpired,
		Schedulable:             account.Schedulable,
		RateLimitedAt:           account.RateLimitedAt,
		RateLimitResetAt:        account.RateLimitResetAt,
		OverloadUntil:           account.OverloadUntil,
		TempUnschedulableUntil:  account.TempUnschedulableUntil,
		TempUnschedulableReason: account.TempUnschedulableReason,
		SessionWindowStart:      account.SessionWindowStart,
		SessionWindowEnd:        account.SessionWindowEnd,
		SessionWindowStatus:     account.SessionWindowStatus,
		ParentAccountID:         account.ParentAccountID,
		QuotaDimension:          account.QuotaDimension,
		AccountGroups:           filterSchedulerAccountGroups(account.AccountGroups),
		GroupIDs:                filterSchedulerGroupIDs(account.GroupIDs, account.AccountGroups),
		Credentials:             filterSchedulerCredentials(account.Credentials),
		Extra:                   filterSchedulerExtra(account.Extra),
	}
}

func filterSchedulerAccountGroups(accountGroups []service.AccountGroup) []service.AccountGroup {
	if len(accountGroups) == 0 {
		return nil
	}

	filtered := make([]service.AccountGroup, 0, len(accountGroups))
	for _, ag := range accountGroups {
		if ag.GroupID <= 0 {
			continue
		}
		filtered = append(filtered, service.AccountGroup{
			AccountID: ag.AccountID,
			GroupID:   ag.GroupID,
			Priority:  ag.Priority,
			CreatedAt: ag.CreatedAt,
		})
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func filterSchedulerGroupIDs(groupIDs []int64, accountGroups []service.AccountGroup) []int64 {
	if len(groupIDs) == 0 && len(accountGroups) == 0 {
		return nil
	}

	seen := make(map[int64]struct{}, len(groupIDs)+len(accountGroups))
	filtered := make([]int64, 0, len(groupIDs)+len(accountGroups))
	for _, id := range groupIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		filtered = append(filtered, id)
	}
	for _, ag := range accountGroups {
		if ag.GroupID <= 0 {
			continue
		}
		if _, ok := seen[ag.GroupID]; ok {
			continue
		}
		seen[ag.GroupID] = struct{}{}
		filtered = append(filtered, ag.GroupID)
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func filterSchedulerCredentials(credentials map[string]any) map[string]any {
	if len(credentials) == 0 {
		return nil
	}
	keys := []string{"model_mapping", "compact_model_mapping", "openai_capabilities", "api_key", "project_id", "oauth_type", "plan_type"}
	filtered := make(map[string]any)
	for _, key := range keys {
		if value, ok := credentials[key]; ok && value != nil {
			filtered[key] = value
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func filterSchedulerExtra(extra map[string]any) map[string]any {
	if len(extra) == 0 {
		return nil
	}
	keys := []string{
		"mixed_scheduling",
		"window_cost_limit",
		"window_cost_sticky_reserve",
		"max_sessions",
		"session_idle_timeout_minutes",
		"openai_oauth_responses_websockets_v2_enabled",
		"openai_oauth_responses_websockets_v2_mode",
		"openai_apikey_responses_websockets_v2_enabled",
		"openai_apikey_responses_websockets_v2_mode",
		"responses_websockets_v2_enabled",
		"openai_ws_enabled",
		"openai_ws_force_http",
		"openai_responses_mode",
		"openai_responses_supported",
		"codex_5h_used_percent",
		"codex_7d_used_percent",
		"codex_5h_reset_at",
		"codex_7d_reset_at",
		"codex_5h_reset_after_seconds",
		"codex_7d_reset_after_seconds",
		"codex_usage_updated_at",
		"auto_pause_5h_threshold",
		"auto_pause_7d_threshold",
		"auto_pause_5h_disabled",
		"auto_pause_7d_disabled",
		"model_rate_limits",
		service.UpstreamBillingProbeExtraKey,
	}
	filtered := make(map[string]any)
	for _, key := range keys {
		if value, ok := extra[key]; ok && value != nil {
			if key == service.UpstreamBillingProbeExtraKey {
				filteredProbe := filterSchedulerUpstreamBillingProbe(value)
				if filteredProbe == nil {
					continue
				}
				value = filteredProbe
			}
			filtered[key] = value
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func filterSchedulerUpstreamBillingProbe(value any) map[string]any {
	source, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	status, ok := source["status"].(string)
	if !ok || status == "" {
		return nil
	}
	filtered := map[string]any{"status": status}
	for _, key := range []string{"received_at", "fresh_until", "next_probe_at"} {
		if field, exists := source[key]; exists && field != nil {
			filtered[key] = field
		}
	}
	data, ok := source["data"].(map[string]any)
	if !ok {
		return filtered
	}
	filteredData := make(map[string]any)
	for _, key := range []string{
		"billing_scope",
		"resolved_rate_multiplier",
		"peak_rate_enabled",
		"peak_start",
		"peak_end",
		"peak_rate_multiplier",
		"timezone",
	} {
		if field, exists := data[key]; exists && field != nil {
			filteredData[key] = field
		}
	}
	if len(filteredData) > 0 {
		filtered["data"] = filteredData
	}
	return filtered
}
