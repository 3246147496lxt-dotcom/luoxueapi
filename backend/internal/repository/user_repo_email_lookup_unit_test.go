package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newUserEntRepo(t *testing.T) (*userRepository, *dbent.Client) {
	t.Helper()

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(10)

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return newUserRepositoryWithSQL(client, db), client
}

func TestUserRepositoryGetByEmailNormalizesLegacySpacingAndCase(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()

	err := repo.Create(ctx, &service.User{
		Email:        " Legacy@Example.com ",
		Username:     "legacy-user",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	require.NoError(t, err)

	got, err := repo.GetByEmail(ctx, "legacy@example.com")
	require.NoError(t, err)
	require.Equal(t, " Legacy@Example.com ", got.Email)
}

func TestUserRepositoryExistsByEmailNormalizesLegacySpacingAndCase(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()

	err := repo.Create(ctx, &service.User{
		Email:        " Legacy@Example.com ",
		Username:     "legacy-user",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	require.NoError(t, err)

	exists, err := repo.ExistsByEmail(ctx, "  LEGACY@example.com  ")
	require.NoError(t, err)
	require.True(t, exists)
}

func TestUserRepositoryExistsByEmailAliasDetectsGmailVariants(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, &service.User{
		Email: "some.one+seed@gmail.com", PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive,
	}))
	found, err := repo.ExistsByEmailAlias(ctx, "someone@gmail.com")
	require.NoError(t, err)
	require.True(t, found)
	found, err = repo.ExistsByEmailAlias(ctx, "other@gmail.com")
	require.NoError(t, err)
	require.False(t, found)
}

func TestUserRepositoryExistsByEmailAliasCoversProviderAndInputVariants(t *testing.T) {
	cases := []struct {
		name   string
		stored string
		probe  string
		want   bool
	}{
		{name: "same address", stored: "someone@gmail.com", probe: "someone@gmail.com", want: true},
		{name: "gmail plus alias", stored: "someone@gmail.com", probe: "someone+bulk294@gmail.com", want: true},
		{name: "gmail dot trick", stored: "d.axis.2026@gmail.com", probe: "daxis2026@gmail.com", want: true},
		{name: "gmail dot trick both sides", stored: "d.axis.2026@gmail.com", probe: "da.xis.2026@gmail.com", want: true},
		{name: "googlemail family", stored: "someone@googlemail.com", probe: "some.one@gmail.com", want: true},
		{name: "root dot on probe", stored: "d.axis.2026@gmail.com", probe: "da.xis.2026@gmail.com.", want: true},
		{name: "root dot on stored row", stored: "d.axis.2026@gmail.com.", probe: "daxis2026@gmail.com", want: true},
		{name: "multiple root dots on stored row", stored: "d.axis.2026@gmail.com...", probe: "daxis2026@gmail.com", want: true},
		{name: "legacy spacing and case", stored: "  D.Axis.2026@Gmail.com  ", probe: "daxis2026@gmail.com", want: true},
		{name: "non-gmail plus alias", stored: "first.last@qq.com", probe: "first.last+tag@qq.com", want: true},
		{name: "plus-prefixed locals remain distinct", stored: "+alice+tag@gmail.com", probe: "+alice@gmail.com", want: false},
		{name: "different gmail inbox", stored: "someone@gmail.com", probe: "someoneelse@gmail.com", want: false},
		{name: "non-gmail dots significant", stored: "first.last@qq.com", probe: "firstlast@qq.com", want: false},
		{name: "different domain", stored: "someone@gmail.com", probe: "someone@qq.com", want: false},
		{name: "distinct plus-prefixed locals", stored: "+alice@gmail.com", probe: "+bob@gmail.com", want: false},
		{name: "underscore is literal", stored: "user_x@qq.com", probe: "userax@qq.com", want: false},
		{name: "percent is literal", stored: "a%b@qq.com", probe: "axxb@qq.com", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, _ := newUserEntRepo(t)
			require.NoError(t, repo.Create(context.Background(), &service.User{
				Email: tc.stored, Username: "alias-test", PasswordHash: "hash",
				Role: service.RoleUser, Status: service.StatusActive,
			}))
			got, err := repo.ExistsByEmailAlias(context.Background(), tc.probe)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestUserRepositoryExistsByEmailAliasIgnoresMalformedInput(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	require.NoError(t, repo.Create(context.Background(), &service.User{
		Email: "someone@gmail.com", Username: "alias-test", PasswordHash: "hash",
		Role: service.RoleUser, Status: service.StatusActive,
	}))
	found, err := repo.ExistsByEmailAlias(context.Background(), "not-an-email")
	require.NoError(t, err)
	require.False(t, found)
}

func TestUserRepositoryExistsByEmailAliasDoesNotTruncateCandidates(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()
	for i := 0; i < 60; i++ {
		require.NoError(t, repo.Create(ctx, &service.User{
			Email:        fmt.Sprintf("crowded+%03d@gmail.com", i),
			Username:     fmt.Sprintf("crowded-%03d", i),
			PasswordHash: "hash",
			Role:         service.RoleUser,
			Status:       service.StatusActive,
		}))
	}

	// All rows resolve to one inbox.  The lookup must still find a conflict
	// when the table contains more than the old fixed 50-row probe limit.
	found, err := repo.ExistsByEmailAlias(ctx, "crowded@gmail.com")
	require.NoError(t, err)
	require.True(t, found)
}

func TestUserRepositoryCreateWithEmailAliasGuardSerializesInboxVariants(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()
	first := &service.User{Email: "alias.user@gmail.com", PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive}
	require.NoError(t, repo.CreateWithEmailAliasGuard(ctx, first))
	second := &service.User{Email: "a.l.i.a.s.u.s.e.r+two@googlemail.com", PasswordHash: "hash", Role: service.RoleUser, Status: service.StatusActive}
	require.ErrorIs(t, repo.CreateWithEmailAliasGuard(ctx, second), service.ErrEmailExists)
}

func TestUserRepositoryUpdateJoinsOuterTransaction(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()

	user := &service.User{
		Email:        "tx-update@example.com",
		Username:     "before",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, user))

	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	txCtx := dbent.NewTxContext(ctx, tx)

	user.Username = "after"
	require.NoError(t, repo.Update(txCtx, user, service.UserUpdateFields{Username: true}))

	// Reads made with the same transaction context must observe the uncommitted
	// update; this also proves Update did not silently open a nested transaction.
	inTx, err := repo.GetByID(txCtx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "after", inTx.Username)

	require.NoError(t, tx.Rollback())
	outside, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, "before", outside.Username)
}

func TestUserRepositoryCreateRejectsNormalizedEmailDuplicate(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()

	err := repo.Create(ctx, &service.User{
		Email:        " Existing@Example.com ",
		Username:     "existing-user",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	require.NoError(t, err)

	err = repo.Create(ctx, &service.User{
		Email:        "existing@example.com",
		Username:     "duplicate-user",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	})
	require.ErrorIs(t, err, service.ErrEmailExists)
}

func TestUserRepositoryUpdateRejectsNormalizedEmailDuplicate(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()

	first := &service.User{
		Email:        " Existing@Example.com ",
		Username:     "existing-user",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, first))

	second := &service.User{
		Email:        "second@example.com",
		Username:     "second-user",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, second))

	second.Email = " existing@example.com "
	err := repo.Update(ctx, second, service.UserUpdateFields{Email: true})
	require.ErrorIs(t, err, service.ErrEmailExists)
}

func TestUserRepositoryUpdateRejectsEmailAliasDuplicate(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()

	first := &service.User{
		Email:        "canonical.inbox@gmail.com",
		Username:     "first-user",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, first))
	second := &service.User{
		Email:        "second-user@example.com",
		Username:     "second-user",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, second))

	second.Email = "c.an.onical.inbox+support@googlemail.com"
	err := repo.Update(ctx, second, service.UserUpdateFields{Email: true})
	require.ErrorIs(t, err, service.ErrEmailExists)

	stored, err := client.User.Get(ctx, second.ID)
	require.NoError(t, err)
	require.Equal(t, "second-user@example.com", stored.Email, "failed alias update must not change the row")
}

func TestUserRepositoryGetByEmailReportsNormalizedEmailConflict(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()

	_, err := client.User.Create().
		SetEmail("Conflict@Example.com").
		SetUsername("conflict-user-1").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.User.Create().
		SetEmail(" conflict@example.com ").
		SetUsername("conflict-user-2").
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	_, err = repo.GetByEmail(ctx, "conflict@example.com")
	require.Error(t, err)
	require.ErrorContains(t, err, "normalized email lookup matched multiple users")
}

func TestUserRepositoryCreateSerializesNormalizedEmailConflictsUnderConcurrency(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()

	firstCreateStarted := make(chan struct{})
	releaseFirstCreate := make(chan struct{})
	var firstCreate sync.Once
	client.User.Use(func(next dbent.Mutator) dbent.Mutator {
		return dbent.MutateFunc(func(ctx context.Context, m dbent.Mutation) (dbent.Value, error) {
			blocked := false
			if m.Op().Is(dbent.OpCreate) {
				firstCreate.Do(func() {
					blocked = true
					close(firstCreateStarted)
				})
			}
			if blocked {
				<-releaseFirstCreate
			}
			return next.Mutate(ctx, m)
		})
	})

	type createResult struct {
		err error
	}

	results := make(chan createResult, 2)
	go func() {
		results <- createResult{err: repo.Create(ctx, &service.User{
			Email:        " Race@Example.com ",
			Username:     "race-user-1",
			PasswordHash: "hash",
			Role:         service.RoleUser,
			Status:       service.StatusActive,
		})}
	}()

	<-firstCreateStarted

	go func() {
		results <- createResult{err: repo.Create(ctx, &service.User{
			Email:        "race@example.com",
			Username:     "race-user-2",
			PasswordHash: "hash",
			Role:         service.RoleUser,
			Status:       service.StatusActive,
		})}
	}()

	time.Sleep(100 * time.Millisecond)
	close(releaseFirstCreate)

	first := <-results
	second := <-results

	errors := []error{first.err, second.err}
	successes := 0
	conflicts := 0
	for _, err := range errors {
		switch err {
		case nil:
			successes++
		case service.ErrEmailExists:
			conflicts++
		default:
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)

	count, err := client.User.Query().Where(userEmailLookupPredicate("race@example.com")).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}
