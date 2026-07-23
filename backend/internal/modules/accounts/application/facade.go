package application

import (
	"context"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/modules/accounts/domain"
	"github.com/Wei-Shaw/sub2api/internal/modules/accounts/ports"
)

var ErrAccountUnitOfWorkUnavailable = errors.New("account unit of work is unavailable")

type Facade struct {
	unitOfWork ports.AccountUnitOfWork
}

func NewFacade(unitOfWork ports.AccountUnitOfWork) *Facade {
	return &Facade{unitOfWork: unitOfWork}
}

func (f *Facade) CreateAccount(ctx context.Context, command domain.CreateAccountCommand) (domain.Account, error) {
	if f == nil || f.unitOfWork == nil {
		return domain.Account{}, ErrAccountUnitOfWorkUnavailable
	}
	command.Credentials = cloneMap(command.Credentials)
	command.Extra = cloneMap(command.Extra)
	command.Groups = append([]domain.GroupBinding(nil), command.Groups...)
	return f.unitOfWork.CreateWithGroups(ctx, command)
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
