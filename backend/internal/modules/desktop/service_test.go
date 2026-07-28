package desktop

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

type memoryPairingStore struct {
	mu      sync.Mutex
	pairing Pairing
}

func (s *memoryPairingStore) Create(_ context.Context, pairing Pairing, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pairing = pairing
	return nil
}

func (s *memoryPairingStore) GetByUserCode(context.Context, string) (*Pairing, error) {
	return s.get()
}

func (s *memoryPairingStore) GetByDeviceCode(context.Context, string) (*Pairing, error) {
	return s.get()
}

func (s *memoryPairingStore) BeginApproval(context.Context, string) (*Pairing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.pairing.ExpiresAt.After(time.Now()) {
		return nil, ErrPairingExpired
	}
	if s.pairing.Status != PairingStatusPending {
		return nil, ErrPairingState
	}
	s.pairing.Status = PairingStatusApproving
	out := s.pairing
	return &out, nil
}

func (s *memoryPairingStore) FinishApproval(_ context.Context, _ string, userID, deviceID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pairing.Status != PairingStatusApproving {
		return ErrPairingState
	}
	s.pairing.Status = PairingStatusApproved
	s.pairing.UserID = userID
	s.pairing.DeviceID = deviceID
	return nil
}

func (s *memoryPairingStore) CancelApproval(context.Context, string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pairing.Status == PairingStatusApproving {
		s.pairing.Status = PairingStatusPending
	}
	return nil
}

func (s *memoryPairingStore) Consume(context.Context, string) (*Pairing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pairing.Status == PairingStatusConsumed {
		return nil, ErrPairingConsumed
	}
	if s.pairing.Status != PairingStatusApproved {
		return nil, ErrPairingState
	}
	s.pairing.Status = PairingStatusConsumed
	out := s.pairing
	return &out, nil
}

func (s *memoryPairingStore) RestoreConsumed(context.Context, string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pairing.Status == PairingStatusConsumed {
		s.pairing.Status = PairingStatusApproved
	}
	return nil
}

func (s *memoryPairingStore) get() (*Pairing, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.pairing.ExpiresAt.After(time.Now()) {
		return nil, ErrPairingExpired
	}
	out := s.pairing
	return &out, nil
}

type memoryRepository struct {
	mu            sync.Mutex
	device        Device
	reserveErr    error
	activationErr error
	managed       map[int64]*ManagedKey
	activations   int
	refreshActive bool
	todayUsage    TodayUsage
	usageDeviceID int64
	usageUserID   int64
	usageStart    time.Time
	usageEnd      time.Time
}

