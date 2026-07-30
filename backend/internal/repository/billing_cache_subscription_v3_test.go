package repository

import (
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func validSubscriptionCacheV3Fields() map[string]string {
	startsAt := time.Date(2026, time.July, 29, 13, 47, 12, 345678901, time.UTC)
	windowStart := startsAt.Add(service.SubscriptionWeeklyWindowDuration)
	windowEnd := windowStart.Add(service.SubscriptionWeeklyWindowDuration)
	expiresAt := startsAt.Add(30 * 24 * time.Hour)
	return map[string]string{
		subFieldSchemaVersion:     strconv.FormatInt(subscriptionCacheSchemaV3, 10),
		subFieldSubscriptionID:    "42",
		subFieldStatus:            service.SubscriptionStatusActive,
		subFieldStartsAt:          strconv.FormatInt(startsAt.Unix(), 10),
		subFieldStartsAtExact:     startsAt.Format(time.RFC3339Nano),
		subFieldExpiresAt:         strconv.FormatInt(expiresAt.Unix(), 10),
		subFieldExpiresAtExact:    expiresAt.Format(time.RFC3339Nano),
		subFieldWeeklyWindowStart: strconv.FormatInt(windowStart.Unix(), 10),
		subFieldWeeklyStartExact:  windowStart.Format(time.RFC3339Nano),
		subFieldWeeklyWindowEnd:   strconv.FormatInt(windowEnd.Unix(), 10),
		subFieldWeeklyEndExact:    windowEnd.Format(time.RFC3339Nano),
		subFieldWeeklyUsage:       "12.5",
		subFieldVersion:           "123456789",
	}
}

func TestParseSubscriptionCacheV3RequiresIdentityAndAuthoritativeFields(t *testing.T) {
	cache := &billingCache{}
	valid := validSubscriptionCacheV3Fields()

	parsed, err := cache.parseSubscriptionCache(valid)
	require.NoError(t, err)
	require.Equal(t, int64(42), parsed.SubscriptionID)
	require.Equal(t, 12.5, parsed.WeeklyUsage)
	require.Equal(t, int64(123456789), parsed.Version)

	tests := []struct {
		name  string
		field string
		value *string
	}{
		{name: "missing subscription id", field: subFieldSubscriptionID},
		{name: "invalid subscription id", field: subFieldSubscriptionID, value: cacheStringPtr("0")},
		{name: "missing weekly usage", field: subFieldWeeklyUsage},
		{name: "malformed weekly usage", field: subFieldWeeklyUsage, value: cacheStringPtr("bad")},
		{name: "non-finite weekly usage", field: subFieldWeeklyUsage, value: cacheStringPtr("NaN")},
		{name: "negative weekly usage", field: subFieldWeeklyUsage, value: cacheStringPtr("-1")},
		{name: "missing version", field: subFieldVersion},
		{name: "invalid version", field: subFieldVersion, value: cacheStringPtr("bad")},
		{name: "missing exact starts at", field: subFieldStartsAtExact},
		{name: "mismatched starts at unix", field: subFieldStartsAt, value: cacheStringPtr("1")},
		{name: "missing exact expires at", field: subFieldExpiresAtExact},
		{name: "missing weekly window start", field: subFieldWeeklyStartExact},
		{name: "missing weekly window end", field: subFieldWeeklyEndExact},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := validSubscriptionCacheV3Fields()
			if tt.value == nil {
				delete(fields, tt.field)
			} else {
				fields[tt.field] = *tt.value
			}
			_, err := cache.parseSubscriptionCache(fields)
			require.Error(t, err)
		})
	}
}

func TestSubscriptionCacheTTLCannotCrossResetOrExpiry(t *testing.T) {
	now := time.Now()
	windowStart := now.Add(-service.SubscriptionWeeklyWindowDuration + 3*time.Second)
	data := &service.SubscriptionCacheData{
		SubscriptionID:    1,
		Status:            service.SubscriptionStatusActive,
		StartsAt:          windowStart,
		ExpiresAt:         now.Add(2 * time.Second),
		WeeklyWindowStart: &windowStart,
		WeeklyWindowEnd:   now.Add(3 * time.Second),
		WeeklyUsage:       1,
		Version:           1,
	}

	ttl := subscriptionCacheTTL(data, now)
	require.Greater(t, ttl, time.Duration(0))
	require.LessOrEqual(t, ttl, 2*time.Second)
}

func cacheStringPtr(value string) *string {
	return &value
}
