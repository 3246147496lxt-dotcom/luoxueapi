package quotaauth

import (
	"context"
	"crypto/hmac"
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
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	pairingTTL          = 10 * time.Minute
	accessTokenTTL      = 15 * time.Minute
	defaultRefreshTTL   = 30 * 24 * time.Hour
	maxRefreshTTL       = 90 * 24 * time.Hour
	maxPairingAttempts  = 8
	maxAccessTokenBytes = 8192
	refreshTokenPrefix  = "qvrt_"
	accessTokenType     = "at+jwt"
	refreshRecoveryTTL  = 10 * time.Minute
)

type Service struct {
	store      PairingStore
	repository Repository
	jwtKey     []byte
	refreshTTL time.Duration
	now        func() time.Time
}

func NewService(store PairingStore, repository Repository, cfg *config.Config) *Service {
	secret := ""
	refreshTTL := defaultRefreshTTL
	if cfg != nil {
		secret = cfg.JWT.Secret
		if cfg.JWT.RefreshTokenExpireDays > 0 {
			refreshTTL = time.Duration(cfg.JWT.RefreshTokenExpireDays) * 24 * time.Hour
			if refreshTTL > maxRefreshTTL {
				refreshTTL = maxRefreshTTL
			}
		}
	}
	return &Service{
		store:      store,
		repository: repository,
		jwtKey:     deriveJWTKey(secret),
		refreshTTL: refreshTTL,
		now:        time.Now,
	}
}

func (s *Service) CreatePairing(ctx context.Context, input CreatePairingInput) (*PairingChallenge, error) {
	input.ClientID = strings.TrimSpace(input.ClientID)
	input.Scope = strings.TrimSpace(input.Scope)
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	input.Architecture = normalizeArchitecture(input.Architecture)
	input.DeviceName = strings.TrimSpace(input.DeviceName)
	input.InstallationID = strings.TrimSpace(input.InstallationID)
	input.OSVersion = strings.TrimSpace(input.OSVersion)
	input.AppVersion = strings.TrimSpace(input.AppVersion)

	if input.ClientID != ClientID || input.Scope != ScopeRead ||
		input.CodeChallengeMethod != "S256" || !validCodeChallenge(input.CodeChallenge) ||
		input.InstallationID == "" || len(input.InstallationID) > 256 ||
		input.DeviceName == "" || len(input.DeviceName) > 100 ||
		(input.Platform != "macos" && input.Platform != "windows") ||
		(input.Architecture != "arm64" && input.Architecture != "x86_64") ||
		len(input.OSVersion) > 50 || len(input.AppVersion) > 50 {
		return nil, ErrInvalidAuthorizationRequest
	}
	if s == nil || s.store == nil || len(s.jwtKey) == 0 {
		return nil, ErrServiceUnavailable
	}

	installationHash := sha256.Sum256([]byte(input.InstallationID))
	for attempt := 0; attempt < maxPairingAttempts; attempt++ {
		deviceCode, err := randomBase64URL(32)
		if err != nil {
			return nil, ErrServiceUnavailable.WithCause(fmt.Errorf("generate quota viewer device code: %w", err))
		}
		userCode, err := randomUserCode()
		if err != nil {
			return nil, ErrServiceUnavailable.WithCause(fmt.Errorf("generate quota viewer user code: %w", err))
		}
		deviceHash := sha256.Sum256([]byte(deviceCode))
		expiresAt := s.now().UTC().Add(pairingTTL)
		pairing := Pairing{
			DeviceCodeHash: hex.EncodeToString(deviceHash[:]),
			UserCode:       userCode,
			CodeChallenge:  input.CodeChallenge,
			ClientID:       ClientID,
			Scope:          ScopeRead,
			InstallationHash: hex.EncodeToString(
				installationHash[:],
			),
			DeviceName:   input.DeviceName,
			Platform:     input.Platform,
			Architecture: input.Architecture,
			OSVersion:    input.OSVersion,
			AppVersion:   input.AppVersion,
			Status:       PairingStatusPending,
			ExpiresAt:    expiresAt,
		}
		if err := s.store.Create(ctx, pairing, pairingTTL); err != nil {
			if errors.Is(err, ErrPairingCollision) {
				continue
			}
			return nil, mapBackendError(err)
		}
		verificationURI := "/quota-viewer/authorize"
		return &PairingChallenge{
			DeviceCode:              deviceCode,
			UserCode:                userCode,
			VerificationURI:         verificationURI,
			VerificationURIComplete: verificationURI + "?user_code=" + url.QueryEscape(userCode),
			ClientID:                ClientID,
			Scope:                   ScopeRead,
			ExpiresIn:               int(pairingTTL.Seconds()),
			Interval:                5,
			ExpiresAt:               expiresAt,
		}, nil
	}
	return nil, ErrPairingCollision
}

