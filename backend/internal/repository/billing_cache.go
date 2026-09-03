package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	billingBalanceKeyPrefix             = "billing:balance:"
	billingBalanceGenerationKeyPrefix   = "billing:balance:generation:"
	billingSubKeyPrefix                 = "billing:sub:"
	billingRateLimitKeyPrefix           = "apikey:rate:"
	billingRateLimitGenerationKeyPrefix = "apikey:rate:generation:"
	subCacheInvalidateChannel           = "subscription:cache:invalidate"
	billingCacheTTL                     = 5 * time.Minute
	billingCacheJitter                  = 30 * time.Second
	rateLimitCacheTTL                   = 7 * 24 * time.Hour // 7 days matches the longest window

	// Rate limit window durations — must match service.RateLimitWindow* constants.
	rateLimitWindow5h = 5 * time.Hour
	rateLimitWindow1d = 24 * time.Hour
	rateLimitWindow7d = 7 * 24 * time.Hour
)

// jitteredTTL 返回带随机抖动的 TTL，防止缓存雪崩
func jitteredTTL() time.Duration {
	// 只做“减法抖动”，确保实际 TTL 不会超过 billingCacheTTL（避免上界预期被打破）。
	if billingCacheJitter <= 0 {
		return billingCacheTTL
	}
	jitter := time.Duration(rand.IntN(int(billingCacheJitter)))
	return billingCacheTTL - jitter
}

// billingBalanceKey generates the Redis key for user balance cache.
func billingBalanceKey(userID int64) string {
	return fmt.Sprintf("%s%d", billingBalanceKeyPrefix, userID)
}

// billingBalanceTaggedKey is the canonical key used by generation-aware
// operations.  Keep billingBalanceKey above unchanged: it is the key format
// written by pre-generation binaries and is therefore part of the rolling
// upgrade contract.  The ID is the hash tag for both the snapshot and its
// generation token, so every Lua script below is safe on Redis Cluster.
func billingBalanceTaggedKey(userID int64) string {
	return fmt.Sprintf("%s{%d}", billingBalanceKeyPrefix, userID)
}

func billingBalanceGenerationKey(userID int64) string {
	return fmt.Sprintf("%s{%d}", billingBalanceGenerationKeyPrefix, userID)
}

// billingSubKey generates the Redis key for subscription cache.
func billingSubKey(userID, groupID int64) string {
	return fmt.Sprintf("%s%d:%d", billingSubKeyPrefix, userID, groupID)
}

const (
	subscriptionCacheSchemaV4  = int64(4)
	subFieldSchemaVersion      = "schema_version"
	subFieldSubscriptionID     = "subscription_id"
	subFieldStatus             = "status"
	subFieldStartsAt           = "starts_at"
	subFieldStartsAtExact      = "starts_at_exact"
	subFieldExpiresAt          = "expires_at"
	subFieldExpiresAtExact     = "expires_at_exact"
	subFieldWeeklyWindowStart  = "weekly_window_start"
	subFieldWeeklyStartExact   = "weekly_window_start_exact"
	subFieldWeeklyWindowEnd    = "weekly_window_end"
	subFieldWeeklyEndExact     = "weekly_window_end_exact"
	subFieldMonthlyWindowStart = "monthly_window_start"
	subFieldMonthlyStartExact  = "monthly_window_start_exact"
	subFieldMonthlyWindowEnd   = "monthly_window_end"
	subFieldMonthlyEndExact    = "monthly_window_end_exact"
	subFieldDailyUsage         = "daily_usage"
	subFieldWeeklyUsage        = "weekly_usage"
	subFieldMonthlyUsage       = "monthly_usage"
	subFieldVersion            = "version"
)

// billingRateLimitKey generates the Redis key for API key rate limit cache.
func billingRateLimitKey(keyID int64) string {
	return fmt.Sprintf("%s%d", billingRateLimitKeyPrefix, keyID)
}

// billingRateLimitTaggedKey is the canonical hash-tagged key used by
// generation-aware rate-limit operations.  billingRateLimitKey remains the
// legacy spelling for old binaries and is mirrored during the rolling window.
func billingRateLimitTaggedKey(keyID int64) string {
	return fmt.Sprintf("%s{%d}", billingRateLimitKeyPrefix, keyID)
}

func billingRateLimitGenerationKey(keyID int64) string {
	return fmt.Sprintf("%s{%d}", billingRateLimitGenerationKeyPrefix, keyID)
}

const (
	rateLimitFieldUsage5h  = "usage_5h"
	rateLimitFieldUsage1d  = "usage_1d"
	rateLimitFieldUsage7d  = "usage_7d"
	rateLimitFieldWindow5h = "window_5h"
	rateLimitFieldWindow1d = "window_1d"
	rateLimitFieldWindow7d = "window_7d"
)

