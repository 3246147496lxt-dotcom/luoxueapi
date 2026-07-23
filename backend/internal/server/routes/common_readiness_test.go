package routes

import (
	"context"
	"encoding/json"
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
	if liveResponse.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("livez content-type = %q", liveResponse.Header().Get("Content-Type"))
	}

	readyResponse := httptest.NewRecorder()
	router.ServeHTTP(readyResponse, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if readyResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz code = %d, want 503", readyResponse.Code)
	}
	if readyResponse.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("readyz cache-control = %q", readyResponse.Header().Get("Cache-Control"))
	}
	if readyResponse.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("readyz content-type = %q", readyResponse.Header().Get("Content-Type"))
	}
	var result lifecycle.ReadinessResult
	if err := json.Unmarshal(readyResponse.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode readyz response: %v", err)
	}
	if result.Status != lifecycle.ReadinessNotReady {
		t.Fatalf("readyz status = %q, want %q", result.Status, lifecycle.ReadinessNotReady)
	}
}

func TestReadinessStatusMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		status     string
		statusCode int
	}{
		{name: "ready", status: lifecycle.ReadinessReady, statusCode: http.StatusOK},
		{name: "degraded", status: lifecycle.ReadinessDegraded, statusCode: http.StatusOK},
		{name: "not_ready", status: lifecycle.ReadinessNotReady, statusCode: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			RegisterHealthRoutes(router, staticReadinessProbe{result: lifecycle.ReadinessResult{
				Status: tt.status,
				Checks: map[string]lifecycle.CheckResult{
					"database": {Status: tt.status},
				},
			}})

			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if response.Code != tt.statusCode {
				t.Fatalf("readyz code = %d, want %d", response.Code, tt.statusCode)
			}

			var result lifecycle.ReadinessResult
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatalf("decode readyz response: %v", err)
			}
			if result.Status != tt.status {
				t.Fatalf("readyz status = %q, want %q", result.Status, tt.status)
			}
		})
	}
}

func TestReadinessWithoutProbeFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHealthRoutes(router, nil)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz code = %d, want 503", response.Code)
	}

	var result lifecycle.ReadinessResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode readyz response: %v", err)
	}
	if result.Status != lifecycle.ReadinessNotReady || result.Checks["probe"].Status != lifecycle.ReadinessNotReady {
		t.Fatalf("unexpected readyz result: %+v", result)
	}
}
