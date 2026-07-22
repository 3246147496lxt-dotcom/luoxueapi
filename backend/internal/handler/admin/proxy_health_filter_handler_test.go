package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type proxyHealthHandlerRepo struct {
	service.ProxyRepository
	items []service.ProxyWithAccountImpact
}

func (r *proxyHealthHandlerRepo) ListAllWithAccountImpact(context.Context) ([]service.ProxyWithAccountImpact, error) {
	return r.items, nil
}

type proxyHealthHandlerCache struct {
	service.ProxyLatencyCache
	items map[int64]*service.ProxyLatencyInfo
}

func (c *proxyHealthHandlerCache) GetProxyLatencies(context.Context, []int64) (map[int64]*service.ProxyLatencyInfo, error) {
	return c.items, nil
}

func TestProxyListHealthFilterKeepsFilteredPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	qualityChecked := now.Unix()
	adminSvc := newStubAdminService()
	adminSvc.proxyCounts = []service.ProxyWithAccountCount{
		{Proxy: service.Proxy{ID: 1, Name: "healthy", Protocol: "http", Status: service.StatusActive}},
		{Proxy: service.Proxy{ID: 2, Name: "failed", Protocol: "http", Status: service.StatusActive}},
	}
	repo := &proxyHealthHandlerRepo{items: []service.ProxyWithAccountImpact{
		{Proxy: adminSvc.proxyCounts[0].Proxy},
		{Proxy: adminSvc.proxyCounts[1].Proxy},
	}}
	cache := &proxyHealthHandlerCache{items: map[int64]*service.ProxyLatencyInfo{
		1: {Success: true, UpdatedAt: now, QualityCheckedAt: &qualityChecked, QualityStatus: "healthy"},
		2: {Success: false, UpdatedAt: now},
	}}
	healthService := service.NewProxyHealthService(adminSvc, repo, cache, nil, nil, nil, nil)
	handler := NewProxyHandler(adminSvc)
	handler.SetProxyHealthService(healthService)
	router := gin.New()
	router.GET("/proxies", handler.List)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/proxies?health=failed&page=1&page_size=1", nil)
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var envelope struct {
		Data struct {
			Items []struct {
				ID            int64  `json:"id"`
				QualityStatus string `json:"quality_status"`
			} `json:"items"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
	require.Equal(t, int64(1), envelope.Data.Total)
	require.Len(t, envelope.Data.Items, 1)
	require.Equal(t, int64(2), envelope.Data.Items[0].ID)
	require.Equal(t, string(service.ProxyHealthFailed), envelope.Data.Items[0].QualityStatus)
}
