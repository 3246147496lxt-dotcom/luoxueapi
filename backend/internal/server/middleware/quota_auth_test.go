package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type stubQuotaAuthenticator struct {
	subject *quotaauth.Subject
	err     error
	calls   int
	token   string
}

func (s *stubQuotaAuthenticator) AuthenticateAccessToken(
	_ context.Context,
	tokenString string,
) (*quotaauth.Subject, error) {
	s.calls++
	s.token = tokenString
	return s.subject, s.err
}

func TestQuotaAuthRequiresDedicatedBearerAndIgnoresCookieAndAPIKeyHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authenticator := new(stubQuotaAuthenticator)
	router := gin.New()
	router.GET(
		"/api/v1/quota/overview",
		gin.HandlerFunc(newQuotaAuthMiddleware(authenticator)),
		RequireQuotaRead(),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	tests := []struct {
		name    string
		prepare func(*http.Request)
	}{
		{
			name: "cookie only",
			prepare: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "session", Value: "website-jwt"})
			},
		},
		{
			name: "x api key only",
			prepare: func(req *http.Request) {
				req.Header.Set("x-api-key", "sk-consumer-key")
			},
		},
		{
			name: "malformed bearer",
			prepare: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer ")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/quota/overview", nil)
			test.prepare(req)
			router.ServeHTTP(recorder, req)
			require.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}
	require.Zero(t, authenticator.calls, "non-Bearer credentials must never reach quota token validation")
}

func TestQuotaAuthPreservesAudienceScopeAndBackendFailureStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "wrong audience", err: quotaauth.ErrAccessTokenAudience, wantStatus: http.StatusForbidden},
		{name: "missing scope", err: quotaauth.ErrAccessTokenScope, wantStatus: http.StatusForbidden},
		{name: "expired", err: quotaauth.ErrAccessTokenExpired, wantStatus: http.StatusUnauthorized},
		{name: "revoked device", err: quotaauth.ErrDeviceUnauthorized, wantStatus: http.StatusUnauthorized},
		{name: "database unavailable", err: quotaauth.ErrServiceUnavailable.WithCause(errors.New("db down")), wantStatus: http.StatusServiceUnavailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authenticator := &stubQuotaAuthenticator{err: test.err}
			router := gin.New()
			router.GET(
				"/api/v1/quota/overview",
				gin.HandlerFunc(newQuotaAuthMiddleware(authenticator)),
				func(c *gin.Context) { c.Status(http.StatusOK) },
			)
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/quota/overview", nil)
			req.Header.Set("Authorization", "Bearer quota-token")
			router.ServeHTTP(recorder, req)
			require.Equal(t, test.wantStatus, recorder.Code)
		})
	}
}

func TestQuotaAuthPublishesOnlyExactReadSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		subject    quotaauth.Subject
		wantStatus int
	}{
		{
			name: "exact quota read",
			subject: quotaauth.Subject{
				DeviceID: 41,
				UserID:   7,
				ClientID: quotaauth.ClientID,
				Scopes:   []string{quotaauth.ScopeRead},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "desktop client",
			subject: quotaauth.Subject{
				DeviceID: 41,
				UserID:   7,
				ClientID: "luoxue-desktop",
				Scopes:   []string{quotaauth.ScopeRead},
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "write scope only",
			subject: quotaauth.Subject{
				DeviceID: 41,
				UserID:   7,
				ClientID: quotaauth.ClientID,
				Scopes:   []string{"managed_keys:write"},
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authenticator := &stubQuotaAuthenticator{subject: &test.subject}
			router := gin.New()
			router.GET(
				"/api/v1/quota/overview",
				gin.HandlerFunc(newQuotaAuthMiddleware(authenticator)),
				RequireQuotaRead(),
				func(c *gin.Context) {
					subject, ok := GetQuotaAuthSubjectFromContext(c)
					require.True(t, ok)
					require.Equal(t, int64(7), subject.UserID)
					c.Status(http.StatusOK)
				},
			)
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/quota/overview", nil)
			req.Header.Set("Authorization", "Bearer dedicated-token")
			router.ServeHTTP(recorder, req)
			require.Equal(t, test.wantStatus, recorder.Code)
			require.Equal(t, "dedicated-token", authenticator.token)
		})
	}
}
