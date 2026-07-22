package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// GetAccountPool returns the read-only account-pool capacity snapshot.
// GET /api/v1/admin/ops/account-pool
//
// Query params:
// - platform: optional platform filter
// - group_id: optional positive group id
// - limit: optional anomaly-list limit (default 5, maximum 20)
func (h *OpsHandler) GetAccountPool(c *gin.Context) {
	if h.opsService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Ops service not available")
		return
	}
	if err := h.opsService.RequireMonitoringEnabled(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	platform := strings.TrimSpace(c.Query("platform"))
	var groupID *int64
	if raw := strings.TrimSpace(c.Query("group_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "Invalid group_id")
			return
		}
		groupID = &id
	}

	limit := service.DefaultOpsAccountPoolAnomalyLimit
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			response.BadRequest(c, "Invalid limit")
			return
		}
		if value > service.MaxOpsAccountPoolAnomalyLimit {
			value = service.MaxOpsAccountPoolAnomalyLimit
		}
		limit = value
	}

	data, err := h.opsService.GetAccountPool(c.Request.Context(), platform, groupID, limit)
	if err != nil {
		if isOpsRealtimeRequestCanceled(c, err) {
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, data)
}
