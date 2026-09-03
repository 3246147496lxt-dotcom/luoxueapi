//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// oauthAliasRaceRepo models the repository boundary used by OAuth signup: the
// initial exact lookup is intentionally racy, while the guarded create is the
// authoritative canonical-inbox reservation.
type oauthAliasRaceRepo struct {
	UserRepository

	mu      sync.Mutex
	nextID  int64
	users   map[string]*User
	created int
}

func (r *oauthAliasRaceRepo) GetByEmail(_ context.Context, email string) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user, ok := r.users[email]; ok {
		clone := *user
		return &clone, nil
	}
	return nil, ErrUserNotFound
}

func (r *oauthAliasRaceRepo) CreateWithEmailAliasGuard(_ context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	identity := NormalizeEmailForAliasDedup(user.Email)
	for email := range r.users {
		if NormalizeEmailForAliasDedup(email) == identity {
			return ErrEmailExists
		}
	}
	r.nextID++
	user.ID = r.nextID
	clone := *user
	r.users[user.Email] = &clone
	r.created++
	return nil
}

func TestAuthServiceCreateEmailOAuthUser_AliasCollisionMapsToEmailExists(t *testing.T) {
	repo := &oauthAliasRaceRepo{nextID: 100, users: map[string]*User{
		"owner@gmail.com": {ID: 7, Email: "owner@gmail.com", Status: StatusActive},
	}}
	svc := newEmailOAuthAutoAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	_, created, err := svc.createEmailOAuthUser(context.Background(), "owner+tag@gmail.com", "oauth", "github", "", "")
	require.ErrorIs(t, err, ErrEmailExists)
	require.False(t, created)
	require.Equal(t, 0, repo.created)
}

func TestAuthServiceLoginOrRegisterOAuth_AliasRaceCreatesOnlyOneUser(t *testing.T) {
	repo := &oauthAliasRaceRepo{users: make(map[string]*User)}
	svc := newEmailOAuthAutoAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	const attempts = 2
	start := make(chan struct{})
	results := make(chan error, attempts)
	emails := []string{"race+tag@gmail.com", "race@gmail.com"}
	for i := 0; i < attempts; i++ {
		email := emails[i]
		go func() {
			<-start
			_, _, err := svc.LoginOrRegisterOAuthWithTokenPair(
				context.Background(), email, "oauth", "", "", "github",
			)
			results <- err
		}()
	}
	close(start)

	var successes, conflicts int
	for i := 0; i < attempts; i++ {
		switch err := <-results; {
		case err == nil:
			successes++
		case errors.Is(err, ErrEmailExists):
			conflicts++
		default:
			t.Fatalf("unexpected OAuth race result: %v", err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)
	require.Equal(t, 1, repo.created)
}

func TestAuthServiceLoginOrRegisterOAuth_AliasCollisionMapsToEmailExists(t *testing.T) {
	repo := &oauthAliasRaceRepo{users: map[string]*User{
		"owner@gmail.com": {ID: 7, Email: "owner@gmail.com", Status: StatusActive},
	}}
	svc := newEmailOAuthAutoAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	_, _, err := svc.LoginOrRegisterOAuthWithTokenPair(
		context.Background(), "owner+tag@gmail.com", "oauth", "", "", "github",
	)
	require.ErrorIs(t, err, ErrEmailExists)
	require.Equal(t, 0, repo.created)
}

func TestAuthServiceLoginOrRegisterOAuthLegacy_AliasCollisionMapsToEmailExists(t *testing.T) {
	repo := &oauthAliasRaceRepo{users: map[string]*User{
		"owner@gmail.com": {ID: 7, Email: "owner@gmail.com", Status: StatusActive},
	}}
	svc := newEmailOAuthAutoAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	_, _, err := svc.LoginOrRegisterOAuth(
		context.Background(), "owner+tag@gmail.com", "oauth",
	)
	require.ErrorIs(t, err, ErrEmailExists)
	require.Equal(t, 0, repo.created)
}

func TestCreateUserAndClaimInvitation_FailsClosedWithoutRedeemRepository(t *testing.T) {
	repo := &oauthAliasRaceRepo{users: make(map[string]*User)}
	svc := newEmailOAuthAutoAuthService(repo, map[string]string{
		SettingKeyRegistrationEnabled: "true",
	}, nil)

	err := svc.createUserAndClaimInvitation(context.Background(), &User{
		Email:  "new@example.com",
		Status: StatusActive,
	}, &RedeemCode{ID: 1, Code: "INVITE"})
	require.ErrorIs(t, err, ErrServiceUnavailable)
	require.Equal(t, 0, repo.created, "missing invitation repository must not create an account")
}
