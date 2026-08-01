package quotaauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const quotaAuthTestDevicePublicID = "41d69f86-6f59-4d70-8f06-18952934a81e"

type memoryQuotaPairingStore struct {
	pairing     Pairing
	consumed    bool
	consumeCall int
}

func (s *memoryQuotaPairingStore) Create(_ context.Context, pairing Pairing, _ time.Duration) error {
	s.pairing = pairing
	return nil
}

func (s *memoryQuotaPairingStore) GetByUserCode(_ context.Context, _ string) (*Pairing, error) {
	pairing := s.pairing
	return &pairing, nil
}

func (s *memoryQuotaPairingStore) GetByDeviceCode(_ context.Context, _ string) (*Pairing, error) {
	pairing := s.pairing
	return &pairing, nil
}

func (s *memoryQuotaPairingStore) BeginApproval(_ context.Context, _ string) (*Pairing, error) {
	if s.pairing.Status != PairingStatusPending {
		return nil, ErrPairingState
	}
	s.pairing.Status = PairingStatusApproving
	pairing := s.pairing
	return &pairing, nil
}

func (s *memoryQuotaPairingStore) FinishApproval(_ context.Context, _ string, userID, deviceID int64) error {
	s.pairing.Status = PairingStatusApproved
	s.pairing.UserID = userID
	s.pairing.DeviceID = deviceID
	return nil
}

func (s *memoryQuotaPairingStore) CancelApproval(_ context.Context, _ string) error {
	s.pairing.Status = PairingStatusPending
	return nil
}

func (s *memoryQuotaPairingStore) Consume(_ context.Context, _ string) (*Pairing, error) {
	s.consumeCall++
	if s.consumed {
		return nil, ErrPairingConsumed
	}
	if s.pairing.Status != PairingStatusApproved {
		return nil, ErrPairingState
	}
	s.consumed = true
	s.pairing.Status = PairingStatusConsumed
	pairing := s.pairing
	return &pairing, nil
}

func (s *memoryQuotaPairingStore) RestoreConsumed(_ context.Context, _ string) error {
	s.consumed = false
	s.pairing.Status = PairingStatusApproved
	return nil
}

type memoryQuotaAuthRepository struct {
	device              Device
	authorizedErr       error
	rotateErr           error
	activatedHash       string
	replacementHash     string
	replacementExpires  time.Time
	recoveryExpires     time.Time
	rotationID          string
	rotationResult      string
	revoked             bool
	pendingDeleteCalled bool
}

func newMemoryQuotaAuthRepository() *memoryQuotaAuthRepository {
	return &memoryQuotaAuthRepository{
		device: Device{
			ID:           41,
			PublicID:     quotaAuthTestDevicePublicID,
			UserID:       7,
			ClientID:     ClientID,
			Scope:        ScopeRead,
			Name:         "Work laptop",
			Platform:     "macos",
			Architecture: "arm64",
			Status:       DeviceStatusActive,
			TokenVersion: 1,
		},
	}
}

func (r *memoryQuotaAuthRepository) ReserveDevice(_ context.Context, userID int64, pairing Pairing) (*Device, error) {
	r.device.UserID = userID
	r.device.Name = pairing.DeviceName
	r.device.Platform = pairing.Platform
	r.device.Architecture = pairing.Architecture
	r.device.Status = DeviceStatusPending
	device := r.device
	return &device, nil
}

func (r *memoryQuotaAuthRepository) DeletePendingDevice(_ context.Context, _ int64) error {
	r.pendingDeleteCalled = true
	return nil
}

func (r *memoryQuotaAuthRepository) ActivateDevice(
	_ context.Context,
	deviceID int64,
	_ string,
	refreshTokenHash string,
	_ time.Time,
) (*Device, error) {
	if deviceID != r.device.ID {
		return nil, ErrDeviceUnauthorized
	}
	r.activatedHash = refreshTokenHash
	r.device.Status = DeviceStatusActive
	device := r.device
	return &device, nil
}

