package service

import (
	"context"
	"errors"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type chatTestUserReader struct {
	user  *User
	err   error
	calls int
}

func (s *chatTestUserReader) GetByID(context.Context, int64) (*User, error) {
	s.calls++
	return s.user, s.err
}

type chatTestGroupAccess struct {
	groups []Group
	err    error
	calls  int
}

func (s *chatTestGroupAccess) GetAvailableGroups(context.Context, int64) ([]Group, error) {
	s.calls++
	return s.groups, s.err
}

type chatTestCatalog struct {
	models           []string
	modelsErr        error
	mapping          ChannelMappingResult
	mappingErr       error
	restricted       bool
	restrictionErr   error
	rateMultiplier   float64
	availableCalls   int
	mappingCalls     int
	restrictionCalls int
}

func (s *chatTestCatalog) GetAvailableModelsStrict(context.Context, *int64, string) ([]string, error) {
	s.availableCalls++
	return s.models, s.modelsErr
}

func (s *chatTestCatalog) ResolveChannelMappingStrict(_ context.Context, _ int64, model string) (ChannelMappingResult, error) {
	s.mappingCalls++
	if s.mapping.MappedModel == "" {
		return ChannelMappingResult{MappedModel: model, BillingModelSource: BillingModelSourceRequested}, s.mappingErr
	}
	return s.mapping, s.mappingErr
}

func (s *chatTestCatalog) IsModelRestrictedStrict(context.Context, int64, string) (bool, error) {
	s.restrictionCalls++
	return s.restricted, s.restrictionErr
}

func (s *chatTestCatalog) ResolveUserGroupRateMultiplier(context.Context, int64, int64, float64) float64 {
	if s.rateMultiplier == 0 {
		return 1
	}
	return s.rateMultiplier
}

type chatTestScheduler struct {
	schedulable bool
	err         error
	calls       int
}

func (s *chatTestScheduler) HasSchedulableChatCompletionsAccount(context.Context, int64, string) (bool, error) {
	s.calls++
	return s.schedulable, s.err
}

type chatTestPricing struct{}

func (chatTestPricing) Resolve(context.Context, PricingInput) *ResolvedPricing {
	return &ResolvedPricing{
		Mode:   BillingModeToken,
		Source: PricingSourceFallback,
		BasePricing: &ModelPricing{
			InputPricePerToken:  0.000001,
			OutputPricePerToken: 0.000002,
		},
	}
}

func (chatTestPricing) GetIntervalPricing(resolved *ResolvedPricing, _ int) *ModelPricing {
	return resolved.BasePricing
}

type chatTestBilling struct {
	balance  float64
	err      error
	calls    int
	platform string
}

func (s *chatTestBilling) PeekWebChatEligibility(_ context.Context, _ int64, platform string) (float64, error) {
	s.calls++
	s.platform = platform
	return s.balance, s.err
}

type chatTestPrincipalProvider struct {
	principal *APIKey
	err       error
	calls     int
}

func (s *chatTestPrincipalProvider) Resolve(context.Context, int64, *Group) (*APIKey, error) {
	s.calls++
	return s.principal, s.err
}

func newChatServiceBehaviorTest() (*ChatService, *chatTestGroupAccess, *chatTestCatalog, *chatTestScheduler, *chatTestBilling, *chatTestPrincipalProvider) {
	groups := &chatTestGroupAccess{groups: []Group{{
		ID:               10,
		Name:             "OpenAI",
		Platform:         PlatformOpenAI,
		Status:           StatusActive,
		SubscriptionType: SubscriptionTypeStandard,
		RateMultiplier:   1,
	}}}
	catalog := &chatTestCatalog{models: []string{"gpt-5.4"}, rateMultiplier: 1}
	scheduler := &chatTestScheduler{schedulable: true}
	billing := &chatTestBilling{balance: 12.5}
	principals := &chatTestPrincipalProvider{principal: &APIKey{ID: 99, UserID: 7}}
	svc := &ChatService{
		users:      &chatTestUserReader{user: &User{ID: 7, Status: StatusActive, Balance: 999}},
		groups:     groups,
		catalog:    catalog,
		scheduler:  scheduler,
		pricing:    chatTestPricing{},
		billing:    billing,
		principals: principals,
	}
	return svc, groups, catalog, scheduler, billing, principals
}

func TestChatServiceListModelsUsesBillingBalanceWithoutCreatingPrincipal(t *testing.T) {
	svc, _, _, _, billing, principals := newChatServiceBehaviorTest()

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, 12.5, result.Balance)
	require.Len(t, result.Models, 1)
	require.Equal(t, "gpt-5.4", result.Models[0].ID)
	require.False(t, result.Models[0].Recommended)
	require.Equal(t, PlatformOpenAI, billing.platform)
	require.Equal(t, 1, billing.calls)
	require.Zero(t, principals.calls, "GET catalog must not create a hidden principal")
}

