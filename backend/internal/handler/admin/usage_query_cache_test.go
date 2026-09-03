package admin

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

func TestUsageStatsCacheKey_StableAndDistinct(t *testing.T) {
	start := time.Date(2026, 5, 29, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	base := usagestats.UsageLogFilters{StartTime: &start, EndTime: &end, Model: "claude-3"}

	k1 := usageStatsCacheKey(base)
	k2 := usageStatsCacheKey(base)
	require.NotEmpty(t, k1)
	require.Equal(t, k1, k2, "same filters must produce same key")

	other := base
	other.Model = "gpt-4o"
	require.NotEqual(t, k1, usageStatsCacheKey(other), "different model must change key")

	withUser := base
	withUser.UserID = 7
	require.NotEqual(t, k1, usageStatsCacheKey(withUser), "different user must change key")

	matched := false
	withMatchedAudit := base
	withMatchedAudit.UpstreamModelMismatch = &matched
	mismatched := true
	withMismatchedAudit := base
	withMismatchedAudit.UpstreamModelMismatch = &mismatched
	require.NotEqual(t, k1, usageStatsCacheKey(withMatchedAudit), "unobserved and explicit match must not share a key")
	require.NotEqual(t, k1, usageStatsCacheKey(withMismatchedAudit), "unobserved and explicit mismatch must not share a key")
	require.NotEqual(t, usageStatsCacheKey(withMatchedAudit), usageStatsCacheKey(withMismatchedAudit), "match and mismatch must not share a key")
}

func TestDashboardCacheKeysDistinguishUpstreamModelAuditTriState(t *testing.T) {
	matched := false
	mismatched := true
	trendBase := dashboardTrendCacheKey{StartTime: "start", EndTime: "end", Granularity: "hour"}
	modelBase := dashboardModelGroupCacheKey{StartTime: "start", EndTime: "end", ModelSource: "requested"}

	for _, keys := range [][3]string{
		{
			mustMarshalDashboardCacheKey(trendBase),
			mustMarshalDashboardCacheKey(func() dashboardTrendCacheKey {
				value := trendBase
				value.UpstreamModelMismatch = &matched
				return value
			}()),
			mustMarshalDashboardCacheKey(func() dashboardTrendCacheKey {
				value := trendBase
				value.UpstreamModelMismatch = &mismatched
				return value
			}()),
		},
		{
			mustMarshalDashboardCacheKey(modelBase),
			mustMarshalDashboardCacheKey(func() dashboardModelGroupCacheKey {
				value := modelBase
				value.UpstreamModelMismatch = &matched
				return value
			}()),
			mustMarshalDashboardCacheKey(func() dashboardModelGroupCacheKey {
				value := modelBase
				value.UpstreamModelMismatch = &mismatched
				return value
			}()),
		},
	} {
		require.NotEqual(t, keys[0], keys[1])
		require.NotEqual(t, keys[0], keys[2])
		require.NotEqual(t, keys[1], keys[2])
	}
}
