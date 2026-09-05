package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminServiceDeepSeekModelListCandidatesIncludeV4Catalog(t *testing.T) {
	svc := &adminServiceImpl{}

	models, err := svc.GetGroupModelsListCandidates(context.Background(), 0, PlatformDeepseek)
	require.NoError(t, err)
	require.Equal(t, []string{
		"deepseek-v4-pro",
		"deepseek-v4-flash",
		"deepseek-v4-flash-vision-exp",
	}, models)
}
