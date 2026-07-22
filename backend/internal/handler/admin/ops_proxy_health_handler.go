package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// GetProxyHealth lists safe proxy/IP health projections.
// GET /api/v1/admin/ops/proxy-health
func (h *OpsHandler) GetProxyHealth(c *gin.Context) {
	if h.proxyHealthService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Proxy health service not available")
		return
	}
	result, err := h.proxyHealthService.List(c.Request.Context(), service.ProxyHealthFilter{
		Health:    strings.TrimSpace(c.Query("health")),
		Lifecycle: strings.TrimSpace(c.Query("lifecycle")),
		Protocol:  strings.TrimSpace(c.Query("protocol")),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetProxyHealthDetail returns a safe health projection for one proxy.
// GET /api/v1/admin/ops/proxy-health/:id
func (h *OpsHandler) GetProxyHealthDetail(c *gin.Context) {
	id, ok := parseProxyHealthID(c)
	if !ok {
		return
	}
	if h.proxyHealthService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Proxy health service not available")
		return
	}
	result, err := h.proxyHealthService.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ReprobeProxyHealth triggers a non-mutating complete quality probe.
// POST /api/v1/admin/ops/proxy-health/:id/reprobe
func (h *OpsHandler) ReprobeProxyHealth(c *gin.Context) {
	id, ok := parseProxyHealthID(c)
	if !ok {
		return
	}
	if h.proxyHealthService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Proxy health service not available")
		return
	}
	result, err := h.proxyHealthService.ProbeNow(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func parseProxyHealthID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid proxy ID")
		return 0, false
	}
	return id, true
}
