package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserRepositoryBalanceMutationsIgnoreSoftDeletedUsers(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()

	user := &service.User{
		Email:        "balance-soft-delete@example.com",
		Username:     "balance-soft-delete",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Balance:      10,
	}
	require.NoError(t, repo.Create(ctx, user))

	_, err := client.User.UpdateOneID(user.ID).SetDeletedAt(time.Now().UTC()).Save(ctx)
	require.NoError(t, err)

	require.ErrorIs(t, repo.UpdateBalance(ctx, user.ID, 2), service.ErrUserNotFound)
	require.ErrorIs(t, repo.DeductBalance(ctx, user.ID, 2), service.ErrUserNotFound)

	deleted, err := repo.GetByIDIncludeDeleted(ctx, user.ID)
	require.NoError(t, err)
	require.InDelta(t, 10.0, deleted.Balance, 1e-9)
}