var (
	// Read-through balance snapshots carry an invalidation generation. The
	// generation key is deliberately retained (rather than TTL'd) so an old
	// snapshot can never become writable again after an idle period.
	balanceWithGenerationScript = redis.NewScript(`
		local balance = redis.call('GET', KEYS[1])
		local generation = redis.call('GET', KEYS[2])
		if generation == false then
			generation = '0'
		end
		return {balance, generation}
	`)

	setBalanceIfGenerationScript = redis.NewScript(`
		local generation = redis.call('GET', KEYS[2])
		if generation == false then
			generation = '0'
		end
		if tostring(generation) ~= tostring(ARGV[2]) then
			return 0
		end
		redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[3])
		return 1
	`)

	invalidateBalanceScript = redis.NewScript(`
		redis.call('INCR', KEYS[2])
		redis.call('DEL', KEYS[1])
		return 1
	`)

	// Rate-limit snapshots are read through from the authoritative api_keys row.
	// Return the six hash fields and the invalidation generation in one Lua
	// invocation so a concurrent post-commit eviction cannot be missed between
	// the cache read and token capture. The first result item is 1 only when the
	// usage hash exists; an empty/missing hash is a cache miss even when the
	// generation key itself exists.
	apiKeyRateLimitWithGenerationScript = redis.NewScript(`
		local exists = redis.call('HEXISTS', KEYS[1], 'usage_5h')
		local generation = redis.call('GET', KEYS[2])
		if generation == false then
			generation = '0'
		end
		if exists == 0 then
			return {0, '', '', '', '', '', '', generation}
		end
		return {
			1,
			redis.call('HGET', KEYS[1], 'usage_5h') or '',
			redis.call('HGET', KEYS[1], 'usage_1d') or '',
			redis.call('HGET', KEYS[1], 'usage_7d') or '',
			redis.call('HGET', KEYS[1], 'window_5h') or '',
			redis.call('HGET', KEYS[1], 'window_1d') or '',
			redis.call('HGET', KEYS[1], 'window_7d') or '',
			generation
		}
	`)

	// A read-through refill is accepted only when the generation captured by
	// the reader still matches. This closes the miss -> DB read -> invalidation
	// -> stale SET race across processes, not just within one process.
	setAPIKeyRateLimitIfGenerationScript = redis.NewScript(`
		local generation = redis.call('GET', KEYS[2])
		if generation == false then
			generation = '0'
		end
		if tostring(generation) ~= tostring(ARGV[7]) then
			return 0
		end
		redis.call('HSET', KEYS[1],
			'usage_5h', ARGV[1],
			'usage_1d', ARGV[2],
			'usage_7d', ARGV[3],
			'window_5h', ARGV[4],
			'window_1d', ARGV[5],
			'window_7d', ARGV[6])
		redis.call('EXPIRE', KEYS[1], ARGV[8])
		return 1
	`)

	// Keep the generation key alive independently of the snapshot TTL. A
	// generation reset to zero after an idle period would otherwise allow an
	// old in-flight DB read to write stale usage back into Redis.
	invalidateAPIKeyRateLimitScript = redis.NewScript(`
		redis.call('INCR', KEYS[2])
		redis.call('DEL', KEYS[1])
		return 1
	`)

	deductBalanceScript = redis.NewScript(`
		local current = redis.call('GET', KEYS[1])
		-- Advance the durable generation before changing the snapshot. If a
		-- later cache command fails, readers still reject DB reloads that began
		-- before this deduction.
		redis.call('INCR', KEYS[2])
		if current == false then
			-- Even a miss needs the bump above: another process may already be in
			-- the miss -> DB reload window with a pre-deduction snapshot.
			return 0
		end
		local newVal = tonumber(current) - tonumber(ARGV[1])
		redis.call('SET', KEYS[1], newVal)
		redis.call('EXPIRE', KEYS[1], ARGV[2])
		return 1
	`)

	// The legacy snapshot is mirrored for rolling upgrades.  Keep this helper
	// to one key: unlike the generation-aware script it must remain usable when
	// the legacy key hashes to a different Redis Cluster slot.
	legacyDeductBalanceScript = redis.NewScript(`
		local current = redis.call('GET', KEYS[1])
		if current == false then
			return 0
		end
		local newVal = tonumber(current) - tonumber(ARGV[1])
		redis.call('SET', KEYS[1], newVal, 'EX', ARGV[2])
		return 1
	`)

	// updateRateLimitUsageScript atomically increments all three rate limit usage counters
	// with window expiration checking. If a window has expired, its usage is reset to cost
	// (instead of accumulated) and the window timestamp is updated, matching the DB-side
	// IncrementRateLimitUsage semantics.
	//
	// ARGV: [1]=cost, [2]=ttl_seconds, [3]=now_unix, [4]=window_5h_seconds, [5]=window_1d_seconds, [6]=window_7d_seconds
	updateRateLimitUsageScript = redis.NewScript(`
		local exists = redis.call('EXISTS', KEYS[1])
		if exists == 0 then
			return 0
		end
		local cost = tonumber(ARGV[1])
		local now = tonumber(ARGV[3])
		local win5h = tonumber(ARGV[4])
		local win1d = tonumber(ARGV[5])
		local win7d = tonumber(ARGV[6])

		-- Helper: check if window is expired and update usage + window accordingly
		-- Returns nothing, modifies the hash in-place.
		local function update_window(usage_field, window_field, window_duration)
			local w = tonumber(redis.call('HGET', KEYS[1], window_field) or 0)
			if w == 0 or (now - w) >= window_duration then
				-- Window expired or never started: reset usage to cost, start new window
				redis.call('HSET', KEYS[1], usage_field, tostring(cost))
				redis.call('HSET', KEYS[1], window_field, tostring(now))
			else
				-- Window still valid: accumulate
				redis.call('HINCRBYFLOAT', KEYS[1], usage_field, cost)
			end
		end

		update_window('usage_5h', 'window_5h', win5h)
		update_window('usage_1d', 'window_1d', win1d)
		update_window('usage_7d', 'window_7d', win7d)
		redis.call('EXPIRE', KEYS[1], ARGV[2])
		return 1
	`)
)

type billingCache struct {
	rdb *redis.Client
}

func NewBillingCache(rdb *redis.Client) service.BillingCache {
	return &billingCache{rdb: rdb}
}

func (c *billingCache) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	// Route all reads through the generation-qualified implementation so direct
	// callers receive the same cross-process stale-hit protection as the billing
	// service.  (The legacy fallback remains inside that method.)
	balance, _, err := c.GetUserBalanceWithGeneration(ctx, userID)
	return balance, err
}

// GetUserBalanceWithGeneration reads the balance and its invalidation token in
// one Redis script. On a miss it still returns the token captured at that
// instant, allowing the caller to reject a stale DB read later.
func (c *billingCache) GetUserBalanceWithGeneration(ctx context.Context, userID int64) (float64, uint64, error) {
	result, err := balanceWithGenerationScript.Run(
		ctx,
		c.rdb,
		[]string{billingBalanceTaggedKey(userID), billingBalanceGenerationKey(userID)},
	).Result()
	if err != nil {
		return 0, 0, err
	}
	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return 0, 0, fmt.Errorf("invalid versioned balance response: %T", result)
	}
	generation, err := parseRedisUint64(values[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse balance generation: %w", err)
	}
	if values[0] == nil {
		// Legacy data is trusted only before this generation namespace has ever
		// been invalidated. Once generation > 0, an old process may have
		// resurrected the untagged key after a new process evicted the canonical
		// key; accepting it would reintroduce the stale-read race we are fencing.
		if generation == 0 {
			legacy, legacyErr := c.rdb.Get(ctx, billingBalanceKey(userID)).Result()
			if legacyErr == nil {
				balance, parseErr := strconv.ParseFloat(legacy, 64)
				if parseErr != nil {
					return 0, generation, parseErr
				}
				// The legacy fallback necessarily takes a second round trip because
				// the untagged key may be in another Cluster slot.  Re-check the
				// canonical token before accepting it so an eviction between the
				// script and this GET cannot turn into a false hit.
				latestGeneration, generationErr := c.GetUserBalanceGeneration(ctx, userID)
				if generationErr != nil {
					return 0, generation, generationErr
				}
				if latestGeneration != generation {
					return 0, latestGeneration, service.ErrBillingCacheGenerationChanged
				}
				return balance, generation, nil
			}
			if !errors.Is(legacyErr, redis.Nil) {
				return 0, generation, legacyErr
			}
		}
		return 0, generation, service.ErrBillingCacheMiss
	}
	balanceText, ok := redisValueString(values[0])
	if !ok {
		return 0, generation, fmt.Errorf("invalid cached balance type: %T", values[0])
	}
	balance, err := strconv.ParseFloat(balanceText, 64)
	if err != nil {
		return 0, generation, err
	}
	return balance, generation, nil
}

