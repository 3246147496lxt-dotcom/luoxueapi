package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type DesktopHandler struct {
	service *desktop.Service
}

func NewDesktopHandler(service *desktop.Service) *DesktopHandler {
	return &DesktopHandler{service: service}
}

type createDesktopPairingRequest struct {
	CodeChallenge       string `json:"code_challenge" binding:"required"`
	CodeChallengeMethod string `json:"code_challenge_method" binding:"required"`
	InstallationID      string `json:"installation_id" binding:"required"`
	DeviceName          string `json:"device_name" binding:"required"`
	Platform            string `json:"platform" binding:"required"`
	Architecture        string `json:"architecture" binding:"required"`
	OSVersion           string `json:"os_version"`
	AppVersion          string `json:"app_version"`
}

type exchangeDesktopPairingRequest struct {
	DeviceCode   string `json:"device_code" binding:"required"`
	CodeVerifier string `json:"code_verifier" binding:"required"`
}

type refreshDesktopSessionRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type renameDesktopDeviceRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *DesktopHandler) CreatePairing(c *gin.Context) {
	var req createDesktopPairingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid desktop pairing request")
		return
	}
	result, err := h.service.CreatePairing(c.Request.Context(), desktop.CreatePairingInput{
		CodeChallenge: req.CodeChallenge, CodeChallengeMethod: req.CodeChallengeMethod,
		InstallationID: req.InstallationID, DeviceName: req.DeviceName, Platform: req.Platform,
		Architecture: req.Architecture, OSVersion: req.OSVersion, AppVersion: req.AppVersion,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, result)
}

func (h *DesktopHandler) PairingPreview(c *gin.Context) {
	preview, err := h.service.GetPairingPreview(c.Request.Context(), c.Param("user_code"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, preview)
}

func (h *DesktopHandler) ApprovePairing(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	middleware.SetAuditAction(c, "desktop.pairing.approve")
	device, err := h.service.ApprovePairing(c.Request.Context(), subject.UserID, c.Param("user_code"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, device)
}

func (h *DesktopHandler) ExchangePairing(c *gin.Context) {
	var req exchangeDesktopPairingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid desktop token exchange request")
		return
	}
	tokens, err := h.service.ExchangePairing(c.Request.Context(), strings.TrimSpace(req.DeviceCode), strings.TrimSpace(req.CodeVerifier))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, tokens)
}

func (h *DesktopHandler) RefreshSession(c *gin.Context) {
	var req refreshDesktopSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid desktop refresh request")
		return
	}
	tokens, err := h.service.RefreshSession(c.Request.Context(), strings.TrimSpace(req.RefreshToken))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, tokens)
}

func (h *DesktopHandler) Me(c *gin.Context) {
	subject, ok := middleware.GetDesktopSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Desktop device not authenticated")
		return
	}
	device, err := h.service.GetDevice(c.Request.Context(), subject)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, device)
}

func (h *DesktopHandler) ListDevices(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	devices, err := h.service.ListDevices(c.Request.Context(), subject.UserID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, devices)
}

func (h *DesktopHandler) RenameDevice(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req renameDesktopDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid desktop device rename request")
		return
	}
	middleware.SetAuditAction(c, "desktop.device.rename")
	device, err := h.service.RenameDevice(c.Request.Context(), subject.UserID, c.Param("device_id"), req.Name)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, device)
}

func (h *DesktopHandler) RevokeDevice(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	middleware.SetAuditAction(c, "desktop.device.revoke")
	device, err := h.service.RevokeDevice(c.Request.Context(), subject.UserID, c.Param("device_id"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, device)
}

func (h *DesktopHandler) Heartbeat(c *gin.Context) {
	subject, ok := middleware.GetDesktopSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Desktop device not authenticated")
		return
	}
	device, err := h.service.Heartbeat(c.Request.Context(), subject)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, device)
}

func (h *DesktopHandler) Activate(c *gin.Context) {
	subject, ok := middleware.GetDesktopSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Desktop device not authenticated")
		return
	}
	device, err := h.service.ActivateCurrentDevice(c.Request.Context(), subject)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, device)
}

func (h *DesktopHandler) RevokeCurrentDevice(c *gin.Context) {
	subject, ok := middleware.GetDesktopSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Desktop device not authenticated")
		return
	}
	device, err := h.service.RevokeCurrentDevice(c.Request.Context(), subject)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, device)
}

func (h *DesktopHandler) Routes(c *gin.Context) {
	subject, ok := middleware.GetDesktopSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Desktop device not authenticated")
		return
	}
	routes, err := h.service.ListRoutes(c.Request.Context(), subject)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, routes)
}

func (h *DesktopHandler) TodayUsage(c *gin.Context) {
	subject, ok := middleware.GetDesktopSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Desktop device not authenticated")
		return
	}
	usage, err := h.service.GetTodayUsage(c.Request.Context(), subject)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, usage)
}

func (h *DesktopHandler) Release(c *gin.Context) {
	subject, ok := middleware.GetDesktopSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Desktop device not authenticated")
		return
	}
	manifest, err := h.service.FindRelease(c.Request.Context(), subject, c.Param("target"), c.Param("arch"), c.Param("current_version"))
	if response.ErrorFrom(c, err) {
		return
	}
	if manifest == nil {
		c.Status(http.StatusNoContent)
		c.Writer.WriteHeaderNow()
		return
	}
	c.JSON(http.StatusOK, manifest)
}

func (h *DesktopHandler) PublicLatestRelease(c *gin.Context) {
	manifest := h.service.PublicLatestRelease()
	if manifest == nil {
		c.Status(http.StatusNoContent)
		c.Writer.WriteHeaderNow()
		return
	}
	c.JSON(http.StatusOK, manifest)
}

func (h *DesktopHandler) PublicLatestReleaseDownload(c *gin.Context) {
	manifest := h.service.PublicLatestRelease()
	if manifest == nil {
		c.Status(http.StatusNotFound)
		c.Writer.WriteHeaderNow()
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, manifest.InstallerURL)
}

func (h *DesktopHandler) UploadDiagnostic(c *gin.Context) {
	subject, ok := middleware.GetDesktopSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Desktop device not authenticated")
		return
	}
	var upload desktop.DiagnosticUpload
	if err := decodeStrictDesktopJSON(c, &upload, desktop.DiagnosticRequestMaxBytes); err != nil {
		response.ErrorFrom(c, desktop.ErrDiagnosticsInvalid.WithCause(err))
		return
	}
	created, err := h.service.UploadDiagnostic(c.Request.Context(), subject, upload)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, created)
}

func decodeStrictDesktopJSON(c *gin.Context, dst any, limit int64) error {
	if c.Request == nil || c.Request.Body == nil {
		return errors.New("request body is required")
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func (h *DesktopHandler) EnsureManagedKey(c *gin.Context) {
	subject, ok := middleware.GetDesktopSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Desktop device not authenticated")
		return
	}
	groupID, err := strconv.ParseInt(c.Param("group_id"), 10, 64)
	if err != nil || groupID <= 0 {
		response.BadRequest(c, "Invalid group ID")
		return
	}
	managed, err := h.service.EnsureManagedKey(c.Request.Context(), subject, groupID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, managed)
}
