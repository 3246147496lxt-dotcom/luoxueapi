package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const QuotaViewerInstallerDownloadAuditAction = "quota_viewer.installer.download"

type QuotaViewerInstallerHandler struct {
	service *service.QuotaViewerInstallerDownloadService
}

func NewQuotaViewerInstallerHandler(service *service.QuotaViewerInstallerDownloadService) *QuotaViewerInstallerHandler {
	return &QuotaViewerInstallerHandler{service: service}
}

// IssueDownload creates a 60-second, single-use same-origin download path for
// any authenticated website user.
func (h *QuotaViewerInstallerHandler) IssueDownload(c *gin.Context) {
	setQuotaViewerInstallerNoStoreHeaders(c)
	middleware.SetAuditAction(c, QuotaViewerInstallerDownloadAuditAction)

	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "authentication required")
		return
	}
	if h == nil || h.service == nil {
		response.ErrorFrom(c, service.ErrQuotaViewerInstallerDownloadUnavailable)
		return
	}

	download, err := h.service.Issue(c.Request.Context(), subject.UserID, c.Param("platform"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, download)
}

// Download atomically consumes a one-time code before redirecting to the exact
// whitelisted GitHub release asset.
func (h *QuotaViewerInstallerHandler) Download(c *gin.Context) {
	setQuotaViewerInstallerNoStoreHeaders(c)
	if h == nil || h.service == nil {
		response.ErrorFrom(c, service.ErrQuotaViewerInstallerDownloadUnavailable)
		return
	}

	asset, err := h.service.Consume(c.Request.Context(), c.Param("platform"), c.Query("code"))
	if response.ErrorFrom(c, err) {
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, asset.URL)
}

func setQuotaViewerInstallerNoStoreHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("Vary", "Authorization")
}