func (r *memoryQuotaAuthRepository) RotateSession(
	_ context.Context,
	rotation RefreshRotation,
) (*RefreshRotationOutcome, error) {
	if r.rotateErr != nil {
		return nil, r.rotateErr
	}
	r.replacementHash = rotation.ReplacementHash
	r.replacementExpires = rotation.ReplacementExpiry
	r.recoveryExpires = rotation.RecoveryExpiry
	r.rotationID = rotation.RotationID
	device := r.device
	result := r.rotationResult
	if result == "" {
		result = RefreshRotationCommitted
	}
	return &RefreshRotationOutcome{Device: &device, Result: result}, nil
}

func (r *memoryQuotaAuthRepository) GetAuthorizedDevice(
	_ context.Context,
	deviceID, tokenVersion int64,
) (*Device, error) {
	if r.authorizedErr != nil {
		return nil, r.authorizedErr
	}
	if r.revoked || r.device.Status != DeviceStatusActive ||
		deviceID != r.device.ID || tokenVersion != r.device.TokenVersion {
		return nil, ErrDeviceUnauthorized
	}
	device := r.device
	return &device, nil
}

func (r *memoryQuotaAuthRepository) ListDevices(_ context.Context, userID int64) ([]Device, error) {
	if userID != r.device.UserID {
		return []Device{}, nil
	}
	return []Device{r.device}, nil
}

func (r *memoryQuotaAuthRepository) RevokeDevice(
	_ context.Context,
	userID int64,
	publicID string,
) (*Device, error) {
	if userID != r.device.UserID || publicID != r.device.PublicID {
		return nil, ErrDeviceNotFound
	}
	if !r.revoked {
		r.device.Status = DeviceStatusRevoked
		r.device.TokenVersion++
		r.revoked = true
	}
	device := r.device
	return &device, nil
}

func TestQuotaAccessTokenHasExactReadOnlyAuthorizationDomain(t *testing.T) {
	repo := newMemoryQuotaAuthRepository()
	cfg := &config.Config{JWT: config.JWTConfig{Secret: strings.Repeat("w", 40)}}
	service := NewService(nil, repo, cfg)
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	service.now = func() time.Time { return now }

	pair, err := service.issueTokenPair(&repo.device, refreshTokenPrefix+strings.Repeat("a", 64))
	require.NoError(t, err)
	require.Equal(t, 15*60, pair.ExpiresIn)
	require.Equal(t, ClientID, pair.ClientID)
	require.Equal(t, ScopeRead, pair.Scope)

	claims := new(accessClaims)
	token, err := jwt.ParseWithClaims(
		pair.AccessToken,
		claims,
		func(token *jwt.Token) (any, error) {
			require.Equal(t, jwt.SigningMethodHS256, token.Method)
			require.Equal(t, accessTokenType, token.Header["typ"])
			return service.jwtKey, nil
		},
		jwt.WithAudience(Audience),
		jwt.WithIssuer(Issuer),
	)
	require.NoError(t, err)
	require.True(t, token.Valid)
	require.Equal(t, ClientID, claims.ClientID)
	require.Equal(t, ScopeRead, claims.Scope)
	require.Equal(t, quotaAuthTestDevicePublicID, claims.Subject)
	require.Equal(t, accessTokenTTL, claims.ExpiresAt.Sub(claims.IssuedAt.Time))

	payload := map[string]any{}
	parts := strings.Split(pair.AccessToken, ".")
	require.Len(t, parts, 3)
	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(decoded, &payload))
	require.NotContains(t, payload, "user_id")
	require.NotContains(t, payload, "email")
	require.NotContains(t, payload, "role")

	subject, err := service.AuthenticateAccessToken(context.Background(), pair.AccessToken)
	require.NoError(t, err)
	require.Equal(t, int64(7), subject.UserID, "user ownership must come from the authorized device record")
	require.Equal(t, []string{ScopeRead}, subject.Scopes)
}

