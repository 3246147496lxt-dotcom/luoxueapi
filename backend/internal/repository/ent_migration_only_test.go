package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyConfiguredMigrationsRejectsNilContext(t *testing.T) {
	var nilContext context.Context
	err := ApplyConfiguredMigrations(nilContext, nil, ConfiguredDatabaseIdentity{})
	require.ErrorContains(t, err, "context is nil")
}

func TestApplyConfiguredMigrationsRequiresExpectedIdentity(t *testing.T) {
	err := ApplyConfiguredMigrations(
		context.Background(),
		nil,
		ConfiguredDatabaseIdentity{},
	)
	require.ErrorContains(t, err, "expected database identity contract")
}

func TestApplyConfiguredMigrationsRejectsNilConfig(t *testing.T) {
	err := ApplyConfiguredMigrations(context.Background(), nil, ConfiguredDatabaseIdentity{
		Contract:         ConfiguredDatabaseIdentityContract,
		Database:         "sub2api",
		SystemIdentifier: "7612345678901234567",
		InRecovery:       false,
	})
	require.ErrorContains(t, err, "database config is nil")
}
