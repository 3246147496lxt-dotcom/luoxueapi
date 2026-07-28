package desktop

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	pairingTTL            = 10 * time.Minute
	accessTokenTTL        = 15 * time.Minute
	defaultRefreshTTL     = 30 * 24 * time.Hour
	maxPairingAttempts    = 8
	maxManagedKeyAttempts = 4
)

var desktopScopes = []string{ScopeProfileRead, ScopeRoutesRead, ScopeManagedKeyWrite, ScopeDiagnosticsWrite}

type Service struct {
	store            PairingStore
	repository       Repository
	routes           RouteCatalog
	invalidator      KeyInvalidator
	jwtKey           []byte
	apiKeyPrefix     string
	refreshTTL       time.Duration
	now              func() time.Time
	diagnostics      DiagnosticsRepository
	diagnosticKey    []byte
	diagnosticKeyErr error
	desktopConfig    config.DesktopConfig
}

func NewService(store PairingStore, repository Repository, routes RouteCatalog, invalidator KeyInvalidator, cfg *config.Config) *Service {
	secret := ""
	prefix := "sk-"
	refreshTTL := defaultRefreshTTL
	if cfg != nil {
		secret = cfg.JWT.Secret
		if cfg.Default.APIKeyPrefix != "" {
			prefix = cfg.Default.APIKeyPrefix
		}
		if cfg.JWT.RefreshTokenExpireDays > 0 {
			refreshTTL = time.Duration(cfg.JWT.RefreshTokenExpireDays) * 24 * time.Hour
		}
	}
	key := sha256.Sum256([]byte("luoxue-desktop-v1\x00" + secret))
	svc := &Service{
		store: store, repository: repository, routes: routes, invalidator: invalidator,
		jwtKey: key[:], apiKeyPrefix: prefix, refreshTTL: refreshTTL, now: time.Now,
	}
	if diagnostics, ok := repository.(DiagnosticsRepository); ok {
		svc.diagnostics = diagnostics
	}
	if cfg != nil {
		svc.desktopConfig = cfg.Desktop
	}
	svc.diagnosticKey, svc.diagnosticKeyErr = resolveDiagnosticKey(cfg)
	return svc
}

func (s *Service) CreatePairing(ctx context.Context, input CreatePairingInput) (*PairingChallenge, error) {
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	input.Architecture = strings.ToLower(strings.TrimSpace(input.Architecture))
	input.DeviceName = strings.TrimSpace(input.DeviceName)
	input.InstallationID = strings.TrimSpace(input.InstallationID)
	if input.CodeChallengeMethod != "S256" || !validCodeChallenge(input.CodeChallenge) ||
		input.InstallationID == "" || len(input.InstallationID) > 256 ||
		input.DeviceName == "" || len(input.DeviceName) > 100 || input.Platform != "macos" ||
		(input.Architecture != "arm64" && input.Architecture != "x86_64") ||
		len(input.OSVersion) > 50 || len(input.AppVersion) > 50 {
		return nil, ErrInvalidPairingRequest
	}

	installationHash := sha256.Sum256([]byte(input.InstallationID))
	for attempt := 0; attempt < maxPairingAttempts; attempt++ {
		deviceCode, err := randomBase64URL(32)
		if err != nil {
			return nil, fmt.Errorf("generate device code: %w", err)
		}
		userCode, err := randomUserCode()
		if err != nil {
			return nil, fmt.Errorf("generate user code: %w", err)
		}
		deviceHash := sha256.Sum256([]byte(deviceCode))
		expiresAt := s.now().Add(pairingTTL)
		pairing := Pairing{
			DeviceCodeHash: hex.EncodeToString(deviceHash[:]), UserCode: userCode,
			CodeChallenge: input.CodeChallenge, InstallationHash: hex.EncodeToString(installationHash[:]),
			DeviceName: input.DeviceName, Platform: input.Platform, Architecture: input.Architecture,
			OSVersion: input.OSVersion, AppVersion: input.AppVersion,
			Status: PairingStatusPending, ExpiresAt: expiresAt,
		}
		if err := s.store.Create(ctx, pairing, pairingTTL); err != nil {
			if errors.Is(err, ErrPairingCollision) {
				continue
			}
			return nil, err
		}
		verificationURI := "/desktop/authorize"
		return &PairingChallenge{
			DeviceCode: deviceCode, UserCode: userCode, VerificationURI: verificationURI,
			VerificationURIComplete: verificationURI + "?user_code=" + url.QueryEscape(userCode),
			ExpiresIn:               int(pairingTTL.Seconds()), Interval: 5, ExpiresAt: expiresAt,
		}, nil
	}
	return nil, ErrPairingCollision
}

