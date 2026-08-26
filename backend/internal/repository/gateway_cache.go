package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const stickySessionPrefix = "sticky_session:"

// The shared Redis Cluster hash tag keeps every key used by admission Lua
// scripts in the same slot.
const transcriptionAdmissionPrefix = "transcription:admission:{transcription}:"

var transcriptionAcquireScript = redis.NewScript(`
local now_ms = tonumber(ARGV[1])
local lease_expiry_ms = tonumber(ARGV[2])
local idempotency_ttl_ms = tonumber(ARGV[3])
local global_limit = tonumber(ARGV[4])
local user_concurrency_limit = tonumber(ARGV[5])
local user_rpm_limit = tonumber(ARGV[6])
local ip_rpm_limit = tonumber(ARGV[7])
local lease_id = ARGV[8]

local user_rpm = tonumber(redis.call('GET', KEYS[2]) or '0')
local ip_rpm = tonumber(redis.call('GET', KEYS[3]) or '0')
if user_rpm >= user_rpm_limit or ip_rpm >= ip_rpm_limit then
  return 'rate_limited'
end
redis.call('INCR', KEYS[2])
redis.call('PEXPIRE', KEYS[2], 120000)
redis.call('INCR', KEYS[3])
redis.call('PEXPIRE', KEYS[3], 120000)

if redis.call('EXISTS', KEYS[1]) == 1 then
  return 'duplicate'
end

redis.call('ZREMRANGEBYSCORE', KEYS[4], '-inf', now_ms)
redis.call('ZREMRANGEBYSCORE', KEYS[5], '-inf', now_ms)
if redis.call('ZCARD', KEYS[4]) >= global_limit or redis.call('ZCARD', KEYS[5]) >= user_concurrency_limit then
  return 'busy'
end

local claimed = redis.call('SET', KEYS[1], lease_id, 'NX', 'PX', idempotency_ttl_ms)
if not claimed then
  return 'duplicate'
end

redis.call('ZADD', KEYS[4], lease_expiry_ms, lease_id)
redis.call('ZADD', KEYS[5], lease_expiry_ms, lease_id)

local function expire_after_latest_lease(key)
  local latest = redis.call('ZREVRANGE', key, 0, 0, 'WITHSCORES')
  local latest_expiry_ms = tonumber(latest[2] or lease_expiry_ms)
  local ttl_ms = latest_expiry_ms - now_ms + 60000
  if ttl_ms < 60000 then
    ttl_ms = 60000
  end
  redis.call('PEXPIRE', key, ttl_ms)
end

expire_after_latest_lease(KEYS[4])
expire_after_latest_lease(KEYS[5])
return 'acquired'
`)

var transcriptionReleaseScript = redis.NewScript(`
redis.call('ZREM', KEYS[1], ARGV[1])
redis.call('ZREM', KEYS[2], ARGV[1])
return 1
`)

var transcriptionDailyScript = redis.NewScript(`
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local addition = tonumber(ARGV[1])
local maximum = tonumber(ARGV[2])
if current + addition > maximum then
  return 0
end
local updated = redis.call('INCRBY', KEYS[1], addition)
redis.call('EXPIRE', KEYS[1], tonumber(ARGV[3]))
return updated
`)

type gatewayCache struct {
	rdb *redis.Client
}

func NewGatewayCache(rdb *redis.Client) service.GatewayCache {
	return &gatewayCache{rdb: rdb}
}

// buildSessionKey 构建 session key，包含 groupID 实现分组隔离
// 格式: sticky_session:{groupID}:{sessionHash}
func buildSessionKey(groupID int64, sessionHash string) string {
	return fmt.Sprintf("%s%d:%s", stickySessionPrefix, groupID, sessionHash)
}

func (c *gatewayCache) GetSessionAccountID(ctx context.Context, groupID int64, sessionHash string) (int64, error) {
	key := buildSessionKey(groupID, sessionHash)
	accountID, err := c.rdb.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, service.ErrStickySessionNotFound
	}
	return accountID, err
}

func (c *gatewayCache) SetSessionAccountID(ctx context.Context, groupID int64, sessionHash string, accountID int64, ttl time.Duration) error {
	key := buildSessionKey(groupID, sessionHash)
	return c.rdb.Set(ctx, key, accountID, ttl).Err()
}

func (c *gatewayCache) RefreshSessionTTL(ctx context.Context, groupID int64, sessionHash string, ttl time.Duration) error {
	key := buildSessionKey(groupID, sessionHash)
	return c.rdb.Expire(ctx, key, ttl).Err()
}

// DeleteSessionAccountID 删除粘性会话与账号的绑定关系。
// 当检测到绑定的账号不可用（如状态错误、禁用、不可调度等）时调用，
// 以便下次请求能够重新选择可用账号。
//
// DeleteSessionAccountID removes the sticky session binding for the given session.
// Called when the bound account becomes unavailable (e.g., error status, disabled,
// or unschedulable), allowing subsequent requests to select a new available account.
func (c *gatewayCache) DeleteSessionAccountID(ctx context.Context, groupID int64, sessionHash string) error {
	key := buildSessionKey(groupID, sessionHash)
	return c.rdb.Del(ctx, key).Err()
}

