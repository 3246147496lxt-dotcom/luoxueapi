//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

type updateMaskAPIKeyRepo struct {
	quotaBaseAPIKeyRepoStub
	key    *APIKey
	fields []APIKeyUpdateFields
}

func (r *updateMaskAPIKeyRepo) GetByID(context.Context, int64) (*APIKey, error) {
	c := *r.key
	return &c, nil
}
func (r *updateMaskAPIKeyRepo) Update(_ context.Context, _ *APIKey, f APIKeyUpdateFields) error {
	r.fields = append(r.fields, f)
	return nil
}
func (r *updateMaskAPIKeyRepo) IncrementQuotaUsed(_ context.Context, _ int64, amount float64) (float64, error) {
	r.key.QuotaUsed += amount
	return r.key.QuotaUsed, nil
}

func TestAPIKeyUpdateUsesExplicitMask(t *testing.T) {
	name := "renamed"
	r := &updateMaskAPIKeyRepo{key: &APIKey{ID: 1, UserID: 2, Key: "sk-test", Name: "old", Status: StatusActive, Quota: 100, QuotaUsed: 5}}
	s := &APIKeyService{apiKeyRepo: r}
	_, err := s.Update(context.Background(), 1, 2, UpdateAPIKeyRequest{Name: &name})
	require.NoError(t, err)
	require.Equal(t, []APIKeyUpdateFields{{Name: true}}, r.fields)
}

func TestUpdateQuotaUsedOnlyWritesStatus(t *testing.T) {
	r := &updateMaskAPIKeyRepo{key: &APIKey{ID: 1, UserID: 2, Key: "sk-test", Status: StatusActive, Quota: 10, QuotaUsed: 9}}
	s := &APIKeyService{apiKeyRepo: r}
	require.NoError(t, s.UpdateQuotaUsed(context.Background(), 1, 2))
	require.Equal(t, []APIKeyUpdateFields{{Status: true}}, r.fields)
}
