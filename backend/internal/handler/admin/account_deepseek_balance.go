package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// QueryDeepSeekBalance probes the configured API-key account's upstream
// DeepSeek balance. The endpoint is a manual refresh; the optional periodic
// pausing/recovery checker reuses the same service underneath.
// GET /api/v1/admin/accounts/:id/balance
func (h *AccountHandler) QueryDeepSeekBalance(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	if h == nil || h.deepSeekBalanceService == nil {
		response.BadRequest(c, "DeepSeek balance service is not enabled")
		return
	}
	result, err := h.deepSeekBalanceService.QueryBalance(c.Request.Context(), accountID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// QueryCNProviderBalance is an alias used by the provider-oriented route. It
// keeps the public endpoint naming compatible with the upstream project while
// sharing the account handler's existing dependency surface.
func (h *AccountHandler) QueryCNProviderBalance(c *gin.Context) {
	h.QueryDeepSeekBalance(c)
}