func TestChatServiceBillingPreflightRejectsGETAndPOSTBeforePrincipalCreation(t *testing.T) {
	for _, operation := range []string{"GET", "POST"} {
		t.Run(operation, func(t *testing.T) {
			svc, groups, catalog, _, billing, principals := newChatServiceBehaviorTest()
			billing.err = ErrInsufficientBalance

			var err error
			if operation == "GET" {
				_, err = svc.ListModels(context.Background(), 7)
			} else {
				_, err = svc.ResolvePrincipal(context.Background(), 7, "gpt-5.4")
			}

			require.ErrorIs(t, err, ErrChatInsufficientBalance)
			require.True(t, infraerrors.IsForbidden(err))
			require.Equal(t, 1, billing.calls)
			require.Zero(t, groups.calls)
			require.Zero(t, catalog.availableCalls)
			require.Zero(t, principals.calls)
		})
	}
}

func TestChatServiceBillingDependencyFailureReturns503(t *testing.T) {
	svc, groups, catalog, scheduler, billing, principals := newChatServiceBehaviorTest()
	billing.err = errors.New("redis and database unavailable")

	_, err := svc.ListModels(context.Background(), 7)
	require.Error(t, err)
	require.True(t, infraerrors.IsServiceUnavailable(err))
	require.Equal(t, "BILLING_SERVICE_ERROR", infraerrors.Reason(err))
	require.Zero(t, groups.calls)
	require.Zero(t, catalog.availableCalls)
	require.Zero(t, scheduler.calls)
	require.Zero(t, principals.calls)
}

func TestChatServiceUserDependencyFailureReturnsCatalog503WithoutSideEffects(t *testing.T) {
	svc, groups, catalog, scheduler, billing, principals := newChatServiceBehaviorTest()
	users, ok := svc.users.(*chatTestUserReader)
	require.True(t, ok)
	dependencyErr := infraerrors.ServiceUnavailable("DATABASE_UNAVAILABLE", "user database unavailable")
	users.err = dependencyErr

	_, err := svc.ListModels(context.Background(), 7)
	require.ErrorIs(t, err, dependencyErr)
	require.True(t, infraerrors.IsServiceUnavailable(err))
	require.Equal(t, "CHAT_CATALOG_UNAVAILABLE", infraerrors.Reason(err))
	require.Equal(t, 1, users.calls)
	require.Zero(t, billing.calls)
	require.Zero(t, groups.calls)
	require.Zero(t, catalog.availableCalls)
	require.Zero(t, scheduler.calls)
	require.Zero(t, principals.calls)
}

func TestChatServiceGroupDependencyFailureReturnsCatalog503WithoutLaterSideEffects(t *testing.T) {
	svc, groups, catalog, scheduler, billing, principals := newChatServiceBehaviorTest()
	users, ok := svc.users.(*chatTestUserReader)
	require.True(t, ok)
	dependencyErr := errors.New("group database unavailable")
	groups.err = dependencyErr

	_, err := svc.ListModels(context.Background(), 7)
	require.ErrorIs(t, err, dependencyErr)
	require.True(t, infraerrors.IsServiceUnavailable(err))
	require.Equal(t, "CHAT_CATALOG_UNAVAILABLE", infraerrors.Reason(err))
	require.Equal(t, 1, users.calls)
	require.Equal(t, 1, billing.calls)
	require.Equal(t, 1, groups.calls)
	require.Zero(t, catalog.availableCalls)
	require.Zero(t, scheduler.calls)
	require.Zero(t, principals.calls)
}