func (r *memoryRepository) ReserveDevice(_ context.Context, userID int64, pairing Pairing) (*Device, error) {
	if r.reserveErr != nil {
		return nil, r.reserveErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.device = Device{
		ID: 41, PublicID: "c7d06b34-7c6a-4a59-ad48-648ade0525e1", UserID: userID,
		Name: pairing.DeviceName, Platform: "macos", Architecture: "arm64",
		Status: DeviceStatusPending, TokenVersion: 1, ReleaseChannel: pairing.ReleaseChannel,
	}
	out := r.device
	return &out, nil
}

func (r *memoryRepository) DeletePendingDevice(context.Context, int64) error { return nil }

func (r *memoryRepository) ActivateDevice(_ context.Context, deviceID int64, _, _ string, _ time.Time) (*Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.activationErr != nil {
		return nil, r.activationErr
	}
	if r.device.ID == 0 {
		r.device = Device{
			ID: deviceID, PublicID: "c7d06b34-7c6a-4a59-ad48-648ade0525e1", UserID: 7,
			Name: "Mac", Platform: "macos", Architecture: "arm64", TokenVersion: 1,
		}
	}
	r.device.Status = DeviceStatusActive
	r.activations++
	out := r.device
	return &out, nil
}

func (r *memoryRepository) RotateSession(context.Context, string, string, time.Time) (*Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.refreshActive || r.device.Status != DeviceStatusActive {
		return nil, ErrRefreshToken
	}
	out := r.device
	return &out, nil
}

func (r *memoryRepository) GetAuthorizedDevice(_ context.Context, deviceID, userID, tokenVersion int64) (*Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.device.ID != deviceID || r.device.UserID != userID || r.device.TokenVersion != tokenVersion || r.device.Status != DeviceStatusActive {
		return nil, ErrDeviceUnauthorized
	}
	out := r.device
	return &out, nil
}

func (r *memoryRepository) ListDevices(_ context.Context, userID int64) ([]Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.device.ID == 0 || r.device.UserID != userID {
		return []Device{}, nil
	}
	return []Device{r.device}, nil
}

func (r *memoryRepository) RenameDevice(_ context.Context, userID int64, publicID, name string) (*Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.device.UserID != userID || r.device.PublicID != publicID || r.device.Status == DeviceStatusRevoked {
		return nil, ErrDeviceNotFound
	}
	r.device.Name = name
	out := r.device
	return &out, nil
}

func (r *memoryRepository) HeartbeatDevice(_ context.Context, deviceID, userID, tokenVersion int64) (*Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.device.ID != deviceID || r.device.UserID != userID || r.device.TokenVersion != tokenVersion || r.device.Status != DeviceStatusActive {
		return nil, ErrDeviceUnauthorized
	}
	now := time.Now()
	r.device.LastSeenAt = &now
	out := r.device
	return &out, nil
}

func (r *memoryRepository) MarkDeviceActivated(_ context.Context, deviceID, userID, tokenVersion int64, activatedAt time.Time) (*Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.device.ID != deviceID || r.device.UserID != userID || r.device.TokenVersion != tokenVersion || r.device.Status != DeviceStatusActive {
		return nil, ErrDeviceUnauthorized
	}
	if r.device.ActivatedAt == nil {
		first := activatedAt
		r.device.ActivatedAt = &first
	}
	lastSeen := activatedAt
	r.device.LastSeenAt = &lastSeen
	out := r.device
	return &out, nil
}

func (r *memoryRepository) RevokeDevice(_ context.Context, userID int64, publicID string) (*DeviceRevocation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.device.UserID != userID || r.device.PublicID != publicID {
		return nil, ErrDeviceNotFound
	}
	if r.device.Status != DeviceStatusRevoked {
		r.device.Status = DeviceStatusRevoked
		r.device.TokenVersion++
	}
	r.refreshActive = false
	keys := make([]string, 0, len(r.managed))
	for _, managed := range r.managed {
		keys = append(keys, managed.Key)
	}
	out := r.device
	return &DeviceRevocation{Device: out, ManagedKeys: keys}, nil
}

func (r *memoryRepository) EnsureManagedKey(_ context.Context, deviceID, userID, groupID int64, generated string) (*ManagedKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.device.ID != deviceID || r.device.UserID != userID || r.device.Status != DeviceStatusActive {
		return nil, ErrDeviceUnauthorized
	}
	if r.managed == nil {
		r.managed = make(map[int64]*ManagedKey)
	}
	if existing := r.managed[groupID]; existing != nil {
		out := *existing
		out.Created = false
		return &out, nil
	}
	created := &ManagedKey{ID: int64(len(r.managed) + 1), GroupID: groupID, Key: generated, Created: true}
	r.managed[groupID] = created
	out := *created
	return &out, nil
}

func (r *memoryRepository) GetTodayUsage(_ context.Context, deviceID, userID int64, startTime, endTime time.Time) (*TodayUsage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usageDeviceID = deviceID
	r.usageUserID = userID
	r.usageStart = startTime
	r.usageEnd = endTime
	out := r.todayUsage
	return &out, nil
}

func (r *memoryRepository) GetReleaseChannel(_ context.Context, deviceID, userID, tokenVersion int64) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.device.ID != deviceID || r.device.UserID != userID || r.device.TokenVersion != tokenVersion || r.device.Status != DeviceStatusActive {
		return "", ErrDeviceUnauthorized
	}
	if r.device.ReleaseChannel == "" {
		return ReleaseChannelStable, nil
	}
	return r.device.ReleaseChannel, nil
}

type staticRoutes []Route

func (r staticRoutes) ListRoutes(context.Context, int64) ([]Route, error) {
	return append([]Route(nil), r...), nil
}

