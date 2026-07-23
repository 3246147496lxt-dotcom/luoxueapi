package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
	"github.com/gin-gonic/gin"
)

func TestSetupRouterHealthSemantics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := newSetupRouter()

	for _, path := range []string{"/health", "/livez"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("%s code = %d, want 200", path, response.Code)
			}
			if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("%s content-type = %q", path, response.Header().Get("Content-Type"))
			}
		})
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz code = %d, want 503", response.Code)
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("readyz content-type = %q", response.Header().Get("Content-Type"))
	}

	var result lifecycle.ReadinessResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode readyz response: %v", err)
	}
	if result.Status != lifecycle.ReadinessNotReady {
		t.Fatalf("readyz status = %q, want %q", result.Status, lifecycle.ReadinessNotReady)
	}
}
