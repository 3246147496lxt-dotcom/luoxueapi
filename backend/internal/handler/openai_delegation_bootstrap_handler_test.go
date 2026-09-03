package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponses_DelegationBootstrapPassesToolOutputValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const envelope = `<codex_delegation><source_thread_id>thread-parent</source_thread_id><input>inspect the request</input></codex_delegation>`
	tests := []struct {
		name             string
		body             string
		wantStatus       int
		wantAcquireCalls int
	}{
		{
			name:             "delegation bootstrap reaches concurrency acquisition",
			body:             `{"model":"gpt-5.1","input":[{"type":"function_call_output","namespace":"codex_app","name":"create_thread","output":"` + envelope + `"}]}`,
			wantStatus:       http.StatusServiceUnavailable,
			wantAcquireCalls: 1,
		},
		{
			name:             "ordinary tool output still requires call id",
			body:             `{"model":"gpt-5.1","input":[{"type":"function_call_output","output":"{}"}]}`,
			wantStatus:       http.StatusBadRequest,
			wantAcquireCalls: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acquireCalls := 0
			cache := &concurrencyCacheMock{
				acquireUserSlotFn: func(context.Context, int64, int, string) (bool, error) {
					acquireCalls++
					// Stop at the first dependency after validation. Asserting this
					// call prevents an unrelated early 503 or panic from passing.
					return false, errors.New("stop after request validation")
				},
			}
			h := newOpenAIHandlerForPreviousResponseIDValidation(t, cache)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			groupID := int64(2)
			c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
				ID:      101,
				GroupID: &groupID,
				User:    &service.User{ID: 1},
			})
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1, Concurrency: 1})

			h.Responses(c)

			require.Equal(t, tt.wantAcquireCalls, acquireCalls, w.Body.String())
			require.Equal(t, tt.wantStatus, w.Code, w.Body.String())
			if tt.wantAcquireCalls > 0 {
				require.NotContains(t, w.Body.String(), "function_call_output requires call_id")
			} else {
				require.Contains(t, w.Body.String(), "function_call_output requires call_id")
			}
		})
	}
}
