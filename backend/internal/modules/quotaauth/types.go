package quotaauth

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	ClientID  = "luoxue-quota-viewer"
	Issuer    = "luoxue-api"
	Audience  = "luoxue-quota-api"
	ScopeRead = "quota:read"

	PairingStatusPending   = "pending"
	PairingStatusApproving = "approving"
	PairingStatusApproved  = "approved"
	PairingStatusConsumed  = "consumed"

	DeviceStatusPending = "pending"
	DeviceStatusActive  = "active"
	DeviceStatusRevoked = "revoked"
)

var (
	ErrInvalidAuthorizationRequest = infraerrors.BadRequest("QUOTA_AUTH_REQUEST_INVALID", "invalid quota viewer authorization request")
	ErrPairingExpired              = infraerrors.BadRequest("QUOTA_AUTH_PAIRING_EXPIRED", "quota viewer authorization has expired")
	ErrPairingCollision            = infraerrors.Conflict("QUOTA_AUTH_PAIRING_COLLISION", "quota viewer authorization code collision")
	ErrPairingState                = infraerrors.Conflict("QUOTA_AUTH_PAIRING_STATE", "quota viewer authorization is not in the required state")
	ErrPairingConsumed             = infraerrors.Conflict("QUOTA_AUTH_PAIRING_CONSUMED", "quota viewer authorization has already been consumed")
	ErrPKCEVerification            = infraerrors.Unauthorized("QUOTA_AUTH_PKCE_INVALID", "PKCE verification failed")
	ErrDeviceLimit                 = infraerrors.Conflict("QUOTA_AUTH_DEVICE_LIMIT", "quota viewer device limit reached")
	ErrDeviceNotFound              = infraerrors.NotFound("QUOTA_AUTH_DEVICE_NOT_FOUND", "quota viewer device not found")
	ErrDeviceUnauthorized          = infraerrors.Unauthorized("QUOTA_AUTH_DEVICE_UNAUTHORIZED", "quota viewer device is not authorized")
	ErrAuthRequired                = infraerrors.Unauthorized("QUOTA_AUTH_REQUIRED", "quota viewer bearer token is required")
	ErrAccessTokenInvalid          = infraerrors.Unauthorized("QUOTA_ACCESS_TOKEN_INVALID", "invalid quota viewer access token")
	ErrAccessTokenExpired          = infraerrors.Unauthorized("QUOTA_ACCESS_TOKEN_EXPIRED", "quota viewer access token has expired")
	ErrAccessTokenAudience         = infraerrors.Forbidden("QUOTA_ACCESS_TOKEN_AUDIENCE_FORBIDDEN", "quota viewer token audience is not allowed")
	ErrAccessTokenClient           = infraerrors.Forbidden("QUOTA_ACCESS_TOKEN_CLIENT_FORBIDDEN", "quota viewer token client is not allowed")
	ErrAccessTokenScope            = infraerrors.Forbidden("QUOTA_ACCESS_TOKEN_SCOPE_FORBIDDEN", "quota viewer token does not grant quota:read")
	ErrRefreshToken                = infraerrors.Unauthorized("QUOTA_REFRESH_TOKEN_INVALID", "invalid quota viewer refresh token")
	ErrRefreshReplay               = infraerrors.Unauthorized("QUOTA_REFRESH_TOKEN_REPLAY", "quota viewer refresh token replay detected")
	ErrRefreshRotationSuperseded   = infraerrors.Unauthorized("QUOTA_REFRESH_ROTATION_SUPERSEDED", "quota viewer refresh rotation has been superseded")
	ErrRefreshRecoveryExpired      = infraerrors.Unauthorized("QUOTA_REFRESH_RECOVERY_EXPIRED", "quota viewer refresh recovery window has expired")
	ErrRefreshOutcomeUnknown       = infraerrors.ServiceUnavailable("QUOTA_REFRESH_OUTCOME_UNKNOWN", "quota viewer refresh outcome is unknown; retry the same rotation")
	ErrActivationOutcomeUnknown    = infraerrors.ServiceUnavailable("QUOTA_AUTH_ACTIVATION_UNKNOWN", "quota viewer authorization outcome is unknown; start a new authorization")
	ErrServiceUnavailable          = infraerrors.ServiceUnavailable("QUOTA_AUTH_SERVICE_UNAVAILABLE", "quota viewer authorization service is unavailable")
)

const (
	RefreshProtocolCandidateV1 = "candidate-v1"

	RefreshRotationCommitted = "committed"
	RefreshRotationRecovered = "recovered"
)