func (s *Service) GetPairingPreview(ctx context.Context, userCode string) (*PairingPreview, error) {
	pairing, err := s.store.GetByUserCode(ctx, normalizeUserCode(userCode))
	if err != nil {
		return nil, err
	}
	if pairing.Status != PairingStatusPending {
		return nil, ErrPairingState
	}
	return &PairingPreview{
		UserCode: pairing.UserCode, DeviceName: pairing.DeviceName, Platform: pairing.Platform,
		Architecture: pairing.Architecture, OSVersion: pairing.OSVersion, AppVersion: pairing.AppVersion,
		RequestedScopes: append([]string(nil), desktopScopes...), ExpiresAt: pairing.ExpiresAt,
	}, nil
}

func (s *Service) ApprovePairing(ctx context.Context, userID int64, userCode string) (*Device, error) {
	userCode = normalizeUserCode(userCode)
	pairing, err := s.store.BeginApproval(ctx, userCode)
	if err != nil {
		return nil, err
	}
	pairing.ReleaseChannel = s.releaseChannelForUser(userID)
	device, err := s.repository.ReserveDevice(ctx, userID, *pairing)
	if err != nil {
		_ = s.store.CancelApproval(ctx, userCode)
		return nil, err
	}
	if err := s.store.FinishApproval(ctx, userCode, userID, device.ID); err != nil {
		_ = s.repository.DeletePendingDevice(ctx, device.ID)
		_ = s.store.CancelApproval(ctx, userCode)
		return nil, err
	}
	return device, nil
}

func (s *Service) ExchangePairing(ctx context.Context, deviceCode, codeVerifier string) (*TokenPair, error) {
	pairing, err := s.store.GetByDeviceCode(ctx, deviceCode)
	if err != nil {
		return nil, err
	}
	if !verifyPKCE(pairing.CodeChallenge, codeVerifier) {
		return nil, ErrPKCEVerification
	}
	pairing, err = s.store.Consume(ctx, deviceCode)
	if err != nil {
		return nil, err
	}
	refreshToken, err := randomOpaqueToken("drt_", 32)
	if err != nil {
		_ = s.store.RestoreConsumed(ctx, deviceCode)
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshHash := hashToken(refreshToken)
	device, err := s.repository.ActivateDevice(ctx, pairing.DeviceID, uuid.NewString(), refreshHash, s.now().Add(s.refreshTTL))
	if err != nil {
		if !errors.Is(err, ErrActivationOutcomeUnknown) {
			_ = s.store.RestoreConsumed(ctx, deviceCode)
		}
		// An ambiguous commit stays consumed rather than risk activating a
		// second durable refresh-token family.
		return nil, err
	}
	return s.issueTokenPair(device, refreshToken)
}

func (s *Service) RefreshSession(ctx context.Context, refreshToken string) (*TokenPair, error) {
	if !strings.HasPrefix(refreshToken, "drt_") || len(refreshToken) > 256 {
		return nil, ErrRefreshToken
	}
	replacement, err := randomOpaqueToken("drt_", 32)
	if err != nil {
		return nil, fmt.Errorf("generate replacement refresh token: %w", err)
	}
	device, err := s.repository.RotateSession(ctx, hashToken(refreshToken), hashToken(replacement), s.now().Add(s.refreshTTL))
	if err != nil {
		return nil, err
	}
	return s.issueTokenPair(device, replacement)
}

func (s *Service) AuthenticateAccessToken(ctx context.Context, tokenString string) (*DesktopSubject, error) {
	if tokenString == "" || len(tokenString) > 8192 {
		return nil, ErrDesktopToken
	}
	claims := new(desktopClaims)
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithAudience(DesktopAudience),
		jwt.WithIssuer(DesktopIssuer),
		jwt.WithExpirationRequired(),
	)
	token, err := parser.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) { return s.jwtKey, nil })
	if err != nil || !token.Valid || claims.DeviceID <= 0 || claims.UserID <= 0 || claims.TokenVersion <= 0 || claims.ID == "" {
		return nil, ErrDesktopToken
	}
	device, err := s.repository.GetAuthorizedDevice(ctx, claims.DeviceID, claims.UserID, claims.TokenVersion)
	if err != nil {
		return nil, err
	}
	if device.PublicID == "" || subtle.ConstantTimeCompare([]byte(device.PublicID), []byte(claims.Subject)) != 1 {
		return nil, ErrDesktopToken
	}
	return &DesktopSubject{
		DeviceID: device.ID, DevicePublicID: device.PublicID, UserID: device.UserID,
		TokenVersion: device.TokenVersion, Scopes: append([]string(nil), claims.Scopes...),
	}, nil
}

