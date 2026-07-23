package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type accountProjectionWriterStub struct{}

func (*accountProjectionWriterStub) SetAccount(context.Context, *service.Account) error {
	return nil
}

func (*accountProjectionWriterStub) DeleteAccount(context.Context, int64) error {
	return nil
}

var (
	_ service.AccountProjectionWriter = (*accountProjectionWriterStub)(nil)
	_ service.AccountProjectionWriter = (*schedulerCache)(nil)
)

func TestAccountRepositoryConstructorUsesNarrowProjectionWriter(t *testing.T) {
	writer := &accountProjectionWriterStub{}
	repository := newAccountRepositoryWithSQL(nil, nil, writer)

	require.Same(t, writer, repository.accountProjection)
}

func TestAccountRepositoryProjectionBoundaryIsWiredExplicitly(t *testing.T) {
	read := func(path string) string {
		source, err := os.ReadFile(path)
		require.NoError(t, err, path)
		return string(source)
	}

	repositorySource := read("account_repo.go")
	require.NotContains(t, repositorySource, "service.SchedulerCache")
	require.Contains(t, repositorySource, "service.AccountProjectionWriter")
	require.Contains(t, repositorySource, "durable correctness and recovery source")

	wireSource := read("wire.go")
	require.Contains(t, wireSource, "ProvideAccountProjectionWriter")
	generatedWireSource := read(filepath.Join("..", "..", "cmd", "server", "wire_gen.go"))
	require.Contains(t, generatedWireSource, "repository.ProvideAccountProjectionWriter(schedulerCache)")
	require.Contains(t, generatedWireSource, "repository.NewAccountRepository(entClient, db, accountProjectionWriter)")
	require.Contains(t, generatedWireSource, "repository.NewAdminAccountRepository(entClient, db, accountProjectionWriter)")
}
