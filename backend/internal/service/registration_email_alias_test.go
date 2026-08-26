//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeEmailForAliasDedupGuardsProviderVariants(t *testing.T) {
	cases := map[string]string{
		"User+tag@gmail.com":     "user@gmail.com",
		"s.o.m.e@googlemail.com": "some@gmail.com",
		"user@example.com.":      "user@example.com",
		"+alice@gmail.com":       "+alice@gmail.com",
		"first.last@qq.com":      "first.last@qq.com",
	}
	for input, want := range cases {
		require.Equal(t, want, NormalizeEmailForAliasDedup(input), input)
	}
}

type aliasDedupRepoStub struct {
	UserRepository
	exists    bool
	existsErr error
	alias     bool
	aliasErr  error
}

func (s *aliasDedupRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	return s.exists, s.existsErr
}
func (s *aliasDedupRepoStub) ExistsByEmailAlias(context.Context, string) (bool, error) {
	return s.alias, s.aliasErr
}

func TestExistsByEmailOrAliasFailsClosedAndShortCircuits(t *testing.T) {
	ctx := context.Background()
	svc := &AuthService{userRepo: &aliasDedupRepoStub{exists: true, aliasErr: errors.New("must not call")}}
	found, err := svc.existsByEmailOrAlias(ctx, "user@gmail.com")
	require.NoError(t, err)
	require.True(t, found)

	svc = &AuthService{userRepo: &aliasDedupRepoStub{aliasErr: errors.New("db down")}}
	_, err = svc.existsByEmailOrAlias(ctx, "user@gmail.com")
	require.Error(t, err)
}
