package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSettingHandlerUpdateSettingsValidatesCustomMenuAuthMode(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		authMode string
	}{
		{name: "unknown mode", url: "https://page.example.com", authMode: "jwt"},
		{name: "exchange requires https", url: "http://page.example.com", authMode: service.CustomMenuAuthModeExchangeCode},
		{name: "markdown cannot exchange", url: "md:terms", authMode: service.CustomMenuAuthModeExchangeCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			repo := &settingHandlerRepoStub{values: map[string]string{service.SettingKeyPromoCodeEnabled: "true"}}
			h := NewSettingHandler(service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}}), nil, nil, nil, nil, nil, nil)
			body, err := json.Marshal(map[string]any{
				"promo_code_enabled": true,
				"custom_menu_items": []map[string]any{{
					"id": "payment", "label": "Payment", "url": tt.url,
					"visibility": "user", "sort_order": 1, "auth_mode": tt.authMode,
				}},
			})
			require.NoError(t, err)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			h.UpdateSettings(c)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.NotContains(t, repo.values, service.SettingKeyCustomMenuItems)
		})
	}
}

func TestSettingHandlerUpdateSettingsNormalizesLegacyCustomMenuAuthMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &settingHandlerRepoStub{values: map[string]string{service.SettingKeyPromoCodeEnabled: "true"}}
	h := NewSettingHandler(service.NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}}), nil, nil, nil, nil, nil, nil)
	body, err := json.Marshal(map[string]any{
		"promo_code_enabled": true,
		"custom_menu_items": []map[string]any{
			{
				"id": "legacy", "label": "Legacy", "url": "https://page.example.com",
				"visibility": "user", "sort_order": 1,
			},
			{
				"id": "payment", "label": "Payment", "url": "https://pay.example.com",
				"visibility": "user", "sort_order": 2, "auth_mode": service.CustomMenuAuthModeExchangeCode,
			},
		},
	})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.UpdateSettings(c)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var stored []dto.CustomMenuItem
	require.NoError(t, json.Unmarshal([]byte(repo.values[service.SettingKeyCustomMenuItems]), &stored))
	require.Len(t, stored, 2)
	require.Equal(t, service.CustomMenuAuthModeNone, stored[0].AuthMode)
	require.Equal(t, service.CustomMenuAuthModeExchangeCode, stored[1].AuthMode)
}
