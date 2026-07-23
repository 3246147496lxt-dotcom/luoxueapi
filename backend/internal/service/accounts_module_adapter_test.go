package service

import (
	"context"
	"testing"
	"time"

	accountsapp "github.com/Wei-Shaw/sub2api/internal/modules/accounts/application"
	accountsdomain "github.com/Wei-Shaw/sub2api/internal/modules/accounts/domain"
	"github.com/stretchr/testify/require"
)

type accountUnitOfWorkRepositoryStub struct {
	account *Account
	groups  []AccountGroup
}

func (s *accountUnitOfWorkRepositoryStub) CreateWithAccountGroups(_ context.Context, account *Account, groups []AccountGroup) error {
	s.account = account
	s.groups = append([]AccountGroup(nil), groups...)
	account.ID = 73
	account.CreatedAt = time.Unix(100, 0).UTC()
	account.UpdatedAt = time.Unix(101, 0).UTC()
	return nil
}

func TestLegacyAccountUnitOfWorkAdapterMapsAggregateBoundary(t *testing.T) {
	repository := &accountUnitOfWorkRepositoryStub{}
	adapter := NewLegacyAccountUnitOfWorkAdapter(repository)
	rateMultiplier := 1.25
	loadFactor := 4

	created, err := adapter.CreateWithGroups(context.Background(), accountsdomain.CreateAccountCommand{
		Name:           "primary",
		Platform:       "openai",
		Type:           "oauth",
		Credentials:    map[string]any{"token": "opaque"},
		Concurrency:    2,
		Priority:       3,
		RateMultiplier: &rateMultiplier,
		LoadFactor:     &loadFactor,
		Status:         "active",
		Schedulable:    true,
		Groups: []accountsdomain.GroupBinding{
			{GroupID: 11, Priority: 5},
			{GroupID: 12, Priority: 6},
		},
	})

	require.NoError(t, err)
	require.EqualValues(t, 73, created.ID)
	require.Equal(t, "primary", repository.account.Name)
	require.Equal(t, &rateMultiplier, repository.account.RateMultiplier)
	require.Equal(t, &loadFactor, repository.account.LoadFactor)
	require.Equal(t, []AccountGroup{
		{GroupID: 11, Priority: 5},
		{GroupID: 12, Priority: 6},
	}, repository.groups)
	require.Equal(t, []accountsdomain.GroupBinding{
		{GroupID: 11, Priority: 5},
		{GroupID: 12, Priority: 6},
	}, created.Groups)
}

func TestLegacyAccountUnitOfWorkAdapterFailsClosedWithoutRepository(t *testing.T) {
	_, err := (*LegacyAccountUnitOfWorkAdapter)(nil).CreateWithGroups(context.Background(), accountsdomain.CreateAccountCommand{})
	require.ErrorIs(t, err, accountsapp.ErrAccountUnitOfWorkUnavailable)
}

func TestCreateAccountWithModuleFacadePreservesLegacyAggregateResult(t *testing.T) {
	repository := &accountUnitOfWorkRepositoryStub{}
	account := &Account{
		Name:        "facade-live-path",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Credentials: map[string]any{"access_token": "opaque"},
		Status:      StatusActive,
		Schedulable: true,
	}

	err := CreateAccountWithModuleFacade(context.Background(), repository, account, []AccountGroup{
		{GroupID: 41, Priority: 1},
		{GroupID: 42, Priority: 2},
	})

	require.NoError(t, err)
	require.EqualValues(t, 73, account.ID)
	require.Equal(t, time.Unix(100, 0).UTC(), account.CreatedAt)
	require.Equal(t, []int64{41, 42}, account.GroupIDs)
	require.Equal(t, []AccountGroup{
		{AccountID: 73, GroupID: 41, Priority: 1},
		{AccountID: 73, GroupID: 42, Priority: 2},
	}, account.AccountGroups)
	require.NotSame(t, account, repository.account)
	require.Equal(t, account.Credentials, repository.account.Credentials)
}
