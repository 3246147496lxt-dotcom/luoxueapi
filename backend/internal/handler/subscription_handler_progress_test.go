//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type subscriptionProgressRepoStub struct {
	service.UserSubscriptionRepository
	active []service.UserSubscription
	byID   map[int64]*service.UserSubscription
	errID  int64
}

func (s *subscriptionProgressRepoStub) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	out := make([]service.UserSubscription, len(s.active))
	copy(out, s.active)
	return out, nil
}

func (s *subscriptionProgressRepoStub) GetByID(_ context.Context, id int64) (*service.UserSubscription, error) {
	if id == s.errID {
		return nil, errors.New("read failed")
	}
	sub, ok := s.byID[id]
	if !ok {
		return nil, service.ErrSubscriptionNotFound
	}
	clone := *sub
	return &clone, nil
}

func newSubscriptionProgressTestContext(t *testing.T, handler *SubscriptionHandler) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/progress", nil)
	ctx.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
	return recorder, ctx
}

func TestSubscriptionProgressContractUsesNestedExplicitDTO(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	windowStart := now.Add(-24 * time.Hour)
	limit := 50.0
	group := &service.Group{
		ID:             7,
		Name:           "Pro",
		WeeklyLimitUSD: &limit,
	}
	sub := service.UserSubscription{
		ID:                9,
		UserID:            42,
		GroupID:           group.ID,
		Status:            service.SubscriptionStatusActive,
		StartsAt:          windowStart,
		ExpiresAt:         now.Add(30 * 24 * time.Hour),
		WeeklyWindowStart: &windowStart,
		WeeklyUsageUSD:    12.5,
		Group:             group,
	}
	repo := &subscriptionProgressRepoStub{
		active: []service.UserSubscription{sub},
		byID:   map[int64]*service.UserSubscription{sub.ID: &sub},
	}
	subscriptionService := service.NewSubscriptionService(nil, repo, nil, nil, nil)
	handler := NewSubscriptionHandler(subscriptionService)
	recorder, ctx := newSubscriptionProgressTestContext(t, handler)

	handler.GetProgress(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Code int `json:"code"`
		Data []struct {
			Subscription struct {
				ID int64 `json:"id"`
			} `json:"subscription"`
			Progress struct {
				ID        int64  `json:"id"`
				GroupName string `json:"group_name"`
				Weekly    struct {
					State        string  `json:"state"`
					LimitUSD     float64 `json:"limit_usd"`
					UsedUSD      float64 `json:"used_usd"`
					RemainingUSD float64 `json:"remaining_usd"`
				} `json:"weekly"`
			} `json:"progress"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, 0, payload.Code)
	require.Len(t, payload.Data, 1)
	require.Equal(t, int64(9), payload.Data[0].Subscription.ID)
	require.Equal(t, int64(9), payload.Data[0].Progress.ID)
	require.Equal(t, "Pro", payload.Data[0].Progress.GroupName)
	require.Equal(t, "active", payload.Data[0].Progress.Weekly.State)
	require.Equal(t, 50.0, payload.Data[0].Progress.Weekly.LimitUSD)
	require.Equal(t, 12.5, payload.Data[0].Progress.Weekly.UsedUSD)
	require.Equal(t, 37.5, payload.Data[0].Progress.Weekly.RemainingUSD)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &raw))
	items := raw["data"].([]any)
	item := items[0].(map[string]any)
	require.NotContains(t, item, "subscription_id")
	require.Contains(t, item, "subscription")
	require.Contains(t, item, "progress")
}

func TestSubscriptionProgressStaleWindowIsUnknownNotZero(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	anchor := now.Add(-15 * 24 * time.Hour)
	staleStart := anchor.Add(7 * 24 * time.Hour)
	limit := 50.0
	group := &service.Group{ID: 7, Name: "Pro", WeeklyLimitUSD: &limit}
	sub := service.UserSubscription{
		ID:                9,
		UserID:            42,
		GroupID:           group.ID,
		Status:            service.SubscriptionStatusActive,
		StartsAt:          anchor,
		ExpiresAt:         now.Add(30 * 24 * time.Hour),
		WeeklyWindowStart: &staleStart,
		WeeklyUsageUSD:    999,
		Group:             group,
	}
	repo := &subscriptionProgressRepoStub{
		active: []service.UserSubscription{sub},
		byID:   map[int64]*service.UserSubscription{sub.ID: &sub},
	}
	handler := NewSubscriptionHandler(
		service.NewSubscriptionService(nil, repo, nil, nil, nil),
	)
	recorder, ctx := newSubscriptionProgressTestContext(t, handler)

	handler.GetProgress(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Data []struct {
			Progress struct {
				Weekly struct {
					State        string   `json:"state"`
					UsedUSD      *float64 `json:"used_usd"`
					RemainingUSD *float64 `json:"remaining_usd"`
					Percentage   *float64 `json:"percentage"`
				} `json:"weekly"`
			} `json:"progress"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Len(t, payload.Data, 1)
	require.Equal(t, "unknown", payload.Data[0].Progress.Weekly.State)
	require.Nil(t, payload.Data[0].Progress.Weekly.UsedUSD)
	require.Nil(t, payload.Data[0].Progress.Weekly.RemainingUSD)
	require.Nil(t, payload.Data[0].Progress.Weekly.Percentage)
}

func TestSubscriptionSummaryPreservesExplicitZeroMonthlyLimit(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	weeklyLimit := 50.0
	monthlyLimit := 0.0
	group := &service.Group{
		ID:              7,
		Name:            "Zero Month",
		WeeklyLimitUSD:  &weeklyLimit,
		MonthlyLimitUSD: &monthlyLimit,
	}
	sub := service.UserSubscription{
		ID:                 9,
		UserID:             42,
		GroupID:            group.ID,
		Status:             service.SubscriptionStatusActive,
		StartsAt:           now,
		ExpiresAt:          now.Add(30 * 24 * time.Hour),
		WeeklyWindowStart:  &now,
		MonthlyWindowStart: &now,
		Group:              group,
	}
	repo := &subscriptionProgressRepoStub{active: []service.UserSubscription{sub}}
	handler := NewSubscriptionHandler(service.NewSubscriptionService(nil, repo, nil, nil, nil))
	recorder, ctx := newSubscriptionProgressTestContext(t, handler)

	handler.GetSummary(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload struct {
		Data struct {
			Subscriptions []map[string]any `json:"subscriptions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Len(t, payload.Data.Subscriptions, 1)
	value, exists := payload.Data.Subscriptions[0]["monthly_limit_usd"]
	require.True(t, exists)
	require.Equal(t, float64(0), value)
}

func TestSubscriptionProgressReadFailureDoesNotReturnPartialSuccess(t *testing.T) {
	now := time.Now().UTC()
	group := &service.Group{ID: 7, Name: "Pro"}
	first := service.UserSubscription{
		ID: 1, UserID: 42, GroupID: group.ID, Group: group,
		Status: service.SubscriptionStatusActive, ExpiresAt: now.Add(time.Hour),
	}
	second := first
	second.ID = 2
	repo := &subscriptionProgressRepoStub{
		active: []service.UserSubscription{first, second},
		byID:   map[int64]*service.UserSubscription{first.ID: &first},
		errID:  second.ID,
	}
	subscriptionService := service.NewSubscriptionService(nil, repo, nil, nil, nil)
	handler := NewSubscriptionHandler(subscriptionService)
	recorder, ctx := newSubscriptionProgressTestContext(t, handler)

	handler.GetProgress(ctx)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.NotContains(t, recorder.Body.String(), `"data"`)
	require.NotContains(t, recorder.Body.String(), `"progress"`)
}
