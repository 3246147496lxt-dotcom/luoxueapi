package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestParseCustomMenuItemsDefaultsLegacyAuthModeToNone(t *testing.T) {
	items := ParseCustomMenuItems(`[{"id":"legacy","label":"Legacy","url":"https://example.com","visibility":"user","sort_order":1}]`)
	require.Len(t, items, 1)
	require.Equal(t, service.CustomMenuAuthModeNone, items[0].AuthMode)
}
