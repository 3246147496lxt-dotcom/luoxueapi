package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	oauthPendingBrowserCookiePath = "/api/v1/auth/oauth"
	oauthPendingBrowserCookieName = "oauth_pending_browser_session"
	oauthPendingSessionCookiePath = "/api/v1/auth/oauth"
	oauthPendingSessionCookieName = "oauth_pending_session"
	oauthPromoCodeCookieName      = "oauth_promo_code"
	oauthPendingCookieMaxAgeSec   = 10 * 60
	oauthPendingChoiceStep        = "choose_account_action_required"

	oauthCompletionResponseKey = "completion_response"
	oauthPromoCodeStateKey     = "promo_code"
)

var pendingOAuthCreateAccountPreCommitHook func(context.Context, *service.PendingAuthSession) error

type oauthPendingSessionPayload struct {
	Intent                 string
	Identity               service.PendingAuthIdentityKey
	TargetUserID           *int64
	ResolvedEmail          string
	RedirectTo             string
	BrowserSessionKey      string
	UpstreamIdentityClaims map[string]any
	CompletionResponse     map[string]any
}

type oauthAdoptionDecisionRequest struct {
	AdoptDisplayName *bool `json:"adopt_display_name,omitempty"`
	AdoptAvatar      *bool `json:"adopt_avatar,omitempty"`
}

type bindPendingOAuthLoginRequest struct {
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required"`
	AdoptDisplayName *bool  `json:"adopt_display_name,omitempty"`
	AdoptAvatar      *bool  `json:"adopt_avatar,omitempty"`
}

type createPendingOAuthAccountRequest struct {
	Email            string `json:"email" binding:"required,email"`
	VerifyCode       string `json:"verify_code,omitempty"`
	Password         string `json:"password" binding:"required,min=6"`
	InvitationCode   string `json:"invitation_code,omitempty"`
	AffCode          string `json:"aff_code,omitempty"`
	AdoptDisplayName *bool  `json:"adopt_display_name,omitempty"`
	AdoptAvatar      *bool  `json:"adopt_avatar,omitempty"`
}

type sendPendingOAuthVerifyCodeRequest struct {
	Email             string `json:"email" binding:"required,email"`
	TurnstileToken    string `json:"turnstile_token,omitempty"`
	PendingAuthToken  string `json:"pending_auth_token,omitempty"`
	PendingOAuthToken string `json:"pending_oauth_token,omitempty"`
}

func (r bindPendingOAuthLoginRequest) adoptionDecision() oauthAdoptionDecisionRequest {
	return oauthAdoptionDecisionRequest{
		AdoptDisplayName: r.AdoptDisplayName,
		AdoptAvatar:      r.AdoptAvatar,
	}
}

func (r createPendingOAuthAccountRequest) adoptionDecision() oauthAdoptionDecisionRequest {
	return oauthAdoptionDecisionRequest{
		AdoptDisplayName: r.AdoptDisplayName,
		AdoptAvatar:      r.AdoptAvatar,
	}
}

func (h *AuthHandler) pendingIdentityService() (PendingIdentityUseCases, error) {
	if h == nil {
		return nil, infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready")
	}
	if h.pendingIdentity != nil {
		return h.pendingIdentity, nil
	}
	if h.authService != nil {
		if pending := h.authService.PendingIdentityUseCases(); pending != nil {
			return pending, nil
		}
	}
	return nil, infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready")
}

func generateOAuthPendingBrowserSession() (string, error) {
	return oauth.GenerateState()
}

