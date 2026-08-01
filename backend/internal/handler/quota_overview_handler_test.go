//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type quotaOverviewUsecaseStub struct {
	overview *service.QuotaOverview
	err      error
	userID   int64
	timezone string
	calls    int
}

func (s *quotaOverviewUsecaseStub) GetOverview(_ context.Context, userID int64, timezone string) (*service.QuotaOverview, error) {
	s.calls++
	s.userID = userID
	s.timezone = timezone
	if s.overview == nil {
		return nil, s.err
	}
	clone := *s.overview
	return &clone, s.err
}

func quotaOverviewHandlerFixture() *service.QuotaOverview {
	asOf := time.Date(2026, 7, 29, 7, 30, 0, 0, time.UTC)
	periodEnd := asOf.Add(2 * time.Minute)
	limit := "200.0000000000"
	used := "12.3456789012"
	remaining := "187.6543210988"
	percent := 6.1728394506
	monthlyPeriodEnd := asOf.Add(29 * 24 * time.Hour)
	monthlyLimit := "800.0000000000"
	monthlyUsed := "125.0000000000"
	monthlyRemaining := "675.0000000000"
	monthlyPercent := 15.625
	totalRequests := int64(58)
	totalTokens := int64(182000)
	cacheHitTokens := int64(92000)
	cacheMissTokens := int64(41000)
	outputTokens := int64(49000)
	subID := "9007199254740995"
	return &service.QuotaOverview{
		SchemaVersion:   1,
		GeneratedAt:     asOf.Add(time.Second),
		AsOf:            asOf,
		FreshUntil:      periodEnd,
		DisplayTimezone: "Asia/Shanghai",
		Freshness:       service.QuotaFreshnessFresh,
		Coverage: service.QuotaOverviewCoverage{
			Included: []string{
				"wallet",
				"api_key_billing_groups",
				"subscription_7d",
				"subscription_period_usage",
			},
			Excluded: []string{"upstream_quota", "codex_quota"},
		},
		Account: service.QuotaOverviewAccount{
			DisplayLabel:      "p***@example.com",
			DataScope:         "all_enabled_api_keys",
			QuotaState:        service.QuotaStateAllAvailable,
			CanMakeRequest:    nil,
			UsableGroupCount:  1,
			BlockedGroupCount: 0,
			UnknownGroupCount: 0,
		},
		Wallet: service.QuotaOverviewWallet{
			Unit: "snow_credit", State: service.QuotaWalletAvailable,
			Available: "128.6400000000", Reserved: "0.0000000000",
			TodaySpend: "1.1600000000", MonthSpend: "10.7400000000",
		},
		BillingGroups: []service.QuotaOverviewBillingGroup{
			{
				ID: "9007199254740994", DisplayName: "Pro 会员",
				BillingMode: service.QuotaBillingSubscription, State: service.QuotaGroupUsable,
				RecommendedAction: "none", FallbackPolicy: "none",
				ResourceRef: service.QuotaOverviewResourceRef{Kind: "subscription", ID: &subID},
				Keys: []service.QuotaOverviewKey{
					{ID: "9007199254740993", Name: "Codex", MaskedKey: "sk-••••42FD", State: service.QuotaGroupUsable},
				},
			},
		},
		Subscriptions: []service.QuotaOverviewSubscription{
			{
				ID: subID, GroupID: "9007199254740994", Name: "Pro 会员",
				Status: service.SubscriptionStatusActive, StartsAt: asOf.Add(-24 * time.Hour),
				ExpiresAt: asOf.Add(30 * 24 * time.Hour),
				WeeklyWindow: service.QuotaOverviewWeeklyWindow{
					Kind: "7d_from_subscription_start", State: service.QuotaWindowActive,
					AnchorAt: asOf.Add(-24 * time.Hour), PeriodStart: timePtr(asOf.Add(-24 * time.Hour)),
					PeriodEnd: &periodEnd, ResetsAt: &periodEnd,
					Limit: &limit, Used: &used, Remaining: &remaining, UsedPercent: &percent,
				},
				MonthlyWindow: service.QuotaOverviewMonthlyWindow{
					Kind: "30d_from_subscription_start", State: service.QuotaWindowActive,
					AnchorAt: asOf.Add(-24 * time.Hour), PeriodStart: timePtr(asOf.Add(-24 * time.Hour)),
					PeriodEnd: &monthlyPeriodEnd, ResetsAt: &monthlyPeriodEnd,
					Limit: &monthlyLimit, Used: &monthlyUsed, Remaining: &monthlyRemaining,
					UsedPercent: &monthlyPercent,
				},
				PeriodUsage: service.QuotaOverviewPeriodUsage{
					State:         service.QuotaPeriodUsageAvailable,
					ObservedUntil: &asOf,
					BucketKind:    service.QuotaPeriodUsageBucketAnchored24h,
					TotalRequests: &totalRequests,
					TotalTokens:   &totalTokens,
					Points: []service.QuotaOverviewPeriodUsagePoint{
						{
							Index: 1, StartAt: asOf.Add(-24 * time.Hour), EndAt: asOf,
							State:    service.QuotaPeriodUsagePointComplete,
							Requests: &totalRequests, CacheHitTokens: &cacheHitTokens,
							CacheMissTokens: &cacheMissTokens, OutputTokens: &outputTokens,
							TotalTokens: &totalTokens,
						},
						{
							Index: 2, StartAt: asOf, EndAt: asOf.Add(24 * time.Hour),
							State: service.QuotaPeriodUsagePointFuture,
						},
					},
				},
			},
		},
		Warnings: []string{},
	}
}

