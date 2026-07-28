package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type DesktopDiagnosticHandler struct{ service *desktop.Service }

func NewDesktopDiagnosticHandler(service *desktop.Service) *DesktopDiagnosticHandler {
	return &DesktopDiagnosticHandler{service: service}
}

func (h *DesktopDiagnosticHandler) List(c *gin.Context) {
	if !requireDesktopDiagnosticAdmin(c) {
		return
	}
	page, pageSize := response.ParsePagination(c)
	if pageSize > 200 {
		pageSize = 200
	}
	result, err := h.service.ListDiagnostics(c.Request.Context(), page, pageSize)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Paginated(c, result.Items, result.Total, page, pageSize)
}

func (h *DesktopDiagnosticHandler) Get(c *gin.Context) {
	if !requireDesktopDiagnosticAdmin(c) {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	middleware.SetAuditAction(c, "admin.desktop_diagnostics.view")
	item, err := h.service.GetDiagnostic(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}

func (h *DesktopDiagnosticHandler) Download(c *gin.Context) {
	if !requireDesktopDiagnosticAdmin(c) {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	middleware.SetAuditAction(c, "admin.desktop_diagnostics.download")
	item, err := h.service.GetDiagnostic(c.Request.Context(), strings.TrimSpace(c.Param("id")))
	if response.ErrorFrom(c, err) {
		return
	}
	payload, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		response.ErrorFrom(c, desktop.ErrDiagnosticsUnavailable.WithCause(err))
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="desktop-diagnostic-%s.json"`, item.ID))
	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}

func requireDesktopDiagnosticAdmin(c *gin.Context) bool {
	role, ok := middleware.GetUserRoleFromContext(c)
	if !ok || role != "admin" {
		response.ErrorWithDetails(c, http.StatusForbidden, "Administrator access required", "ADMIN_ACCESS_REQUIRED", nil)
		return false
	}
	return true
}
