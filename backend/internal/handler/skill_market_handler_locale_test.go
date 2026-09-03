package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSkillMarketRequestLocalePrefersQueryOverHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("GET", "/api/v1/catalog/skills?lang=zh-CN", nil)
	context.Request.Header.Set("Accept-Language", "en-US")

	require.Equal(t, "zh-CN", skillMarketRequestLocale(context))
}

func TestSkillMarketRequestLocaleFallsBackToHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("GET", "/api/v1/catalog/skills", nil)
	context.Request.Header.Set("Accept-Language", "zh-Hans")

	require.Equal(t, "zh-Hans", skillMarketRequestLocale(context))
}
