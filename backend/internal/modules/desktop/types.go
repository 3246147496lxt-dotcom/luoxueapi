package desktop

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	PairingStatusPending   = "pending"
	PairingStatusApproving = "approving"
	PairingStatusApproved  = "approved"
	PairingStatusConsumed  = "consumed"

	DeviceStatusPending = "pending"
	DeviceStatusActive  = "active"
	DeviceStatusRevoked = "revoked"

	ScopeProfileRead      = "profile:read"
	ScopeRoutesRead       = "routes:read"
	ScopeManagedKeyWrite  = "managed_keys:write"
	ScopeDiagnosticsWrite = "diagnostics:write"

	ReleaseChannelStable   = "stable"
	ReleaseChannelInternal = "internal"
	ReleaseTargetDarwin    = "darwin"
	ReleaseArchUniversal   = "universal"

	DesktopAudience = "luoxue-desktop"
	DesktopIssuer   = "luoxue-api"
)

var (
	ErrInvalidPairingRequest    = infraerrors.BadRequest("DESKTOP_PAIRING_INVALID", "invalid desktop pairing request")
	ErrPairingExpired           = infraerrors.BadRequest("DESKTOP_PAIRING_EXPIRED", "desktop pairing has expired")
	ErrPairingCollision         = infraerrors.Conflict("DESKTOP_PAIRING_COLLISION", "desktop pairing code collision")
	ErrPairingState             = infraerrors.Conflict("DESKTOP_PAIRING_STATE", "desktop pairing is not in the required state")
	ErrPairingConsumed          = infraerrors.Conflict("DESKTOP_PAIRING_CONSUMED", "desktop pairing has already been consumed")
	ErrPKCEVerification         = infraerrors.Unauthorized("DESKTOP_PKCE_INVALID", "PKCE verification failed")
	ErrDeviceLimit              = infraerrors.Conflict("DESKTOP_DEVICE_LIMIT", "desktop device limit reached")
	ErrDeviceNotFound           = infraerrors.NotFound("DESKTOP_DEVICE_NOT_FOUND", "desktop device not found")
	ErrInvalidDeviceName        = infraerrors.BadRequest("DESKTOP_DEVICE_NAME_INVALID", "invalid desktop device name")
	ErrDeviceUnauthorized       = infraerrors.Forbidden("DESKTOP_DEVICE_FORBIDDEN", "desktop device is not authorized")
	ErrDesktopToken             = infraerrors.Unauthorized("DESKTOP_TOKEN_INVALID", "invalid desktop access token")
	ErrDesktopScope             = infraerrors.Forbidden("DESKTOP_SCOPE_FORBIDDEN", "desktop token does not grant the required scope")
	ErrRefreshToken             = infraerrors.Unauthorized("DESKTOP_REFRESH_INVALID", "invalid desktop refresh token")
	ErrRefreshReplay            = infraerrors.Unauthorized("DESKTOP_REFRESH_REPLAY", "desktop refresh token replay detected")
	ErrRouteUnavailable         = infraerrors.Forbidden("DESKTOP_ROUTE_UNAVAILABLE", "desktop route is not available to this user")
	ErrManagedKeyCollision      = infraerrors.Conflict("DESKTOP_KEY_COLLISION", "generated managed key collided with an existing key")
	ErrManagedKeyCache          = infraerrors.ServiceUnavailable("DESKTOP_KEY_CACHE_INVALIDATION_FAILED", "desktop managed-key cache invalidation failed; retry the request")
	ErrActivationOutcomeUnknown = infraerrors.ServiceUnavailable("DESKTOP_ACTIVATION_UNKNOWN", "desktop activation outcome is unknown; start a new pairing")
	ErrReleaseParameters        = infraerrors.BadRequest("DESKTOP_RELEASE_PARAMETERS_INVALID", "invalid desktop release parameters")
	ErrDiagnosticsInvalid       = infraerrors.BadRequest("DESKTOP_DIAGNOSTICS_INVALID", "invalid desktop diagnostics payload")
	ErrDiagnosticsSensitive     = infraerrors.BadRequest("DESKTOP_DIAGNOSTICS_SENSITIVE", "desktop diagnostics contain forbidden sensitive data")
	ErrDiagnosticsUnavailable   = infraerrors.ServiceUnavailable("DESKTOP_DIAGNOSTICS_UNAVAILABLE", "desktop diagnostics are unavailable")
	ErrDiagnosticNotFound       = infraerrors.NotFound("DESKTOP_DIAGNOSTIC_NOT_FOUND", "desktop diagnostic not found")
)