func (s *Service) GetPairingPreview(ctx context.Context, userCode string) (*PairingPreview, error) {
	if s == nil || s.store == nil {
		return nil, ErrServiceUnavailable
	}
	pairing, err := s.store.GetByUserCode(ctx, normalizeUserCode(userCode))
	if err != nil {
		return nil, mapBackendError(err)
	}
	if pairing.Status != PairingStatusPending || pairing.ClientID != ClientID || pairing.Scope != ScopeRead {
		return nil, ErrPairingState
	}
	return &PairingPreview{
		UserCode:     pairing.UserCode,
		ClientID:     ClientID,
		Scope:        ScopeRead,
		DeviceName:   pairing.DeviceName,
		Platform:     pairing.Platform,
		Architecture: pairing.Architecture,
		OSVersion:    pairing.OSVersion,
		AppVersion:   pairing.AppVersion,
		ReadOnly:     true,
		ExpiresAt:    pairing.ExpiresAt,
	}, nil
}

func (s *Service) ApprovePairing(ctx context.Context, userID int64, userCode string) (*Device, error) {
	if userID <= 0 {
		return nil, ErrDeviceUnauthorized
	}
	if s == nil || s.store == nil || s.repository == nil || len(s.jwtKey) == 0 {
		return nil, ErrServiceUnavailable
	}
	userCode = normalizeUserCode(userCode)
	pairing, err := s.store.BeginApproval(ctx, userCode)
	if err != nil {
		return nil, mapBackendError(err)
	}
	if pairing.ClientID != ClientID || pairing.Scope != ScopeRead {
		_ = s.store.CancelApproval(ctx, userCode)
		return nil, ErrPairingState
	}
	device, err := s.repository.ReserveDevice(ctx, userID, *pairing)
	if err != nil {
		_ = s.store.CancelApproval(ctx, userCode)
		return nil, mapBackendError(err)
	}
	if err := s.store.FinishApproval(ctx, userCode, userID, device.ID); err != nil {
		_ = s.repository.DeletePendingDevice(ctx, device.ID)
		_ = s.store.CancelApproval(ctx, userCode)
		return nil, mapBackendError(err)
	}
	return device, nil
}

func (s *Service) ExchangePairing(ctx context.Context, clientID, deviceCode, codeVerifier string) (*TokenPair, error) {
	if strings.TrimSpace(clientID) != ClientID {
		return nil, ErrInvalidAuthorizationRequest
	}
	if s == nil || s.store == nil || s.repository == nil || len(s.jwtKey) == 0 {
		return nil, ErrServiceUnavailable
	}
	deviceCode = strings.TrimSpace(deviceCode)
	pairing, err := s.store.GetByDeviceCode(ctx, deviceCode)
	if err != nil {
		return nil, mapBackendError(err)
	}
	if pairing.ClientID != ClientID || pairing.Scope != ScopeRead {
		return nil, ErrPairingState
	}
	if !verifyPKCE(pairing.CodeChallenge, strings.TrimSpace(codeVerifier)) {
		return nil, ErrPKCEVerification
	}
	pairing, err = s.store.Consume(ctx, deviceCode)
	if err != nil {
		return nil, mapBackendError(err)
	}
	refreshToken, err := randomOpaqueToken(refreshTokenPrefix, 32)
	if err != nil {
		_ = s.store.RestoreConsumed(ctx, deviceCode)
		return nil, ErrServiceUnavailable.WithCause(fmt.Errorf("generate quota viewer refresh token: %w", err))
	}
	device, err := s.repository.ActivateDevice(
		ctx,
		pairing.DeviceID,
		uuid.NewString(),
		hashToken(refreshToken),
		s.now().UTC().Add(s.refreshTTL),
	)
	if err != nil {
		if !errors.Is(err, ErrActivationOutcomeUnknown) {
			_ = s.store.RestoreConsumed(ctx, deviceCode)
		}
		return nil, mapBackendError(err)
	}
	return s.issueTokenPair(device, refreshToken)
}