type recordingInvalidator struct {
	mu   sync.Mutex
	keys []string
	err  error
}

func (i *recordingInvalidator) InvalidateManagedKey(_ context.Context, key string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.keys = append(i.keys, key)
	return i.err
}

func newDesktopTestService(store PairingStore, repo Repository) *Service {
	cfg := &config.Config{}
	cfg.JWT.Secret = "web-signing-secret"
	return NewService(store, repo, staticRoutes{{GroupID: 9, Name: "OpenAI"}}, &recordingInvalidator{}, cfg)
}

const testDevicePublicID = "c7d06b34-7c6a-4a59-ad48-648ade0525e1"

func approvedPairing(verifier string) Pairing {
	digest := sha256.Sum256([]byte(verifier))
	return Pairing{
		CodeChallenge: base64.RawURLEncoding.EncodeToString(digest[:]), Status: PairingStatusApproved,
		UserID: 7, DeviceID: 41, DeviceName: "Mac", ExpiresAt: time.Now().Add(time.Minute),
	}
}

func TestExchangePairingRejectsExpiredPairing(t *testing.T) {
	store := &memoryPairingStore{pairing: Pairing{Status: PairingStatusApproved, ExpiresAt: time.Now().Add(-time.Second)}}
	svc := newDesktopTestService(store, &memoryRepository{})

	_, err := svc.ExchangePairing(context.Background(), "device-code", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.ErrorIs(t, err, ErrPairingExpired)
}

func TestExchangePairingRejectsPKCEAndDoesNotConsume(t *testing.T) {
	verifier := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	store := &memoryPairingStore{pairing: approvedPairing(verifier)}
	repo := &memoryRepository{}
	svc := newDesktopTestService(store, repo)

	_, err := svc.ExchangePairing(context.Background(), "device-code", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	require.ErrorIs(t, err, ErrPKCEVerification)
	require.Equal(t, PairingStatusApproved, store.pairing.Status)
	require.Zero(t, repo.activations)
}

func TestExchangePairingIsSingleUse(t *testing.T) {
	verifier := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	store := &memoryPairingStore{pairing: approvedPairing(verifier)}
	repo := &memoryRepository{}
	svc := newDesktopTestService(store, repo)

	first, err := svc.ExchangePairing(context.Background(), "device-code", verifier)
	require.NoError(t, err)
	require.NotEmpty(t, first.AccessToken)
	require.Nil(t, first.Device.ActivatedAt, "token exchange must not mark first real use")
	_, err = svc.ExchangePairing(context.Background(), "device-code", verifier)
	require.ErrorIs(t, err, ErrPairingConsumed)
	require.Equal(t, 1, repo.activations)
}

func TestExchangePairingRestoresOnlyKnownPreCommitFailures(t *testing.T) {
	verifier := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	t.Run("known rollback", func(t *testing.T) {
		store := &memoryPairingStore{pairing: approvedPairing(verifier)}
		repo := &memoryRepository{activationErr: errors.New("insert failed")}
		svc := newDesktopTestService(store, repo)

		_, err := svc.ExchangePairing(context.Background(), "device-code", verifier)
		require.Error(t, err)
		require.Equal(t, PairingStatusApproved, store.pairing.Status)
	})

	t.Run("ambiguous commit", func(t *testing.T) {
		store := &memoryPairingStore{pairing: approvedPairing(verifier)}
		repo := &memoryRepository{activationErr: ErrActivationOutcomeUnknown}
		svc := newDesktopTestService(store, repo)

		_, err := svc.ExchangePairing(context.Background(), "device-code", verifier)
		require.ErrorIs(t, err, ErrActivationOutcomeUnknown)
		require.Equal(t, PairingStatusConsumed, store.pairing.Status)
	})
}

func TestApprovePairingEnforcesDeviceLimitAndReleasesClaim(t *testing.T) {
	store := &memoryPairingStore{pairing: Pairing{
		Status: PairingStatusPending, DeviceName: "Mac", ExpiresAt: time.Now().Add(time.Minute),
	}}
	repo := &memoryRepository{reserveErr: ErrDeviceLimit}
	svc := newDesktopTestService(store, repo)

	_, err := svc.ApprovePairing(context.Background(), 7, "ABCD-EFGH")
	require.ErrorIs(t, err, ErrDeviceLimit)
	require.Equal(t, PairingStatusPending, store.pairing.Status)
}

func TestDesktopTokenUsesSeparateSigningDomainFromWebJWT(t *testing.T) {
	verifier := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	store := &memoryPairingStore{pairing: approvedPairing(verifier)}
	repo := &memoryRepository{}
	svc := newDesktopTestService(store, repo)

	tokens, err := svc.ExchangePairing(context.Background(), "device-code", verifier)
	require.NoError(t, err)
	_, err = svc.AuthenticateAccessToken(context.Background(), tokens.AccessToken)
	require.NoError(t, err)

	_, err = jwt.Parse(tokens.AccessToken, func(*jwt.Token) (any, error) {
		return []byte("web-signing-secret"), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	require.Error(t, err, "a Desktop token must fail validation with the Web JWT key")
}

func TestEnsureManagedKeyRejectsAnotherDevice(t *testing.T) {
	repo := &memoryRepository{device: Device{
		ID: 41, PublicID: "device-a", UserID: 7, Status: DeviceStatusActive, TokenVersion: 1,
	}}
	svc := newDesktopTestService(&memoryPairingStore{}, repo)

	_, err := svc.EnsureManagedKey(context.Background(), DesktopSubject{
		DeviceID: 42, UserID: 7, Scopes: []string{ScopeManagedKeyWrite},
	}, 9)
	require.ErrorIs(t, err, ErrDeviceUnauthorized)
}

func TestConcurrentManagedKeyEnsureConverges(t *testing.T) {
	repo := &memoryRepository{device: Device{
		ID: 41, PublicID: "device-a", UserID: 7, Status: DeviceStatusActive, TokenVersion: 1,
	}}
	svc := newDesktopTestService(&memoryPairingStore{}, repo)
	subject := DesktopSubject{DeviceID: 41, UserID: 7, Scopes: []string{ScopeManagedKeyWrite}}

	const workers = 24
	keys := make(chan string, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			managed, err := svc.EnsureManagedKey(context.Background(), subject, 9)
			if err != nil {
				errs <- err
				return
			}
			keys <- managed.Key
		}()
	}
	wg.Wait()
	close(keys)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var expected string
	for key := range keys {
		if expected == "" {
			expected = key
		}
		require.Equal(t, expected, key)
	}
	require.NotEmpty(t, expected)
	require.Len(t, repo.managed, 1)
}

func TestAuthenticateRejectsRevokedDevice(t *testing.T) {
	repo := &memoryRepository{device: Device{
		ID: 41, PublicID: "device-a", UserID: 7, Status: DeviceStatusActive, TokenVersion: 1,
	}}
	svc := newDesktopTestService(&memoryPairingStore{}, repo)
	tokens, err := svc.issueTokenPair(&repo.device, "refresh")
	require.NoError(t, err)
	repo.device.Status = "revoked"

	_, err = svc.AuthenticateAccessToken(context.Background(), tokens.AccessToken)
	require.True(t, errors.Is(err, ErrDeviceUnauthorized))
}

func TestWebDeviceOperationsEnforceOwnership(t *testing.T) {
	repo := &memoryRepository{device: Device{
		ID: 41, PublicID: testDevicePublicID, UserID: 7, Name: "Mac",
		Status: DeviceStatusActive, TokenVersion: 1,
	}}
	svc := newDesktopTestService(&memoryPairingStore{}, repo)

	devices, err := svc.ListDevices(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, devices, 1)

	_, err = svc.RenameDevice(context.Background(), 8, testDevicePublicID, "Other Mac")
	require.ErrorIs(t, err, ErrDeviceNotFound)
	_, err = svc.RevokeDevice(context.Background(), 8, testDevicePublicID)
	require.ErrorIs(t, err, ErrDeviceNotFound)
	require.Equal(t, DeviceStatusActive, repo.device.Status)

	renamed, err := svc.RenameDevice(context.Background(), 7, testDevicePublicID, "  Work Mac  ")
	require.NoError(t, err)
	require.Equal(t, "Work Mac", renamed.Name)
	_, err = svc.RenameDevice(context.Background(), 7, testDevicePublicID, " ")
	require.ErrorIs(t, err, ErrInvalidDeviceName)
}

func TestHeartbeatOnlyTouchesAuthenticatedDevice(t *testing.T) {
	repo := &memoryRepository{device: Device{
		ID: 41, PublicID: testDevicePublicID, UserID: 7,
		Status: DeviceStatusActive, TokenVersion: 1,
	}}
	svc := newDesktopTestService(&memoryPairingStore{}, repo)

	_, err := svc.Heartbeat(context.Background(), DesktopSubject{DeviceID: 41, UserID: 7, TokenVersion: 1})
	require.ErrorIs(t, err, ErrDesktopScope)
	_, err = svc.Heartbeat(context.Background(), DesktopSubject{
		DeviceID: 42, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeProfileRead},
	})
	require.ErrorIs(t, err, ErrDeviceUnauthorized)
	device, err := svc.Heartbeat(context.Background(), DesktopSubject{
		DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeProfileRead},
	})
	require.NoError(t, err)
	require.NotNil(t, device.LastSeenAt)
}

func TestActivateCurrentDeviceIsIdempotentAndRequiresCurrentAuthorizedDevice(t *testing.T) {
	repo := &memoryRepository{device: Device{
		ID: 41, PublicID: testDevicePublicID, UserID: 7,
		Status: DeviceStatusActive, TokenVersion: 1,
	}}
	svc := newDesktopTestService(&memoryPairingStore{}, repo)
	firstAt := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return firstAt }

	_, err := svc.ActivateCurrentDevice(context.Background(), DesktopSubject{DeviceID: 41, UserID: 7})
	require.ErrorIs(t, err, ErrDesktopScope)
	_, err = svc.ActivateCurrentDevice(context.Background(), DesktopSubject{
		DeviceID: 42, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeProfileRead},
	})
	require.ErrorIs(t, err, ErrDeviceUnauthorized)

	activated, err := svc.ActivateCurrentDevice(context.Background(), DesktopSubject{
		DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeProfileRead},
	})
	require.NoError(t, err)
	require.Equal(t, firstAt, *activated.ActivatedAt)

	svc.now = func() time.Time { return firstAt.Add(time.Hour) }
	again, err := svc.ActivateCurrentDevice(context.Background(), DesktopSubject{
		DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeProfileRead},
	})
	require.NoError(t, err)
	require.Equal(t, firstAt, *again.ActivatedAt, "activation timestamp must be write-once")
	require.Equal(t, firstAt.Add(time.Hour), *again.LastSeenAt)

	repo.device.Status = DeviceStatusRevoked
	_, err = svc.ActivateCurrentDevice(context.Background(), DesktopSubject{
		DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeProfileRead},
	})
	require.ErrorIs(t, err, ErrDeviceUnauthorized)
}