func (s *Service) GetDevice(ctx context.Context, subject DesktopSubject) (*Device, error) {
	return s.repository.GetAuthorizedDevice(ctx, subject.DeviceID, subject.UserID, subject.TokenVersion)
}

func (s *Service) ListDevices(ctx context.Context, userID int64) ([]Device, error) {
	return s.repository.ListDevices(ctx, userID)
}

func (s *Service) RenameDevice(ctx context.Context, userID int64, publicID, name string) (*Device, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 {
		return nil, ErrInvalidDeviceName
	}
	publicID = strings.TrimSpace(publicID)
	if _, err := uuid.Parse(publicID); err != nil {
		return nil, ErrDeviceNotFound
	}
	return s.repository.RenameDevice(ctx, userID, publicID, name)
}

func (s *Service) Heartbeat(ctx context.Context, subject DesktopSubject) (*Device, error) {
	if !subject.HasScope(ScopeProfileRead) {
		return nil, ErrDesktopScope
	}
	return s.repository.HeartbeatDevice(ctx, subject.DeviceID, subject.UserID, subject.TokenVersion)
}

func (s *Service) ActivateCurrentDevice(ctx context.Context, subject DesktopSubject) (*Device, error) {
	if !subject.HasScope(ScopeProfileRead) {
		return nil, ErrDesktopScope
	}
	return s.repository.MarkDeviceActivated(
		ctx, subject.DeviceID, subject.UserID, subject.TokenVersion, s.now(),
	)
}

func (s *Service) RevokeDevice(ctx context.Context, userID int64, publicID string) (*Device, error) {
	return s.revokeDevice(ctx, userID, strings.TrimSpace(publicID))
}

func (s *Service) RevokeCurrentDevice(ctx context.Context, subject DesktopSubject) (*Device, error) {
	if !subject.HasScope(ScopeProfileRead) {
		return nil, ErrDesktopScope
	}
	return s.revokeDevice(ctx, subject.UserID, subject.DevicePublicID)
}

func (s *Service) revokeDevice(ctx context.Context, userID int64, publicID string) (*Device, error) {
	if _, err := uuid.Parse(publicID); err != nil {
		return nil, ErrDeviceNotFound
	}
	revocation, err := s.repository.RevokeDevice(ctx, userID, publicID)
	if err != nil {
		return nil, err
	}
	if s.invalidator != nil {
		invalidationCtx := context.WithoutCancel(ctx)
		var invalidationErrors []error
		for _, key := range revocation.ManagedKeys {
			if err := s.invalidator.InvalidateManagedKey(invalidationCtx, key); err != nil {
				invalidationErrors = append(invalidationErrors, err)
			}
		}
		if err := errors.Join(invalidationErrors...); err != nil {
			return nil, ErrManagedKeyCache.WithCause(err)
		}
	}
	return &revocation.Device, nil
}

