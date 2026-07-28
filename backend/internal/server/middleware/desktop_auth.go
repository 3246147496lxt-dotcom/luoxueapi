package middleware

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

const desktopSubjectContextKey = "desktop_subject"

type DesktopAuthMiddleware gin.HandlerFunc

func NewDesktopAuthMiddleware(service *desktop.Service) DesktopAuthMiddleware {
	return DesktopAuthMiddleware(func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			abortDesktopError(c, 401, "DESKTOP_AUTH_REQUIRED", "Desktop bearer token is required")
			return
		}
		subject, err := service.AuthenticateAccessToken(c.Request.Context(), strings.TrimSpace(parts[1]))
		if err != nil {
			abortDesktopError(c, 401, "DESKTOP_TOKEN_INVALID", "Invalid desktop access token")
			return
		}
		c.Set(desktopSubjectContextKey, *subject)
		c.Next()
	})
}

func GetDesktopSubjectFromContext(c *gin.Context) (desktop.DesktopSubject, bool) {
	value, ok := c.Get(desktopSubjectContextKey)
	if !ok {
		return desktop.DesktopSubject{}, false
	}
	subject, ok := value.(desktop.DesktopSubject)
	return subject, ok
}

func RequireDesktopScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		subject, ok := GetDesktopSubjectFromContext(c)
		if !ok {
			abortDesktopError(c, 401, "DESKTOP_AUTH_REQUIRED", "Desktop authentication required")
			return
		}
		if !subject.HasScope(scope) {
			abortDesktopError(c, 403, "DESKTOP_SCOPE_FORBIDDEN", "Desktop token does not grant the required scope")
			return
		}
		c.Next()
	}
}

func abortDesktopError(c *gin.Context, status int, reason, message string) {
	response.ErrorWithDetails(c, status, message, reason, nil)
	c.Abort()
}