// GetUserBalanceGeneration performs a cheap second-token read used to close
// the small window between a generation-qualified cache hit and the service
// returning it.  A missing token is the initial generation zero.
func (c *billingCache) GetUserBalanceGeneration(ctx context.Context, userID int64) (uint64, error) {
	value, err := c.rdb.Get(ctx, billingBalanceGenerationKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(value, 10, 64)
}

func (c *billingCache) SetUserBalance(ctx context.Context, userID int64, balance float64) error {
	ttl := jitteredTTL()
	// Mirror the value into both namespaces.  The tagged key is authoritative
	// for generation-aware readers; the untagged key keeps older binaries
	// functional until the rolling upgrade is complete.  These are deliberately
	// separate commands because the two legacy/canonical keys need not share a
	// Redis Cluster slot.
	return errors.Join(
		c.rdb.Set(ctx, billingBalanceTaggedKey(userID), balance, ttl).Err(),
		c.rdb.Set(ctx, billingBalanceKey(userID), balance, ttl).Err(),
	)
}

// SetUserBalanceIfGeneration conditionally writes a read-through value. A
// concurrent invalidation increments the generation and makes this a no-op;
// surface that outcome so the caller cannot return the stale DB snapshot.
func (c *billingCache) SetUserBalanceIfGeneration(ctx context.Context, userID int64, balance float64, generation uint64) error {
	ttl := jitteredTTL()
	result, err := setBalanceIfGenerationScript.Run(
		ctx,
		c.rdb,
		[]string{billingBalanceTaggedKey(userID), billingBalanceGenerationKey(userID)},
		strconv.FormatFloat(balance, 'f', -1, 64),
		strconv.FormatUint(generation, 10),
		int(ttl.Seconds()),
	).Result()
	if err != nil {
		return err
	}
	accepted, err := redisResultInt64(result)
	if err != nil {
		return fmt.Errorf("parse balance generation fence result: %w", err)
	}
	if accepted == 0 {
		return service.ErrBillingCacheGenerationChanged
	}
	// Best-effort rolling-upgrade mirror.  The tagged write above is the
	// generation-fenced source of truth; an error mirroring the legacy key is
	// surfaced so callers can observe cache degradation, but never invalidates
	// the accepted canonical snapshot.
	return c.rdb.Set(ctx, billingBalanceKey(userID), balance, ttl).Err()
}

func (c *billingCache) DeductUserBalance(ctx context.Context, userID int64, amount float64) error {
	ttl := jitteredTTL()
	_, canonicalErr := deductBalanceScript.Run(
		ctx,
		c.rdb,
		[]string{billingBalanceTaggedKey(userID), billingBalanceGenerationKey(userID)},
		amount,
		int(ttl.Seconds()),
	).Result()
	if canonicalErr != nil && !errors.Is(canonicalErr, redis.Nil) {
		log.Printf("Warning: deduct canonical balance cache failed for user %d: %v", userID, canonicalErr)
	}

	// Keep the pre-generation key in sync for older processes.  This is a
	// one-key script so it remains Cluster-safe even though the legacy key is in
	// a different slot from the canonical pair.  A missing legacy key is a
	// normal no-op (for example after a fresh canonical-only warmup).
	_, legacyErr := legacyDeductBalanceScript.Run(
		ctx,
		c.rdb,
		[]string{billingBalanceKey(userID)},
		amount,
		int(ttl.Seconds()),
	).Result()
	if legacyErr != nil && !errors.Is(legacyErr, redis.Nil) {
		log.Printf("Warning: deduct legacy balance cache failed for user %d: %v", userID, legacyErr)
	}
	if canonicalErr != nil && !errors.Is(canonicalErr, redis.Nil) {
		return canonicalErr
	}
	if legacyErr != nil && !errors.Is(legacyErr, redis.Nil) {
		return legacyErr
	}
	return nil
}

func (c *billingCache) InvalidateUserBalance(ctx context.Context, userID int64) error {
	_, canonicalErr := invalidateBalanceScript.Run(
		ctx,
		c.rdb,
		[]string{billingBalanceTaggedKey(userID), billingBalanceGenerationKey(userID)},
	).Result()
	legacyErr := c.rdb.Del(ctx, billingBalanceKey(userID)).Err()
	return errors.Join(canonicalErr, legacyErr)
}

func redisValueString(value interface{}) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case []byte:
		return string(typed), true
	default:
		return "", false
	}
}

func parseRedisUint64(value interface{}) (uint64, error) {
	text, ok := redisValueString(value)
	if !ok {
		return 0, fmt.Errorf("invalid generation type: %T", value)
	}
	return strconv.ParseUint(text, 10, 64)
}

func redisResultInt64(value interface{}) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case uint64:
		if typed > math.MaxInt64 {
			return 0, fmt.Errorf("integer overflow: %d", typed)
		}
		return int64(typed), nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	default:
		return 0, fmt.Errorf("invalid integer type: %T", value)
	}
}

func redisResultInt64AllowEmpty(value interface{}) (int64, error) {
	text, ok := redisValueString(value)
	if ok && text == "" {
		return 0, nil
	}
	return redisResultInt64(value)
}

func redisResultFloat64(value interface{}) (float64, error) {
	text, ok := redisValueString(value)
	if ok && text == "" {
		return 0, nil
	}
	if !ok {
		return 0, fmt.Errorf("invalid float type: %T", value)
	}
	return strconv.ParseFloat(text, 64)
}

func (c *billingCache) GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*service.SubscriptionCacheData, error) {
	key := billingSubKey(userID, groupID)
	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, service.ErrBillingCacheMiss
	}
	return c.parseSubscriptionCache(result)
}