func (s *Service) RefreshSession(ctx context.Context, clientID, refreshToken string) (*TokenPair, error) {
	return s.RefreshSessionWithInput(ctx, RefreshSessionInput{
		ClientID:     clientID,
		RefreshToken: refreshToken,
	})
}

func (s *Service) RefreshSessionWithInput(ctx context.Context, input RefreshSessionInput) (*TokenPair, error) {
	if strings.TrimSpace(input.ClientID) != ClientID {
		return nil, ErrInvalidAuthorizationRequest
	}
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if !validRefreshToken(refreshToken) {
		return nil, ErrRefreshToken
	}
	if s == nil || s.repository == nil || len(s.jwtKey) == 0 {
		return nil, ErrServiceUnavailable
	}

	hasRotationID := input.RotationID != nil
	hasCandidate := input.CandidateRefreshToken != nil
	if hasRotationID != hasCandidate {
		return nil, ErrInvalidAuthorizationRequest
	}

	replacement := ""
	rotationID := ""
	if hasRotationID {
		rotationID = *input.RotationID
		replacement = *input.CandidateRefreshToken
		parsedRotationID, err := uuid.Parse(rotationID)
		if err != nil || parsedRotationID.Version() != uuid.Version(4) ||
			parsedRotationID.Variant() != uuid.RFC4122 ||
			parsedRotationID.String() != rotationID ||
			!validCandidateRefreshToken(replacement) ||
			subtle.ConstantTimeCompare([]byte(replacement), []byte(refreshToken)) == 1 {
			return nil, ErrInvalidAuthorizationRequest
		}
	} else {
		var err error
		replacement, err = randomOpaqueToken(refreshTokenPrefix, 32)
		if err != nil {
			return nil, ErrServiceUnavailable.WithCause(fmt.Errorf("generate replacement quota viewer refresh token: %w", err))
		}
	}

	now := s.now().UTC()
	outcome, err := s.repository.RotateSession(ctx, RefreshRotation{
		PredecessorHash:   hashToken(refreshToken),
		ReplacementHash:   hashToken(replacement),
		ReplacementExpiry: now.Add(s.refreshTTL),
		RotationID:        rotationID,
		RecoveryExpiry:    now.Add(refreshRecoveryTTL),
	})
	if err != nil {
		return nil, mapBackendError(err)
	}
	if outcome == nil || outcome.Device == nil {
		return nil, ErrServiceUnavailable
	}
	pair, err := s.issueTokenPair(outcome.Device, replacement)
	if err != nil {
		return nil, err
	}
	if hasRotationID {
		if outcome.Result != RefreshRotationCommitted && outcome.Result != RefreshRotationRecovered {
			return nil, ErrServiceUnavailable
		}
		pair.RefreshProtocol = RefreshProtocolCandidateV1
		pair.RotationID = rotationID
		pair.RotationResult = outcome.Result
	}
	return pair, nil
}

func (s *Service) AuthenticateAccessToken(ctx context.Context, tokenString string) (*Subject, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" || len(tokenString) > maxAccessTokenBytes {
		return nil, ErrAccessTokenInvalid
	}
	if s == nil || s.repository == nil || len(s.jwtKey) == 0 {
		return nil, ErrServiceUnavailable
	}

	claims := new(accessClaims)
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
		jwt.WithAudience(Audience),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	token, err := parser.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return s.jwtKey, nil
	})
	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenSignatureInvalid):
			return nil, ErrAccessTokenInvalid
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrAccessTokenExpired
		case errors.Is(err, jwt.ErrTokenInvalidAudience):
			return nil, ErrAccessTokenAudience
		default:
			return nil, ErrAccessTokenInvalid
		}
	}
	if !token.Valid || token.Header["typ"] != accessTokenType ||
		claims.DeviceID <= 0 || claims.TokenVersion <= 0 ||
		claims.Subject == "" || claims.ID == "" ||
		claims.ExpiresAt == nil || claims.IssuedAt == nil || claims.NotBefore == nil {
		return nil, ErrAccessTokenInvalid
	}
	lifetime := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if lifetime <= 0 || lifetime > accessTokenTTL {
		return nil, ErrAccessTokenInvalid
	}
	if claims.ClientID != ClientID {
		return nil, ErrAccessTokenClient
	}
	if claims.Scope != ScopeRead {
		return nil, ErrAccessTokenScope
	}

	device, err := s.repository.GetAuthorizedDevice(ctx, claims.DeviceID, claims.TokenVersion)
	if err != nil {
		return nil, mapBackendError(err)
	}
	if device.PublicID == "" ||
		subtle.ConstantTimeCompare([]byte(device.PublicID), []byte(claims.Subject)) != 1 ||
		device.ClientID != ClientID || device.Scope != ScopeRead {
		return nil, ErrAccessTokenInvalid
	}
	return &Subject{
		DeviceID:       device.ID,
		DevicePublicID: device.PublicID,
		UserID:         device.UserID,
		TokenVersion:   device.TokenVersion,
		ClientID:       ClientID,
		Scopes:         []string{ScopeRead},
	}, nil
}