func setOAuthPendingBrowserCookie(c *gin.Context, sessionKey string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthPendingBrowserCookieName,
		Value:    encodeCookieValue(sessionKey),
		Path:     oauthPendingBrowserCookiePath,
		MaxAge:   oauthPendingCookieMaxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthPendingBrowserCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthPendingBrowserCookieName,
		Value:    "",
		Path:     oauthPendingBrowserCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func readOAuthPendingBrowserCookie(c *gin.Context) (string, error) {
	return readCookieDecoded(c, oauthPendingBrowserCookieName)
}

func setOAuthPendingSessionCookie(c *gin.Context, sessionToken string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthPendingSessionCookieName,
		Value:    encodeCookieValue(sessionToken),
		Path:     oauthPendingSessionCookiePath,
		MaxAge:   oauthPendingCookieMaxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthPendingSessionCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthPendingSessionCookieName,
		Value:    "",
		Path:     oauthPendingSessionCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func readOAuthPendingSessionCookie(c *gin.Context) (string, error) {
	return readCookieDecoded(c, oauthPendingSessionCookieName)
}

func captureOAuthPromoCode(c *gin.Context, secure bool) {
	promoCode := strings.TrimSpace(c.Query("promo_code"))
	if promoCode == "" {
		clearOAuthPromoCodeCookie(c, secure)
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthPromoCodeCookieName,
		Value:    encodeCookieValue(promoCode),
		Path:     oauthPendingBrowserCookiePath,
		MaxAge:   oauthPendingCookieMaxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearOAuthPromoCodeCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     oauthPromoCodeCookieName,
		Value:    "",
		Path:     oauthPendingBrowserCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func readOAuthPromoCode(c *gin.Context) string {
	if c == nil {
		return ""
	}
	promoCode, err := readCookieDecoded(c, oauthPromoCodeCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(promoCode)
}

func pendingOAuthPromoCode(session *service.PendingAuthSession) string {
	if session == nil {
		return ""
	}
	return pendingSessionStringValue(session.LocalFlowState, oauthPromoCodeStateKey)
}

func redirectToFrontendCallback(c *gin.Context, frontendCallback string) {
	u, err := url.Parse(frontendCallback)
	if err != nil {
		c.Redirect(http.StatusFound, linuxDoOAuthDefaultRedirectTo)
		return
	}
	if u.Scheme != "" && !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		c.Redirect(http.StatusFound, linuxDoOAuthDefaultRedirectTo)
		return
	}
	u.Fragment = ""
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Redirect(http.StatusFound, u.String())
}

func (h *AuthHandler) createOAuthPendingSession(c *gin.Context, payload oauthPendingSessionPayload) error {
	svc, err := h.pendingIdentityService()
	if err != nil {
		return err
	}

	localFlowState := map[string]any{
		oauthCompletionResponseKey: payload.CompletionResponse,
	}
	if promoCode := readOAuthPromoCode(c); promoCode != "" {
		localFlowState[oauthPromoCodeStateKey] = promoCode
	}

	session, err := svc.CreatePendingSession(c.Request.Context(), service.CreatePendingAuthSessionInput{
		Intent:                 strings.TrimSpace(payload.Intent),
		Identity:               payload.Identity,
		TargetUserID:           payload.TargetUserID,
		ResolvedEmail:          strings.TrimSpace(payload.ResolvedEmail),
		RedirectTo:             strings.TrimSpace(payload.RedirectTo),
		BrowserSessionKey:      strings.TrimSpace(payload.BrowserSessionKey),
		UpstreamIdentityClaims: payload.UpstreamIdentityClaims,
		LocalFlowState:         localFlowState,
	})
	if err != nil {
		slog.Error("pending auth session create failed",
			"intent", strings.TrimSpace(payload.Intent),
			"provider_type", strings.TrimSpace(payload.Identity.ProviderType),
			"provider_key", strings.TrimSpace(payload.Identity.ProviderKey),
			"provider_subject_len", len(strings.TrimSpace(payload.Identity.ProviderSubject)),
			"resolved_email_len", len(strings.TrimSpace(payload.ResolvedEmail)),
			"has_target_user", payload.TargetUserID != nil,
			"error", err.Error())
		return infraerrors.InternalServer("PENDING_AUTH_SESSION_CREATE_FAILED", "failed to create pending auth session").WithCause(err)
	}

	setOAuthPendingSessionCookie(c, session.SessionToken, isRequestHTTPS(c))
	return nil
}

func readCompletionResponse(session map[string]any) (map[string]any, bool) {
	if len(session) == 0 {
		return nil, false
	}
	value, ok := session[oauthCompletionResponseKey]
	if !ok {
		return nil, false
	}
	result, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	return result, true
}

func clonePendingMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func mergePendingCompletionResponse(session *service.PendingAuthSession, overrides map[string]any) map[string]any {
	payload, _ := readCompletionResponse(session.LocalFlowState)
	merged := clonePendingMap(payload)
	if strings.TrimSpace(session.RedirectTo) != "" {
		if _, exists := merged["redirect"]; !exists {
			merged["redirect"] = session.RedirectTo
		}
	}
	for key, value := range overrides {
		if value == nil {
			delete(merged, key)
			continue
		}
		merged[key] = value
	}
	applySuggestedProfileToCompletionResponse(merged, session.UpstreamIdentityClaims)
	return merged
}

func pendingSessionStringValue(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	raw, ok := values[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func pendingSessionWantsInvitation(payload map[string]any) bool {
	return strings.EqualFold(strings.TrimSpace(pendingSessionStringValue(payload, "error")), "invitation_required")
}

// pendingSessionRequiresEmailCompletion 判断 callback 写入的 completion payload 是否处于"补邮箱"状态。
// 钉钉跨组织/staff 邮箱缺失时进入此状态：前端跳到补邮箱页，exchange 不应走 adoption apply。
func pendingSessionRequiresEmailCompletion(payload map[string]any) bool {
	if v, ok := payload["requires_email_completion"].(bool); ok && v {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(pendingSessionStringValue(payload, "step")), "email_completion")
}

// pendingSessionRequiresBindLogin 判断 callback 写入的 completion payload 是否处于"必须绑定已有账户"状态。
// 钉钉 signupBlocked=true（注册关 + 钉钉企业豁免关）时进入此状态：前端渲染 bind_login 表单，
// exchange 不应消费 session，否则后续 /pending/bind-login 找不到 session。
func pendingSessionRequiresBindLogin(payload map[string]any) bool {
	return strings.EqualFold(strings.TrimSpace(pendingSessionStringValue(payload, "step")), "bind_login_required")
}

func pendingOAuthCompletionCanIssueTokenPair(session *service.PendingAuthSession, payload map[string]any) bool {
	if session == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(session.Intent), oauthIntentLogin) {
		return false
	}
	if session.TargetUserID == nil || *session.TargetUserID <= 0 {
		return false
	}
	if pendingSessionWantsInvitation(payload) {
		return false
	}
	return strings.TrimSpace(pendingSessionStringValue(payload, "step")) == ""
}

func ensurePendingOAuthCompleteRegistrationSession(session *service.PendingAuthSession) error {
	if session == nil {
		return infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth registration context is invalid")
	}
	if strings.TrimSpace(session.Intent) != oauthIntentLogin {
		return infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth registration context is invalid")
	}
	if session.TargetUserID != nil && *session.TargetUserID > 0 {
		return infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth registration context is invalid")
	}
	payload, _ := readCompletionResponse(session.LocalFlowState)
	if strings.EqualFold(strings.TrimSpace(pendingSessionStringValue(payload, "step")), "bind_login_required") {
		return infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth registration context is invalid")
	}
	return nil
}

func buildLegacyCompleteRegistrationPendingResponse(
	session *service.PendingAuthSession,
	forceEmailOnSignup bool,
	emailVerificationRequired bool,
) map[string]any {
	completionResponse := normalizePendingOAuthCompletionResponse(mergePendingCompletionResponse(session, map[string]any{
		"step":                   oauthPendingChoiceStep,
		"adoption_required":      true,
		"create_account_allowed": true,
		"force_email_on_signup":  forceEmailOnSignup,
	}))

	if email := strings.TrimSpace(session.ResolvedEmail); email != "" {
		if _, exists := completionResponse["email"]; !exists {
			completionResponse["email"] = email
		}
		if _, exists := completionResponse["resolved_email"]; !exists {
			completionResponse["resolved_email"] = email
		}
	}
	if _, exists := completionResponse["choice_reason"]; !exists {
		switch {
		case forceEmailOnSignup:
			completionResponse["choice_reason"] = "force_email_on_signup"
		case emailVerificationRequired:
			completionResponse["choice_reason"] = "email_verification_required"
		default:
			completionResponse["choice_reason"] = "third_party_signup"
		}
	}
	return completionResponse
}

func (h *AuthHandler) legacyCompleteRegistrationSessionStatus(
	c *gin.Context,
	session *service.PendingAuthSession,
) (*service.PendingAuthSession, bool, error) {
	if session == nil {
		return nil, false, infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth registration context is invalid")
	}

	payload := normalizePendingOAuthCompletionResponse(mergePendingCompletionResponse(session, nil))
	if step := pendingSessionStringValue(payload, "step"); step != "" {
		return session, true, nil
	}

	emailVerificationRequired := h != nil && h.signupCases() != nil && h.signupCases().IsEmailVerifyEnabled(c.Request.Context())
	forceEmailOnSignup := h.isForceEmailOnThirdPartySignup(c.Request.Context())
	if !emailVerificationRequired && !forceEmailOnSignup {
		return session, false, nil
	}

	svc, err := h.pendingIdentityService()
	if err != nil {
		return nil, false, err
	}
	updatedSession, err := svc.UpdateSessionProgress(c.Request.Context(), service.UpdatePendingAuthSessionProgressInput{
		SessionID:          session.ID,
		Intent:             strings.TrimSpace(session.Intent),
		ResolvedEmail:      strings.TrimSpace(session.ResolvedEmail),
		CompletionResponse: buildLegacyCompleteRegistrationPendingResponse(session, forceEmailOnSignup, emailVerificationRequired),
	})
	if err != nil {
		return nil, false, infraerrors.InternalServer("PENDING_AUTH_SESSION_UPDATE_FAILED", "failed to update pending oauth session").WithCause(err)
	}
	return updatedSession, true, nil
}

func (r oauthAdoptionDecisionRequest) hasDecision() bool {
	return r.AdoptDisplayName != nil || r.AdoptAvatar != nil
}

func bindOptionalOAuthAdoptionDecision(c *gin.Context) (oauthAdoptionDecisionRequest, error) {
	var req oauthAdoptionDecisionRequest
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return req, nil
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return req, nil
		}
		return req, err
	}
	return req, nil
}

func cloneOAuthMetadata(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func applySuggestedProfileToCompletionResponse(payload map[string]any, upstream map[string]any) {
	if len(payload) == 0 || len(upstream) == 0 {
		return
	}

	displayName := pendingSessionStringValue(upstream, "suggested_display_name")
	avatarURL := pendingSessionStringValue(upstream, "suggested_avatar_url")
	if displayName != "" {
		if _, exists := payload["suggested_display_name"]; !exists {
			payload["suggested_display_name"] = displayName
		}
	}
	if avatarURL != "" {
		if _, exists := payload["suggested_avatar_url"]; !exists {
			payload["suggested_avatar_url"] = avatarURL
		}
	}
	if displayName != "" || avatarURL != "" {
		payload["adoption_required"] = true
	}
}

func (h *AuthHandler) isForceEmailOnThirdPartySignup(ctx context.Context) bool {
	if h == nil || h.settingSvc == nil {
		return false
	}
	defaults, err := h.settingSvc.GetAuthSourceDefaultSettings(ctx)
	if err != nil || defaults == nil {
		return false
	}
	return defaults.ForceEmailOnThirdPartySignup
}

func (h *AuthHandler) findOAuthIdentityUser(ctx context.Context, identity service.PendingAuthIdentityKey) (*service.AuthIdentityUser, error) {
	svc, err := h.pendingIdentityService()
	if err != nil {
		return nil, err
	}
	return svc.FindIdentityUser(ctx, identity)
}

func (h *AuthHandler) findPendingUserByNormalizedEmail(ctx context.Context, email string) (*service.AuthIdentityUser, error) {
	svc, err := h.pendingIdentityService()
	if err != nil {
		return nil, err
	}
	return svc.FindUserByNormalizedEmail(ctx, email)
}

func (h *AuthHandler) ensurePendingRegistrationIdentityAvailable(ctx context.Context, session *service.PendingAuthSession) error {
	svc, err := h.pendingIdentityService()
	if err != nil {
		return err
	}
	return svc.EnsureRegistrationIdentityAvailable(ctx, session)
}

func (h *AuthHandler) pendingIdentityDefaultApplier() service.PendingIdentityDefaultApplier {
	if h == nil || h.authService == nil {
		return nil
	}
	return h.authService
}

func (h *AuthHandler) pendingIdentityAvatarWriter() service.PendingIdentityAvatarWriter {
	if h == nil || h.userService == nil {
		return nil
	}
	return h.userService
}

func (h *AuthHandler) applyPendingIdentityBinding(
	ctx context.Context,
	session *service.PendingAuthSession,
	decision *service.PendingIdentityDecision,
	overrideUserID *int64,
	forceBind bool,
	applyFirstBindDefaults bool,
) error {
	svc, err := h.pendingIdentityService()
	if err != nil {
		return err
	}
	return svc.ApplyBinding(ctx, service.ApplyPendingIdentityBindingInput{
		Session:                session,
		Decision:               decision,
		OverrideUserID:         overrideUserID,
		ForceBind:              forceBind,
		ApplyFirstBindDefaults: applyFirstBindDefaults,
		DefaultApplier:         h.pendingIdentityDefaultApplier(),
		AvatarWriter:           h.pendingIdentityAvatarWriter(),
	})
}

func (h *AuthHandler) applyPendingIdentityBindingAndConsume(
	ctx context.Context,
	session *service.PendingAuthSession,
	decision *service.PendingIdentityDecision,
	userID int64,
) error {
	svc, err := h.pendingIdentityService()
	if err != nil {
		return err
	}
	return svc.ApplyBindingAndConsume(ctx, service.ApplyPendingIdentityBindingInput{
		Session:        session,
		Decision:       decision,
		OverrideUserID: &userID,
		DefaultApplier: h.pendingIdentityDefaultApplier(),
		AvatarWriter:   h.pendingIdentityAvatarWriter(),
	})
}

func (h *AuthHandler) BindLinuxDoOAuthLogin(c *gin.Context) { h.bindPendingOAuthLogin(c, "linuxdo") }
func (h *AuthHandler) BindOIDCOAuthLogin(c *gin.Context)    { h.bindPendingOAuthLogin(c, "oidc") }
func (h *AuthHandler) BindWeChatOAuthLogin(c *gin.Context)  { h.bindPendingOAuthLogin(c, "wechat") }
func (h *AuthHandler) BindPendingOAuthLogin(c *gin.Context) { h.bindPendingOAuthLogin(c, "") }

func (h *AuthHandler) CreateLinuxDoOAuthAccount(c *gin.Context) {
	h.createPendingOAuthAccount(c, "linuxdo")
}

func (h *AuthHandler) CreateOIDCOAuthAccount(c *gin.Context) { h.createPendingOAuthAccount(c, "oidc") }

func (h *AuthHandler) CreateWeChatOAuthAccount(c *gin.Context) {
	h.createPendingOAuthAccount(c, "wechat")
}

func (h *AuthHandler) CreatePendingOAuthAccount(c *gin.Context) {
	h.createPendingOAuthAccount(c, "")
}

// SendPendingOAuthVerifyCode sends a verification code for a browser-bound
// pending OAuth account-creation flow.
// POST /api/v1/auth/oauth/pending/send-verify-code
func (h *AuthHandler) SendPendingOAuthVerifyCode(c *gin.Context) {
	var req sendPendingOAuthVerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.verifyLoginChallenge(c.Request.Context(), req.TurnstileToken, ip.GetClientIP(c)); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	_, session, _, err := readPendingOAuthBrowserSession(c, h)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := ensurePendingOAuthCompleteRegistrationSession(session); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if existingUser, err := h.findPendingUserByNormalizedEmail(c.Request.Context(), email); err == nil && existingUser != nil {
		session, err = h.transitionPendingOAuthAccountToChoiceState(c, session, existingUser, email)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		c.JSON(http.StatusOK, buildPendingOAuthSessionStatusPayload(session))
		return
	} else if err != nil && !errors.Is(err, service.ErrUserNotFound) {
		response.ErrorFrom(c, err)
		return
	}

	result, err := h.signupCases().SendPendingOAuthVerifyCode(c.Request.Context(), req.Email, c.GetHeader("Accept-Language"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, SendVerifyCodeResponse{
		Message:   "Verification code sent successfully",
		Countdown: result.Countdown,
	})
}

func (h *AuthHandler) upsertPendingOAuthAdoptionDecision(
	c *gin.Context,
	sessionID int64,
	req oauthAdoptionDecisionRequest,
) (*service.PendingIdentityDecision, error) {
	svc, err := h.pendingIdentityService()
	if err != nil {
		return nil, err
	}
	existing, err := svc.GetAdoptionDecision(c.Request.Context(), sessionID)
	if err != nil {
		return nil, infraerrors.InternalServer("PENDING_AUTH_ADOPTION_LOAD_FAILED", "failed to load oauth profile adoption decision").WithCause(err)
	}
	if existing != nil && !req.hasDecision() {
		return existing, nil
	}
	if existing == nil && !req.hasDecision() {
		return nil, nil
	}

	input := service.PendingIdentityAdoptionDecisionInput{
		PendingAuthSessionID: sessionID,
	}
	if existing != nil {
		input.AdoptDisplayName = existing.AdoptDisplayName
		input.AdoptAvatar = existing.AdoptAvatar
		input.IdentityID = existing.IdentityID
	}
	if req.AdoptDisplayName != nil {
		input.AdoptDisplayName = *req.AdoptDisplayName
	}
	if req.AdoptAvatar != nil {
		input.AdoptAvatar = *req.AdoptAvatar
	}

	decision, err := svc.UpsertAdoptionDecision(c.Request.Context(), input)
	if err != nil {
		return nil, infraerrors.InternalServer("PENDING_AUTH_ADOPTION_SAVE_FAILED", "failed to save oauth profile adoption decision").WithCause(err)
	}
	return decision, nil
}

func (h *AuthHandler) ensurePendingOAuthAdoptionDecision(
	c *gin.Context,
	sessionID int64,
	req oauthAdoptionDecisionRequest,
) (*service.PendingIdentityDecision, error) {
	decision, err := h.upsertPendingOAuthAdoptionDecision(c, sessionID, req)
	if err != nil {
		return nil, err
	}
	if decision != nil {
		return decision, nil
	}

	svc, err := h.pendingIdentityService()
	if err != nil {
		return nil, err
	}
	decision, err = svc.UpsertAdoptionDecision(c.Request.Context(), service.PendingIdentityAdoptionDecisionInput{
		PendingAuthSessionID: sessionID,
	})
	if err != nil {
		return nil, infraerrors.InternalServer("PENDING_AUTH_ADOPTION_SAVE_FAILED", "failed to save oauth profile adoption decision").WithCause(err)
	}
	return decision, nil
}

func (h *AuthHandler) shouldSkipPendingOAuthAdoptionPrompt(
	ctx context.Context,
	session *service.PendingAuthSession,
	payload map[string]any,
) (bool, error) {
	if session == nil || len(payload) == 0 {
		return false, nil
	}
	if !pendingOAuthCompletionCanIssueTokenPair(session, payload) {
		return false, nil
	}
	if pendingSessionStringValue(session.UpstreamIdentityClaims, "suggested_display_name") == "" &&
		pendingSessionStringValue(session.UpstreamIdentityClaims, "suggested_avatar_url") == "" {
		return false, nil
	}

	svc, err := h.pendingIdentityService()
	if err != nil {
		return false, err
	}
	var compatibleKeys []string
	if strings.EqualFold(strings.TrimSpace(session.ProviderType), "wechat") {
		compatibleKeys = wechatCompatibleProviderKeys(session.ProviderKey)
	}
	return svc.IdentityExistsForUser(ctx, session, *session.TargetUserID, compatibleKeys)
}

func readPendingOAuthBrowserSession(c *gin.Context, h *AuthHandler) (PendingIdentityUseCases, *service.PendingAuthSession, func(), error) {
	secureCookie := isRequestHTTPS(c)
	clearCookies := func() {
		clearOAuthPendingSessionCookie(c, secureCookie)
		clearOAuthPendingBrowserCookie(c, secureCookie)
	}

	sessionToken, err := readOAuthPendingSessionCookie(c)
	if err != nil || strings.TrimSpace(sessionToken) == "" {
		clearCookies()
		return nil, nil, clearCookies, service.ErrPendingAuthSessionNotFound
	}
	browserSessionKey, err := readOAuthPendingBrowserCookie(c)
	if err != nil || strings.TrimSpace(browserSessionKey) == "" {
		clearCookies()
		return nil, nil, clearCookies, service.ErrPendingAuthBrowserMismatch
	}

	svc, err := h.pendingIdentityService()
	if err != nil {
		clearCookies()
		return nil, nil, clearCookies, err
	}

	session, err := svc.GetBrowserSession(c.Request.Context(), sessionToken, browserSessionKey)
	if err != nil {
		clearCookies()
		return nil, nil, clearCookies, err
	}

	return svc, session, clearCookies, nil
}

func (h *AuthHandler) consumePendingOAuthSessionOnLogout(c *gin.Context) {
	if c == nil || c.Request == nil {
		return
	}

	sessionToken, err := readOAuthPendingSessionCookie(c)
	if err != nil || strings.TrimSpace(sessionToken) == "" {
		return
	}
	browserSessionKey, err := readOAuthPendingBrowserCookie(c)
	if err != nil || strings.TrimSpace(browserSessionKey) == "" {
		return
	}

	if h != nil && h.authModule != nil {
		_, _ = h.authModule.ConsumePendingIdentity(c.Request.Context(), sessionToken, browserSessionKey)
		return
	}
	svc, err := h.pendingIdentityService()
	if err == nil {
		_, _ = svc.ConsumeBrowserSession(c.Request.Context(), sessionToken, browserSessionKey)
	}
}

func clearOAuthLogoutCookies(c *gin.Context) {
	secureCookie := isRequestHTTPS(c)

	clearOAuthPendingSessionCookie(c, secureCookie)
	clearOAuthPendingBrowserCookie(c, secureCookie)
	clearOAuthBindAccessTokenCookie(c, secureCookie)

	clearCookie(c, linuxDoOAuthStateCookieName, secureCookie)
	clearCookie(c, linuxDoOAuthVerifierCookie, secureCookie)
	clearCookie(c, linuxDoOAuthRedirectCookie, secureCookie)
	clearCookie(c, linuxDoOAuthIntentCookieName, secureCookie)
	clearCookie(c, linuxDoOAuthBindUserCookieName, secureCookie)

	oidcClearCookie(c, oidcOAuthStateCookieName, secureCookie)
	oidcClearCookie(c, oidcOAuthVerifierCookie, secureCookie)
	oidcClearCookie(c, oidcOAuthRedirectCookie, secureCookie)
	oidcClearCookie(c, oidcOAuthNonceCookie, secureCookie)
	oidcClearCookie(c, oidcOAuthIntentCookieName, secureCookie)
	oidcClearCookie(c, oidcOAuthBindUserCookieName, secureCookie)

	wechatClearCookie(c, wechatOAuthStateCookieName, secureCookie)
	wechatClearCookie(c, wechatOAuthRedirectCookieName, secureCookie)
	wechatClearCookie(c, wechatOAuthIntentCookieName, secureCookie)
	wechatClearCookie(c, wechatOAuthModeCookieName, secureCookie)
	wechatClearCookie(c, wechatOAuthBindUserCookieName, secureCookie)

	wechatPaymentClearCookie(c, wechatPaymentOAuthStateName, secureCookie)
	wechatPaymentClearCookie(c, wechatPaymentOAuthRedirect, secureCookie)
	wechatPaymentClearCookie(c, wechatPaymentOAuthContextName, secureCookie)
	wechatPaymentClearCookie(c, wechatPaymentOAuthScope, secureCookie)
}

func buildPendingOAuthSessionStatusPayload(session *service.PendingAuthSession) gin.H {
	completionResponse := normalizePendingOAuthCompletionResponse(mergePendingCompletionResponse(session, nil))
	payload := gin.H{
		"auth_result": "pending_session",
		"provider":    strings.TrimSpace(session.ProviderType),
		"intent":      strings.TrimSpace(session.Intent),
	}
	for key, value := range completionResponse {
		payload[key] = value
	}
	if email := strings.TrimSpace(session.ResolvedEmail); email != "" {
		payload["email"] = email
	}
	return payload
}

func normalizePendingOAuthCompletionResponse(payload map[string]any) map[string]any {
	normalized := clonePendingMap(payload)
	for _, key := range []string{"access_token", "refresh_token", "expires_in", "token_type"} {
		delete(normalized, key)
	}
	step := strings.ToLower(strings.TrimSpace(pendingSessionStringValue(normalized, "step")))
	// 把多种 choice 别名归一为 oauthPendingChoiceStep；bind_login_required 是独立终态
	// （前端渲染 needsBindLogin 而非 needsChooser），故不能并入归一化列表。
	switch step {
	case "choice", "choose_account_action", "choose_account", "choose", "email_required":
		normalized["step"] = oauthPendingChoiceStep
	}
	if strings.EqualFold(strings.TrimSpace(pendingSessionStringValue(normalized, "step")), oauthPendingChoiceStep) {
		normalized["adoption_required"] = true
	}
	if _, exists := normalized["adoption_required"]; !exists {
		if _, hasChoiceFields := normalized["email_binding_required"]; hasChoiceFields {
			normalized["adoption_required"] = true
		}
	}
	return normalized
}

func pendingOAuthChoiceCompletionResponse(session *service.PendingAuthSession, email string) map[string]any {
	response := mergePendingCompletionResponse(session, map[string]any{
		"step":                      oauthPendingChoiceStep,
		"adoption_required":         true,
		"force_email_on_signup":     true,
		"email_binding_required":    true,
		"existing_account_bindable": true,
	})
	if email = strings.TrimSpace(email); email != "" {
		response["email"] = email
		response["resolved_email"] = email
	}
	return response
}

func (h *AuthHandler) transitionPendingOAuthAccountToChoiceState(
	c *gin.Context,
	session *service.PendingAuthSession,
	targetUser *service.AuthIdentityUser,
	email string,
) (*service.PendingAuthSession, error) {
	completionResponse := pendingOAuthChoiceCompletionResponse(session, email)
	var targetUserID *int64
	if targetUser != nil && targetUser.ID > 0 {
		targetUserID = &targetUser.ID
	}
	svc, err := h.pendingIdentityService()
	if err != nil {
		return nil, err
	}
	session, err = svc.UpdateSessionProgress(c.Request.Context(), service.UpdatePendingAuthSessionProgressInput{
		SessionID:          session.ID,
		Intent:             strings.TrimSpace(session.Intent),
		ResolvedEmail:      email,
		TargetUserID:       targetUserID,
		CompletionResponse: completionResponse,
	})
	if err != nil {
		return nil, infraerrors.InternalServer("PENDING_AUTH_SESSION_UPDATE_FAILED", "failed to update pending oauth session").WithCause(err)
	}
	return session, nil
}

func writeOAuthTokenPairResponse(c *gin.Context, tokenPair *service.TokenPair) {
	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"token_type":    "Bearer",
	})
}

func (h *AuthHandler) bindPendingOAuthLogin(c *gin.Context, provider string) {
	var req bindPendingOAuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	pendingSvc, session, clearCookies, err := readPendingOAuthBrowserSession(c, h)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if strings.TrimSpace(provider) != "" && !strings.EqualFold(strings.TrimSpace(session.ProviderType), provider) {
		response.BadRequest(c, "Pending oauth session provider mismatch")
		return
	}

	user, err := h.loginCases().ValidatePasswordCredentials(c.Request.Context(), strings.TrimSpace(req.Email), req.Password)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if session.TargetUserID != nil && *session.TargetUserID > 0 && user.ID != *session.TargetUserID {
		response.ErrorFrom(c, infraerrors.Conflict("PENDING_AUTH_TARGET_USER_MISMATCH", "pending oauth session must be completed by the targeted user"))
		return
	}
	if err := h.ensureBackendModeAllowsUser(c.Request.Context(), user); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	decision, err := h.ensurePendingOAuthAdoptionDecision(c, session.ID, req.adoptionDecision())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if h.totpService != nil && h.settingSvc.IsTotpEnabled(c.Request.Context()) && user.TotpEnabled {
		tempToken, err := h.totpService.CreatePendingOAuthBindLoginSession(
			c.Request.Context(),
			user.ID,
			user.Email,
			session.SessionToken,
			session.BrowserSessionKey,
		)
		if err != nil {
			response.InternalError(c, "Failed to create 2FA session")
			return
		}
		response.Success(c, TotpLoginResponse{
			Requires2FA:     true,
			TempToken:       tempToken,
			UserEmailMasked: service.MaskEmail(user.Email),
		})
		return
	}
	if err := h.applyPendingIdentityBinding(c.Request.Context(), session, decision, &user.ID, true, true); err != nil {
		respondPendingOAuthBindingApplyError(c, err)
		return
	}

	h.loginCases().RecordSuccessfulLogin(c.Request.Context(), user.ID)
	// bindPendingOAuthLogin = 绑定已有账户登录，不动 users.username（用户已有自己的名字）
	h.maybeSyncDingTalkAfterLogin(c.Request.Context(), session, user.ID)
	tokenPair, err := h.loginCases().GenerateTokenPair(c.Request.Context(), user, "")
	if err != nil {
		response.InternalError(c, "Failed to generate token pair")
		return
	}
	if _, err := pendingSvc.ConsumeBrowserSession(c.Request.Context(), session.SessionToken, session.BrowserSessionKey); err != nil {
		clearCookies()
		response.ErrorFrom(c, err)
		return
	}

	clearCookies()
	writeOAuthTokenPairResponse(c, tokenPair)
}

func respondPendingOAuthBindingApplyError(c *gin.Context, err error) {
	if code := infraerrors.Code(err); code >= http.StatusBadRequest && code < http.StatusInternalServerError {
		response.ErrorFrom(c, err)
		return
	}
	response.ErrorFrom(c, infraerrors.InternalServer("PENDING_AUTH_BIND_APPLY_FAILED", "failed to bind pending oauth identity").WithCause(err))
}

func (h *AuthHandler) createPendingOAuthAccount(c *gin.Context, provider string) {
	var req createPendingOAuthAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	_, session, clearCookies, err := readPendingOAuthBrowserSession(c, h)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := ensurePendingOAuthCompleteRegistrationSession(session); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if strings.TrimSpace(provider) != "" && !strings.EqualFold(strings.TrimSpace(session.ProviderType), provider) {
		response.BadRequest(c, "Pending oauth session provider mismatch")
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	existingUser, err := h.findPendingUserByNormalizedEmail(c.Request.Context(), email)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			existingUser = nil
		case infraerrors.Code(err) >= http.StatusBadRequest && infraerrors.Code(err) < http.StatusInternalServerError:
			response.ErrorFrom(c, err)
			return
		default:
			response.ErrorFrom(c, infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "service temporarily unavailable"))
			return
		}
	}
	if existingUser != nil {
		session, err = h.transitionPendingOAuthAccountToChoiceState(c, session, existingUser, email)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		c.JSON(http.StatusOK, buildPendingOAuthSessionStatusPayload(session))
		return
	}
	if err := h.ensureBackendModeAllowsNewUserLogin(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	tokenPair, user, err := h.authService.RegisterOAuthEmailAccount(
		c.Request.Context(),
		email,
		req.Password,
		strings.TrimSpace(req.VerifyCode),
		strings.TrimSpace(req.InvitationCode),
		strings.TrimSpace(session.ProviderType),
	)
	if err != nil {
		if errors.Is(err, service.ErrEmailExists) {
			existingUser, lookupErr := h.findPendingUserByNormalizedEmail(c.Request.Context(), email)
			if lookupErr != nil {
				response.ErrorFrom(c, lookupErr)
				return
			}
			session, err = h.transitionPendingOAuthAccountToChoiceState(c, session, existingUser, email)
			if err != nil {
				response.ErrorFrom(c, err)
				return
			}
			c.JSON(http.StatusOK, buildPendingOAuthSessionStatusPayload(session))
			return
		}
		response.ErrorFrom(c, err)
		return
	}

	rollbackCreatedUser := func(originalErr error) bool {
		if user == nil || user.ID <= 0 {
			return false
		}
		if rollbackErr := h.authService.RollbackOAuthEmailAccountCreation(
			c.Request.Context(),
			user.ID,
			strings.TrimSpace(req.InvitationCode),
		); rollbackErr != nil {
			response.ErrorFrom(c, infraerrors.InternalServer(
				"PENDING_AUTH_ACCOUNT_ROLLBACK_FAILED",
				"failed to rollback pending oauth account creation",
			).WithCause(fmt.Errorf("original error: %w; rollback error: %v", originalErr, rollbackErr)))
			return true
		}
		user = nil
		return false
	}

	decision, err := h.ensurePendingOAuthAdoptionDecision(c, session.ID, req.adoptionDecision())
	if err != nil {
		if rollbackCreatedUser(err) {
			return
		}
		response.ErrorFrom(c, err)
		return
	}

	if err := h.authService.FinalizePendingOAuthAccount(c.Request.Context(), service.FinalizePendingOAuthAccountInput{
		Session:        session,
		Decision:       decision,
		User:           user,
		InvitationCode: req.InvitationCode,
		ProviderType:   session.ProviderType,
		AffiliateCode:  req.AffCode,
		AvatarWriter:   h.pendingIdentityAvatarWriter(),
		BeforeCommit:   pendingOAuthCreateAccountPreCommitHook,
	}); err != nil {
		if rollbackCreatedUser(err) {
			return
		}
		respondPendingOAuthBindingApplyError(c, err)
		return
	}

	h.authService.ApplyOAuthSignupPromoCode(c.Request.Context(), user.ID, pendingOAuthPromoCode(session))
	h.loginCases().RecordSuccessfulLogin(c.Request.Context(), user.ID)
	// createPendingOAuthAccount = 注册新账户，需要把钉钉昵称同步到 users.username 作为初始值
	h.maybeSyncDingTalkAfterRegistration(c.Request.Context(), session, user.ID)
	clearCookies()
	writeOAuthTokenPairResponse(c, tokenPair)
}

// ExchangePendingOAuthCompletion redeems a pending OAuth browser session into a frontend-safe payload.
// POST /api/v1/auth/oauth/pending/exchange
func (h *AuthHandler) ExchangePendingOAuthCompletion(c *gin.Context) {
	secureCookie := isRequestHTTPS(c)
	clearCookies := func() {
		clearOAuthPendingSessionCookie(c, secureCookie)
		clearOAuthPendingBrowserCookie(c, secureCookie)
	}
	adoptionDecision, err := bindOptionalOAuthAdoptionDecision(c)
	if err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	sessionToken, err := readOAuthPendingSessionCookie(c)
	if err != nil || strings.TrimSpace(sessionToken) == "" {
		clearCookies()
		response.ErrorFrom(c, service.ErrPendingAuthSessionNotFound)
		return
	}
	browserSessionKey, err := readOAuthPendingBrowserCookie(c)
	if err != nil || strings.TrimSpace(browserSessionKey) == "" {
		clearCookies()
		response.ErrorFrom(c, service.ErrPendingAuthBrowserMismatch)
		return
	}

	svc, err := h.pendingIdentityService()
	if err != nil {
		clearCookies()
		response.ErrorFrom(c, err)
		return
	}

	session, err := svc.GetBrowserSession(c.Request.Context(), sessionToken, browserSessionKey)
	if err != nil {
		clearCookies()
		response.ErrorFrom(c, err)
		return
	}

	payload, ok := readCompletionResponse(session.LocalFlowState)
	if !ok {
		clearCookies()
		response.ErrorFrom(c, infraerrors.InternalServer("PENDING_AUTH_COMPLETION_INVALID", "pending auth completion payload is invalid"))
		return
	}
	payload = normalizePendingOAuthCompletionResponse(payload)
	if strings.TrimSpace(session.RedirectTo) != "" {
		if _, exists := payload["redirect"]; !exists {
			payload["redirect"] = session.RedirectTo
		}
	}
	applySuggestedProfileToCompletionResponse(payload, session.UpstreamIdentityClaims)

	canIssueTokenPair := pendingOAuthCompletionCanIssueTokenPair(session, payload)
	var loginUser *service.User
	if canIssueTokenPair {
		loginUser, err = h.userService.GetByID(c.Request.Context(), *session.TargetUserID)
		if err != nil {
			clearCookies()
			response.ErrorFrom(c, err)
			return
		}
		if err := ensureLoginUserActive(loginUser); err != nil {
			clearCookies()
			response.ErrorFrom(c, err)
			return
		}
		if err := h.ensureBackendModeAllowsUser(c.Request.Context(), loginUser); err != nil {
			clearCookies()
			response.ErrorFrom(c, err)
			return
		}
	}
	skipAdoptionPrompt, err := h.shouldSkipPendingOAuthAdoptionPrompt(c.Request.Context(), session, payload)
	if err != nil {
		clearCookies()
		response.ErrorFrom(c, err)
		return
	}
	if skipAdoptionPrompt {
		delete(payload, "adoption_required")
	}

	if pendingSessionWantsInvitation(payload) {
		if adoptionDecision.hasDecision() {
			decision, err := h.upsertPendingOAuthAdoptionDecision(c, session.ID, adoptionDecision)
			if err != nil {
				response.ErrorFrom(c, err)
				return
			}
			_ = decision
		}
		response.Success(c, payload)
		return
	}
	if pendingSessionRequiresEmailCompletion(payload) {
		response.Success(c, payload)
		return
	}
	if pendingSessionRequiresBindLogin(payload) {
		response.Success(c, payload)
		return
	}
	if !adoptionDecision.hasDecision() {
		adoptionRequired, _ := payload["adoption_required"].(bool)
		if adoptionRequired {
			response.Success(c, payload)
			return
		}
	}

	decisionReq := adoptionDecision
	if !decisionReq.hasDecision() {
		adoptDisplayName := false
		adoptAvatar := false
		decisionReq = oauthAdoptionDecisionRequest{
			AdoptDisplayName: &adoptDisplayName,
			AdoptAvatar:      &adoptAvatar,
		}
	}

	decision, err := h.ensurePendingOAuthAdoptionDecision(c, session.ID, decisionReq)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := h.applyPendingIdentityBinding(c.Request.Context(), session, decision, session.TargetUserID, false, strings.EqualFold(strings.TrimSpace(session.Intent), "bind_current_user")); err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("PENDING_AUTH_ADOPTION_APPLY_FAILED", "failed to apply oauth profile adoption").WithCause(err))
		return
	}

	if _, err := svc.ConsumeBrowserSession(c.Request.Context(), sessionToken, browserSessionKey); err != nil {
		clearCookies()
		response.ErrorFrom(c, err)
		return
	}

	if canIssueTokenPair {
		tokenPair, err := h.loginCases().GenerateTokenPair(c.Request.Context(), loginUser, "")
		if err != nil {
			clearCookies()
			response.InternalError(c, "Failed to generate token pair")
			return
		}
		h.loginCases().RecordSuccessfulLogin(c.Request.Context(), loginUser.ID)
		payload["access_token"] = tokenPair.AccessToken
		payload["refresh_token"] = tokenPair.RefreshToken
		payload["expires_in"] = tokenPair.ExpiresIn
		payload["token_type"] = "Bearer"
	}

	clearCookies()
	response.Success(c, payload)
}