func (c *billingCache) parseSubscriptionCache(data map[string]string) (*service.SubscriptionCacheData, error) {
	result := &service.SubscriptionCacheData{}

	schemaVersion, err := strconv.ParseInt(data[subFieldSchemaVersion], 10, 64)
	if err != nil || schemaVersion != subscriptionCacheSchemaV4 {
		return nil, errors.New("invalid cache: unsupported subscription schema")
	}

	subscriptionID, err := strconv.ParseInt(data[subFieldSubscriptionID], 10, 64)
	if err != nil || subscriptionID <= 0 {
		return nil, errors.New("invalid cache: invalid subscription_id")
	}
	result.SubscriptionID = subscriptionID

	result.Status = data[subFieldStatus]
	if result.Status == "" {
		return nil, errors.New("invalid cache: missing status")
	}

	startsAt, err := time.Parse(time.RFC3339Nano, data[subFieldStartsAtExact])
	if err != nil || startsAt.IsZero() {
		return nil, errors.New("invalid cache: invalid starts_at")
	}
	result.StartsAt = startsAt
	if err := validateUnixTimeField(data, subFieldStartsAt, startsAt); err != nil {
		return nil, err
	}

	expiresAt, err := time.Parse(time.RFC3339Nano, data[subFieldExpiresAtExact])
	if err != nil || expiresAt.IsZero() || !expiresAt.After(startsAt) {
		return nil, errors.New("invalid cache: invalid expires_at")
	}
	result.ExpiresAt = expiresAt
	if err := validateUnixTimeField(data, subFieldExpiresAt, expiresAt); err != nil {
		return nil, err
	}

	windowStart, err := time.Parse(time.RFC3339Nano, data[subFieldWeeklyStartExact])
	if err != nil || windowStart.IsZero() {
		return nil, errors.New("invalid cache: invalid weekly_window_start")
	}
	result.WeeklyWindowStart = &windowStart
	if err := validateUnixTimeField(data, subFieldWeeklyWindowStart, windowStart); err != nil {
		return nil, err
	}

	windowEnd, err := time.Parse(time.RFC3339Nano, data[subFieldWeeklyEndExact])
	if err != nil || !windowEnd.Equal(windowStart.Add(service.SubscriptionWeeklyWindowDuration)) {
		return nil, errors.New("invalid cache: invalid weekly_window_end")
	}
	result.WeeklyWindowEnd = windowEnd
	if err := validateUnixTimeField(data, subFieldWeeklyWindowEnd, windowEnd); err != nil {
		return nil, err
	}
	expectedStart, expectedEnd, ok := service.AnchoredWeeklyWindow(startsAt, windowStart)
	if !ok || !windowStart.Equal(expectedStart) || !windowEnd.Equal(expectedEnd) {
		return nil, errors.New("invalid cache: weekly window is not anchored")
	}

	monthlyWindowStart, err := time.Parse(time.RFC3339Nano, data[subFieldMonthlyStartExact])
	if err != nil || monthlyWindowStart.IsZero() {
		return nil, errors.New("invalid cache: invalid monthly_window_start")
	}
	result.MonthlyWindowStart = &monthlyWindowStart
	if err := validateUnixTimeField(data, subFieldMonthlyWindowStart, monthlyWindowStart); err != nil {
		return nil, err
	}

	monthlyWindowEnd, err := time.Parse(time.RFC3339Nano, data[subFieldMonthlyEndExact])
	if err != nil || !monthlyWindowEnd.Equal(monthlyWindowStart.Add(service.SubscriptionMonthlyWindowDuration)) {
		return nil, errors.New("invalid cache: invalid monthly_window_end")
	}
	result.MonthlyWindowEnd = monthlyWindowEnd
	if err := validateUnixTimeField(data, subFieldMonthlyWindowEnd, monthlyWindowEnd); err != nil {
		return nil, err
	}
	expectedMonthlyStart, expectedMonthlyEnd, ok := service.AnchoredMonthlyWindow(startsAt, monthlyWindowStart)
	if !ok ||
		!monthlyWindowStart.Equal(expectedMonthlyStart) ||
		!monthlyWindowEnd.Equal(expectedMonthlyEnd) {
		return nil, errors.New("invalid cache: monthly window is not anchored")
	}

	if dailyStr, ok := data[subFieldDailyUsage]; ok {
		result.DailyUsage, err = parseNonnegativeFiniteFloat(dailyStr)
		if err != nil {
			return nil, errors.New("invalid cache: invalid daily_usage")
		}
	}

	weeklyStr, ok := data[subFieldWeeklyUsage]
	if !ok {
		return nil, errors.New("invalid cache: missing weekly_usage")
	}
	result.WeeklyUsage, err = parseNonnegativeFiniteFloat(weeklyStr)
	if err != nil {
		return nil, errors.New("invalid cache: invalid weekly_usage")
	}

	monthlyStr, ok := data[subFieldMonthlyUsage]
	if !ok {
		return nil, errors.New("invalid cache: missing monthly_usage")
	}
	result.MonthlyUsage, err = parseNonnegativeFiniteFloat(monthlyStr)
	if err != nil {
		return nil, errors.New("invalid cache: invalid monthly_usage")
	}

	versionStr, ok := data[subFieldVersion]
	if !ok {
		return nil, errors.New("invalid cache: missing version")
	}
	result.Version, err = strconv.ParseInt(versionStr, 10, 64)
	if err != nil || result.Version <= 0 {
		return nil, errors.New("invalid cache: invalid version")
	}

	return result, nil
}

func validateUnixTimeField(data map[string]string, field string, exact time.Time) error {
	value, ok := data[field]
	if !ok {
		return fmt.Errorf("invalid cache: missing %s", field)
	}
	unix, err := strconv.ParseInt(value, 10, 64)
	if err != nil || unix != exact.Unix() {
		return fmt.Errorf("invalid cache: invalid %s", field)
	}
	return nil
}

func parseNonnegativeFiniteFloat(value string) (float64, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < 0 {
		return 0, errors.New("invalid nonnegative finite float")
	}
	return parsed, nil
}

