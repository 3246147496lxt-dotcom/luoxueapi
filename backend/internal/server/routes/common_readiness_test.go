package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
	"github.com/gin-gonic/gin"
)

type staticReadinessProbe struct {
	result lifecycle.ReadinessResult
}

func (p staticReadinessProbe) Probe(context.Context) lifecycle.ReadinessResult { return p.result }

func TestCommonHealthCompatibilityAndReadinessStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterCommonRoutesWithReadiness(router, staticReadinessProbe{result: lifecycle.ReadinessResult{
		Status: lifecycle.ReadinessNotReady,
	}})

	healthResponse := httptest.NewRecorder()
	router.ServeHTTP(healthResponse, httptest.NewRequest(http.MethodGet, "/health", nil))
	if healthResponse.Code != http.StatusOK || healthResponse.Body.String() != "{\"status\":\"ok\"}" {
		t.Fatalf("legacy health response changed: code=%d body=%q", healthResponse.Code, healthResponse.Body.String())
	}

	liveResponse := httptest.NewRecorder()
	router.ServeHTTP(liveResponse, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if liveResponse.Code != http.StatusOK {
		t.Fatalf("livez code = %d", liveResponse.Code)
	}

	readyResponse := httptest.NewRecorder()
	router.ServeHTTP(readyResponse, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if readyResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz code = %d, want 503", readyResponse.Code)
	}
	if readyResponse.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("readyz cache-control = %q", readyResponse.Header().Get("Cache-Control"))
	}
}