func TestGetTodayUsageUsesCurrentDeviceAndConfiguredTimezoneDay(t *testing.T) {
	repo := &memoryRepository{todayUsage: TodayUsage{
		Requests: 3, Tokens: 1234, Cost: 0.42, Balance: 8.75,
	}}
	svc := newDesktopTestService(&memoryPairingStore{}, repo)
	now := time.Date(2026, 7, 28, 18, 45, 0, 0, timezone.Location())
	svc.now = func() time.Time { return now }

	_, err := svc.GetTodayUsage(context.Background(), DesktopSubject{DeviceID: 41, UserID: 7})
	require.ErrorIs(t, err, ErrDesktopScope)

	usage, err := svc.GetTodayUsage(context.Background(), DesktopSubject{
		DeviceID: 41, UserID: 7, Scopes: []string{ScopeProfileRead},
	})
	require.NoError(t, err)
	require.Equal(t, &repo.todayUsage, usage)
	require.Equal(t, int64(41), repo.usageDeviceID)
	require.Equal(t, int64(7), repo.usageUserID)
	expectedStart := timezone.StartOfDay(now)
	require.Equal(t, expectedStart, repo.usageStart)
	require.Equal(t, expectedStart.AddDate(0, 0, 1), repo.usageEnd)
}

func TestRevocationInvalidatesAccessRefreshAndManagedKeys(t *testing.T) {
	repo := &memoryRepository{
		device: Device{
			ID: 41, PublicID: testDevicePublicID, UserID: 7,
			Status: DeviceStatusActive, TokenVersion: 1,
		},
		refreshActive: true,
		managed: map[int64]*ManagedKey{
			9: {ID: 1, GroupID: 9, Key: "sk-desktop-managed"},
		},
	}
	invalidator := &recordingInvalidator{}
	cfg := &config.Config{}
	cfg.JWT.Secret = "web-signing-secret"
	svc := NewService(&memoryPairingStore{}, repo, staticRoutes{{GroupID: 9, Name: "OpenAI"}}, invalidator, cfg)

	tokens, err := svc.issueTokenPair(&repo.device, "drt_current")
	require.NoError(t, err)
	_, err = svc.AuthenticateAccessToken(context.Background(), tokens.AccessToken)
	require.NoError(t, err)
	_, err = svc.RefreshSession(context.Background(), "drt_current")
	require.NoError(t, err)

	revoked, err := svc.RevokeDevice(context.Background(), 7, testDevicePublicID)
	require.NoError(t, err)
	require.Equal(t, DeviceStatusRevoked, revoked.Status)
	require.Equal(t, int64(2), revoked.TokenVersion)
	require.Equal(t, []string{"sk-desktop-managed"}, invalidator.keys)

	_, err = svc.AuthenticateAccessToken(context.Background(), tokens.AccessToken)
	require.ErrorIs(t, err, ErrDeviceUnauthorized)
	_, err = svc.RefreshSession(context.Background(), "drt_current")
	require.ErrorIs(t, err, ErrRefreshToken)

	_, err = svc.RevokeDevice(context.Background(), 7, testDevicePublicID)
	require.NoError(t, err)
	require.Equal(t, int64(2), repo.device.TokenVersion, "idempotent revocation must increment token_version only once")
	require.Equal(t, []string{"sk-desktop-managed", "sk-desktop-managed"}, invalidator.keys,
		"a retry must invalidate managed-key caches again")
}

