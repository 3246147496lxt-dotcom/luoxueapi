package admin

import (
	"io"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AccountHandler) accountHealth() *service.AccountHealthService {
	if h == nil {
		return service.NewAccountHealthService(nil, nil)
	}
	if h.accountHealthSvc != nil {
		return h.accountHealthSvc
	}
	svc := service.NewAccountHealthServiceFrom(h.adminService, h.accountTestService)
	svc.SetSettingsRepository(h.accountHealthSettingsRepo)
	if importer, ok := h.grokOAuthService.(service.AccountHealthGrokSSOImporter); ok {
		svc.SetGrokSSOImporter(importer)
	}
	h.accountHealthSvc = svc
	return svc
}

// GetAccountHealthSettings returns lifecycle policy and routing defaults.
func (h *AccountHandler) GetAccountHealthSettings(c *gin.Context) {
	cfg, err := h.accountHealth().GetSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

// UpdateAccountHealthSettings persists lifecycle policy and routing defaults.
func (h *AccountHandler) UpdateAccountHealthSettings(c *gin.Context) {
	var cfg service.AccountHealthSettings
	if !bindAccountHealthJSON(c, &cfg) {
		return
	}
	updated, err := h.accountHealth().UpdateSettings(c.Request.Context(), &cfg)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, updated)
}

// GetImportWatcherSettings returns registration-tool output discovery settings.
func (h *AccountHandler) GetImportWatcherSettings(c *gin.Context) {
	cfg, err := h.accountHealth().GetImportWatcherSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

// UpdateImportWatcherSettings persists registration-tool output discovery settings.
func (h *AccountHandler) UpdateImportWatcherSettings(c *gin.Context) {
	var cfg service.AccountImportWatcherSettings
	if !bindAccountHealthJSON(c, &cfg) {
		return
	}
	updated, err := h.accountHealth().UpdateImportWatcherSettings(c.Request.Context(), &cfg)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

// ScanImportWatcher discovers registration exports and never returns their contents.
func (h *AccountHandler) ScanImportWatcher(c *gin.Context) {
	result, err := h.accountHealth().ScanImportWatcher(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type importWatcherImportRequest struct {
	File        string  `json:"file"`
	Indexes     []int   `json:"indexes"`
	ProxyID     *int64  `json:"proxy_id,omitempty"`
	GroupIDs    []int64 `json:"group_ids,omitempty"`
	Confirm     string  `json:"confirm"`
	OperationID string  `json:"operation_id,omitempty"`
}

// ImportWatcherFile reads an export server-side and converts it through the
// existing account-health importer, keeping SSO credentials out of the UI.
func (h *AccountHandler) ImportWatcherFile(c *gin.Context) {
	var req importWatcherImportRequest
	if !bindAccountHealthJSON(c, &req) {
		return
	}
	result, err := h.accountHealth().ImportWatcherFile(c.Request.Context(), req.File, req.Indexes, req.ProxyID, req.GroupIDs, req.Confirm, req.OperationID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func bindAccountHealthJSON(c *gin.Context, dest any) bool {
	if err := c.ShouldBindJSON(dest); err != nil && err != io.EOF {
		response.BadRequest(c, "Invalid request body")
		return false
	}
	return true
}

// ScanAccountHealth lists expired/error accounts and optionally live-tests a subset.
// POST /api/v1/admin/accounts/health/scan
func (h *AccountHandler) ScanAccountHealth(c *gin.Context) {
	var req service.AccountHealthScanRequest
	if !bindAccountHealthJSON(c, &req) {
		return
	}
	result, err := h.accountHealth().Scan(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// CleanupAccountHealth deletes explicitly confirmed account IDs.
// POST /api/v1/admin/accounts/health/cleanup
func (h *AccountHandler) CleanupAccountHealth(c *gin.Context) {
	var req service.AccountHealthCleanupRequest
	if !bindAccountHealthJSON(c, &req) {
		return
	}
	result, err := h.accountHealth().Cleanup(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// AccountHealthAssistant handles constrained scan/test/cleanup/add chat turns.
// POST /api/v1/admin/accounts/health/assistant
func (h *AccountHandler) AccountHealthAssistant(c *gin.Context) {
	var req service.AccountHealthAssistantRequest
	if !bindAccountHealthJSON(c, &req) {
		return
	}
	if strings.TrimSpace(req.Intent) == "" && len(req.Messages) == 0 && req.PendingProposal == nil && strings.TrimSpace(req.Handoff) == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("ACCOUNT_HEALTH_CHAT_REQUIRED", "messages or intent is required"))
		return
	}
	result, err := h.accountHealth().Chat(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// AddAccountHealth creates confirmed API-key accounts from a credential handoff.
// POST /api/v1/admin/accounts/health/add
func (h *AccountHandler) AddAccountHealth(c *gin.Context) {
	var req service.AccountHealthAddRequest
	if !bindAccountHealthJSON(c, &req) {
		return
	}
	result, err := h.accountHealth().Add(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
