package repository

import (
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func validSubscriptionCacheV4Fields() map[string]string {
	startsAt := time.Date(2026, time.July, 29, 13, 47, 12, 345678901, time.UTC)
	weeklyWindowStart := startsAt.Add(service.SubscriptionWeeklyWindowDuration)
	weeklyWindowEnd := weeklyWindowStart.Add(service.SubscriptionWeeklyWindowDuration)
	monthlyWindowStart := startsAt
	monthlyWindowEnd := monthlyWindowStart.Add(service.SubscriptionMonthlyWindowDuration)
	expiresAt := startsAt.Add(60 * 24 * time.Hour)
	return map[string]string{
		subFieldSchemaVersion:      strconv.FormatInt(subscriptionCacheSchemaV4, 10),
		subFieldSubscriptionID:     "42",
		subFieldStatus:             service.SubscriptionStatusActive,
		subFieldStartsAt:           strconv.FormatInt(startsAt.Unix(), 10),
		subFieldStartsAtExact:      startsAt.Format(time.RFC3339Nano),
		subFieldExpiresAt:          strconv.FormatInt(expiresAt.Unix(), 10),
		subFieldExpiresAtExact:     expiresAt.Format(time.RFC3339Nano),
		subFieldWeeklyWindowStart:  strconv.FormatInt(weeklyWindowStart.Unix(), 10),
		subFieldWeeklyStartExact:   weeklyWindowStart.Format(time.RFC3339Nano),
		subFieldWeeklyWindowEnd:    strconv.FormatInt(weeklyWindowEnd.Unix(), 10),
		subFieldWeeklyEndExact:     weeklyWindowEnd.Format(time.RFC3339Nano),
		subFieldMonthlyWindowStart: strconv.FormatInt(monthlyWindowStart.Unix(), 10),
		subFieldMonthlyStartExact:  monthlyWindowStart.Format(time.RFC3339Nano),
		subFieldMonthlyWindowEnd:   strconv.FormatInt(monthlyWindowEnd.Unix(), 10),
		subFieldMonthlyEndExact:    monthlyWindowEnd.Format(time.RFC3339Nano),
		subFieldWeeklyUsage:        "12.5",
		subFieldMonthlyUsage:       "41.25",
		subFieldVersion:            "123456789",
	}
}

func TestParseSubscriptionCacheV4RequiresIdentityAndAuthoritativeFields(t *testing.T) {
	cache := &billingCache{}
	valid := validSubscriptionCacheV4Fields()

	parsed, err := cache.parseSubscriptionCache(valid)
	require.NoError(t, err)
	require.Equal(t, int64(42), parsed.SubscriptionID)
	require.Equal(t, 12.5, parsed.WeeklyUsage)
	require.Equal(t, 41.25, parsed.MonthlyUsage)
	require.NotNil(t, parsed.MonthlyWindowStart)
	require.True(t, parsed.MonthlyWindowEnd.Equal(parsed.MonthlyWindowStart.Add(service.SubscriptionMonthlyWindowDuration)))
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
		{name: "missing monthly usage", field: subFieldMonthlyUsage},
		{name: "malformed monthly usage", field: subFieldMonthlyUsage, value: cacheStringPtr("bad")},
		{name: "non-finite monthly usage", field: subFieldMonthlyUsage, value: cacheStringPtr("+Inf")},
		{name: "negative monthly usage", field: subFieldMonthlyUsage, value: cacheStringPtr("-1")},
		{name: "missing version", field: subFieldVersion},
		{name: "invalid version", field: subFieldVersion, value: cacheStringPtr("bad")},
		{name: "missing exact starts at", field: subFieldStartsAtExact},
		{name: "mismatched starts at unix", field: subFieldStartsAt, value: cacheStringPtr("1")},
		{name: "missing exact expires at", field: subFieldExpiresAtExact},
		{name: "missing weekly window start", field: subFieldWeeklyStartExact},
		{name: "missing weekly window end", field: subFieldWeeklyEndExact},
		{name: "missing monthly window start", field: subFieldMonthlyStartExact},
		{name: "mismatched monthly window start unix", field: subFieldMonthlyWindowStart, value: cacheStringPtr("1")},
		{name: "missing monthly window end", field: subFieldMonthlyEndExact},
		{name: "mismatched monthly window end unix", field: subFieldMonthlyWindowEnd, value: cacheStringPtr("1")},
		{name: "old schema", field: subFieldSchemaVersion, value: cacheStringPtr("3")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := validSubscriptionCacheV4Fields()
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

func TestParseSubscriptionCacheV4RejectsUnanchoredMonthlyWindow(t *testing.T) {
	cache := &billingCache{}
	fields := validSubscriptionCacheV4Fields()
	startsAt, err := time.Parse(time.RFC3339Nano, fields[subFieldStartsAtExact])
	require.NoError(t, err)
	monthlyWindowStart := startsAt.Add(time.Hour)
	monthlyWindowEnd := monthlyWindowStart.Add(service.SubscriptionMonthlyWindowDuration)
	fields[subFieldMonthlyWindowStart] = strconv.FormatInt(monthlyWindowStart.Unix(), 10)
	fields[subFieldMonthlyStartExact] = monthlyWindowStart.Format(time.RFC3339Nano)
	fields[subFieldMonthlyWindowEnd] = strconv.FormatInt(monthlyWindowEnd.Unix(), 10)
	fields[subFieldMonthlyEndExact] = monthlyWindowEnd.Format(time.RFC3339Nano)

	_, err = cache.parseSubscriptionCache(fields)

	require.EqualError(t, err, "invalid cache: monthly window is not anchored")
}

func TestValidateSubscriptionCacheDataRequiresAuthoritativeMonthlyFields(t *testing.T) {
	fields := validSubscriptionCacheV4Fields()
	cache := &billingCache{}
	valid, err := cache.parseSubscriptionCache(fields)
	require.NoError(t, err)
	require.NoError(t, validateSubscriptionCacheData(valid))

	missingWindow := *valid
	missingWindow.MonthlyWindowStart = nil
	require.EqualError(
		t,
		validateSubscriptionCacheData(&missingWindow),
		"invalid subscription cache data: invalid monthly window",
	)

	invalidUsage := *valid
	invalidUsage.MonthlyUsage = -1
	require.EqualError(
		t,
		validateSubscriptionCacheData(&invalidUsage),
		"invalid subscription cache data: invalid monthly usage",
	)
}

func TestSubscriptionCacheTTLCannotCrossResetOrExpiry(t *testing.T) {
	now := time.Now()
	startsAt := now.Add(-service.SubscriptionMonthlyWindowDuration + 3*time.Second)
	weeklyWindowStart, weeklyWindowEnd, weeklyOK := service.AnchoredWeeklyWindow(startsAt, now)
	require.True(t, weeklyOK)
	monthlyWindowStart, monthlyWindowEnd, monthlyOK := service.AnchoredMonthlyWindow(startsAt, now)
	require.True(t, monthlyOK)
	data := &service.SubscriptionCacheData{
		SubscriptionID:     1,
		Status:             service.SubscriptionStatusActive,
		StartsAt:           startsAt,
		ExpiresAt:          now.Add(10 * time.Second),
		WeeklyWindowStart:  &weeklyWindowStart,
		WeeklyWindowEnd:    weeklyWindowEnd,
		MonthlyWindowStart: &monthlyWindowStart,
		MonthlyWindowEnd:   monthlyWindowEnd,
		WeeklyUsage:        1,
		MonthlyUsage:       2,
		Version:            1,
	}

	ttl := subscriptionCacheTTL(data, now)
	require.Greater(t, ttl, time.Duration(0))
	require.LessOrEqual(t, ttl, 3*time.Second)
}

func cacheStringPtr(value string) *string {
	return &value
}