func TestQuotaAccessTokenCannotValidateInWebOrDesktopSigningDomains(t *testing.T) {
	secret := strings.Repeat("s", 40)
	repo := newMemoryQuotaAuthRepository()
	service := NewService(nil, repo, &config.Config{JWT: config.JWTConfig{Secret: secret}})
	pair, err := service.issueTokenPair(&repo.device, refreshTokenPrefix+strings.Repeat("b", 64))
	require.NoError(t, err)

	_, err = jwt.Parse(pair.AccessToken, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	require.Error(t, err, "quota token must fail under the website JWT key")

	desktopKey := sha256.Sum256([]byte("luoxue-desktop-v1\x00" + secret))
	_, err = jwt.Parse(pair.AccessToken, func(*jwt.Token) (any, error) {
		return desktopKey[:], nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	require.Error(t, err, "quota token must fail under the Desktop JWT key")
}

func TestQuotaAccessTokenRejectsWrongAudienceClientScopeAndExpiry(t *testing.T) {
	repo := newMemoryQuotaAuthRepository()
	service := NewService(nil, repo, &config.Config{JWT: config.JWTConfig{Secret: strings.Repeat("t", 40)}})
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	service.now = func() time.Time { return now }

	tests := []struct {
		name   string
		mutate func(*accessClaims)
		want   error
	}{
		{
			name: "wrong audience",
			mutate: func(claims *accessClaims) {
				claims.Audience = jwt.ClaimStrings{"another-api"}
			},
			want: ErrAccessTokenAudience,
		},
		{
			name: "wrong issuer",
			mutate: func(claims *accessClaims) {
				claims.Issuer = "another-issuer"
			},
			want: ErrAccessTokenInvalid,
		},
		{
			name: "wrong client",
			mutate: func(claims *accessClaims) {
				claims.ClientID = "luoxue-desktop"
			},
			want: ErrAccessTokenClient,
		},
		{
			name: "missing scope",
			mutate: func(claims *accessClaims) {
				claims.Scope = ""
			},
			want: ErrAccessTokenScope,
		},
		{
			name: "extra scope",
			mutate: func(claims *accessClaims) {
				claims.Scope = ScopeRead + " managed_keys:write"
			},
			want: ErrAccessTokenScope,
		},
		{
			name: "expired",
			mutate: func(claims *accessClaims) {
				claims.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Second))
				claims.IssuedAt = jwt.NewNumericDate(now.Add(-accessTokenTTL - time.Second))
				claims.NotBefore = jwt.NewNumericDate(now.Add(-accessTokenTTL - time.Second))
			},
			want: ErrAccessTokenExpired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claims := validQuotaClaims(now, &repo.device)
			test.mutate(&claims)
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			token.Header["typ"] = accessTokenType
			raw, err := token.SignedString(service.jwtKey)
			require.NoError(t, err)

			_, err = service.AuthenticateAccessToken(context.Background(), raw)
			require.ErrorIs(t, err, test.want)
			if test.want == ErrAccessTokenAudience || test.want == ErrAccessTokenClient || test.want == ErrAccessTokenScope {
				require.Equal(t, 403, infraerrors.Code(err))
			} else {
				require.Equal(t, 401, infraerrors.Code(err))
			}
		})
	}
}

func TestQuotaAccessTokenRequiresCurrentActiveDeviceAndFailsUnavailableClosed(t *testing.T) {
	repo := newMemoryQuotaAuthRepository()
	service := NewService(nil, repo, &config.Config{JWT: config.JWTConfig{Secret: strings.Repeat("u", 40)}})
	pair, err := service.issueTokenPair(&repo.device, refreshTokenPrefix+strings.Repeat("c", 64))
	require.NoError(t, err)

	repo.revoked = true
	_, err = service.AuthenticateAccessToken(context.Background(), pair.AccessToken)
	require.ErrorIs(t, err, ErrDeviceUnauthorized)

	repo.revoked = false
	repo.authorizedErr = errors.New("database unavailable")
	_, err = service.AuthenticateAccessToken(context.Background(), pair.AccessToken)
	require.ErrorIs(t, err, ErrServiceUnavailable)
	require.Equal(t, 503, infraerrors.Code(err))
}