func (c *billingCache) SetSubscriptionCache(ctx context.Context, userID, groupID int64, data *service.SubscriptionCacheData) error {
	if err := validateSubscriptionCacheData(data); err != nil {
		return err
	}

	key := billingSubKey(userID, groupID)

	fields := map[string]any{
		subFieldSchemaVersion:      subscriptionCacheSchemaV4,
		subFieldSubscriptionID:     data.SubscriptionID,
		subFieldStatus:             data.Status,
		subFieldStartsAt:           data.StartsAt.Unix(),
		subFieldStartsAtExact:      data.StartsAt.Format(time.RFC3339Nano),
		subFieldExpiresAt:          data.ExpiresAt.Unix(),
		subFieldExpiresAtExact:     data.ExpiresAt.Format(time.RFC3339Nano),
		subFieldWeeklyWindowStart:  data.WeeklyWindowStart.Unix(),
		subFieldWeeklyStartExact:   data.WeeklyWindowStart.Format(time.RFC3339Nano),
		subFieldWeeklyWindowEnd:    data.WeeklyWindowEnd.Unix(),
		subFieldWeeklyEndExact:     data.WeeklyWindowEnd.Format(time.RFC3339Nano),
		subFieldMonthlyWindowStart: data.MonthlyWindowStart.Unix(),
		subFieldMonthlyStartExact:  data.MonthlyWindowStart.Format(time.RFC3339Nano),
		subFieldMonthlyWindowEnd:   data.MonthlyWindowEnd.Unix(),
		subFieldMonthlyEndExact:    data.MonthlyWindowEnd.Format(time.RFC3339Nano),
		subFieldDailyUsage:         0,
		subFieldWeeklyUsage:        data.WeeklyUsage,
		subFieldMonthlyUsage:       data.MonthlyUsage,
		subFieldVersion:            data.Version,
	}
	ttl := subscriptionCacheTTL(data, time.Now())
	if ttl <= 0 {
		return c.rdb.Del(ctx, key).Err()
	}

	// MULTI/EXEC keeps the hash contents and boundary-capped TTL inseparable:
	// a transport failure must not leave a valid-looking key without expiry.
	pipe := c.rdb.TxPipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func validateSubscriptionCacheData(data *service.SubscriptionCacheData) error {
	if data == nil || data.SubscriptionID <= 0 {
		return errors.New("invalid subscription cache data: missing subscription_id")
	}
	if data.Status == "" || data.StartsAt.IsZero() || !data.ExpiresAt.After(data.StartsAt) {
		return errors.New("invalid subscription cache data: invalid entitlement")
	}
	if data.WeeklyWindowStart == nil ||
		!data.WeeklyWindowEnd.Equal(data.WeeklyWindowStart.Add(service.SubscriptionWeeklyWindowDuration)) {
		return errors.New("invalid subscription cache data: invalid weekly window")
	}
	expectedStart, expectedEnd, ok := service.AnchoredWeeklyWindow(data.StartsAt, *data.WeeklyWindowStart)
	if !ok || !data.WeeklyWindowStart.Equal(expectedStart) || !data.WeeklyWindowEnd.Equal(expectedEnd) {
		return errors.New("invalid subscription cache data: weekly window is not anchored")
	}
	if data.MonthlyWindowStart == nil ||
		!data.MonthlyWindowEnd.Equal(data.MonthlyWindowStart.Add(service.SubscriptionMonthlyWindowDuration)) {
		return errors.New("invalid subscription cache data: invalid monthly window")
	}
	expectedMonthlyStart, expectedMonthlyEnd, ok := service.AnchoredMonthlyWindow(data.StartsAt, *data.MonthlyWindowStart)
	if !ok ||
		!data.MonthlyWindowStart.Equal(expectedMonthlyStart) ||
		!data.MonthlyWindowEnd.Equal(expectedMonthlyEnd) {
		return errors.New("invalid subscription cache data: monthly window is not anchored")
	}
	if _, err := parseNonnegativeFiniteFloat(strconv.FormatFloat(data.WeeklyUsage, 'g', -1, 64)); err != nil {
		return errors.New("invalid subscription cache data: invalid weekly usage")
	}
	if _, err := parseNonnegativeFiniteFloat(strconv.FormatFloat(data.MonthlyUsage, 'g', -1, 64)); err != nil {
		return errors.New("invalid subscription cache data: invalid monthly usage")
	}
	if data.Version <= 0 {
		return errors.New("invalid subscription cache data: invalid version")
	}
	return nil
}

func (c *billingCache) UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64) error {
	// The DB increment is authoritative. Applying a detached HINCR here can
	// double-count when another request repopulates Redis from the committed DB
	// before this call arrives, so every legacy "update" is an eviction.
	_ = cost
	if err := c.InvalidateSubscriptionCache(ctx, userID, groupID); err != nil {
		log.Printf("Warning: invalidate subscription usage cache failed for user %d group %d: %v", userID, groupID, err)
		return err
	}
	return nil
}

func subscriptionCacheTTL(data *service.SubscriptionCacheData, now time.Time) time.Duration {
	if data == nil {
		return 0
	}
	ttl := jitteredTTL()
	if untilReset := data.WeeklyWindowEnd.Sub(now); untilReset < ttl {
		ttl = untilReset
	}
	if untilReset := data.MonthlyWindowEnd.Sub(now); untilReset < ttl {
		ttl = untilReset
	}
	if untilExpiry := data.ExpiresAt.Sub(now); untilExpiry < ttl {
		ttl = untilExpiry
	}
	if ttl <= 0 {
		return 0
	}
	// Redis expiration is second-granular here. Truncation ensures a cache key
	// cannot survive beyond an entitlement boundary.
	return time.Duration(int64(ttl.Seconds())) * time.Second
}

func (c *billingCache) InvalidateSubscriptionCache(ctx context.Context, userID, groupID int64) error {
	key := billingSubKey(userID, groupID)
	return c.rdb.Del(ctx, key).Err()
}

func (c *billingCache) PublishSubscriptionCacheInvalidation(ctx context.Context, cacheKey string) error {
	return c.rdb.Publish(ctx, subCacheInvalidateChannel, cacheKey).Err()
}

func (c *billingCache) SubscribeSubscriptionCacheInvalidation(ctx context.Context, handler func(cacheKey string)) error {
	pubsub := c.rdb.Subscribe(ctx, subCacheInvalidateChannel)

	_, err := pubsub.Receive(ctx)
	if err != nil {
		_ = pubsub.Close()
		return fmt.Errorf("subscribe to subscription cache invalidation: %w", err)
	}

	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Printf("Warning: failed to close subscription cache invalidation pubsub: %v", err)
		}
	}()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if msg != nil {
				handler(msg.Payload)
			}
		}
	}
}

func (c *billingCache) GetAPIKeyRateLimit(ctx context.Context, keyID int64) (*service.APIKeyRateLimitCacheData, error) {
	data, _, err := c.GetAPIKeyRateLimitWithGeneration(ctx, keyID)
	if errors.Is(err, service.ErrBillingCacheMiss) {
		// Preserve the native miss sentinel expected by legacy callers of this
		// method while the versioned extension uses service.ErrBillingCacheMiss.
		return nil, redis.Nil
	}
	return data, err
}

