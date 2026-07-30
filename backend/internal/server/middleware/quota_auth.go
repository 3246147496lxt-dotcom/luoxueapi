package middleware

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const quotaAuthSubjectContextKey = "quota_auth_subject"

type QuotaAuthMiddleware gin.HandlerFunc

type quotaAccessTokenAuthenticator interface {
	AuthenticateAccessToken(ctx context.Context, tokenString string) (*quotaauth.Subject, error)
}

func NewQuotaAuthMiddleware(service *quotaauth.Service) QuotaAuthMiddleware {
	return newQuotaAuthMiddleware(service)
}

func newQuotaAuthMiddleware(service quotaAccessTokenAuthenticator) QuotaAuthMiddleware {
	return QuotaAuthMiddleware(func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") ||
			strings.TrimSpace(parts[1]) == "" {
			abortQuotaAuthError(c, quotaauth.ErrAuthRequired)
			return
		}
		if service == nil {
			abortQuotaAuthError(c, quotaauth.ErrServiceUnavailable)
			return
		}
		subject, err := service.AuthenticateAccessToken(
			c.Request.Context(),
			strings.TrimSpace(parts[1]),
		)
		if err != nil {
			abortQuotaAuthError(c, err)
			return
		}
		if subject == nil {
			abortQuotaAuthError(c, quotaauth.ErrAccessTokenInvalid)
			return
		}
		c.Set(quotaAuthSubjectContextKey, *subject)
		c.Next()
	})
}

func GetQuotaAuthSubjectFromContext(c *gin.Context) (quotaauth.Subject, bool) {
	value, ok := c.Get(quotaAuthSubjectContextKey)
	if !ok {
		return quotaauth.Subject{}, false
	}
	subject, ok := value.(quotaauth.Subject)
	return subject, ok
}

func RequireQuotaScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		subject, ok := GetQuotaAuthSubjectFromContext(c)
		if !ok {
			abortQuotaAuthError(c, quotaauth.ErrAuthRequired)
			return
		}
		if subject.ClientID != quotaauth.ClientID || !subject.HasScope(scope) {
			abortQuotaAuthError(c, quotaauth.ErrAccessTokenScope)
			return
		}
		c.Next()
	}
}

func RequireQuotaRead() gin.HandlerFunc {
	return RequireQuotaScope(quotaauth.ScopeRead)
}

func abortQuotaAuthError(c *gin.Context, err error) {
	response.ErrorFrom(c, err)
	c.Abort()
}