type Pairing struct {
	DeviceCodeHash   string    `json:"device_code_hash"`
	UserCode         string    `json:"user_code"`
	CodeChallenge    string    `json:"code_challenge"`
	ClientID         string    `json:"client_id"`
	Scope            string    `json:"scope"`
	InstallationHash string    `json:"installation_hash"`
	DeviceName       string    `json:"device_name"`
	Platform         string    `json:"platform"`
	Architecture     string    `json:"architecture"`
	OSVersion        string    `json:"os_version"`
	AppVersion       string    `json:"app_version"`
	Status           string    `json:"status"`
	UserID           int64     `json:"user_id,omitempty"`
	DeviceID         int64     `json:"device_id,omitempty"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type CreatePairingInput struct {
	ClientID            string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
	InstallationID      string
	DeviceName          string
	Platform            string
	Architecture        string
	OSVersion           string
	AppVersion          string
}

type PairingChallenge struct {
	DeviceCode              string    `json:"device_code"`
	UserCode                string    `json:"user_code"`
	VerificationURI         string    `json:"verification_uri"`
	VerificationURIComplete string    `json:"verification_uri_complete"`
	ClientID                string    `json:"client_id"`
	Scope                   string    `json:"scope"`
	ExpiresIn               int       `json:"expires_in"`
	Interval                int       `json:"interval"`
	ExpiresAt               time.Time `json:"expires_at"`
}

type PairingPreview struct {
	UserCode     string    `json:"user_code"`
	ClientID     string    `json:"client_id"`
	Scope        string    `json:"scope"`
	DeviceName   string    `json:"device_name"`
	Platform     string    `json:"platform"`
	Architecture string    `json:"architecture"`
	OSVersion    string    `json:"os_version"`
	AppVersion   string    `json:"app_version"`
	ReadOnly     bool      `json:"read_only"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Device struct {
	ID           int64      `json:"-"`
	PublicID     string     `json:"id"`
	UserID       int64      `json:"-"`
	ClientID     string     `json:"client_id"`
	Scope        string     `json:"scope"`
	Name         string     `json:"name"`
	Platform     string     `json:"platform"`
	Architecture string     `json:"architecture"`
	OSVersion    string     `json:"os_version"`
	AppVersion   string     `json:"app_version"`
	Status       string     `json:"status"`
	TokenVersion int64      `json:"-"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	ActivatedAt  *time.Time `json:"activated_at,omitempty"`
	LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

type Subject struct {
	DeviceID       int64
	DevicePublicID string
	UserID         int64
	TokenVersion   int64
	ClientID       string
	Scopes         []string
}

func (s Subject) HasScope(required string) bool {
	for _, scope := range s.Scopes {
		if scope == required {
			return true
		}
	}
	return false
}

type TokenPair struct {
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int    `json:"expires_in"`
	ClientID        string `json:"client_id"`
	Scope           string `json:"scope"`
	Device          Device `json:"device"`
	RefreshProtocol string `json:"refresh_protocol,omitempty"`
	RotationID      string `json:"rotation_id,omitempty"`
	RotationResult  string `json:"rotation_result,omitempty"`
}

type RefreshSessionInput struct {
	ClientID              string
	RefreshToken          string
	RotationID            *string
	CandidateRefreshToken *string
}

type RefreshRotation struct {
	PredecessorHash   string
	ReplacementHash   string
	ReplacementExpiry time.Time
	RotationID        string
	RecoveryExpiry    time.Time
}

type RefreshRotationOutcome struct {
	Device *Device
	Result string
}

type PairingStore interface {
	Create(ctx context.Context, pairing Pairing, ttl time.Duration) error
	GetByUserCode(ctx context.Context, userCode string) (*Pairing, error)
	GetByDeviceCode(ctx context.Context, deviceCode string) (*Pairing, error)
	BeginApproval(ctx context.Context, userCode string) (*Pairing, error)
	FinishApproval(ctx context.Context, userCode string, userID, deviceID int64) error
	CancelApproval(ctx context.Context, userCode string) error
	Consume(ctx context.Context, deviceCode string) (*Pairing, error)
	RestoreConsumed(ctx context.Context, deviceCode string) error
}

type Repository interface {
	ReserveDevice(ctx context.Context, userID int64, pairing Pairing) (*Device, error)
	DeletePendingDevice(ctx context.Context, deviceID int64) error
	ActivateDevice(ctx context.Context, deviceID int64, familyID, refreshTokenHash string, expiresAt time.Time) (*Device, error)
	RotateSession(ctx context.Context, rotation RefreshRotation) (*RefreshRotationOutcome, error)
	GetAuthorizedDevice(ctx context.Context, deviceID, tokenVersion int64) (*Device, error)
	ListDevices(ctx context.Context, userID int64) ([]Device, error)
	RevokeDevice(ctx context.Context, userID int64, publicID string) (*Device, error)
}
