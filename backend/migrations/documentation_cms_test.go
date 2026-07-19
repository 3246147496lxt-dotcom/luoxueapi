package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocumentationCMSMigrationIsEmbedded(t *testing.T) {
	contents, err := FS.ReadFile("182_documentation_cms.sql")
	require.NoError(t, err)
	sql := string(contents)
	require.Contains(t, sql, "documentation_documents")
	require.Contains(t, sql, "documentation_revisions")
	require.Contains(t, sql, "documentation_assets")
}
