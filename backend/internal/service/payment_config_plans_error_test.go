//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	entintercept "github.com/Wei-Shaw/sub2api/ent/intercept"
	"github.com/stretchr/testify/require"
)

func TestPaymentConfigServiceGetGroupInfoMapPropagatesQueryError(t *testing.T) {
	client := newPaymentConfigServiceTestClient(t)
	queryErr := errors.New("group query failed")
	client.Group.Intercept(entintercept.Func(func(context.Context, entintercept.Query) error {
		return queryErr
	}))

	svc := NewPaymentConfigService(client, nil, nil)
	_, err := svc.GetGroupInfoMap(context.Background(), []*dbent.SubscriptionPlan{{GroupID: 42}})

	require.ErrorIs(t, err, queryErr)
	require.ErrorContains(t, err, "query plan groups")
}

func TestPaymentConfigServiceGetGroupInfoMapSkipsQueryWithoutPlans(t *testing.T) {
	client := newPaymentConfigServiceTestClient(t)
	client.Group.Intercept(entintercept.Func(func(context.Context, entintercept.Query) error {
		return errors.New("group query should not run")
	}))

	svc := NewPaymentConfigService(client, nil, nil)
	groupInfo, err := svc.GetGroupInfoMap(context.Background(), nil)

	require.NoError(t, err)
	require.Empty(t, groupInfo)
}