type Pairing struct {
	DeviceCodeHash   string    `json:"device_code_hash"`
	UserCode         string    `json:"user_code"`
	CodeChallenge    string    `json:"code_challenge"`
	InstallationHash string    `json:"installation_hash"`
	DeviceName       string    `json:"device_name"`
	Platform         string    `json:"platform"`
	Architecture     string    `json:"architecture"`
	OSVersion        string    `json:"os_version"`
	AppVersion       string    `json:"app_version"`
	Status           string    `json:"status"`
	UserID           int64     `json:"user_id,omitempty"`
	DeviceID         int64     `json:"device_id,omitempty"`
	ReleaseChannel   string    `json:"-"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type CreatePairingInput struct {
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
	ExpiresIn               int       `json:"expires_in"`
	Interval                int       `json:"interval"`
	ExpiresAt               time.Time `json:"expires_at"`
}

type PairingPreview struct {
	UserCode        string    `json:"user_code"`
	DeviceName      string    `json:"device_name"`
	Platform        string    `json:"platform"`
	Architecture    string    `json:"architecture"`
	OSVersion       string    `json:"os_version"`
	AppVersion      string    `json:"app_version"`
	RequestedScopes []string  `json:"requested_scopes"`
	ExpiresAt       time.Time `json:"expires_at"`
}

type Device struct {
	ID             int64      `json:"-"`
	PublicID       string     `json:"id"`
	UserID         int64      `json:"-"`
	Name           string     `json:"name"`
	Platform       string     `json:"platform"`
	Architecture   string     `json:"architecture"`
	OSVersion      string     `json:"os_version"`
	AppVersion     string     `json:"app_version"`
	Status         string     `json:"status"`
	TokenVersion   int64      `json:"-"`
	ReleaseChannel string     `json:"-"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
	ActivatedAt    *time.Time `json:"activated_at,omitempty"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
}

type DesktopSubject struct {
	DeviceID       int64
	DevicePublicID string
	UserID         int64
	TokenVersion   int64
	Scopes         []string
}

func (s DesktopSubject) HasScope(required string) bool {
	for _, scope := range s.Scopes {
		if scope == required {
			return true
		}
	}
	return false
}

type TokenPair struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int      `json:"expires_in"`
	Scope        []string `json:"scope"`
	Device       Device   `json:"device"`
}

type Route struct {
	GroupID        int64    `json:"group_id"`
	Name           string   `json:"name"`
	RateMultiplier float64  `json:"rate_multiplier"`
	Models         []string `json:"models"`
}

type ManagedKey struct {
	ID      int64  `json:"id"`
	GroupID int64  `json:"group_id"`
	Key     string `json:"key"`
	Created bool   `json:"created"`
}

type TodayUsage struct {
	Requests int64   `json:"requests"`
	Tokens   int64   `json:"tokens"`
	Cost     float64 `json:"cost"`
	Balance  float64 `json:"balance"`
}

type CleanupResult struct {
	PendingDevicesDeleted int64
	SessionsDeleted       int64
	DiagnosticsDeleted    int64
}

// DeviceRevocation carries secrets only between the repository and service so
// authentication caches can be invalidated after the database commit.
type DeviceRevocation struct {
	Device      Device
	ManagedKeys []string
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
	RotateSession(ctx context.Context, refreshTokenHash, replacementHash string, replacementExpiresAt time.Time) (*Device, error)
	GetAuthorizedDevice(ctx context.Context, deviceID, userID, tokenVersion int64) (*Device, error)
	ListDevices(ctx context.Context, userID int64) ([]Device, error)
	RenameDevice(ctx context.Context, userID int64, publicID, name string) (*Device, error)
	HeartbeatDevice(ctx context.Context, deviceID, userID, tokenVersion int64) (*Device, error)
	MarkDeviceActivated(ctx context.Context, deviceID, userID, tokenVersion int64, activatedAt time.Time) (*Device, error)
	RevokeDevice(ctx context.Context, userID int64, publicID string) (*DeviceRevocation, error)
	EnsureManagedKey(ctx context.Context, deviceID, userID, groupID int64, generatedKey string) (*ManagedKey, error)
	GetTodayUsage(ctx context.Context, deviceID, userID int64, startTime, endTime time.Time) (*TodayUsage, error)
	GetReleaseChannel(ctx context.Context, deviceID, userID, tokenVersion int64) (string, error)
}

type CleanupRepository interface {
	CleanupExpired(ctx context.Context, now, sessionRetentionCutoff time.Time, limit int) (*CleanupResult, error)
}

type RouteCatalog interface {
	ListRoutes(ctx context.Context, userID int64) ([]Route, error)
}

type KeyInvalidator interface {
	InvalidateManagedKey(ctx context.Context, key string) error
}