func (s *Service) ListRoutes(ctx context.Context, subject DesktopSubject) ([]Route, error) {
	if !subject.HasScope(ScopeRoutesRead) {
		return nil, ErrDesktopScope
	}
	return s.routes.ListRoutes(ctx, subject.UserID)
}

func (s *Service) GetTodayUsage(ctx context.Context, subject DesktopSubject) (*TodayUsage, error) {
	if !subject.HasScope(ScopeProfileRead) {
		return nil, ErrDesktopScope
	}
	start := timezone.StartOfDay(s.now())
	end := start.AddDate(0, 0, 1)
	return s.repository.GetTodayUsage(ctx, subject.DeviceID, subject.UserID, start, end)
}

func (s *Service) EnsureManagedKey(ctx context.Context, subject DesktopSubject, groupID int64) (*ManagedKey, error) {
	if !subject.HasScope(ScopeManagedKeyWrite) || groupID <= 0 {
		return nil, ErrDesktopScope
	}
	routes, err := s.routes.ListRoutes(ctx, subject.UserID)
	if err != nil {
		return nil, err
	}
	allowed := false
	for _, route := range routes {
		if route.GroupID == groupID {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, ErrRouteUnavailable
	}
	for attempt := 0; attempt < maxManagedKeyAttempts; attempt++ {
		generated, err := randomOpaqueToken(s.apiKeyPrefix, 32)
		if err != nil {
			return nil, fmt.Errorf("generate managed key: %w", err)
		}
		managed, err := s.repository.EnsureManagedKey(ctx, subject.DeviceID, subject.UserID, groupID, generated)
		if errors.Is(err, ErrManagedKeyCollision) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if s.invalidator != nil {
			if err := s.invalidator.InvalidateManagedKey(ctx, managed.Key); err != nil {
				return nil, ErrManagedKeyCache.WithCause(err)
			}
		}
		return managed, nil
	}
	return nil, ErrManagedKeyCollision
}

type desktopClaims struct {
	DeviceID     int64    `json:"device_id"`
	UserID       int64    `json:"user_id"`
	TokenVersion int64    `json:"token_version"`
	Scopes       []string `json:"scope"`
	jwt.RegisteredClaims
}

func (s *Service) issueTokenPair(device *Device, refreshToken string) (*TokenPair, error) {
	if device == nil || device.ID <= 0 || device.UserID <= 0 || device.PublicID == "" {
		return nil, ErrDeviceUnauthorized
	}
	now := s.now()
	claims := desktopClaims{
		DeviceID: device.ID, UserID: device.UserID, TokenVersion: device.TokenVersion,
		Scopes: append([]string(nil), desktopScopes...),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: DesktopIssuer, Subject: device.PublicID,
			Audience:  jwt.ClaimStrings{DesktopAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)), IssuedAt: jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now), ID: uuid.NewString(),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtKey)
	if err != nil {
		return nil, fmt.Errorf("sign desktop access token: %w", err)
	}
	return &TokenPair{
		AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer",
		ExpiresIn: int(accessTokenTTL.Seconds()), Scope: append([]string(nil), desktopScopes...), Device: *device,
	}, nil
}

func validCodeChallenge(challenge string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(challenge)
	return err == nil && len(decoded) == sha256.Size && len(challenge) == 43
}

func verifyPKCE(challenge, verifier string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	digest := sha256.Sum256([]byte(verifier))
	actual := base64.RawURLEncoding.EncodeToString(digest[:])
	return subtle.ConstantTimeCompare([]byte(actual), []byte(challenge)) == 1
}

func normalizeUserCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, " ", "")
	if len(code) == 8 && !strings.Contains(code, "-") {
		code = code[:4] + "-" + code[4:]
	}
	return code
}

func randomUserCode() (string, error) {
	const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	buf := make([]byte, 8)
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = alphabet[int(random[i])%len(alphabet)]
	}
	return string(buf[:4]) + "-" + string(buf[4:]), nil
}

func randomBase64URL(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomOpaqueToken(prefix string, size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
