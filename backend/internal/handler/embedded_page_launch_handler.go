package handler

import (
	"net"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type EmbeddedPageLaunchHandler struct {
	service *service.EmbeddedPageLaunchService
}

func NewEmbeddedPageLaunchHandler(service *service.EmbeddedPageLaunchService) *EmbeddedPageLaunchHandler {
	return &EmbeddedPageLaunchHandler{service: service}
}

type launchEmbeddedPageRequest struct {
	Theme  string `json:"theme"`
	Lang   string `json:"lang"`
	UIMode string `json:"ui_mode"`
}

type exchangeEmbeddedPageRequest struct {
	ClientID string `json:"client_id"`
	Code     string `json:"code"`
}

// Launch issues a 60-second, one-time code for a single custom-page opening.
// POST /api/v1/user/custom-pages/:id/launch
func (h *EmbeddedPageLaunchHandler) Launch(c *gin.Context) {
	setEmbeddedPageNoStoreHeaders(c)

	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req launchEmbeddedPageRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid launch request")
			return
		}
	}
	role, _ := servermiddleware.GetUserRoleFromContext(c)
	result, err := h.service.Issue(
		c.Request.Context(),
		subject.UserID,
		role,
		c.Param("id"),
		service.EmbeddedPageLaunchOptions{
			Theme:      req.Theme,
			Lang:       req.Lang,
			UIMode:     req.UIMode,
			SourceHost: embeddedPageSourceOrigin(c),
		},
	)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

// Exchange atomically consumes a launch code. It intentionally rejects browser
// Origin requests: external pages must exchange from their own backend.
// POST /api/v1/embedded-pages/exchange
func (h *EmbeddedPageLaunchHandler) Exchange(c *gin.Context) {
	setEmbeddedPageNoStoreHeaders(c)
	clearEmbeddedPageCORSHeaders(c)
	if strings.TrimSpace(c.GetHeader("Origin")) != "" {
		response.ErrorWithDetails(c, http.StatusForbidden, "browser exchange is not allowed", "EMBEDDED_PAGE_SERVER_EXCHANGE_REQUIRED", nil)
		return
	}

	var req exchangeEmbeddedPageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrInvalidEmbedLaunchCode)
		return
	}
	result, err := h.service.Exchange(c.Request.Context(), req.ClientID, req.Code)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func setEmbeddedPageNoStoreHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Referrer-Policy", "no-referrer")
}

func clearEmbeddedPageCORSHeaders(c *gin.Context) {
	for _, header := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
		"Access-Control-Expose-Headers",
		"Access-Control-Max-Age",
	} {
		c.Writer.Header().Del(header)
	}
}

func embeddedPageSourceOrigin(c *gin.Context) string {
	if origin := strings.TrimSpace(c.GetHeader("Origin")); origin != "" {
		return origin
	}

	proto := "http"
	if c.Request.TLS != nil {
		proto = "https"
	} else if embeddedPageTrustsForwardedHeaders(c) {
		forwardedProto := strings.ToLower(firstForwardedValue(c.GetHeader("X-Forwarded-Proto")))
		if forwardedProto == "http" || forwardedProto == "https" {
			proto = forwardedProto
		}
	}

	// Reverse proxies preserve the request Host by default. Do not accept
	// X-Forwarded-Host here: unlike Host, it is not part of the authority that
	// selected this route and is easy for an untrusted peer to spoof.
	host := c.Request.Host
	if strings.TrimSpace(host) == "" {
		return ""
	}
	return proto + "://" + host
}

func embeddedPageTrustsForwardedHeaders(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	remoteIP := net.ParseIP(c.RemoteIP())
	if remoteIP == nil {
		return false
	}
	// The documented Caddy deployment proxies over localhost. Treat the local
	// peer as trusted so its HTTPS scheme survives the hop even when the global
	// trusted_proxies list is intentionally empty.
	if remoteIP.IsLoopback() {
		return true
	}

	// For non-local proxies, Gin only returns a different ClientIP after the
	// direct peer passes Engine.SetTrustedProxies and its forwarded chain parses.
	clientIP := net.ParseIP(c.ClientIP())
	return clientIP != nil && !clientIP.Equal(remoteIP)
}

func firstForwardedValue(value string) string {
	if before, _, ok := strings.Cut(value, ","); ok {
		return strings.TrimSpace(before)
	}
	return strings.TrimSpace(value)
}
