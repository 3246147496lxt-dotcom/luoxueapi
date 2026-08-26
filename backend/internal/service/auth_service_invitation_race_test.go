//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// invitationRaceUserRepo models the repository-side uniqueness guard used by
// registration. The embedded interface keeps this fixture narrow while the
// overridden methods make concurrent registration deterministic.
type invitationRaceUserRepo struct {
	UserRepository

	mu      sync.Mutex
	nextID  int64
	byEmail map[string]*User
	byID    map[int64]*User
}

func newInvitationRaceUserRepo() *invitationRaceUserRepo {
	return &invitationRaceUserRepo{
		nextID:  1,
		byEmail: make(map[string]*User),
		byID:    make(map[int64]*User),
	}
}

func (r *invitationRaceUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.byEmail[email]
	return ok, nil
}

func (r *invitationRaceUserRepo) ExistsByEmailAlias(ctx context.Context, email string) (bool, error) {
	identity := NormalizeEmailForAliasDedup(email)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, user := range r.byEmail {
		if NormalizeEmailForAliasDedup(user.Email) == identity {
			return true, nil
		}
	}
	return false, nil
}

func (r *invitationRaceUserRepo) CreateWithEmailAliasGuard(_ context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	identity := NormalizeEmailForAliasDedup(user.Email)
	for _, existing := range r.byEmail {
		if NormalizeEmailForAliasDedup(existing.Email) == identity {
			return ErrEmailExists
		}
	}
	user.ID = r.nextID
	r.nextID++
	clone := *user
	r.byEmail[user.Email] = &clone
	r.byID[user.ID] = &clone
	return nil
}

func (r *invitationRaceUserRepo) GetByEmail(_ context.Context, email string) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.byEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

func (r *invitationRaceUserRepo) GetByID(_ context.Context, id int64) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.byID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *user
	return &clone, nil
}

type invitationRaceRedeemRepo struct {
	RedeemCodeRepository

	mu    sync.Mutex
	codes map[string]*RedeemCode
}

func (r *invitationRaceRedeemRepo) GetByCode(_ context.Context, code string) (*RedeemCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	redeem, ok := r.codes[code]
	if !ok {
		return nil, ErrRedeemCodeNotFound
	}
	clone := *redeem
	return &clone, nil
}

func (r *invitationRaceRedeemRepo) Use(_ context.Context, id, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, redeem := range r.codes {
		if redeem.ID != id {
			continue
		}
		if redeem.Status != StatusUnused {
			return ErrRedeemCodeUsed
		}
		now := time.Now().UTC()
		redeem.Status = StatusUsed
		redeem.UsedBy = &userID
		redeem.UsedAt = &now
		return nil
	}
	return ErrRedeemCodeNotFound
}

func TestAuthServiceRegisterInvitationCodeFailsClosedWithoutTransactionCoordinator(t *testing.T) {
	const code = "INV-RACE-001"
	userRepo := newInvitationRaceUserRepo()
	redeemRepo := &invitationRaceRedeemRepo{codes: map[string]*RedeemCode{
		code: {ID: 1, Code: code, Type: RedeemTypeInvitation, Status: StatusUnused},
	}}
	svc := newOAuthEmailFlowAuthService(
		userRepo,
		redeemRepo,
		&refreshTokenCacheStub{},
		map[string]string{
			"registration_enabled":    "true",
			"invitation_code_enabled": "true",
		},
		nil,
		&userPlatformQuotaRepoStub{},
	)

	const attempts = 8
	start := make(chan struct{})
	results := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_, _, err := svc.RegisterWithVerification(
				context.Background(),
				fmt.Sprintf("race-%d@example.com", index),
				"Password123!", "", "", code, "",
			)
			results <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	serviceUnavailable := 0
	for err := range results {
		if errors.Is(err, ErrServiceUnavailable) {
			serviceUnavailable++
			continue
		}
		t.Fatalf("unexpected registration error: %v", err)
	}
	require.Equal(t, attempts, serviceUnavailable)
	userRepo.mu.Lock()
	require.Empty(t, userRepo.byEmail, "fail-closed registration must not create orphan users")
	userRepo.mu.Unlock()

	claimed, err := redeemRepo.GetByCode(context.Background(), code)
	require.NoError(t, err)
	require.Equal(t, StatusUnused, claimed.Status)
	require.Nil(t, claimed.UsedBy)
}

func TestAuthServiceRegisterRejectsAlreadyUsedInvitationCode(t *testing.T) {
	const code = "INV-RACE-002"
	userRepo := newInvitationRaceUserRepo()
	redeemRepo := &invitationRaceRedeemRepo{codes: map[string]*RedeemCode{
		code: {ID: 2, Code: code, Type: RedeemTypeInvitation, Status: StatusUsed},
	}}
	svc := newOAuthEmailFlowAuthService(
		userRepo,
		redeemRepo,
		&refreshTokenCacheStub{},
		map[string]string{
			"registration_enabled":    "true",
			"invitation_code_enabled": "true",
		},
		nil,
		&userPlatformQuotaRepoStub{},
	)

	_, _, err := svc.RegisterWithVerification(
		context.Background(), "later@example.com", "Password123!", "", "", code, "",
	)
	require.ErrorIs(t, err, ErrInvitationCodeInvalid)
}
