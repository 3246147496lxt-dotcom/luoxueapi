package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type dashboardStatsHandlerRepoStub struct {
	service.UsageLogRepository
	stats *usagestats.DashboardStats
}

func (r *dashboardStatsHandlerRepoStub) GetDashboardStats(context.Context) (*usagestats.DashboardStats, error) {
	return r.stats, nil
}

func TestDashboardHandlerGetStats_ExposesActiveAPIKeyWeekContract(t *testing.T) {
	stats := &usagestats.DashboardStats{
		HealthyAccounts:                     53,
		TodayNewUsers:                       126,
		CurrentDayNewUsers:                  126,
		PreviousDaySamePeriodNewUsers:       116,
		TodayRequests:                       182460,
		CurrentDayRequests:                  182460,
		PreviousDaySamePeriodRequests:       174102,
		CurrentDayStartAt:                   "2026-07-21T00:00:00+08:00",
		CurrentDayEndAt:                     "2026-07-21T15:04:05+08:00",
		PreviousDaySamePeriodStartAt:        "2026-07-20T00:00:00+08:00",
		PreviousDaySamePeriodEndAt:          "2026-07-20T15:04:05+08:00",
		CurrentWeekActiveAPIKeys:            17,
		PreviousWeekSamePeriodActiveAPIKeys: 11,
		CurrentWeekStartAt:                  "2026-07-20T00:00:00+08:00",
		CurrentWeekEndAt:                    "2026-07-21T15:04:05+08:00",
		PreviousWeekSamePeriodStartAt:       "2026-07-13T00:00:00+08:00",
		PreviousWeekSamePeriodEndAt:         "2026-07-14T15:04:05+08:00",
		StatsTimezone:                       "Asia/Shanghai",
	}
	repo := &dashboardStatsHandlerRepoStub{stats: stats}
	handler := NewDashboardHandler(service.NewDashboardService(repo, nil, nil, nil), nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/admin/dashboard/stats", handler.GetStats)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/admin/dashboard/stats", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Data struct {
			HealthyAccounts                     int64  `json:"healthy_accounts"`
			TodayNewUsers                       int64  `json:"today_new_users"`
			CurrentDayNewUsers                  int64  `json:"current_day_new_users"`
			PreviousDaySamePeriodNewUsers       int64  `json:"previous_day_same_period_new_users"`
			TodayRequests                       int64  `json:"today_requests"`
			CurrentDayRequests                  int64  `json:"current_day_requests"`
			PreviousDaySamePeriodRequests       int64  `json:"previous_day_same_period_requests"`
			CurrentDayStartAt                   string `json:"current_day_start_at"`
			CurrentDayEndAt                     string `json:"current_day_end_at"`
			PreviousDaySamePeriodStartAt        string `json:"previous_day_same_period_start_at"`
			PreviousDaySamePeriodEndAt          string `json:"previous_day_same_period_end_at"`
			CurrentWeekActiveAPIKeys            int64  `json:"current_week_active_api_keys"`
			PreviousWeekSamePeriodActiveAPIKeys int64  `json:"previous_week_same_period_active_api_keys"`
			CurrentWeekStartAt                  string `json:"current_week_start_at"`
			CurrentWeekEndAt                    string `json:"current_week_end_at"`
			PreviousWeekSamePeriodStartAt       string `json:"previous_week_same_period_start_at"`
			PreviousWeekSamePeriodEndAt         string `json:"previous_week_same_period_end_at"`
			StatsTimezone                       string `json:"stats_timezone"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, stats.HealthyAccounts, response.Data.HealthyAccounts)
	require.Equal(t, stats.TodayNewUsers, response.Data.TodayNewUsers)
	require.Equal(t, stats.CurrentDayNewUsers, response.Data.CurrentDayNewUsers)
	require.Equal(t, stats.PreviousDaySamePeriodNewUsers, response.Data.PreviousDaySamePeriodNewUsers)
	require.Equal(t, stats.TodayRequests, response.Data.TodayRequests)
	require.Equal(t, stats.CurrentDayRequests, response.Data.CurrentDayRequests)
	require.Equal(t, stats.PreviousDaySamePeriodRequests, response.Data.PreviousDaySamePeriodRequests)
	require.Equal(t, stats.CurrentDayStartAt, response.Data.CurrentDayStartAt)
	require.Equal(t, stats.CurrentDayEndAt, response.Data.CurrentDayEndAt)
	require.Equal(t, stats.PreviousDaySamePeriodStartAt, response.Data.PreviousDaySamePeriodStartAt)
	require.Equal(t, stats.PreviousDaySamePeriodEndAt, response.Data.PreviousDaySamePeriodEndAt)
	require.Equal(t, stats.CurrentWeekActiveAPIKeys, response.Data.CurrentWeekActiveAPIKeys)
	require.Equal(t, stats.PreviousWeekSamePeriodActiveAPIKeys, response.Data.PreviousWeekSamePeriodActiveAPIKeys)
	require.Equal(t, stats.CurrentWeekStartAt, response.Data.CurrentWeekStartAt)
	require.Equal(t, stats.CurrentWeekEndAt, response.Data.CurrentWeekEndAt)
	require.Equal(t, stats.PreviousWeekSamePeriodStartAt, response.Data.PreviousWeekSamePeriodStartAt)
	require.Equal(t, stats.PreviousWeekSamePeriodEndAt, response.Data.PreviousWeekSamePeriodEndAt)
	require.Equal(t, stats.StatsTimezone, response.Data.StatsTimezone)
}

func TestDashboardSnapshotV2Stats_ExposesDayRequestComparison(t *testing.T) {
	stats := dashboardSnapshotV2Stats{
		DashboardStats: usagestats.DashboardStats{
			TodayNewUsers:                 126,
			CurrentDayNewUsers:            126,
			PreviousDaySamePeriodNewUsers: 116,
			TodayRequests:                 182460,
			CurrentDayRequests:            182460,
			PreviousDaySamePeriodRequests: 174102,
			CurrentDayStartAt:             "2026-07-21T00:00:00+08:00",
			CurrentDayEndAt:               "2026-07-21T15:04:05+08:00",
			PreviousDaySamePeriodStartAt:  "2026-07-20T00:00:00+08:00",
			PreviousDaySamePeriodEndAt:    "2026-07-20T15:04:05+08:00",
		},
	}

	payload, err := json.Marshal(stats)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(payload, &got))
	require.Equal(t, float64(stats.TodayNewUsers), got["today_new_users"])
	require.Equal(t, float64(stats.CurrentDayNewUsers), got["current_day_new_users"])
	require.Equal(t, float64(stats.PreviousDaySamePeriodNewUsers), got["previous_day_same_period_new_users"])
	require.Equal(t, float64(stats.TodayRequests), got["today_requests"])
	require.Equal(t, float64(stats.CurrentDayRequests), got["current_day_requests"])
	require.Equal(t, float64(stats.PreviousDaySamePeriodRequests), got["previous_day_same_period_requests"])
	require.Equal(t, stats.CurrentDayStartAt, got["current_day_start_at"])
	require.Equal(t, stats.CurrentDayEndAt, got["current_day_end_at"])
	require.Equal(t, stats.PreviousDaySamePeriodStartAt, got["previous_day_same_period_start_at"])
	require.Equal(t, stats.PreviousDaySamePeriodEndAt, got["previous_day_same_period_end_at"])
}
