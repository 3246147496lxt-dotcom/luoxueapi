package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestWriteConfiguredDatabaseIdentityRejectsNilWriter(t *testing.T) {
	require.ErrorContains(t, writeConfiguredDatabaseIdentity(nil), "writer is nil")
}

func TestReadExpectedDatabaseIdentityFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
  "contract": "sub2api-database-identity/v2",
  "database": "sub2api",
  "system_identifier": "7612345678901234567",
  "in_recovery": false
}`), 0o600))

	identity, err := readExpectedDatabaseIdentityFile(path)
	require.NoError(t, err)
	require.Equal(t, repository.ConfiguredDatabaseIdentity{
		Contract:         repository.ConfiguredDatabaseIdentityContract,
		Database:         "sub2api",
		SystemIdentifier: "7612345678901234567",
		InRecovery:       false,
	}, identity)
}

func TestReadExpectedDatabaseIdentityFileFailsClosed(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		contains string
	}{
		{
			name:     "unknown field",
			content:  `{"contract":"sub2api-database-identity/v2","database":"sub2api","system_identifier":"1","in_recovery":false,"extra":true}`,
			contains: "unknown field",
		},
		{
			name:     "multiple values",
			content:  `{"contract":"sub2api-database-identity/v2","database":"sub2api","system_identifier":"1","in_recovery":false} {}`,
			contains: "multiple JSON values",
		},
		{
			name:     "standby",
			content:  `{"contract":"sub2api-database-identity/v2","database":"sub2api","system_identifier":"1","in_recovery":true}`,
			contains: "writable primary",
		},
		{
			name:     "missing recovery state",
			content:  `{"contract":"sub2api-database-identity/v2","database":"sub2api","system_identifier":"1"}`,
			contains: "in_recovery is required",
		},
		{
			name:     "duplicate field",
			content:  `{"contract":"sub2api-database-identity/v2","database":"sub2api","database":"other","system_identifier":"1","in_recovery":false}`,
			contains: `duplicate field "database"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "identity.json")
			require.NoError(t, os.WriteFile(path, []byte(tt.content), 0o600))
			_, err := readExpectedDatabaseIdentityFile(path)
			require.ErrorContains(t, err, tt.contains)
		})
	}
}
