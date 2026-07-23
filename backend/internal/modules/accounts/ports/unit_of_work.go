package ports

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/modules/accounts/domain"
)

// AccountUnitOfWork owns the atomic boundary for an account aggregate: account
// row, group bindings, and its durable scheduling-change notification.
type AccountUnitOfWork interface {
	CreateWithGroups(ctx context.Context, command domain.CreateAccountCommand) (domain.Account, error)
}
