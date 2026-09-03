package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type skillLocalizationRepositoryStub struct {
	SkillMarketRepository
	listLocale string
	getLocale  string
}

func (r *skillLocalizationRepositoryStub) ListPublished(
	_ context.Context,
	filter SkillListFilter,
) ([]Skill, int64, error) {
	r.listLocale = filter.Locale
	return []Skill{{
		ID: 1, Slug: "find-skills", DisplayName: "查找 Skill",
		Summary: "发现并安装合适的 Skill。", Description: "根据任务寻找可安装的 Skill。",
	}}, 1, nil
}

func (r *skillLocalizationRepositoryStub) ListPublishedCategories(context.Context) ([]string, error) {
	return []string{}, nil
}

func (r *skillLocalizationRepositoryStub) GetPublishedBySlugLocalized(
	_ context.Context,
	_ string,
	locale string,
) (*Skill, error) {
	r.getLocale = locale
	return &Skill{
		ID: 1, Slug: "find-skills", DisplayName: "查找 Skill",
		Summary: "发现并安装合适的 Skill。", Description: "根据任务寻找可安装的 Skill。",
	}, nil
}

func (r *skillLocalizationRepositoryStub) ListVersions(context.Context, int64, bool) ([]SkillVersion, error) {
	return []SkillVersion{}, nil
}

func TestSkillMarketPublicListNormalizesCatalogLocale(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "Chinese browser locale", raw: "zh-CN,zh;q=0.9", want: "zh-CN"},
		{name: "English browser locale", raw: "en-US,en;q=0.9", want: "en"},
		{name: "Chinese default", raw: "", want: "zh-CN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &skillLocalizationRepositoryStub{}
			market := NewSkillMarketService(repo, nil)
			result, err := market.ListPublic(context.Background(), SkillListFilter{
				Locale: tt.raw, Page: 1, PageSize: 12,
			})
			require.NoError(t, err)
			require.Equal(t, tt.want, repo.listLocale)
			require.Equal(t, "查找 Skill", result.Items[0].DisplayName)
		})
	}
}

func TestSkillMarketPublicDetailUsesLocalizedRepositoryProjection(t *testing.T) {
	repo := &skillLocalizationRepositoryStub{}
	market := NewSkillMarketService(repo, nil)

	result, err := market.GetPublic(context.Background(), "FIND-SKILLS", "zh-Hans")
	require.NoError(t, err)
	require.Equal(t, "zh-CN", repo.getLocale)
	require.Equal(t, "查找 Skill", result.DisplayName)
	require.Equal(t, "发现并安装合适的 Skill。", result.Summary)
	require.Equal(t, "根据任务寻找可安装的 Skill。", result.Description)
}