func quotaOverviewTestRouter(subject *quotaauth.Subject, usecase *quotaOverviewUsecaseStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RequestLogger())
	if subject != nil {
		router.Use(func(c *gin.Context) {
			c.Set("quota_auth_subject", *subject)
			c.Next()
		})
	}
	handler := &QuotaOverviewHandler{service: usecase}
	router.GET(
		"/api/v1/quota/overview",
		middleware.RequireQuotaRead(),
		handler.GetOverview,
	)
	return router
}

func TestQuotaOverviewHandlerUsesDeviceSubjectAndSafeWireContract(t *testing.T) {
	usecase := &quotaOverviewUsecaseStub{overview: quotaOverviewHandlerFixture()}
	expectedAsOf := usecase.overview.AsOf
	router := quotaOverviewTestRouter(&quotaauth.Subject{
		UserID: 84, ClientID: quotaauth.ClientID, Scopes: []string{quotaauth.ScopeRead},
	}, usecase)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/quota/overview?timezone=Asia%2FShanghai",
		nil,
	)
	request.Header.Set("Authorization", "Bearer quota-viewer-token")
	request.Header.Set("X-Request-ID", "request-safe-1")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "Authorization", recorder.Header().Get("Vary"))
	require.Equal(t, int64(84), usecase.userID)
	require.Equal(t, "Asia/Shanghai", usecase.timezone)
	require.NotContains(t, recorder.Body.String(), "sk-secret")
	require.NotContains(t, recorder.Body.String(), `"key":`)
	require.NotContains(t, recorder.Body.String(), `"email":`)

	var payload struct {
		Data struct {
			RequestID string `json:"request_id"`
			Account   struct {
				CanMakeRequest any `json:"can_make_request"`
			} `json:"account"`
			Wallet struct {
				Available  string `json:"available"`
				TodaySpend string `json:"today_spend"`
				MonthSpend string `json:"month_spend"`
			} `json:"wallet"`
			BillingGroups []struct {
				ID   string `json:"id"`
				Keys []struct {
					ID        string `json:"id"`
					MaskedKey string `json:"masked_key"`
				} `json:"keys"`
			} `json:"billing_groups"`
			Subscriptions []struct {
				ID           string `json:"id"`
				WeeklyWindow struct {
					Limit string `json:"limit"`
					Used  string `json:"used"`
				} `json:"weekly_window"`
				MonthlyWindow struct {
					Kind        string  `json:"kind"`
					State       string  `json:"state"`
					Limit       string  `json:"limit"`
					Used        string  `json:"used"`
					Remaining   string  `json:"remaining"`
					UsedPercent float64 `json:"used_percent"`
				} `json:"monthly_window"`
				PeriodUsage struct {
					State         string    `json:"state"`
					ObservedUntil time.Time `json:"observed_until"`
					BucketKind    string    `json:"bucket_kind"`
					TotalRequests int64     `json:"total_requests"`
					TotalTokens   int64     `json:"total_tokens"`
					Points        []struct {
						Index           int    `json:"index"`
						State           string `json:"state"`
						Requests        *int64 `json:"requests"`
						CacheHitTokens  *int64 `json:"cache_hit_tokens"`
						CacheMissTokens *int64 `json:"cache_miss_tokens"`
						OutputTokens    *int64 `json:"output_tokens"`
						TotalTokens     *int64 `json:"total_tokens"`
					} `json:"points"`
				} `json:"period_usage"`
			} `json:"subscriptions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "request-safe-1", payload.Data.RequestID)
	require.Nil(t, payload.Data.Account.CanMakeRequest)
	require.Equal(t, "128.6400000000", payload.Data.Wallet.Available)
	require.Equal(t, "1.1600000000", payload.Data.Wallet.TodaySpend)
	require.Equal(t, "10.7400000000", payload.Data.Wallet.MonthSpend)
	require.Equal(t, "9007199254740994", payload.Data.BillingGroups[0].ID)
	require.Equal(t, "9007199254740993", payload.Data.BillingGroups[0].Keys[0].ID)
	require.Equal(t, "sk-••••42FD", payload.Data.BillingGroups[0].Keys[0].MaskedKey)
	require.Equal(t, "9007199254740995", payload.Data.Subscriptions[0].ID)
	require.Equal(t, "200.0000000000", payload.Data.Subscriptions[0].WeeklyWindow.Limit)
	require.Equal(t, "12.3456789012", payload.Data.Subscriptions[0].WeeklyWindow.Used)
	require.Equal(t, "30d_from_subscription_start", payload.Data.Subscriptions[0].MonthlyWindow.Kind)
	require.Equal(t, service.QuotaWindowActive, payload.Data.Subscriptions[0].MonthlyWindow.State)
	require.Equal(t, "800.0000000000", payload.Data.Subscriptions[0].MonthlyWindow.Limit)
	require.Equal(t, "125.0000000000", payload.Data.Subscriptions[0].MonthlyWindow.Used)
	require.Equal(t, "675.0000000000", payload.Data.Subscriptions[0].MonthlyWindow.Remaining)
	require.Equal(t, 15.625, payload.Data.Subscriptions[0].MonthlyWindow.UsedPercent)
	require.Equal(t, service.QuotaPeriodUsageAvailable, payload.Data.Subscriptions[0].PeriodUsage.State)
	require.Equal(t, expectedAsOf, payload.Data.Subscriptions[0].PeriodUsage.ObservedUntil)
	require.Equal(t, service.QuotaPeriodUsageBucketAnchored24h, payload.Data.Subscriptions[0].PeriodUsage.BucketKind)
	require.Equal(t, int64(58), payload.Data.Subscriptions[0].PeriodUsage.TotalRequests)
	require.Equal(t, int64(182000), payload.Data.Subscriptions[0].PeriodUsage.TotalTokens)
	require.Len(t, payload.Data.Subscriptions[0].PeriodUsage.Points, 2)
	require.Equal(t, int64(92000), *payload.Data.Subscriptions[0].PeriodUsage.Points[0].CacheHitTokens)
	require.Equal(t, int64(41000), *payload.Data.Subscriptions[0].PeriodUsage.Points[0].CacheMissTokens)
	require.Equal(t, int64(49000), *payload.Data.Subscriptions[0].PeriodUsage.Points[0].OutputTokens)
	require.Nil(t, payload.Data.Subscriptions[0].PeriodUsage.Points[1].Requests)
	require.Nil(t, payload.Data.Subscriptions[0].PeriodUsage.Points[1].TotalTokens)
}

func TestQuotaOverviewHandlerRejectsWrongAuthAndUnsupportedQuery(t *testing.T) {
	usecase := &quotaOverviewUsecaseStub{overview: quotaOverviewHandlerFixture()}
	router := quotaOverviewTestRouter(&quotaauth.Subject{
		UserID: 84, ClientID: quotaauth.ClientID, Scopes: []string{"other:scope"},
	}, usecase)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/quota/overview", nil)
	request.Header.Set("Authorization", "Bearer wrong-scope")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Zero(t, usecase.calls)

	usecase = &quotaOverviewUsecaseStub{overview: quotaOverviewHandlerFixture()}
	router = quotaOverviewTestRouter(&quotaauth.Subject{
		UserID: 84, ClientID: quotaauth.ClientID, Scopes: []string{quotaauth.ScopeRead},
	}, usecase)
	request = httptest.NewRequest(http.MethodGet, "/api/v1/quota/overview?user_id=42", nil)
	request.Header.Set("Authorization", "Bearer quota-viewer-token")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Zero(t, usecase.calls)

	request = httptest.NewRequest(http.MethodGet, "/api/v1/quota/overview?timezone=Not%2FAZone", nil)
	request.Header.Set("Authorization", "Bearer quota-viewer-token")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Zero(t, usecase.calls)

	router = quotaOverviewTestRouter(nil, usecase)
	request = httptest.NewRequest(http.MethodGet, "/api/v1/quota/overview", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Zero(t, usecase.calls)
}

func TestQuotaOverviewHandlerSerializesUnknownPeriodUsageAsNullNotZero(t *testing.T) {
	overview := quotaOverviewHandlerFixture()
	overview.Subscriptions[0].PeriodUsage = service.QuotaOverviewPeriodUsage{
		State:      service.QuotaPeriodUsageUnknown,
		BucketKind: service.QuotaPeriodUsageBucketAnchored24h,
	}
	usecase := &quotaOverviewUsecaseStub{overview: overview}
	router := quotaOverviewTestRouter(&quotaauth.Subject{
		UserID: 84, ClientID: quotaauth.ClientID, Scopes: []string{quotaauth.ScopeRead},
	}, usecase)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/quota/overview", nil)
	request.Header.Set("Authorization", "Bearer quota-viewer-token")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Data struct {
			Coverage struct {
				Included []string `json:"included"`
			} `json:"coverage"`
			Subscriptions []struct {
				PeriodUsage struct {
					State         string          `json:"state"`
					ObservedUntil json.RawMessage `json:"observed_until"`
					TotalRequests json.RawMessage `json:"total_requests"`
					TotalTokens   json.RawMessage `json:"total_tokens"`
					Points        json.RawMessage `json:"points"`
				} `json:"period_usage"`
			} `json:"subscriptions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Contains(t, payload.Data.Coverage.Included, "subscription_period_usage")
	require.Len(t, payload.Data.Subscriptions, 1)
	usage := payload.Data.Subscriptions[0].PeriodUsage
	require.Equal(t, service.QuotaPeriodUsageUnknown, usage.State)
	require.JSONEq(t, "null", string(usage.ObservedUntil))
	require.JSONEq(t, "null", string(usage.TotalRequests))
	require.JSONEq(t, "null", string(usage.TotalTokens))
	require.JSONEq(t, "null", string(usage.Points))
}

func timePtr(value time.Time) *time.Time {
	return &value
}
