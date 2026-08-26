//go:build integration

package repository

import (
	"context"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/redeemcode"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// TestCreateWithEmailAliasGuardJoinsOuterTransaction verifies that user
// creation joins an ent transaction opened by the caller.  This is the basis
// for the registration path's "create user + claim invitation" atomicity:
// rolling back the outer transaction must undo both writes, while committing
// it must persist both writes together.
func TestCreateWithEmailAliasGuardJoinsOuterTransaction(t *testing.T) {
	client := testEntClient(t)
	userRepo := NewUserRepository(client, integrationDB)
	redeemRepo := NewRedeemCodeRepository(client)

	ctx := context.Background()

	// The commit case intentionally leaves a small amount of data until the
	// test cleanup so this test does not affect other integration tests.
	var committedUserEmails []string
	var seededCodeIDs []int64
	t.Cleanup(func() {
		if len(committedUserEmails) > 0 {
			_, _ = client.User.Delete().Where(user.EmailIn(committedUserEmails...)).Exec(ctx)
		}
		if len(seededCodeIDs) > 0 {
			_, _ = client.RedeemCode.Delete().Where(redeemcode.IDIn(seededCodeIDs...)).Exec(ctx)
		}
	})

	seedCode := func(code string) int64 {
		_, err := client.RedeemCode.Create().
			SetCode(code).
			SetType(service.RedeemTypeInvitation).
			SetStatus(service.StatusUnused).
			SetValue(0).
			Save(ctx)
		require.NoError(t, err, "seed redeem code")
		c, err := client.RedeemCode.Query().Where(redeemcode.CodeEQ(code)).Only(ctx)
		require.NoError(t, err)
		seededCodeIDs = append(seededCodeIDs, c.ID)
		return c.ID
	}

	t.Run("rollback removes user and releases claim", func(t *testing.T) {
		codeID := seedCode("ITX-RACE-ROLLBACK-001")
		tx, err := client.Tx(ctx)
		require.NoError(t, err)
		txCtx := dbent.NewTxContext(ctx, tx)

		u := &service.User{
			Email:        "itx-rollback@example.com",
			PasswordHash: "test-password-hash",
			Role:         service.RoleUser,
			Status:       service.StatusActive,
			Balance:      0,
			Concurrency:  1,
		}
		guardRepo, ok := userRepo.(interface {
			CreateWithEmailAliasGuard(context.Context, *service.User) error
		})
		require.True(t, ok, "repository must expose the guarded registration capability")
		require.NoError(t, guardRepo.CreateWithEmailAliasGuard(txCtx, u))
		require.Greater(t, u.ID, int64(0), "create should populate user ID")
		require.NoError(t, redeemRepo.Use(txCtx, codeID, u.ID))
		require.NoError(t, tx.Rollback())

		exists, err := userRepo.ExistsByEmail(ctx, "itx-rollback@example.com")
		require.NoError(t, err)
		require.False(t, exists, "rollback must not leave an orphan user")

		after, err := client.RedeemCode.Get(ctx, codeID)
		require.NoError(t, err)
		require.Equal(t, service.StatusUnused, after.Status, "rollback must release the invitation")
	})

	t.Run("commit persists user and claim together", func(t *testing.T) {
		codeID := seedCode("ITX-RACE-COMMIT-001")
		tx, err := client.Tx(ctx)
		require.NoError(t, err)
		txCtx := dbent.NewTxContext(ctx, tx)

		u := &service.User{
			Email:        "itx-commit@example.com",
			PasswordHash: "test-password-hash",
			Role:         service.RoleUser,
			Status:       service.StatusActive,
			Balance:      0,
			Concurrency:  1,
		}
		guardRepo, ok := userRepo.(interface {
			CreateWithEmailAliasGuard(context.Context, *service.User) error
		})
		require.True(t, ok, "repository must expose the guarded registration capability")
		require.NoError(t, guardRepo.CreateWithEmailAliasGuard(txCtx, u))
		require.NoError(t, redeemRepo.Use(txCtx, codeID, u.ID))
		require.NoError(t, tx.Commit())
		committedUserEmails = append(committedUserEmails, u.Email)

		exists, err := userRepo.ExistsByEmail(ctx, "itx-commit@example.com")
		require.NoError(t, err)
		require.True(t, exists, "commit must persist the user")

		after, err := client.RedeemCode.Get(ctx, codeID)
		require.NoError(t, err)
		require.Equal(t, service.StatusUsed, after.Status, "commit must persist the invitation claim")
	})
}