func TestQuotaPairingRequiresExactClientScopeAndPKCE(t *testing.T) {
	store := new(memoryQuotaPairingStore)
	repo := newMemoryQuotaAuthRepository()
	service := NewService(store, repo, &config.Config{JWT: config.JWTConfig{Secret: strings.Repeat("v", 40)}})
	verifier := strings.Repeat("z", 48)
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	input := CreatePairingInput{
		ClientID:            ClientID,
		Scope:               ScopeRead,
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		InstallationID:      uuid.NewString(),
		DeviceName:          "Windows workstation",
		Platform:            "windows",
		Architecture:        "amd64",
		OSVersion:           "11",
		AppVersion:          "1.0.0",
	}

	badClient := input
	badClient.ClientID = "luoxue-desktop"
	_, err := service.CreatePairing(context.Background(), badClient)
	require.ErrorIs(t, err, ErrInvalidAuthorizationRequest)

	badScope := input
	badScope.Scope = ScopeRead + " managed_keys:write"
	_, err = service.CreatePairing(context.Background(), badScope)
	require.ErrorIs(t, err, ErrInvalidAuthorizationRequest)

	challengeResult, err := service.CreatePairing(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, ClientID, challengeResult.ClientID)
	require.Equal(t, ScopeRead, challengeResult.Scope)
	require.Equal(t, "x86_64", store.pairing.Architecture)

	_, err = service.ApprovePairing(context.Background(), 7, challengeResult.UserCode)
	require.NoError(t, err)

	_, err = service.ExchangePairing(context.Background(), ClientID, challengeResult.DeviceCode, "wrong-verifier")
	require.ErrorIs(t, err, ErrPKCEVerification)
	require.Zero(t, store.consumeCall, "failed PKCE must not consume the device code")

	pair, err := service.ExchangePairing(context.Background(), ClientID, challengeResult.DeviceCode, verifier)
	require.NoError(t, err)
	require.Equal(t, ScopeRead, pair.Scope)
	require.NotEmpty(t, repo.activatedHash)
	require.NotEqual(t, hashToken(pair.RefreshToken), pair.RefreshToken)
	require.Equal(t, hashToken(pair.RefreshToken), repo.activatedHash)
}

func TestQuotaRefreshRotationAndDeviceRevocation(t *testing.T) {
	repo := newMemoryQuotaAuthRepository()
	service := NewService(nil, repo, &config.Config{
		JWT: config.JWTConfig{
			Secret:                 strings.Repeat("x", 40),
			RefreshTokenExpireDays: 30,
		},
	})
	current := refreshTokenPrefix + strings.Repeat("d", 64)

	_, err := service.RefreshSession(context.Background(), ClientID, current)
	require.ErrorIs(t, err, ErrInvalidAuthorizationRequest)
	require.Empty(t, repo.replacementHash, "refresh without a persisted candidate must not rotate")

	candidate := refreshTokenPrefix + strings.Repeat("e", 64)
	rotationID := "fb3b5595-9c6f-4891-8860-216003304151"
	refreshInput := RefreshSessionInput{
		ClientID:              ClientID,
		RefreshToken:          current,
		RotationID:            &rotationID,
		CandidateRefreshToken: &candidate,
	}
	pair, err := service.RefreshSessionWithInput(context.Background(), refreshInput)
	require.NoError(t, err)
	require.Equal(t, candidate, pair.RefreshToken)
	require.Equal(t, RefreshProtocolCandidateV1, pair.RefreshProtocol)
	require.Equal(t, hashToken(candidate), repo.replacementHash)

	repo.rotateErr = ErrRefreshReplay
	_, err = service.RefreshSessionWithInput(context.Background(), refreshInput)
	require.ErrorIs(t, err, ErrRefreshReplay)

	accessPair, err := service.issueTokenPair(&repo.device, refreshTokenPrefix+strings.Repeat("f", 64))
	require.NoError(t, err)
	revoked, err := service.RevokeDevice(context.Background(), repo.device.UserID, repo.device.PublicID)
	require.NoError(t, err)
	require.Equal(t, DeviceStatusRevoked, revoked.Status)
	_, err = service.AuthenticateAccessToken(context.Background(), accessPair.AccessToken)
	require.ErrorIs(t, err, ErrDeviceUnauthorized)
}