func TestChatServiceUserBusinessRejectionsKeepNotFoundSemantics(t *testing.T) {
	t.Run("repository not found", func(t *testing.T) {
		svc, groups, catalog, scheduler, billing, principals := newChatServiceBehaviorTest()
		users, ok := svc.users.(*chatTestUserReader)
		require.True(t, ok)
		users.err = ErrUserNotFound

		_, err := svc.ListModels(context.Background(), 7)
		require.ErrorIs(t, err, ErrUserNotFound)
		require.True(t, infraerrors.IsNotFound(err))
		require.Equal(t, 1, users.calls)
		require.Zero(t, billing.calls)
		require.Zero(t, groups.calls)
		require.Zero(t, catalog.availableCalls)
		require.Zero(t, scheduler.calls)
		require.Zero(t, principals.calls)
	})

	t.Run("inactive user", func(t *testing.T) {
		svc, groups, catalog, scheduler, billing, principals := newChatServiceBehaviorTest()
		users, ok := svc.users.(*chatTestUserReader)
		require.True(t, ok)
		users.user.Status = StatusDisabled

		_, err := svc.ListModels(context.Background(), 7)
		require.ErrorIs(t, err, ErrUserNotFound)
		require.True(t, infraerrors.IsNotFound(err))
		require.Equal(t, 1, users.calls)
		require.Zero(t, billing.calls)
		require.Zero(t, groups.calls)
		require.Zero(t, catalog.availableCalls)
		require.Zero(t, scheduler.calls)
		require.Zero(t, principals.calls)
	})

	t.Run("group lookup reports user not found", func(t *testing.T) {
		svc, groups, catalog, scheduler, billing, principals := newChatServiceBehaviorTest()
		groups.err = ErrUserNotFound

		_, err := svc.ListModels(context.Background(), 7)
		require.ErrorIs(t, err, ErrUserNotFound)
		require.True(t, infraerrors.IsNotFound(err))
		require.Equal(t, 1, billing.calls)
		require.Equal(t, 1, groups.calls)
		require.Zero(t, catalog.availableCalls)
		require.Zero(t, scheduler.calls)
		require.Zero(t, principals.calls)
	})
}

func TestChatServiceDependencyCancellationIsNotCatalogFailure(t *testing.T) {
	t.Run("user lookup canceled", func(t *testing.T) {
		svc, groups, catalog, scheduler, billing, principals := newChatServiceBehaviorTest()
		users, ok := svc.users.(*chatTestUserReader)
		require.True(t, ok)
		users.err = context.Canceled

		_, err := svc.ListModels(context.Background(), 7)
		require.ErrorIs(t, err, context.Canceled)
		require.False(t, infraerrors.IsServiceUnavailable(err))
		require.NotEqual(t, "CHAT_CATALOG_UNAVAILABLE", infraerrors.Reason(err))
		require.Zero(t, billing.calls)
		require.Zero(t, groups.calls)
		require.Zero(t, catalog.availableCalls)
		require.Zero(t, scheduler.calls)
		require.Zero(t, principals.calls)
	})

	t.Run("group lookup deadline", func(t *testing.T) {
		svc, groups, catalog, scheduler, billing, principals := newChatServiceBehaviorTest()
		groups.err = context.DeadlineExceeded

		_, err := svc.ListModels(context.Background(), 7)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.False(t, infraerrors.IsServiceUnavailable(err))
		require.NotEqual(t, "CHAT_CATALOG_UNAVAILABLE", infraerrors.Reason(err))
		require.Equal(t, 1, billing.calls)
		require.Equal(t, 1, groups.calls)
		require.Zero(t, catalog.availableCalls)
		require.Zero(t, scheduler.calls)
		require.Zero(t, principals.calls)
	})
}

func TestChatServiceNoAvailableGroupsReturnsEmptyCatalog(t *testing.T) {
	svc, groups, catalog, scheduler, billing, principals := newChatServiceBehaviorTest()
	groups.groups = nil

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Models)
	require.Equal(t, 12.5, result.Balance)
	require.Equal(t, 1, billing.calls)
	require.Equal(t, 1, groups.calls)
	require.Zero(t, catalog.availableCalls)
	require.Zero(t, scheduler.calls)
	require.Zero(t, principals.calls)
}

