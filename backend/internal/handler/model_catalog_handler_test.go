package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type modelCatalogHandlerSettingRepoStub struct {
	service.SettingRepository
	values map[string]string
}

func (s *modelCatalogHandlerSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

type modelCatalogHandlerRepoStub struct {
	service.ModelCatalogRepository
	models []service.ModelCatalogModel
}

func (s *modelCatalogHandlerRepoStub) ListPublished(context.Context) ([]service.ModelCatalogModel, error) {
	return append([]service.ModelCatalogModel(nil), s.models...), nil
}

type modelCatalogHandlerChannelRepoStub struct {
	service.ChannelRepository
	channels []service.Channel
}

func (s *modelCatalogHandlerChannelRepoStub) ListAll(context.Context) ([]service.Channel, error) {
	return append([]service.Channel(nil), s.channels...), nil
}

type modelCatalogHandlerGroupRepoStub struct {
	service.GroupRepository
	groups []service.Group
}

func (s *modelCatalogHandlerGroupRepoStub) ListActive(context.Context) ([]service.Group, error) {
	return append([]service.Group(nil), s.groups...), nil
}

func TestRequestETagMatches(t *testing.T) {
	etag := `"catalog-etag"`
	tests := []struct {
		name   string
		header string
		etag   string
		want   bool
	}{
		{name: "exact", header: etag, etag: etag, want: true},
		{name: "weak", header: `W/"catalog-etag"`, etag: etag, want: true},
		{name: "list", header: `"other", W/"catalog-etag"`, etag: etag, want: true},
		{name: "wildcard", header: "*", etag: etag, want: true},
		{name: "mismatch", header: `"other"`, etag: etag, want: false},
		{name: "empty current etag", header: "*", etag: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, requestETagMatches(tt.header, tt.etag))
		})
	}
}

func TestModelCatalogHandlerListReturnsEmpty304WithCacheHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	groupID := int64(7)
	inputPrice := 0.000001
	catalogService := service.NewModelCatalogService(
		&modelCatalogHandlerRepoStub{models: []service.ModelCatalogModel{{
			ID: 1, Slug: "openai-test", Model: "test-model", Platform: service.PlatformOpenAI,
			DisplayNameZH: "测试模型", SummaryZH: "公开模型", PublicGroupID: &groupID,
			Status: service.ModelCatalogStatusPublished, UpdatedAt: now,
		}}},
		&modelCatalogHandlerChannelRepoStub{channels: []service.Channel{{
			ID: 9, Status: service.StatusActive, GroupIDs: []int64{groupID}, UpdatedAt: now,
			ModelPricing: []service.ChannelModelPricing{{
				Platform: service.PlatformOpenAI, Models: []string{"test-model"}, InputPrice: &inputPrice,
			}},
		}}},
		&modelCatalogHandlerGroupRepoStub{groups: []service.Group{{
			ID: groupID, Name: "public", Platform: service.PlatformOpenAI, Status: service.StatusActive,
			SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1, UpdatedAt: now,
		}}},
		nil,
	)
	settingService := service.NewSettingService(&modelCatalogHandlerSettingRepoStub{values: map[string]string{
		service.SettingKeyPublicModelCatalogEnabled: "true",
	}}, nil)
	h := NewModelCatalogHandler(catalogService, settingService)
	router := gin.New()
	router.GET("/api/v1/catalog/models", h.RequireEnabled, h.List)

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v1/catalog/models", nil))
	require.Equal(t, http.StatusOK, first.Code)
	etag := first.Header().Get("ETag")
	require.NotEmpty(t, etag)

	secondRequest := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/models", nil)
	secondRequest.Header.Set("If-None-Match", `"other", W/`+etag)
	second := httptest.NewRecorder()
	router.ServeHTTP(second, secondRequest)

	require.Equal(t, http.StatusNotModified, second.Code)
	require.Empty(t, second.Body.Bytes())
	require.Equal(t, etag, second.Header().Get("ETag"))
	require.Equal(t, "public, max-age=30, must-revalidate", second.Header().Get("Cache-Control"))
	require.Contains(t, second.Header().Values("Vary"), "Accept-Language")
}

func TestModelCatalogRequireEnabledReturnsNoStore404BeforeNextHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name   string
		values map[string]string
	}{
		{name: "disabled", values: map[string]string{}},
		{name: "backend mode", values: map[string]string{
			service.SettingKeyPublicModelCatalogEnabled: "true",
			service.SettingKeyBackendModeEnabled:        "true",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settingService := service.NewSettingService(&modelCatalogHandlerSettingRepoStub{values: tt.values}, nil)
			h := NewModelCatalogHandler(&service.ModelCatalogService{}, settingService)
			reached := false
			router := gin.New()
			router.GET("/api/v1/catalog/models", h.RequireEnabled, func(c *gin.Context) {
				reached = true
				c.Status(http.StatusNoContent)
			})

			for range 2 {
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/catalog/models", nil))
				require.Equal(t, http.StatusNotFound, recorder.Code)
				require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
				require.False(t, reached)
				require.True(t, strings.Contains(recorder.Body.String(), "Not found"))
			}
		})
	}
}
