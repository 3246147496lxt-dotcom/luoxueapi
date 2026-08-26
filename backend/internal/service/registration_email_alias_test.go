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
		"user@example.com":       "user@example.com",
		"  User@Example.COM ":    "user@example.com",
		"user+tag@example.com":   "user@example.com",
		"User+tag@gmail.com":     "user@gmail.com",
		"s.o.m.e@googlemail.com": "some@gmail.com",
		"some.one@gmail.com":     "someone@gmail.com",
		"s.o.m.e+x@gmail.com":    "some@gmail.com",
		"user@example.com.":      "user@example.com",
		"first.last@qq.com.":     "first.last@qq.com",
		"+alice@gmail.com":       "+alice@gmail.com",
		"...@gmail.com":          "...@gmail.com",
		"not-an-email":           "not-an-email",
	}
	for input, want := range cases {
		require.Equal(t, want, NormalizeEmailForAliasDedup(input), input)
	}
}

func TestNormalizeEmailForAliasDedupKeepsDistinctInboxes(t *testing.T) {
	require.NotEqual(t,
		NormalizeEmailForAliasDedup("+alice@gmail.com"),
		NormalizeEmailForAliasDedup("+bob@gmail.com"),
	)
	require.NotEqual(t,
		NormalizeEmailForAliasDedup("alice@gmail.com"),
		NormalizeEmailForAliasDedup("bob@gmail.com"),
	)
}

func TestEmailAliasDedupProbes(t *testing.T) {
	require.ElementsMatch(t,
		[]EmailAliasProbe{{Local: "someone", Domain: "gmailcom"}, {Local: "someone", Domain: "googlemailcom"}},
		EmailAliasDedupProbes("Some.One+tag@gmail.com"),
	)
	require.ElementsMatch(t,
		[]EmailAliasProbe{{Local: "daxis2026", Domain: "gmailcom"}, {Local: "daxis2026", Domain: "googlemailcom"}},
		EmailAliasDedupProbes("d.axis.2026@googlemail.com."),
	)
	require.Equal(t,
		[]EmailAliasProbe{{Local: "firstlast", Domain: "qqcom"}},
		EmailAliasDedupProbes("first.last+tag@qq.com"),
	)
	require.Nil(t, EmailAliasDedupProbes("not-an-email"))
	require.Nil(t, EmailAliasDedupProbes("...@gmail.com"))
}

type aliasDedupRepoStub struct {
	UserRepository
	exists      bool
	existsErr   error
	stored      []string
	aliasErr    error
	aliasChecks []string
}

func (s *aliasDedupRepoStub) ExistsByEmail(context.Context, string) (bool, error) {
	return s.exists, s.existsErr
}
func (s *aliasDedupRepoStub) ExistsByEmailAlias(_ context.Context, email string) (bool, error) {
	s.aliasChecks = append(s.aliasChecks, email)
	if s.aliasErr != nil {
		return false, s.aliasErr
	}
	identity := NormalizeEmailForAliasDedup(email)
	for _, candidate := range s.stored {
		if NormalizeEmailForAliasDedup(candidate) == identity {
			return true, nil
		}
	}
	return false, nil
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

func TestExistsByEmailOrAliasDetectsProviderVariants(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		stored string
		query  string
	}{
		{name: "plus alias", stored: "someone+bulk294@gmail.com", query: "someone@gmail.com"},
		{name: "gmail dot alias", stored: "some.one@gmail.com", query: "someone@gmail.com"},
		{name: "fqdn root dot", stored: "d.axis.2026@gmail.com", query: "da.xis.2026@gmail.com."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &aliasDedupRepoStub{stored: []string{tc.stored}}
			svc := &AuthService{userRepo: repo}
			found, err := svc.existsByEmailOrAlias(ctx, tc.query)
			require.NoError(t, err)
			require.True(t, found)
		})
	}
}

func TestExistsByEmailOrAliasAllowsDistinctProviderInboxes(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		stored string
		query  string
	}{
		{name: "different gmail inbox", stored: "other@gmail.com", query: "user@gmail.com"},
		{name: "distinct plus-prefixed locals", stored: "+alice@gmail.com", query: "+bob@gmail.com"},
		{name: "non-gmail dots remain significant", stored: "first.last@qq.com", query: "firstlast@qq.com"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &aliasDedupRepoStub{stored: []string{tc.stored}}
			svc := &AuthService{userRepo: repo}
			found, err := svc.existsByEmailOrAlias(ctx, tc.query)
			require.NoError(t, err)
			require.False(t, found)
		})
	}
}