func TestChatServiceCatalogAndSchedulerFailuresReturn503(t *testing.T) {
	t.Run("catalog", func(t *testing.T) {
		svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
		catalog.modelsErr = errors.New("account snapshot unavailable")

		_, err := svc.ListModels(context.Background(), 7)
		require.True(t, infraerrors.IsServiceUnavailable(err))
		require.ErrorIs(t, err, catalog.modelsErr)
	})

	t.Run("scheduler", func(t *testing.T) {
		svc, _, _, scheduler, _, _ := newChatServiceBehaviorTest()
		scheduler.err = errors.New("account repository unavailable")

		_, err := svc.ListModels(context.Background(), 7)
		require.True(t, infraerrors.IsServiceUnavailable(err))
		require.ErrorIs(t, err, scheduler.err)
	})
}

func TestChatServiceUnschedulableModelCannotResolvePrincipal(t *testing.T) {
	svc, _, _, scheduler, _, principals := newChatServiceBehaviorTest()
	scheduler.schedulable = false

	models, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	require.Empty(t, models.Models)

	_, err = svc.ResolvePrincipal(context.Background(), 7, "gpt-5.4")
	require.ErrorIs(t, err, ErrChatModelNotAvailable)
	require.Zero(t, principals.calls)
}

func TestChatServiceDedicatedCapabilityModelsAreNotListedOrResolved(t *testing.T) {
	models := []string{
		"gpt-4o-search-preview",
		"gpt-image-2",
		"gpt-4o-audio-preview",
		"gpt-4o-realtime-preview",
		"gpt-4o-transcribe",
		"gpt-4o-mini-tts",
		"gpt-embedding-private",
		"gpt-moderation-private",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			svc, _, catalog, scheduler, _, principals := newChatServiceBehaviorTest()
			catalog.models = []string{model}

			result, err := svc.ListModels(context.Background(), 7)
			require.NoError(t, err)
			require.Empty(t, result.Models)

			_, err = svc.ResolvePrincipal(context.Background(), 7, model)
			require.ErrorIs(t, err, ErrChatModelNotAvailable)
			require.Zero(t, scheduler.calls)
			require.Zero(t, principals.calls)
		})
	}
}

func TestChatServiceGPT52FamilyIsNotListedOrResolved(t *testing.T) {
	models := []string{
		"gpt-5.2",
		"gpt-5.2-2025-12-11",
		"gpt-5.2-chat-latest",
		"gpt-5.2-pro",
		"gpt-5.2-codex",
		"gpt-5.2_custom",
		"openai/gpt-5.2-codex-max",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			svc, _, catalog, scheduler, _, principals := newChatServiceBehaviorTest()
			catalog.models = []string{model}

			result, err := svc.ListModels(context.Background(), 7)
			require.NoError(t, err)
			require.Empty(t, result.Models)

			_, err = svc.ResolvePrincipal(context.Background(), 7, model)
			require.ErrorIs(t, err, ErrChatModelNotAvailable)
			require.Zero(t, scheduler.calls)
			require.Zero(t, principals.calls)
		})
	}
}

func TestRetiredGPT52FamilyBoundary(t *testing.T) {
	for _, model := range []string{"gpt-5.20", "gpt-5.2x", "gpt-5.2preview"} {
		require.False(t, isRetiredGPT52ChatModel(model), model)
	}
	for _, model := range []string{"gpt-5.2", "gpt-5.2-codex", "gpt-5.2_custom", "gpt-5.2:preview"} {
		require.True(t, isRetiredGPT52ChatModel(model), model)
	}
}

func TestChatServiceRecommendsGPT55WhenAvailable(t *testing.T) {
	svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-5.6-sol", "gpt-5.4", "gpt-5.5"}

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, result.Models, 3)
	require.Equal(t, "gpt-5.5", result.Models[0].ID)
	require.True(t, result.Models[0].Recommended)
	require.False(t, result.Models[1].Recommended)
	require.False(t, result.Models[2].Recommended)
}

func TestChatServiceDoesNotRecommendAnotherModelWhenGPT55IsUnavailable(t *testing.T) {
	svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-5.6-sol", "gpt-5.4"}

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, result.Models, 2)
	for _, model := range result.Models {
		require.False(t, model.Recommended)
	}
}

