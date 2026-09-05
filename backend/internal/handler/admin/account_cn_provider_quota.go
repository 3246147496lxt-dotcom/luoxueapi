package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// QueryCNProviderQuota probes a Zhipu GLM Coding Plan account and returns the
// current 5-hour/weekly rolling windows. The service persists a sanitized
// snapshot in account.extra for scheduling and UI display.
// GET /api/v1/admin/cn-providers/accounts/:id/quota
func (h *AccountHandler) QueryCNProviderQuota(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if h == nil || h.cnProviderQuotaService == nil {
		response.BadRequest(c, "cn provider quota service is not enabled")
		return
	}
	result, err := h.cnProviderQuotaService.QueryUsage(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// QueryZhipuQuota is a semantic alias for callers that expose provider-specific
// routes while retaining the shared CN provider implementation.
func (h *AccountHandler) QueryZhipuQuota(c *gin.Context) {
	h.QueryCNProviderQuota(c)
}