func TestQuotaRefreshCandidateV1CommitsAndRecoversTheExactCandidate(t *testing.T) {
	repo := newMemoryQuotaAuthRepository()
	service := NewService(nil, repo, &config.Config{
		JWT: config.JWTConfig{
			Secret:                 strings.Repeat("y", 40),
			RefreshTokenExpireDays: 30,
		},
	})
	now := time.Date(2026, time.July, 30, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	current := refreshTokenPrefix + strings.Repeat("a", 64)
	candidate := refreshTokenPrefix + strings.Repeat("b", 64)
	rotationID := "af69f4b4-5824-4f48-9149-3e55f1b5ca2d"

	pair, err := service.RefreshSessionWithInput(context.Background(), RefreshSessionInput{
		ClientID:              ClientID,
		RefreshToken:          current,
		RotationID:            &rotationID,
		CandidateRefreshToken: &candidate,
	})
	require.NoError(t, err)
	require.Equal(t, candidate, pair.RefreshToken)
	require.Equal(t, RefreshProtocolCandidateV1, pair.RefreshProtocol)
	require.Equal(t, rotationID, pair.RotationID)
	require.Equal(t, RefreshRotationCommitted, pair.RotationResult)
	require.Equal(t, rotationID, repo.rotationID)
	require.Equal(t, hashToken(candidate), repo.replacementHash)
	require.Equal(t, now.Add(refreshRecoveryTTL), repo.recoveryExpires)
	candidateJSON, err := json.Marshal(pair)
	require.NoError(t, err)
	require.Contains(t, string(candidateJSON), `"refresh_protocol":"candidate-v1"`)
	require.Contains(t, string(candidateJSON), `"rotation_result":"committed"`)

	repo.rotationResult = RefreshRotationRecovered
	recovered, err := service.RefreshSessionWithInput(context.Background(), RefreshSessionInput{
		ClientID:              ClientID,
		RefreshToken:          current,
		RotationID:            &rotationID,
		CandidateRefreshToken: &candidate,
	})
	require.NoError(t, err)
	require.Equal(t, candidate, recovered.RefreshToken)
	require.Equal(t, RefreshRotationRecovered, recovered.RotationResult)
}

func TestQuotaRefreshCandidateV1StrictValidation(t *testing.T) {
	repo := newMemoryQuotaAuthRepository()
	service := NewService(nil, repo, &config.Config{
		JWT: config.JWTConfig{Secret: strings.Repeat("z", 40)},
	})
	current := refreshTokenPrefix + strings.Repeat("c", 64)
	validCandidate := refreshTokenPrefix + strings.Repeat("d", 64)
	validRotationID := "97e583b5-4a68-4a60-8f76-0fb07aecc5d0"

	tests := []struct {
		name       string
		rotationID *string
		candidate  *string
	}{
		{name: "candidate-v1 tuple missing"},
		{name: "rotation id without candidate", rotationID: &validRotationID},
		{name: "candidate without rotation id", candidate: &validCandidate},
		{name: "empty candidate", rotationID: &validRotationID, candidate: stringPointer("")},
		{name: "uppercase candidate hex", rotationID: &validRotationID, candidate: stringPointer(refreshTokenPrefix + strings.Repeat("A", 64))},
		{name: "non hex candidate", rotationID: &validRotationID, candidate: stringPointer(refreshTokenPrefix + strings.Repeat("g", 64))},
		{name: "candidate equals current", rotationID: &validRotationID, candidate: &current},
		{name: "non v4 rotation id", rotationID: stringPointer("00000000-0000-1000-8000-000000000000"), candidate: &validCandidate},
		{name: "non canonical rotation id", rotationID: stringPointer(strings.ToUpper(validRotationID)), candidate: &validCandidate},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.RefreshSessionWithInput(context.Background(), RefreshSessionInput{
				ClientID:              ClientID,
				RefreshToken:          current,
				RotationID:            test.rotationID,
				CandidateRefreshToken: test.candidate,
			})
			require.ErrorIs(t, err, ErrInvalidAuthorizationRequest)
		})
	}
}

func stringPointer(value string) *string {
	return &value
}

func validQuotaClaims(now time.Time, device *Device) accessClaims {
	return accessClaims{
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
}