func (s *Service) ListDevices(ctx context.Context, userID int64) ([]Device, error) {
	if userID <= 0 {
		return nil, ErrDeviceUnauthorized
	}
	if s == nil || s.repository == nil {
		return nil, ErrServiceUnavailable
	}
	devices, err := s.repository.ListDevices(ctx, userID)
	if err != nil {
		return nil, mapBackendError(err)
	}
	return devices, nil
}

func (s *Service) RevokeDevice(ctx context.Context, userID int64, publicID string) (*Device, error) {
	if userID <= 0 {
		return nil, ErrDeviceUnauthorized
	}
	publicID = strings.TrimSpace(publicID)
	if _, err := uuid.Parse(publicID); err != nil {
		return nil, ErrDeviceNotFound
	}
	if s == nil || s.repository == nil {
		return nil, ErrServiceUnavailable
	}
	device, err := s.repository.RevokeDevice(ctx, userID, publicID)
	if err != nil {
		return nil, mapBackendError(err)
	}
	return device, nil
}

type accessClaims struct {
	DeviceID     int64  `json:"device_id"`
	TokenVersion int64  `json:"token_version"`
	ClientID     string `json:"client_id"`
	Scope        string `json:"scope"`
	jwt.RegisteredClaims
}

func (s *Service) issueTokenPair(device *Device, refreshToken string) (*TokenPair, error) {
	if s == nil || len(s.jwtKey) == 0 {
		return nil, ErrServiceUnavailable
	}
	if device == nil || device.ID <= 0 || device.UserID <= 0 || device.PublicID == "" ||
		device.Status != DeviceStatusActive || device.ClientID != ClientID || device.Scope != ScopeRead ||
		!validRefreshToken(refreshToken) {
		return nil, ErrDeviceUnauthorized
	}
	now := s.now().UTC()
	claims := accessClaims{
		DeviceID:     device.ID,
		TokenVersion: device.TokenVersion,
		ClientID:     ClientID,
		Scope:        ScopeRead,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   device.PublicID,
			Audience:  jwt.ClaimStrings{Audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["typ"] = accessTokenType
	accessToken, err := token.SignedString(s.jwtKey)
	if err != nil {
		return nil, ErrServiceUnavailable.WithCause(fmt.Errorf("sign quota viewer access token: %w", err))
	}
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTokenTTL.Seconds()),
		ClientID:     ClientID,
		Scope:        ScopeRead,
		Device:       *device,
	}, nil
}

func deriveJWTKey(secret string) []byte {
	if len([]byte(strings.TrimSpace(secret))) < 32 {
		return nil
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("luoxue-quota-viewer/access-token/v1"))
	return mac.Sum(nil)
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

func normalizeArchitecture(architecture string) string {
	architecture = strings.ToLower(strings.TrimSpace(architecture))
	if architecture == "amd64" {
		return "x86_64"
	}
	return architecture
}

func randomUserCode() (string, error) {
	const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	buf := make([]byte, len(random))
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

func validRefreshToken(token string) bool {
	return strings.HasPrefix(token, refreshTokenPrefix) &&
		len(token) == len(refreshTokenPrefix)+(sha256.Size*2)
}

func validCandidateRefreshToken(token string) bool {
	if !validRefreshToken(token) {
		return false
	}
	for _, char := range token[len(refreshTokenPrefix):] {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func hashToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func mapBackendError(err error) error {
	if err == nil {
		return nil
	}
	var appErr *infraerrors.ApplicationError
	if errors.As(err, &appErr) {
		return err
	}
	return ErrServiceUnavailable.WithCause(err)
}
