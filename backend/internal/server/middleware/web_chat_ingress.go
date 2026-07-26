package middleware

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
)

// WebChatIngress marks the route scope before JWT authentication runs.
// It is intentionally distinct from ctxkey.WebChat, which is trusted only
// after the platform-managed chat principal has been bound.
func WebChatIngress() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request != nil {
			ctx := context.WithValue(c.Request.Context(), ctxkey.WebChatIngress, true)
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}
