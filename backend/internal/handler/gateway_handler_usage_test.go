package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUsageUnrestrictedIncludesWeeklyWindowStart(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/usage", nil)

	weeklyWindowStart := time.Date(2026, time.July, 13, 0, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	c.Set(string(middleware.ContextKeySubscription), &service.UserSubscription{
		WeeklyWindowStart: &weeklyWindowStart,
	})

	handler := &GatewayHandler{}
	handler.usageUnrestricted(
		c,
		context.Background(),
		&service.APIKey{Group: &service.Group{
			Name:             "Weekly plan",
			SubscriptionType: service.SubscriptionTypeSubscription,
		}},
		middleware.AuthSubject{},
		nil,
		nil,
		nil,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Unit         string `json:"unit"`
		Subscription struct {
			WeeklyWindowStart *time.Time `json:"weekly_window_start"`
		} `json:"subscription"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "CREDIT", response.Unit)
	require.NotNil(t, response.Subscription.WeeklyWindowStart)
	require.True(t, weeklyWindowStart.Equal(*response.Subscription.WeeklyWindowStart))
}

func TestUsageQuotaLimitedReportsCreditUnit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/usage", nil)

	handler := &GatewayHandler{}
	handler.usageQuotaLimited(
		c,
		context.Background(),
		&service.APIKey{
			Status:    service.StatusAPIKeyActive,
			Quota:     500,
			QuotaUsed: 125,
		},
		nil,
		nil,
		nil,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Unit  string `json:"unit"`
		Quota struct {
			Unit      string  `json:"unit"`
			Limit     float64 `json:"limit"`
			Used      float64 `json:"used"`
			Remaining float64 `json:"remaining"`
		} `json:"quota"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "CREDIT", response.Unit)
	require.Equal(t, "CREDIT", response.Quota.Unit)
	require.Equal(t, 500.0, response.Quota.Limit)
	require.Equal(t, 125.0, response.Quota.Used)
	require.Equal(t, 375.0, response.Quota.Remaining)
}

type gatewayUsageUserRepoStub struct {
	service.UserRepository
	user *service.User
}

func (s *gatewayUsageUserRepoStub) GetByID(context.Context, int64) (*service.User, error) {
	cloned := *s.user
	return &cloned, nil
}

func (s *gatewayUsageUserRepoStub) GetUserAvatar(context.Context, int64) (*service.UserAvatar, error) {
	return nil, nil
}

func TestUsageUnrestrictedWalletReportsCreditUnit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/usage", nil)

	userRepo := &gatewayUsageUserRepoStub{
		user: &service.User{ID: 42, Balance: 123.45},
	}
	handler := &GatewayHandler{
		userService: service.NewUserService(userRepo, nil, nil, nil),
	}
	handler.usageUnrestricted(
		c,
		context.Background(),
		&service.APIKey{},
		middleware.AuthSubject{UserID: 42},
		nil,
		nil,
		nil,
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Unit      string  `json:"unit"`
		Balance   float64 `json:"balance"`
		Remaining float64 `json:"remaining"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "CREDIT", response.Unit)
	require.Equal(t, 123.45, response.Balance)
	require.Equal(t, 123.45, response.Remaining)
}
