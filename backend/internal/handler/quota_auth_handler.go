package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const quotaAuthRequestBodyLimit int64 = 64 << 10

type QuotaAuthHandler struct {
	service *quotaauth.Service
}

func NewQuotaAuthHandler(service *quotaauth.Service) *QuotaAuthHandler {
	return &QuotaAuthHandler{service: service}
}

type createQuotaPairingRequest struct {
	ClientID            string `json:"client_id"`
	Scope               string `json:"scope"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	InstallationID      string `json:"installation_id"`
	DeviceName          string `json:"device_name"`
	Platform            string `json:"platform"`
	Architecture        string `json:"architecture"`
	OSVersion           string `json:"os_version"`
	AppVersion          string `json:"app_version"`
}

type exchangeQuotaPairingRequest struct {
	ClientID     string `json:"client_id"`
	DeviceCode   string `json:"device_code"`
	CodeVerifier string `json:"code_verifier"`
}

type refreshQuotaSessionRequest struct {
	ClientID              string  `json:"client_id"`
	RefreshToken          string  `json:"refresh_token"`
	RotationID            *string `json:"rotation_id"`
	CandidateRefreshToken *string `json:"candidate_refresh_token"`
}

func (h *QuotaAuthHandler) CreatePairing(c *gin.Context) {
	setQuotaAuthNoStoreHeaders(c)
	var req createQuotaPairingRequest
	if err := decodeStrictQuotaAuthJSON(c, &req); err != nil {
		response.ErrorFrom(c, quotaauth.ErrInvalidAuthorizationRequest.WithCause(err))
		return
	}
	result, err := h.service.CreatePairing(c.Request.Context(), quotaauth.CreatePairingInput{
		ClientID:            req.ClientID,
		Scope:               req.Scope,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		InstallationID:      req.InstallationID,
		DeviceName:          req.DeviceName,
		Platform:            req.Platform,
		Architecture:        req.Architecture,
		OSVersion:           req.OSVersion,
		AppVersion:          req.AppVersion,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, result)
}

func (h *QuotaAuthHandler) PairingPreview(c *gin.Context) {
	setQuotaAuthNoStoreHeaders(c)
	preview, err := h.service.GetPairingPreview(c.Request.Context(), c.Param("user_code"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, preview)
}

func (h *QuotaAuthHandler) ApprovePairing(c *gin.Context) {
	setQuotaAuthNoStoreHeaders(c)
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.ErrorFrom(c, quotaauth.ErrAuthRequired)
		return
	}
	middleware.SetAuditAction(c, "quota_viewer.authorization.approve")
	device, err := h.service.ApprovePairing(c.Request.Context(), subject.UserID, c.Param("user_code"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, device)
}

func (h *QuotaAuthHandler) ExchangePairing(c *gin.Context) {
	setQuotaAuthNoStoreHeaders(c)
	var req exchangeQuotaPairingRequest
	if err := decodeStrictQuotaAuthJSON(c, &req); err != nil {
		response.ErrorFrom(c, quotaauth.ErrInvalidAuthorizationRequest.WithCause(err))
		return
	}
	tokens, err := h.service.ExchangePairing(
		c.Request.Context(),
		req.ClientID,
		strings.TrimSpace(req.DeviceCode),
		strings.TrimSpace(req.CodeVerifier),
	)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, tokens)
}

func (h *QuotaAuthHandler) RefreshSession(c *gin.Context) {
	setQuotaAuthNoStoreHeaders(c)
	var req refreshQuotaSessionRequest
	if err := decodeStrictQuotaAuthJSON(c, &req); err != nil {
		response.ErrorFrom(c, quotaauth.ErrInvalidAuthorizationRequest.WithCause(err))
		return
	}
	if req.RotationID == nil || req.CandidateRefreshToken == nil {
		response.ErrorFrom(c, quotaauth.ErrInvalidAuthorizationRequest)
		return
	}
	tokens, err := h.service.RefreshSessionWithInput(c.Request.Context(), quotaauth.RefreshSessionInput{
		ClientID:              req.ClientID,
		RefreshToken:          strings.TrimSpace(req.RefreshToken),
		RotationID:            req.RotationID,
		CandidateRefreshToken: req.CandidateRefreshToken,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, tokens)
}

func (h *QuotaAuthHandler) ListDevices(c *gin.Context) {
	setQuotaAuthNoStoreHeaders(c)
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.ErrorFrom(c, quotaauth.ErrAuthRequired)
		return
	}
	devices, err := h.service.ListDevices(c.Request.Context(), subject.UserID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, devices)
}

func (h *QuotaAuthHandler) RevokeDevice(c *gin.Context) {
	setQuotaAuthNoStoreHeaders(c)
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.ErrorFrom(c, quotaauth.ErrAuthRequired)
		return
	}
	middleware.SetAuditAction(c, "quota_viewer.device.revoke")
	device, err := h.service.RevokeDevice(
		c.Request.Context(),
		subject.UserID,
		c.Param("device_id"),
	)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, device)
}

func decodeStrictQuotaAuthJSON(c *gin.Context, dst any) error {
	if c.Request == nil || c.Request.Body == nil {
		return errors.New("request body is required")
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, quotaAuthRequestBodyLimit)
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

func setQuotaAuthNoStoreHeaders(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Vary", "Authorization")
}
