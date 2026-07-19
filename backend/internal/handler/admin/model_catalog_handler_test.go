package admin

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestFilterModelCatalogAdminItems(t *testing.T) {
	items := []service.ModelCatalogAdminItem{
		{ModelCatalogModel: service.ModelCatalogModel{Model: "gpt-5", Platform: "openai", Provider: "OpenAI", DisplayNameZH: "通用模型", Status: service.ModelCatalogStatusPublished}},
		{ModelCatalogModel: service.ModelCatalogModel{Model: "claude", Platform: "anthropic", Category: "reasoning", DisplayNameEN: "Claude Reasoner", Status: service.ModelCatalogStatusDraft}},
	}

	filtered := filterModelCatalogAdminItems(items, service.ModelCatalogStatusPublished, "OPENAI")
	require.Len(t, filtered, 1)
	require.Equal(t, "gpt-5", filtered[0].Model)
	filtered = filterModelCatalogAdminItems(items, "", "reasoner")
	require.Len(t, filtered, 1)
	require.Equal(t, "claude", filtered[0].Model)
}
