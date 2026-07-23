package application

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/modules/accounts/domain"
	"github.com/stretchr/testify/require"
)

type accountUnitOfWorkStub struct {
	command domain.CreateAccountCommand
}

func (s *accountUnitOfWorkStub) CreateWithGroups(_ context.Context, command domain.CreateAccountCommand) (domain.Account, error) {
	s.command = command
	return domain.Account{ID: 41, Name: command.Name, Groups: command.Groups}, nil
}

func TestCreateAccountDelegatesOneAggregateCommand(t *testing.T) {
	uow := &accountUnitOfWorkStub{}
	facade := NewFacade(uow)
	credentials := map[string]any{"token": "secret"}
	command := domain.CreateAccountCommand{
		Name:        "account",
		Credentials: credentials,
		Groups:      []domain.GroupBinding{{GroupID: 7, Priority: 1}},
	}

	created, err := facade.CreateAccount(context.Background(), command)
	credentials["token"] = "mutated"

	require.NoError(t, err)
	require.EqualValues(t, 41, created.ID)
	require.Equal(t, "secret", uow.command.Credentials["token"])
	require.Equal(t, []domain.GroupBinding{{GroupID: 7, Priority: 1}}, uow.command.Groups)
}