func TestCurrentDeviceSelfRevocationUsesSameBoundary(t *testing.T) {
	repo := &memoryRepository{device: Device{
		ID: 41, PublicID: testDevicePublicID, UserID: 7,
		Status: DeviceStatusActive, TokenVersion: 1,
	}}
	svc := newDesktopTestService(&memoryPairingStore{}, repo)

	_, err := svc.RevokeCurrentDevice(context.Background(), DesktopSubject{
		DeviceID: 41, DevicePublicID: testDevicePublicID, UserID: 7, TokenVersion: 1,
	})
	require.ErrorIs(t, err, ErrDesktopScope)
	revoked, err := svc.RevokeCurrentDevice(context.Background(), DesktopSubject{
		DeviceID: 41, DevicePublicID: testDevicePublicID, UserID: 7, TokenVersion: 1,
		Scopes: []string{ScopeProfileRead},
	})
	require.NoError(t, err)
	require.Equal(t, DeviceStatusRevoked, revoked.Status)
}

func TestRevocationSurfacesManagedKeyCacheInvalidationFailure(t *testing.T) {
	repo := &memoryRepository{
		device: Device{
			ID: 41, PublicID: testDevicePublicID, UserID: 7,
			Status: DeviceStatusActive, TokenVersion: 1,
		},
		managed: map[int64]*ManagedKey{9: {ID: 1, GroupID: 9, Key: "sk-desktop-managed"}},
	}
	invalidator := &recordingInvalidator{err: errors.New("redis unavailable")}
	svc := NewService(&memoryPairingStore{}, repo, staticRoutes{}, invalidator, &config.Config{})

	_, err := svc.RevokeDevice(context.Background(), 7, testDevicePublicID)
	require.ErrorIs(t, err, ErrManagedKeyCache)
	require.Equal(t, DeviceStatusRevoked, repo.device.Status,
		"database revocation remains authoritative when cache delivery fails")
	require.Equal(t, []string{"sk-desktop-managed"}, invalidator.keys)
}