func TestChatServiceRejectsAliasMappedByChannelToRetiredGPT52(t *testing.T) {
	for _, target := range []string{"gpt-5.2", "gpt-5.2-codex", "openai/gpt-5.2-pro"} {
		t.Run(target, func(t *testing.T) {
			svc, _, catalog, scheduler, _, principals := newChatServiceBehaviorTest()
			catalog.models = []string{"gpt-safe-alias"}
			catalog.mapping = ChannelMappingResult{
				MappedModel:        target,
				Mapped:             true,
				BillingModelSource: BillingModelSourceChannelMapped,
			}

			result, err := svc.ListModels(context.Background(), 7)
			require.NoError(t, err)
			require.Empty(t, result.Models)

			_, err = svc.ResolvePrincipal(context.Background(), 7, "gpt-safe-alias")
			require.ErrorIs(t, err, ErrChatModelNotAvailable)
			require.Zero(t, scheduler.calls)
			require.Zero(t, principals.calls)
		})
	}
}

func TestChatServiceRejectsAliasMappedByChannelToDedicatedCapabilityModel(t *testing.T) {
	for _, target := range []string{
		"gpt-4o-search-preview",
		"gpt-image-2",
		"gpt-4o-audio-preview",
	} {
		t.Run(target, func(t *testing.T) {
			svc, _, catalog, scheduler, _, principals := newChatServiceBehaviorTest()
			catalog.models = []string{"gpt-safe-alias"}
			catalog.mapping = ChannelMappingResult{
				MappedModel:        target,
				Mapped:             true,
				BillingModelSource: BillingModelSourceChannelMapped,
			}

			result, err := svc.ListModels(context.Background(), 7)
			require.NoError(t, err)
			require.Empty(t, result.Models)

			_, err = svc.ResolvePrincipal(context.Background(), 7, "gpt-safe-alias")
			require.ErrorIs(t, err, ErrChatModelNotAvailable)
			require.Zero(t, scheduler.calls)
			require.Zero(t, principals.calls)
		})
	}
}

func TestChatServiceModelCandidatesRespectDefaultAndCustomCatalogs(t *testing.T) {
	t.Run("empty account mappings expose default GPT models", func(t *testing.T) {
		svc, groups, _, scheduler, _, _ := newChatServiceBehaviorTest()
		repo := &chatSchedAccountRepo{accounts: []Account{chatSchedulableAccount(1, PlatformOpenAI, nil)}}
		svc.catalog = &GatewayService{
			accountRepo:    repo,
			channelService: chatChannelServiceFromCache(BillingModelSourceRequested, false, nil, nil),
		}
		scheduler.schedulable = true
		groups.groups[0].ID = 1

		result, err := svc.ListModels(context.Background(), 7)
		require.NoError(t, err)
		modelIDs := make([]string, 0, len(result.Models))
		for _, model := range result.Models {
			modelIDs = append(modelIDs, model.ID)
		}
		require.Contains(t, modelIDs, "gpt-5.4")
		require.NotContains(t, modelIDs, "gpt-image-1")
		for _, model := range result.Models {
			if model.ID == "gpt-5.4" {
				require.Equal(t, "GPT-5.4", model.DisplayName)
			}
		}
	})

	t.Run("custom group uses only its configured model list", func(t *testing.T) {
		svc, groups, _, scheduler, _, _ := newChatServiceBehaviorTest()
		repo := &chatSchedAccountRepo{accounts: []Account{chatSchedulableAccount(1, PlatformOpenAI, nil)}}
		svc.catalog = &GatewayService{
			accountRepo:    repo,
			channelService: chatChannelServiceFromCache(BillingModelSourceRequested, false, nil, nil),
		}
		scheduler.schedulable = true
		groups.groups[0].ID = 1
		groups.groups[0].ModelsListConfig = GroupModelsListConfig{
			Enabled: true,
			Models:  []string{"gpt-private-alias"},
		}

		result, err := svc.ListModels(context.Background(), 7)
		require.NoError(t, err)
		require.Len(t, result.Models, 1)
		require.Equal(t, "gpt-private-alias", result.Models[0].ID)
		require.Equal(t, "gpt-private-alias", result.Models[0].DisplayName)
	})
}