// parseAPIKeyRateLimitHash decodes a legacy/canonical HGETALL result.  The
// legacy reader historically tolerated malformed individual fields by treating
// them as zero; retain that behavior for rolling compatibility.
func parseAPIKeyRateLimitHash(result map[string]string) (*service.APIKeyRateLimitCacheData, uint64, error) {
	data := &service.APIKeyRateLimitCacheData{}
	if v, ok := result[rateLimitFieldUsage5h]; ok {
		data.Usage5h, _ = strconv.ParseFloat(v, 64)
	}
	if v, ok := result[rateLimitFieldUsage1d]; ok {
		data.Usage1d, _ = strconv.ParseFloat(v, 64)
	}
	if v, ok := result[rateLimitFieldUsage7d]; ok {
		data.Usage7d, _ = strconv.ParseFloat(v, 64)
	}
	if v, ok := result[rateLimitFieldWindow5h]; ok {
		data.Window5h, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := result[rateLimitFieldWindow1d]; ok {
		data.Window1d, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := result[rateLimitFieldWindow7d]; ok {
		data.Window7d, _ = strconv.ParseInt(v, 10, 64)
	}
	return data, 0, nil
}

func parseAPIKeyRateLimitFields(values []interface{}) (*service.APIKeyRateLimitCacheData, error) {
	if len(values) != 6 {
		return nil, fmt.Errorf("invalid api key rate-limit field count: %d", len(values))
	}
	data := &service.APIKeyRateLimitCacheData{}
	var err error
	data.Usage5h, err = redisResultFloat64(values[0])
	if err != nil {
		return nil, fmt.Errorf("parse usage_5h: %w", err)
	}
	data.Usage1d, err = redisResultFloat64(values[1])
	if err != nil {
		return nil, fmt.Errorf("parse usage_1d: %w", err)
	}
	data.Usage7d, err = redisResultFloat64(values[2])
	if err != nil {
		return nil, fmt.Errorf("parse usage_7d: %w", err)
	}
	data.Window5h, err = redisResultInt64AllowEmpty(values[3])
	if err != nil {
		return nil, fmt.Errorf("parse window_5h: %w", err)
	}
	data.Window1d, err = redisResultInt64AllowEmpty(values[4])
	if err != nil {
		return nil, fmt.Errorf("parse window_1d: %w", err)
	}
	data.Window7d, err = redisResultInt64AllowEmpty(values[5])
	if err != nil {
		return nil, fmt.Errorf("parse window_7d: %w", err)
	}
	return data, nil
}

// GetAPIKeyRateLimitWithGeneration reads a rate-limit snapshot and its
// invalidation generation atomically. The generation is returned even on a
// cache miss so callers can conditionally refill after loading the DB.
func (c *billingCache) GetAPIKeyRateLimitWithGeneration(ctx context.Context, keyID int64) (*service.APIKeyRateLimitCacheData, uint64, error) {
	result, err := apiKeyRateLimitWithGenerationScript.Run(
		ctx,
		c.rdb,
		[]string{billingRateLimitTaggedKey(keyID), billingRateLimitGenerationKey(keyID)},
	).Result()
	if err != nil {
		return nil, 0, err
	}
	values, ok := result.([]interface{})
	if !ok || len(values) != 8 {
		return nil, 0, fmt.Errorf("invalid versioned api key rate-limit response: %T", result)
	}

	hit, err := redisResultInt64(values[0])
	if err != nil {
		return nil, 0, fmt.Errorf("parse api key rate-limit cache presence: %w", err)
	}
	generation, err := parseRedisUint64(values[7])
	if err != nil {
		return nil, 0, fmt.Errorf("parse api key rate-limit generation: %w", err)
	}
	if hit == 0 {
		// As with balances, only consult the untagged snapshot while the new
		// generation namespace is still at zero.  Once an invalidation has
		// advanced it, an old process may have recreated the legacy hash with a
		// stale value and must not bypass the fence.
		if generation == 0 {
			legacy, legacyErr := c.rdb.HGetAll(ctx, billingRateLimitKey(keyID)).Result()
			if legacyErr != nil {
				return nil, generation, legacyErr
			}
			if len(legacy) > 0 {
				data, _, parseErr := parseAPIKeyRateLimitHash(legacy)
				if parseErr != nil {
					return nil, generation, parseErr
				}
				latestGeneration, generationErr := c.GetAPIKeyRateLimitGeneration(ctx, keyID)
				if generationErr != nil {
					return nil, generation, generationErr
				}
				if latestGeneration != generation {
					return nil, latestGeneration, service.ErrBillingCacheGenerationChanged
				}
				return data, generation, nil
			}
		}
		return nil, generation, service.ErrBillingCacheMiss
	}

	data, parseErr := parseAPIKeyRateLimitFields(values[1:7])
	if parseErr != nil {
		return nil, generation, parseErr
	}
	return data, generation, nil
}

// GetAPIKeyRateLimitGeneration is the cheap companion token read used to
// verify a generation-qualified cache hit immediately before evaluation.
func (c *billingCache) GetAPIKeyRateLimitGeneration(ctx context.Context, keyID int64) (uint64, error) {
	value, err := c.rdb.Get(ctx, billingRateLimitGenerationKey(keyID)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(value, 10, 64)
}

func (c *billingCache) SetAPIKeyRateLimit(ctx context.Context, keyID int64, data *service.APIKeyRateLimitCacheData) error {
	if data == nil {
		return nil
	}
	fields := map[string]any{
		rateLimitFieldUsage5h:  data.Usage5h,
		rateLimitFieldUsage1d:  data.Usage1d,
		rateLimitFieldUsage7d:  data.Usage7d,
		rateLimitFieldWindow5h: data.Window5h,
		rateLimitFieldWindow1d: data.Window1d,
		rateLimitFieldWindow7d: data.Window7d,
	}
	set := func(key string) error {
		pipe := c.rdb.Pipeline()
		pipe.HSet(ctx, key, fields)
		pipe.Expire(ctx, key, rateLimitCacheTTL)
		_, err := pipe.Exec(ctx)
		return err
	}
	// Keep both namespaces populated during rolling upgrades.  The tagged hash
	// is authoritative for generation-aware readers; the legacy hash is for
	// older binaries and is intentionally written in a separate pipeline so a
	// Cluster never receives cross-slot KEYS in one script/transaction.
	return errors.Join(
		set(billingRateLimitTaggedKey(keyID)),
		set(billingRateLimitKey(keyID)),
	)
}

// SetAPIKeyRateLimitIfGeneration conditionally refills a read-through rate
// limit snapshot. A return value of zero from the Lua script means an
// invalidation advanced the generation while the DB read was in flight; surface
// that result so the service retries instead of evaluating the stale snapshot.
func (c *billingCache) SetAPIKeyRateLimitIfGeneration(ctx context.Context, keyID int64, data *service.APIKeyRateLimitCacheData, generation uint64) error {
	if data == nil {
		return nil
	}
	result, err := setAPIKeyRateLimitIfGenerationScript.Run(
		ctx,
		c.rdb,
		[]string{billingRateLimitTaggedKey(keyID), billingRateLimitGenerationKey(keyID)},
		strconv.FormatFloat(data.Usage5h, 'f', -1, 64),
		strconv.FormatFloat(data.Usage1d, 'f', -1, 64),
		strconv.FormatFloat(data.Usage7d, 'f', -1, 64),
		strconv.FormatInt(data.Window5h, 10),
		strconv.FormatInt(data.Window1d, 10),
		strconv.FormatInt(data.Window7d, 10),
		strconv.FormatUint(generation, 10),
		int(rateLimitCacheTTL.Seconds()),
	).Result()
	if err != nil {
		return err
	}
	accepted, err := redisResultInt64(result)
	if err != nil {
		return fmt.Errorf("parse api key rate-limit generation fence result: %w", err)
	}
	if accepted == 0 {
		return service.ErrBillingCacheGenerationChanged
	}
	// Mirror after the fenced canonical write for old binaries.  A mirror error
	// is observable but cannot invalidate the accepted canonical snapshot.
	fields := map[string]any{
		rateLimitFieldUsage5h:  data.Usage5h,
		rateLimitFieldUsage1d:  data.Usage1d,
		rateLimitFieldUsage7d:  data.Usage7d,
		rateLimitFieldWindow5h: data.Window5h,
		rateLimitFieldWindow1d: data.Window1d,
		rateLimitFieldWindow7d: data.Window7d,
	}
	pipe := c.rdb.Pipeline()
	pipe.HSet(ctx, billingRateLimitKey(keyID), fields)
	pipe.Expire(ctx, billingRateLimitKey(keyID), rateLimitCacheTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *billingCache) UpdateAPIKeyRateLimitUsage(ctx context.Context, keyID int64, cost float64) error {
	// The api_keys row is the source of truth. A relative Redis increment is
	// unsafe because a concurrent miss can refill a pre-charge snapshot before
	// this method runs, causing the same DB charge to be counted twice. Keep the
	// old method for interface compatibility but make it an eviction.
	_ = cost
	if err := c.InvalidateAPIKeyRateLimit(ctx, keyID); err != nil {
		log.Printf("Warning: invalidate rate limit usage cache failed for api key %d: %v", keyID, err)
		return err
	}
	return nil
}

func (c *billingCache) InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error {
	_, canonicalErr := invalidateAPIKeyRateLimitScript.Run(
		ctx,
		c.rdb,
		[]string{billingRateLimitTaggedKey(keyID), billingRateLimitGenerationKey(keyID)},
	).Result()
	legacyErr := c.rdb.Del(ctx, billingRateLimitKey(keyID)).Err()
	return errors.Join(canonicalErr, legacyErr)
}

// ============================================
// user × platform quota 缓存
// ============================================

// userPlatformQuotaCacheKey 构造 Redis key
func userPlatformQuotaCacheKey(userID int64, platform string) string {
	return fmt.Sprintf("billing:user_platform_quota:%d:%s", userID, platform)
}

// parseUserPlatformQuotaHash 将 Redis HGETALL 返回的 map[string]string 反序列化为
// *service.UserPlatformQuotaCacheEntry。空 map（key 不存在）返回 nil。
// GetUserPlatformQuotaCache 和 BatchGetUserPlatformQuotaCache 共用此函数，确保解析逻辑一致。
func parseUserPlatformQuotaHash(m map[string]string) *service.UserPlatformQuotaCacheEntry {
	if len(m) == 0 {
		return nil
	}
	parseFloat := func(s string) float64 {
		if s == "" {
			return 0
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			log.Printf("billing_cache: corrupt quota usage field %q (using 0): %v", s, err)
			return 0
		}
		return f
	}
	parseFloatPtr := func(s string) *float64 {
		if s == "" {
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil
		}
		return &f
	}
	parseTimePtr := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil
		}
		t := time.Unix(n, 0).UTC()
		return &t
	}
	parseInt64 := func(s string) int64 {
		n, _ := strconv.ParseInt(s, 10, 64)
		return n
	}
	return &service.UserPlatformQuotaCacheEntry{
		DailyUsageUSD:      parseFloat(m["daily_usage"]),
		WeeklyUsageUSD:     parseFloat(m["weekly_usage"]),
		MonthlyUsageUSD:    parseFloat(m["monthly_usage"]),
		Version:            parseInt64(m["version"]),
		SchemaVersion:      parseInt64(m["schema_version"]),
		DailyLimitUSD:      parseFloatPtr(m["daily_limit"]),
		WeeklyLimitUSD:     parseFloatPtr(m["weekly_limit"]),
		MonthlyLimitUSD:    parseFloatPtr(m["monthly_limit"]),
		DailyWindowStart:   parseTimePtr(m["daily_window_start"]),
		WeeklyWindowStart:  parseTimePtr(m["weekly_window_start"]),
		MonthlyWindowStart: parseTimePtr(m["monthly_window_start"]),
	}
}

func (c *billingCache) GetUserPlatformQuotaCache(ctx context.Context, userID int64, platform string) (*service.UserPlatformQuotaCacheEntry, bool, error) {
	key := userPlatformQuotaCacheKey(userID, platform)
	m, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, false, err
	}
	entry := parseUserPlatformQuotaHash(m)
	if entry == nil {
		// 空 map → key 不存在 → MISS
		return nil, false, nil
	}
	return entry, true, nil
}

func (c *billingCache) SetUserPlatformQuotaCache(ctx context.Context, userID int64, platform string, entry *service.UserPlatformQuotaCacheEntry, ttl time.Duration) error {
	if entry == nil {
		return nil
	}
	key := userPlatformQuotaCacheKey(userID, platform)
	pipe := c.rdb.TxPipeline()

	// 浮点可空字段：nil → 空字符串（读取时 parseFloatPtr 返回 nil，表示无限额）
	fmtFloatPtr := func(p *float64) string {
		if p == nil {
			return ""
		}
		return strconv.FormatFloat(*p, 'f', -1, 64)
	}
	// time.Time 可空字段：nil → 空字符串；有值 → unix 秒
	fmtTimePtr := func(p *time.Time) string {
		if p == nil {
			return ""
		}
		return strconv.FormatInt(p.Unix(), 10)
	}

	pipe.HSet(ctx, key,
		"daily_usage", entry.DailyUsageUSD,
		"weekly_usage", entry.WeeklyUsageUSD,
		"monthly_usage", entry.MonthlyUsageUSD,
		"version", entry.Version,
		"schema_version", entry.SchemaVersion,
		"daily_limit", fmtFloatPtr(entry.DailyLimitUSD),
		"weekly_limit", fmtFloatPtr(entry.WeeklyLimitUSD),
		"monthly_limit", fmtFloatPtr(entry.MonthlyLimitUSD),
		"daily_window_start", fmtTimePtr(entry.DailyWindowStart),
		"weekly_window_start", fmtTimePtr(entry.WeeklyWindowStart),
		"monthly_window_start", fmtTimePtr(entry.MonthlyWindowStart),
	)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *billingCache) DeleteUserPlatformQuotaCache(ctx context.Context, userID int64, platform string) error {
	return c.rdb.Del(ctx, userPlatformQuotaCacheKey(userID, platform)).Err()
}

// updateUserPlatformQuotaUsageScript 缓存累加：EXISTS + schema_version 双重守卫。
// 旧版 entry（schema_version != ARGV[3]，包括缺字段的 0 值）不参与累加，由上层走 DB fallback 后
// SetCache 重建为新版 entry —— 若此处仍累加，上层覆盖时会丢失这部分增量，导致 Redis usage 比真实偏小。
// key 不存在同样跳过（由下次 SetCache 重建）。
// KEYS[1] = hash key
// KEYS[2] = 脏集 key（dirty set）
// ARGV[1] = cost (string float)
// ARGV[2] = ttl seconds
// ARGV[3] = expected schema_version (Go 侧 UserPlatformQuotaCacheSchemaV1)
// ARGV[4] = dirty set member（空串则不 SADD）
// ARGV[5] = 脏集兜底 TTL 秒
const updateUserPlatformQuotaUsageScript = `
if redis.call("EXISTS", KEYS[1]) == 0 then
    return 0
end
local ver = redis.call("HGET", KEYS[1], "schema_version")
if ver == false or tonumber(ver) ~= tonumber(ARGV[3]) then
    return 0
end
redis.call("HINCRBYFLOAT", KEYS[1], "daily_usage", ARGV[1])
redis.call("HINCRBYFLOAT", KEYS[1], "weekly_usage", ARGV[1])
redis.call("HINCRBYFLOAT", KEYS[1], "monthly_usage", ARGV[1])
redis.call("HINCRBY", KEYS[1], "version", 1)
redis.call("EXPIRE", KEYS[1], ARGV[2])
if ARGV[4] ~= "" then
    redis.call("SADD", KEYS[2], ARGV[4])
    redis.call("EXPIRE", KEYS[2], ARGV[5])
end
return 1
`

// userPlatformQuotaDirtySetKey 返回脏集（dirty set）的 Redis key。
// 使用与 userPlatformQuotaCacheKey 相同的前缀 "billing:"。
func userPlatformQuotaDirtySetKey() string { return "billing:" + "upq:dirty" }

// userPlatformQuotaDirtyTTLSeconds 脏集兜底 TTL（秒）：初始 SADD（Lua）与 Readd 共用，
// 确保 flusher 长期停摆时脏集最终过期；正常运行因持续 SADD 不断续期。
const userPlatformQuotaDirtyTTLSeconds = 86400

// userPlatformQuotaDirtyMember 构造脏集成员字符串 "userID:platform"。
func userPlatformQuotaDirtyMember(userID int64, platform string) string {
	return strconv.FormatInt(userID, 10) + ":" + platform
}

func (c *billingCache) IncrUserPlatformQuotaUsageCache(ctx context.Context, userID int64, platform string, cost float64, ttl time.Duration, markDirty bool) error {
	member := ""
	if markDirty {
		member = userPlatformQuotaDirtyMember(userID, platform)
	}
	_, err := c.rdb.Eval(ctx, updateUserPlatformQuotaUsageScript,
		[]string{userPlatformQuotaCacheKey(userID, platform), userPlatformQuotaDirtySetKey()},
		strconv.FormatFloat(cost, 'f', -1, 64),
		int(ttl.Seconds()),
		service.UserPlatformQuotaCacheSchemaV1,
		member,
		userPlatformQuotaDirtyTTLSeconds,
	).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	return nil
}

// parseUserPlatformQuotaDirtyMember 将脏集成员字符串 "userID:platform" 解析为
// service.UserPlatformQuotaKey。解析失败返回 ok=false。
func parseUserPlatformQuotaDirtyMember(m string) (service.UserPlatformQuotaKey, bool) {
	parts := strings.SplitN(m, ":", 2)
	if len(parts) != 2 {
		return service.UserPlatformQuotaKey{}, false
	}
	uid, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return service.UserPlatformQuotaKey{}, false
	}
	return service.UserPlatformQuotaKey{UserID: uid, Platform: parts[1]}, true
}