func (c *gatewayCache) AcquireTranscriptionAdmission(ctx context.Context, input service.TranscriptionAdmissionCacheInput) (service.TranscriptionAdmissionDecision, error) {
	if c == nil || c.rdb == nil || input.UserID <= 0 || input.IPHash == "" || input.IdempotencyKeyHash == "" || input.LeaseID == "" ||
		input.LeaseTTL <= 0 || input.IdempotencyTTL <= 0 || input.MaxConcurrentGlobal <= 0 || input.MaxConcurrentPerUser <= 0 ||
		input.UserRequestsPerMinute <= 0 || input.IPRequestsPerMinute <= 0 {
		return "", fmt.Errorf("invalid transcription admission input")
	}
	serverTime, err := c.rdb.Time(ctx).Result()
	if err != nil {
		return "", fmt.Errorf("transcription admission redis time: %w", err)
	}
	minute := serverTime.Unix() / 60
	nowMS := serverTime.UnixMilli()
	leaseTTLMS := input.LeaseTTL.Milliseconds()
	keys := []string{
		fmt.Sprintf("%sidempotency:%d:%s", transcriptionAdmissionPrefix, input.UserID, input.IdempotencyKeyHash),
		fmt.Sprintf("%srpm:user:%d:%d", transcriptionAdmissionPrefix, input.UserID, minute),
		fmt.Sprintf("%srpm:ip:%s:%d", transcriptionAdmissionPrefix, input.IPHash, minute),
		transcriptionAdmissionPrefix + "concurrency:global",
		fmt.Sprintf("%sconcurrency:user:%d", transcriptionAdmissionPrefix, input.UserID),
	}
	result, err := transcriptionAcquireScript.Run(ctx, c.rdb, keys,
		nowMS,
		nowMS+leaseTTLMS,
		input.IdempotencyTTL.Milliseconds(),
		input.MaxConcurrentGlobal,
		input.MaxConcurrentPerUser,
		input.UserRequestsPerMinute,
		input.IPRequestsPerMinute,
		input.LeaseID,
	).Text()
	if err != nil {
		return "", fmt.Errorf("transcription admission acquire: %w", err)
	}
	decision := service.TranscriptionAdmissionDecision(result)
	switch decision {
	case service.TranscriptionAdmissionAcquired, service.TranscriptionAdmissionDuplicate, service.TranscriptionAdmissionRate, service.TranscriptionAdmissionBusy:
		return decision, nil
	default:
		return "", fmt.Errorf("transcription admission returned invalid decision")
	}
}

func (c *gatewayCache) ReleaseTranscriptionAdmission(ctx context.Context, userID int64, leaseID string) error {
	if c == nil || c.rdb == nil || userID <= 0 || leaseID == "" {
		return fmt.Errorf("invalid transcription admission release")
	}
	keys := []string{
		transcriptionAdmissionPrefix + "concurrency:global",
		fmt.Sprintf("%sconcurrency:user:%d", transcriptionAdmissionPrefix, userID),
	}
	if err := transcriptionReleaseScript.Run(ctx, c.rdb, keys, leaseID).Err(); err != nil {
		return fmt.Errorf("transcription admission release: %w", err)
	}
	return nil
}

func (c *gatewayCache) ReserveTranscriptionDailyAudio(ctx context.Context, userID int64, seconds, limit int) (bool, error) {
	if c == nil || c.rdb == nil || userID <= 0 || seconds <= 0 || limit <= 0 {
		return false, fmt.Errorf("invalid transcription daily quota input")
	}
	serverTime, err := c.rdb.Time(ctx).Result()
	if err != nil {
		return false, fmt.Errorf("transcription daily quota redis time: %w", err)
	}
	utcNow := serverTime.UTC()
	nextDay := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day()+1, 0, 0, 0, 0, time.UTC)
	ttlSeconds := int(nextDay.Sub(utcNow).Seconds()) + 60
	key := fmt.Sprintf("%sdaily:%d:%s", transcriptionAdmissionPrefix, userID, utcNow.Format("20060102"))
	result, err := transcriptionDailyScript.Run(ctx, c.rdb, []string{key}, seconds, limit, ttlSeconds).Int64()
	if err != nil {
		return false, fmt.Errorf("transcription daily quota reserve: %w", err)
	}
	return result > 0, nil
}

var _ service.TranscriptionAdmissionCache = (*gatewayCache)(nil)

// Compile-time assertion: gatewayCache must implement CyberSessionBlockStore.
var _ service.CyberSessionBlockStore = (*gatewayCache)(nil)

const cyberSessionBlockPrefix = "cyber_session_block:"

// SetCyberSessionBlocked 把被 cyber_policy 命中的会话写入屏蔽表（TTL 自动过期）。
// 存储值 "1" 作为存在标记（IsCyberSessionBlocked 只检查 key 是否存在，不读值）。
func (c *gatewayCache) SetCyberSessionBlocked(ctx context.Context, key string, ttl time.Duration) error {
	return c.rdb.Set(ctx, cyberSessionBlockPrefix+key, "1", ttl).Err()
}

// IsCyberSessionBlocked 查询会话是否在屏蔽表中。
func (c *gatewayCache) IsCyberSessionBlocked(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Exists(ctx, cyberSessionBlockPrefix+key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
