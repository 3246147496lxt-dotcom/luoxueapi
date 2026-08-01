package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyConfiguredMigrationsRejectsNilContext(t *testing.T) {
	err := ApplyConfiguredMigrations(nil, nil)
	require.ErrorContains(t, err, "context is nil")
}

func TestApplyConfiguredMigrationsRejectsNilConfig(t *testing.T) {
	err := ApplyConfiguredMigrations(context.Background(), nil)
	require.ErrorContains(t, err, "database config is nil")
}