// PopDirtyUserPlatformQuotaKeys 从脏集随机弹出最多 n 个 key。
// 脏集为空时返回 (nil, nil)。
func (c *billingCache) PopDirtyUserPlatformQuotaKeys(ctx context.Context, n int) ([]service.UserPlatformQuotaKey, error) {
	members, err := c.rdb.SPopN(ctx, userPlatformQuotaDirtySetKey(), int64(n)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	keys := make([]service.UserPlatformQuotaKey, 0, len(members))
	for _, m := range members {
		k, ok := parseUserPlatformQuotaDirtyMember(m)
		if !ok {
			log.Printf("billing_cache: skipping invalid dirty member %q", m)
			continue
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// ReaddDirtyUserPlatformQuotaKeys 将 keys 重新加入脏集（flush 失败时回填）。
// 通过 pipeline 同时执行 SAdd + Expire，确保 Readd 后脏集具有兜底 TTL。
// 空切片时直接返回 nil。
func (c *billingCache) ReaddDirtyUserPlatformQuotaKeys(ctx context.Context, keys []service.UserPlatformQuotaKey) error {
	if len(keys) == 0 {
		return nil
	}
	dirtyKey := userPlatformQuotaDirtySetKey()
	members := make([]any, len(keys))
	for i, k := range keys {
		members[i] = userPlatformQuotaDirtyMember(k.UserID, k.Platform)
	}
	pipe := c.rdb.Pipeline()
	pipe.SAdd(ctx, dirtyKey, members...)
	pipe.Expire(ctx, dirtyKey, userPlatformQuotaDirtyTTLSeconds*time.Second)
	_, err := pipe.Exec(ctx)
	return err
}

// BatchGetUserPlatformQuotaCache 通过 Pipeline 批量 HGETALL 获取多个 user×platform 的
// quota cache。返回切片与 keys 顺序、长度对齐；MISS 或解析失败位置返回 nil。
func (c *billingCache) BatchGetUserPlatformQuotaCache(ctx context.Context, keys []service.UserPlatformQuotaKey) ([]*service.UserPlatformQuotaCacheEntry, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	pipe := c.rdb.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(keys))
	for i, k := range keys {
		cmds[i] = pipe.HGetAll(ctx, userPlatformQuotaCacheKey(k.UserID, k.Platform))
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	results := make([]*service.UserPlatformQuotaCacheEntry, len(keys))
	for i, cmd := range cmds {
		m, err := cmd.Result()
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				log.Printf("billing_cache: BatchGet HGETALL cmd[%d] failed: %v (skip, self-heal)", i, err)
			}
			// 单个命令失败 → 对应位置 nil，继续
			continue
		}
		results[i] = parseUserPlatformQuotaHash(m)
	}
	return results, nil
}
