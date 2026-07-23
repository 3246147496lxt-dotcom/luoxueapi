package service

import (
	"context"

	accountsapp "github.com/Wei-Shaw/sub2api/internal/modules/accounts/application"
	accountsdomain "github.com/Wei-Shaw/sub2api/internal/modules/accounts/domain"
	accountsports "github.com/Wei-Shaw/sub2api/internal/modules/accounts/ports"
)

type LegacyAccountUnitOfWorkAdapter struct {
	repository AccountDuplicateRepository
}

func NewLegacyAccountUnitOfWorkAdapter(repository AccountDuplicateRepository) *LegacyAccountUnitOfWorkAdapter {
	return &LegacyAccountUnitOfWorkAdapter{repository: repository}
}

func NewAccountsModuleFacade(repository AccountDuplicateRepository) *accountsapp.Facade {
	return accountsapp.NewFacade(NewLegacyAccountUnitOfWorkAdapter(repository))
}

// CreateAccountWithModuleFacade routes the live account-creation path through
// the modular AccountUnitOfWork while preserving the legacy service model used
// by existing handlers and post-create workflows. The adapter remains the only
// place that translates between the legacy aggregate and the module domain.
func CreateAccountWithModuleFacade(
	ctx context.Context,
	repository AccountDuplicateRepository,
	account *Account,
	groups []AccountGroup,
) error {
	if account == nil {
		return ErrAccountNilInput
	}
	bindings := make([]accountsdomain.GroupBinding, 0, len(groups))
	for _, group := range groups {
		bindings = append(bindings, accountsdomain.GroupBinding{
			GroupID:  group.GroupID,
			Priority: group.Priority,
		})
	}

	created, err := NewAccountsModuleFacade(repository).CreateAccount(ctx, accountsdomain.CreateAccountCommand{
		Name:                account.Name,
		Notes:               account.Notes,
		Platform:            account.Platform,
		Type:                account.Type,
		Credentials:         account.Credentials,
		Extra:               account.Extra,
		ProxyID:             account.ProxyID,
		Concurrency:         account.Concurrency,
		Priority:            account.Priority,
		RateMultiplier:      account.RateMultiplier,
		LoadFactor:          account.LoadFactor,
		Status:              account.Status,
		Schedulable:         account.Schedulable,
		ExpiresAt:           account.ExpiresAt,
		AutoPauseOnExpired:  account.AutoPauseOnExpired,
		ParentAccountID:     account.ParentAccountID,
		QuotaDimension:      account.QuotaDimension,
		LastUsedAt:          account.LastUsedAt,
		RateLimitedAt:       account.RateLimitedAt,
		RateLimitResetAt:    account.RateLimitResetAt,
		OverloadUntil:       account.OverloadUntil,
		SessionWindowStart:  account.SessionWindowStart,
		SessionWindowEnd:    account.SessionWindowEnd,
		SessionWindowStatus: account.SessionWindowStatus,
		Groups:              bindings,
	})
	if err != nil {
		return err
	}

	account.ID = created.ID
	account.CreatedAt = created.CreatedAt
	account.UpdatedAt = created.UpdatedAt
	account.GroupIDs = make([]int64, 0, len(created.Groups))
	account.AccountGroups = make([]AccountGroup, 0, len(created.Groups))
	for _, binding := range created.Groups {
		account.GroupIDs = append(account.GroupIDs, binding.GroupID)
		account.AccountGroups = append(account.AccountGroups, AccountGroup{
			AccountID: account.ID,
			GroupID:   binding.GroupID,
			Priority:  binding.Priority,
		})
	}
	return nil
}

func (a *LegacyAccountUnitOfWorkAdapter) CreateWithGroups(ctx context.Context, command accountsdomain.CreateAccountCommand) (accountsdomain.Account, error) {
	if a == nil || a.repository == nil {
		return accountsdomain.Account{}, accountsapp.ErrAccountUnitOfWorkUnavailable
	}
	account := &Account{
		Name:                command.Name,
		Notes:               command.Notes,
		Platform:            command.Platform,
		Type:                command.Type,
		Credentials:         command.Credentials,
		Extra:               command.Extra,
		ProxyID:             command.ProxyID,
		Concurrency:         command.Concurrency,
		Priority:            command.Priority,
		RateMultiplier:      command.RateMultiplier,
		LoadFactor:          command.LoadFactor,
		Status:              command.Status,
		Schedulable:         command.Schedulable,
		ExpiresAt:           command.ExpiresAt,
		AutoPauseOnExpired:  command.AutoPauseOnExpired,
		ParentAccountID:     command.ParentAccountID,
		QuotaDimension:      command.QuotaDimension,
		LastUsedAt:          command.LastUsedAt,
		RateLimitedAt:       command.RateLimitedAt,
		RateLimitResetAt:    command.RateLimitResetAt,
		OverloadUntil:       command.OverloadUntil,
		SessionWindowStart:  command.SessionWindowStart,
		SessionWindowEnd:    command.SessionWindowEnd,
		SessionWindowStatus: command.SessionWindowStatus,
	}
	groups := make([]AccountGroup, 0, len(command.Groups))
	for _, binding := range command.Groups {
		groups = append(groups, AccountGroup{GroupID: binding.GroupID, Priority: binding.Priority})
	}
	if err := a.repository.CreateWithAccountGroups(ctx, account, groups); err != nil {
		return accountsdomain.Account{}, err
	}
	resultGroups := make([]accountsdomain.GroupBinding, 0, len(groups))
	for _, binding := range groups {
		resultGroups = append(resultGroups, accountsdomain.GroupBinding{GroupID: binding.GroupID, Priority: binding.Priority})
	}
	return accountsdomain.Account{
		ID:        account.ID,
		Name:      account.Name,
		Platform:  account.Platform,
		Type:      account.Type,
		CreatedAt: account.CreatedAt,
		UpdatedAt: account.UpdatedAt,
		Groups:    resultGroups,
	}, nil
}

var _ accountsports.AccountUnitOfWork = (*LegacyAccountUnitOfWorkAdapter)(nil)
