package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/stretchr/testify/require"
)

func TestWriteMigrationManifestReturnsValidatedEmbeddedSet(t *testing.T) {
	var output bytes.Buffer
	require.NoError(t, writeMigrationManifest(&output))

	var manifest repository.MigrationManifest
	require.NoError(t, json.Unmarshal(output.Bytes(), &manifest))
	require.Equal(t, repository.EmbeddedMigrationManifestContract, manifest.Contract)
	require.Len(t, manifest.SetSHA256, 64)
	require.NotEmpty(t, manifest.Entries)

	var migration195 *repository.MigrationManifestEntry
	for index := range manifest.Entries {
		entry := &manifest.Entries[index]
		require.NotEmpty(t, entry.Filename)
		require.Len(t, entry.SHA256, 64)
		if entry.Filename == "195_subscription_anchored_monthly_quota.sql" {
			migration195 = entry
		}
	}
	require.NotNil(t, migration195)
}

func TestWriteMigrationManifestRejectsNilWriter(t *testing.T) {
	require.ErrorContains(t, writeMigrationManifest(nil), "writer is nil")
}
