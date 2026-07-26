package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNewStreamHeaderWriterPreservesExistingCacheControl(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Header("Cache-Control", "private, no-store")
	upstream := http.Header{
		"Cache-Control": []string{"public, max-age=3600", "no-cache"},
	}

	service := &OpenAIGatewayService{
		responseHeaderFilter: responseheaders.CompileHeaderFilter(config.ResponseHeaderConfig{}),
	}
	service.newStreamHeaderWriter(c, upstream)()

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, []string{"private, no-store"}, recorder.Header().Values("Cache-Control"))
}

func TestNewStreamHeaderWriterDefaultsCacheControl(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	(&OpenAIGatewayService{}).newStreamHeaderWriter(c, nil)()

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "no-cache", recorder.Header().Get("Cache-Control"))
}

func TestNewStreamHeaderWriterOverridesUpstreamCacheControlForOrdinaryGateway(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	upstream := http.Header{
		"Cache-Control": []string{"public, max-age=3600"},
	}

	service := &OpenAIGatewayService{
		responseHeaderFilter: responseheaders.CompileHeaderFilter(config.ResponseHeaderConfig{}),
	}
	service.newStreamHeaderWriter(c, upstream)()

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, []string{"no-cache"}, recorder.Header().Values("Cache-Control"))
}
