package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
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
	schedulable          bool
	err                  error
	calls                int
	reasoningCalls       int
	reasoningOptions     []WebChatReasoningOptions
	reasoningSchedulable *bool
	reasoningByGroup     map[int64]bool
	reasoningByOptions   map[WebChatReasoningOptions]bool
	transcriptionByGroup map[int64]bool
	transcriptionCalls   []int64
}

func (s *chatTestScheduler) HasSchedulableTranscriptionAccount(_ context.Context, groupID int64, _ string) (bool, error) {
	s.transcriptionCalls = append(s.transcriptionCalls, groupID)
	return s.transcriptionByGroup[groupID], s.err
}

func (s *chatTestScheduler) HasSchedulableChatCompletionsAccount(context.Context, int64, string) (bool, error) {
	s.calls++
	return s.schedulable, s.err
}

func (s *chatTestScheduler) HasSchedulableWebChatReasoningAccount(_ context.Context, groupID int64, _ string, options WebChatReasoningOptions) (bool, error) {
	s.reasoningCalls++
	s.reasoningOptions = append(s.reasoningOptions, options)
	if s.reasoningByOptions != nil {
		return s.reasoningByOptions[options], s.err
	}
	if s.reasoningByGroup != nil {
		return s.reasoningByGroup[groupID], s.err
	}
	if s.reasoningSchedulable != nil {
		return *s.reasoningSchedulable, s.err
	}
	return s.schedulable, s.err
}

type chatCapabilitiesSchedulerStub struct {
	config             config.TranscriptionConfig
	configErr          error
	configCalls        int
	chatScheduleCalls  int
	audioScheduleCalls int
}

func (s *chatCapabilitiesSchedulerStub) EffectiveTranscriptionConfig(context.Context) (config.TranscriptionConfig, error) {
	s.configCalls++
	return s.config, s.configErr
}

func (s *chatCapabilitiesSchedulerStub) HasSchedulableChatCompletionsAccount(context.Context, int64, string) (bool, error) {
	s.chatScheduleCalls++
	return false, nil
}

func (s *chatCapabilitiesSchedulerStub) HasSchedulableWebChatReasoningAccount(context.Context, int64, string, WebChatReasoningOptions) (bool, error) {
	s.chatScheduleCalls++
	return false, nil
}

func (s *chatCapabilitiesSchedulerStub) HasSchedulableTranscriptionAccount(context.Context, int64, string) (bool, error) {
	s.audioScheduleCalls++
	return false, nil
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
	balance           float64
	err               error
	calls             int
	platform          string
	subscriptionErr   error
	subscriptionCalls int
}

func (s *chatTestBilling) PeekWebChatEligibility(_ context.Context, _ int64, platform string) (float64, error) {
	s.calls++
	s.platform = platform
	return s.balance, s.err
}

func (s *chatTestBilling) PeekWebChatSubscriptionEligibility(
	_ context.Context,
	_ int64,
	_ *Group,
	_ *UserSubscription,
) error {
	s.subscriptionCalls++
	return s.subscriptionErr
}

type chatTestSubscriptions struct {
	sub              *UserSubscription
	getErr           error
	validateErr      error
	needsMaintenance bool
	maintenanceErr   error
	getCalls         int
	validateCalls    int
	maintenanceCalls int
}

func (s *chatTestSubscriptions) GetActiveSubscription(context.Context, int64, int64) (*UserSubscription, error) {
	s.getCalls++
	return s.sub, s.getErr
}

func (s *chatTestSubscriptions) ValidateAndCheckLimits(*UserSubscription, *Group) (bool, error) {
	s.validateCalls++
	needsMaintenance := s.needsMaintenance
	s.needsMaintenance = false
	return needsMaintenance, s.validateErr
}

func (s *chatTestSubscriptions) EnsureWindowMaintenance(context.Context, *UserSubscription) (*UserSubscription, error) {
	s.maintenanceCalls++
	return s.sub, s.maintenanceErr
}

type chatTestPrincipalProvider struct {
	principal *APIKey
	err       error
	calls     int
	groupIDs  []int64
	subIDs    []int64
}

func (s *chatTestPrincipalProvider) Resolve(
	_ context.Context,
	_ int64,
	group *Group,
	subscription *UserSubscription,
) (*APIKey, error) {
	s.calls++
	if group != nil {
		s.groupIDs = append(s.groupIDs, group.ID)
	}
	if subscription != nil {
		s.subIDs = append(s.subIDs, subscription.ID)
	}
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
		subs:       &chatTestSubscriptions{},
		principals: principals,
	}
	return svc, groups, catalog, scheduler, billing, principals
}

func configureChatSubscriptionAndWalletGroups(groups *chatTestGroupAccess) {
	groups.groups = []Group{
		{
			ID:               10,
			Name:             "Wallet",
			Platform:         PlatformOpenAI,
			Status:           StatusActive,
			SubscriptionType: SubscriptionTypeStandard,
			RateMultiplier:   1,
		},
		{
			ID:               20,
			Name:             "Pro",
			Platform:         PlatformOpenAI,
			Status:           StatusActive,
			SubscriptionType: SubscriptionTypeSubscription,
			RateMultiplier:   1,
		},
	}
}

func activeChatTestSubscription() *UserSubscription {
	now := time.Now()
	return &UserSubscription{
		ID:        700,
		UserID:    7,
		GroupID:   20,
		Status:    SubscriptionStatusActive,
		StartsAt:  now.Add(-time.Hour),
		ExpiresAt: now.Add(time.Hour),
	}
}

func TestChatServiceCapabilitiesDoesNotRunUserOrRuntimeAdmission(t *testing.T) {
	svc, groups, catalog, _, billing, principals := newChatServiceBehaviorTest()
	users, ok := svc.users.(*chatTestUserReader)
	require.True(t, ok)
	scheduler := &chatCapabilitiesSchedulerStub{
		config: config.TranscriptionConfig{
			Enabled:            true,
			MaxUploadBytes:     8 << 20,
			MaxDurationSeconds: 90,
			AcceptedMIMETypes:  []string{"audio/webm", "audio/mp4"},
			GroupIDs:           []int64{10, 20},
			Model:              "gpt-4o-mini-transcribe",
		},
	}
	svc.scheduler = scheduler

	result, err := svc.Capabilities(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Transcription.Enabled)
	require.Equal(t, int64(8<<20), result.Transcription.MaxUploadBytes)
	require.Equal(t, 90, result.Transcription.MaxDurationSeconds)
	require.Equal(t, []string{"audio/webm", "audio/mp4"}, result.Transcription.AcceptedMIMETypes)
	require.Equal(t, 1, scheduler.configCalls)
	require.Zero(t, scheduler.chatScheduleCalls)
	require.Zero(t, scheduler.audioScheduleCalls)
	require.Zero(t, users.calls)
	require.Zero(t, groups.calls)
	require.Zero(t, catalog.availableCalls)
	require.Zero(t, catalog.mappingCalls)
	require.Zero(t, catalog.restrictionCalls)
	require.Zero(t, billing.calls)
	require.Zero(t, principals.calls)
}

func TestChatServiceCapabilitiesFailsClosedWhenProductConfigIsUnavailable(t *testing.T) {
	svc, groups, catalog, _, billing, principals := newChatServiceBehaviorTest()
	users, ok := svc.users.(*chatTestUserReader)
	require.True(t, ok)
	svc.transcribe = config.TranscriptionConfig{
		Enabled:            true,
		MaxUploadBytes:     4 << 20,
		MaxDurationSeconds: 60,
		AcceptedMIMETypes:  []string{"audio/webm"},
	}
	scheduler := &chatCapabilitiesSchedulerStub{configErr: errors.New("settings unavailable")}
	svc.scheduler = scheduler

	result, err := svc.Capabilities(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Transcription.Enabled)
	require.Equal(t, int64(4<<20), result.Transcription.MaxUploadBytes)
	require.Equal(t, 60, result.Transcription.MaxDurationSeconds)
	require.Equal(t, []string{"audio/webm"}, result.Transcription.AcceptedMIMETypes)
	require.Equal(t, 1, scheduler.configCalls)
	require.Zero(t, scheduler.chatScheduleCalls)
	require.Zero(t, scheduler.audioScheduleCalls)
	require.Zero(t, users.calls)
	require.Zero(t, groups.calls)
	require.Zero(t, catalog.availableCalls)
	require.Zero(t, billing.calls)
	require.Zero(t, principals.calls)
}

func TestChatServiceResolveTranscriptionPrincipalUsesWhitelistIntersectionOrder(t *testing.T) {
	svc, groups, _, scheduler, billing, principals := newChatServiceBehaviorTest()
	groups.groups = []Group{
		{ID: 10, Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
		{ID: 30, Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
		{ID: 40, Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription},
	}
	svc.transcribe = config.TranscriptionConfig{
		Enabled:  true,
		Model:    "gpt-4o-mini-transcribe",
		GroupIDs: []int64{99, 10, 40, 30},
	}
	scheduler.transcriptionByGroup = map[int64]bool{10: false, 30: true, 40: true}
	selectedGroupID := int64(30)
	principals.principal = &APIKey{ID: 99, UserID: 7, GroupID: &selectedGroupID}

	principal, err := svc.ResolveTranscriptionPrincipal(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, int64(99), principal.ID)
	require.Equal(t, []int64{10, 30}, scheduler.transcriptionCalls)
	require.Equal(t, []int64{30}, principals.groupIDs)
	require.Equal(t, 1, billing.calls)
}

func TestChatServiceListModelsUsesBillingBalanceWithoutCreatingPrincipal(t *testing.T) {
	svc, _, catalog, _, billing, principals := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-5.5"}

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, 12.5, result.Balance)
	require.Len(t, result.Models, 1)
	require.Equal(t, "gpt-5.5", result.Models[0].ID)
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
			require.Equal(t, 1, groups.calls)
			require.Equal(t, 1, catalog.availableCalls)
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
	require.Equal(t, 1, groups.calls)
	require.Equal(t, 1, catalog.availableCalls)
	require.Equal(t, 1, scheduler.calls)
	require.Zero(t, principals.calls)
}

func TestChatServiceCompletionEntitlementPrefersSubscription(t *testing.T) {
	svc, groups, _, _, billing, principals := newChatServiceBehaviorTest()
	configureChatSubscriptionAndWalletGroups(groups)
	subscriptions, ok := svc.subs.(*chatTestSubscriptions)
	require.True(t, ok)
	subscriptions.sub = activeChatTestSubscription()

	principal, err := svc.ResolvePrincipal(context.Background(), 7, "gpt-5.4")
	require.NoError(t, err)
	require.NotNil(t, principal)
	require.Same(t, principals.principal, principal.APIKey)
	require.Same(t, subscriptions.sub, principal.Subscription)
	require.Equal(t, []int64{20}, principals.groupIDs)
	require.Equal(t, []int64{700}, principals.subIDs)
	require.Equal(t, 1, billing.subscriptionCalls)
}

func TestChatServiceCompletionEntitlementAllowsSubscriptionWhenWalletUnavailable(t *testing.T) {
	for _, walletErr := range []error{
		ErrInsufficientBalance,
		errors.New("wallet cache and database unavailable"),
	} {
		t.Run(walletErr.Error(), func(t *testing.T) {
			svc, groups, _, _, billing, principals := newChatServiceBehaviorTest()
			configureChatSubscriptionAndWalletGroups(groups)
			subscriptions, ok := svc.subs.(*chatTestSubscriptions)
			require.True(t, ok)
			subscriptions.sub = activeChatTestSubscription()
			billing.err = walletErr

			principal, err := svc.ResolvePrincipal(context.Background(), 7, "gpt-5.4")
			require.NoError(t, err)
			require.NotNil(t, principal)
			require.Same(t, subscriptions.sub, principal.Subscription)
			require.Equal(t, []int64{20}, principals.groupIDs)
			require.Equal(t, 1, billing.subscriptionCalls)
		})
	}
}

func TestChatServiceCompletionEntitlementFallsBackToWalletAfterSubscriptionExhaustion(t *testing.T) {
	svc, groups, _, _, billing, principals := newChatServiceBehaviorTest()
	configureChatSubscriptionAndWalletGroups(groups)
	subscriptions, ok := svc.subs.(*chatTestSubscriptions)
	require.True(t, ok)
	subscriptions.sub = activeChatTestSubscription()
	subscriptions.validateErr = ErrWeeklyLimitExceeded

	principal, err := svc.ResolvePrincipal(context.Background(), 7, "gpt-5.4")
	require.NoError(t, err)
	require.NotNil(t, principal)
	require.Nil(t, principal.Subscription)
	require.Equal(t, []int64{10}, principals.groupIDs)
	require.Empty(t, principals.subIDs)
	require.Zero(t, billing.subscriptionCalls)
}

func TestChatServiceCompletionEntitlementReportsInsufficientOnlyWhenEverySourceIsExhausted(t *testing.T) {
	svc, groups, _, _, billing, principals := newChatServiceBehaviorTest()
	configureChatSubscriptionAndWalletGroups(groups)
	subscriptions, ok := svc.subs.(*chatTestSubscriptions)
	require.True(t, ok)
	subscriptions.sub = activeChatTestSubscription()
	subscriptions.validateErr = ErrMonthlyLimitExceeded
	billing.err = ErrInsufficientBalance

	_, err := svc.ResolvePrincipal(context.Background(), 7, "gpt-5.4")
	require.ErrorIs(t, err, ErrChatInsufficientBalance)
	require.True(t, infraerrors.IsForbidden(err))
	require.Zero(t, principals.calls)
}

func TestChatServiceCompletionEntitlementDoesNotMaskDependencyFailureAsInsufficient(t *testing.T) {
	t.Run("subscription dependency failure blocks wallet fallback", func(t *testing.T) {
		svc, groups, _, _, _, principals := newChatServiceBehaviorTest()
		configureChatSubscriptionAndWalletGroups(groups)
		subscriptions, ok := svc.subs.(*chatTestSubscriptions)
		require.True(t, ok)
		subscriptions.getErr = errors.New("subscription database unavailable")

		_, err := svc.ResolvePrincipal(context.Background(), 7, "gpt-5.4")
		require.Error(t, err)
		require.True(t, infraerrors.IsServiceUnavailable(err))
		require.Equal(t, "BILLING_SERVICE_ERROR", infraerrors.Reason(err))
		require.NotErrorIs(t, err, ErrChatInsufficientBalance)
		require.Zero(t, principals.calls)
	})

	t.Run("wallet dependency failure wins when subscription is exhausted", func(t *testing.T) {
		svc, groups, _, _, billing, principals := newChatServiceBehaviorTest()
		configureChatSubscriptionAndWalletGroups(groups)
		subscriptions, ok := svc.subs.(*chatTestSubscriptions)
		require.True(t, ok)
		subscriptions.sub = activeChatTestSubscription()
		subscriptions.validateErr = ErrWeeklyLimitExceeded
		billing.err = errors.New("wallet database unavailable")

		_, err := svc.ResolvePrincipal(context.Background(), 7, "gpt-5.4")
		require.Error(t, err)
		require.True(t, infraerrors.IsServiceUnavailable(err))
		require.Equal(t, "BILLING_SERVICE_ERROR", infraerrors.Reason(err))
		require.NotErrorIs(t, err, ErrChatInsufficientBalance)
		require.Zero(t, principals.calls)
	})
}

func TestChatServiceCompletionEntitlementMaintainsSubscriptionWindowsBeforeAdmission(t *testing.T) {
	svc, groups, _, _, _, _ := newChatServiceBehaviorTest()
	configureChatSubscriptionAndWalletGroups(groups)
	subscriptions, ok := svc.subs.(*chatTestSubscriptions)
	require.True(t, ok)
	subscriptions.sub = activeChatTestSubscription()
	subscriptions.needsMaintenance = true

	principal, err := svc.ResolvePrincipal(context.Background(), 7, "gpt-5.4")
	require.NoError(t, err)
	require.Same(t, subscriptions.sub, principal.Subscription)
	require.Equal(t, 2, subscriptions.validateCalls)
	require.Equal(t, 1, subscriptions.maintenanceCalls)
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

func TestChatServiceRecommendsExactGPT56SolWhenAvailable(t *testing.T) {
	svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-5.6-sol", "gpt-5.4", "gpt-5.5"}

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, result.Models, 2)
	require.Equal(t, "gpt-5.6-sol", result.Models[0].ID)
	require.True(t, result.Models[0].Recommended)
	require.False(t, result.Models[1].Recommended)
}

func TestChatServiceDoesNotRecommendFallbackWhenExactGPT56SolIsUnavailable(t *testing.T) {
	svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-5.6", "gpt-5.5", "gpt-5.4"}

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, result.Models, 1)
	for _, model := range result.Models {
		require.False(t, model.Recommended)
	}
}

func TestChatServiceCatalogHidesProductExcludedModelsWithoutRevokingAuthorization(t *testing.T) {
	svc, _, catalog, _, _, principals := newChatServiceBehaviorTest()
	catalog.models = []string{
		"gpt-5.6",
		"gpt-5.6-sol",
		"gpt-5.5",
		"gpt-5.6-luna",
		"gpt-5.6-terra",
		"gpt-5.3-codex-spark",
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.4-2026-03-05",
		"gpt-5.40",
		"gpt-5.4x",
	}

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	modelIDs := make([]string, 0, len(result.Models))
	for _, model := range result.Models {
		modelIDs = append(modelIDs, model.ID)
	}
	require.NotContains(t, modelIDs, "gpt-5.6")
	require.NotContains(t, modelIDs, "gpt-5.3-codex-spark")
	require.NotContains(t, modelIDs, "gpt-5.4")
	require.NotContains(t, modelIDs, "gpt-5.4-mini")
	require.NotContains(t, modelIDs, "gpt-5.4-2026-03-05")
	require.Contains(t, modelIDs, "gpt-5.6-sol")
	require.Contains(t, modelIDs, "gpt-5.5")
	require.Contains(t, modelIDs, "gpt-5.6-luna")
	require.Contains(t, modelIDs, "gpt-5.6-terra")
	require.Contains(t, modelIDs, "gpt-5.40", "numeric lookalikes must not be treated as GPT-5.4")
	require.Contains(t, modelIDs, "gpt-5.4x", "alphabetic lookalikes must not be treated as GPT-5.4")

	for _, model := range result.Models {
		require.Equal(t, model.ID == "gpt-5.6-sol", model.Recommended, model.ID)
	}

	for _, historicalModel := range []string{
		"gpt-5.6",
		"gpt-5.3-codex-spark",
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.4-2026-03-05",
	} {
		principal, resolveErr := svc.ResolvePrincipal(context.Background(), 7, historicalModel)
		require.NoError(t, resolveErr)
		require.NotNil(t, principal)
	}
	require.Equal(t, 5, principals.calls)
}

func TestHiddenWebChatCatalogModelIDNormalizesGPT54FamilyWithBoundary(t *testing.T) {
	for _, model := range []string{
		"GPT-5.4",
		"openai/GPT-5.4-Mini",
		"gpt5.4mini",
		"gpt-5.4:preview",
		"gpt-5.4_2026-03-05",
	} {
		require.True(t, isHiddenWebChatCatalogModelID(model), model)
	}
	for _, model := range []string{
		"gpt-5.40",
		"gpt-5.4x",
		"gpt-5.41-mini",
		"gpt-15.4",
		"gpt-5.5",
		"gpt-5.6-luna",
	} {
		require.False(t, isHiddenWebChatCatalogModelID(model), model)
	}
}

func TestChatServiceReasoningSliderIsAvailableForEveryCatalogModel(t *testing.T) {
	for _, tt := range []struct {
		name      string
		requested string
		mapping   ChannelMappingResult
	}{
		{
			name:      "exact Sol",
			requested: "gpt-5.6-sol",
		},
		{
			name:      "bare GPT-5.6 normalizes to Sol",
			requested: "gpt-5.6",
		},
		{
			name:      "channel alias maps to Sol",
			requested: "gpt-safe-alias",
			mapping: ChannelMappingResult{
				MappedModel:        "openai/gpt-5.6-sol",
				Mapped:             true,
				BillingModelSource: BillingModelSourceChannelMapped,
			},
		},
		{
			name:      "non-Sol model",
			requested: "gpt-5.5",
		},
		{
			name:      "upstream mapping cannot prove Sol",
			requested: "gpt-safe-alias",
			mapping: ChannelMappingResult{
				MappedModel:        "gpt-5.6-sol",
				Mapped:             true,
				BillingModelSource: BillingModelSourceUpstream,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
			catalog.models = []string{tt.requested}
			catalog.mapping = tt.mapping

			_, _, choices, err := svc.authorizedModelChoices(context.Background(), 7)
			require.NoError(t, err)
			require.Len(t, choices, 1)
			require.True(t, choices[0].model.SupportsReasoningSlider)
		})
	}
}

func TestChatServiceReasoningEffortDefaultsToLowAndPreservesSupportedLevels(t *testing.T) {
	for _, effort := range []string{"", "low", "medium", "high", "xhigh"} {
		t.Run("effort="+effort, func(t *testing.T) {
			svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
			catalog.models = []string{"gpt-5.6-sol"}

			got, err := svc.NormalizeReasoningEffort(context.Background(), 7, "gpt-5.6-sol", effort)
			require.NoError(t, err)
			if effort == "" {
				require.Equal(t, "low", got)
			} else {
				require.Equal(t, effort, got)
			}
		})
	}
}

func TestChatServiceNormalizesWebChatReasoningModeWithoutChangingEffort(t *testing.T) {
	for _, tt := range []struct {
		name       string
		mode       string
		effort     string
		wantMode   string
		wantEffort string
	}{
		{name: "omitted mode defaults standard", effort: "high", wantMode: WebChatReasoningModeStandard, wantEffort: "high"},
		{name: "explicit standard", mode: " STANDARD ", effort: "medium", wantMode: WebChatReasoningModeStandard, wantEffort: "medium"},
		{name: "Pro preserves effort", mode: " PRO ", effort: "xhigh", wantMode: WebChatReasoningModePro, wantEffort: "xhigh"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
			catalog.models = []string{"gpt-5.6-sol"}

			got, err := svc.NormalizeWebChatReasoning(context.Background(), 7, "gpt-5.6-sol", WebChatReasoningOptions{
				Mode: tt.mode, Effort: tt.effort,
			})
			require.NoError(t, err)
			require.Equal(t, WebChatReasoningOptions{Mode: tt.wantMode, Effort: tt.wantEffort}, got)
		})
	}
}

func TestChatServiceRejectsInvalidWebChatReasoningMode(t *testing.T) {
	svc, _, _, scheduler, _, _ := newChatServiceBehaviorTest()

	got, err := svc.NormalizeWebChatReasoning(context.Background(), 7, "gpt-5.6-sol", WebChatReasoningOptions{
		Mode: "turbo", Effort: "medium",
	})
	require.Empty(t, got)
	require.ErrorIs(t, err, ErrChatReasoningModeInvalid)
	require.Equal(t, "CHAT_REASONING_MODE_INVALID", infraerrors.Reason(err))
	require.Zero(t, scheduler.calls)
}

func TestChatServiceProReasoningUnavailableDoesNotDowngrade(t *testing.T) {
	t.Run("model does not support Pro", func(t *testing.T) {
		svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
		catalog.models = []string{"gpt-5.5"}

		got, err := svc.NormalizeWebChatReasoning(context.Background(), 7, "gpt-5.5", WebChatReasoningOptions{
			Mode: WebChatReasoningModePro, Effort: "medium",
		})
		require.Empty(t, got)
		require.ErrorIs(t, err, ErrChatProReasoningUnavailable)
		require.Equal(t, "PRO_REASONING_UNAVAILABLE", infraerrors.Reason(err))
	})

	t.Run("Responses account is unavailable", func(t *testing.T) {
		svc, _, catalog, scheduler, _, _ := newChatServiceBehaviorTest()
		catalog.models = []string{"gpt-5.6-sol"}
		unavailable := false
		scheduler.reasoningSchedulable = &unavailable

		got, err := svc.NormalizeWebChatReasoning(context.Background(), 7, "gpt-5.6-sol", WebChatReasoningOptions{
			Mode: WebChatReasoningModePro, Effort: "high",
		})
		require.Empty(t, got)
		require.ErrorIs(t, err, ErrChatProReasoningUnavailable)
		require.Equal(t, "PRO_REASONING_UNAVAILABLE", infraerrors.Reason(err))
	})
}

func TestChatServiceReasoningPrincipalUsesCapabilityQualifiedGroup(t *testing.T) {
	svc, groups, catalog, scheduler, _, principals := newChatServiceBehaviorTest()
	groups.groups = []Group{
		{ID: 10, Name: "raw-cc", Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard, SortOrder: 1, RateMultiplier: 1},
		{ID: 20, Name: "responses-pro", Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard, SortOrder: 2, RateMultiplier: 1},
	}
	catalog.models = []string{"gpt-5.6-sol"}
	scheduler.reasoningByGroup = map[int64]bool{10: false, 20: true}
	options := WebChatReasoningOptions{Mode: WebChatReasoningModePro, Effort: "low"}

	normalized, err := svc.NormalizeWebChatReasoning(context.Background(), 7, "gpt-5.6-sol", options)
	require.NoError(t, err)
	require.Equal(t, options, normalized)

	principal, err := svc.ResolveWebChatPrincipal(context.Background(), 7, "gpt-5.6-sol", normalized)
	require.NoError(t, err)
	require.NotNil(t, principal)
	require.Equal(t, []int64{20}, principals.groupIDs)
}

func TestChatServiceProLowPreservesEffortIndependently(t *testing.T) {
	for _, model := range []string{"gpt-5.6", "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		t.Run(model, func(t *testing.T) {
			svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
			catalog.models = []string{model}

			got, err := svc.NormalizeWebChatReasoning(context.Background(), 7, model, WebChatReasoningOptions{
				Mode: WebChatReasoningModePro, Effort: "low",
			})
			require.NoError(t, err)
			require.Equal(t, WebChatReasoningOptions{Mode: WebChatReasoningModePro, Effort: "low"}, got)
		})
	}
}

func TestChatServiceCatalogPublishesReasoningCapabilities(t *testing.T) {
	svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-5.6-sol"}

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, result.Models, 1)
	model := result.Models[0]
	require.True(t, model.SupportsResponses)
	require.True(t, model.SupportsReasoningSummary)
	require.True(t, model.SupportsReasoningProMode)
	require.Equal(t, []string{"low", "medium", "high", "xhigh"}, model.SupportedReasoningEfforts)

	payload, err := json.Marshal(model)
	require.NoError(t, err)
	require.Contains(t, string(payload), `"supports_responses":true`)
	require.Contains(t, string(payload), `"supports_reasoning_summary":true`)
	require.Contains(t, string(payload), `"supports_reasoning_pro_mode":true`)
	require.Contains(t, string(payload), `"supported_reasoning_efforts"`)
}

func TestChatServiceCatalogPublishesProForVisibleGPT56FamilyOnly(t *testing.T) {
	svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna", "gpt-5.5"}

	result, err := svc.ListModels(context.Background(), 7)
	require.NoError(t, err)
	byID := make(map[string]ChatModel, len(result.Models))
	for _, model := range result.Models {
		byID[model.ID] = model
	}
	for _, model := range []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		require.True(t, byID[model].SupportsReasoningProMode, model)
	}
	require.False(t, byID["gpt-5.5"].SupportsReasoningProMode)
}

func TestChatServiceCatalogPublishesOnlySchedulableReasoningEffortPairs(t *testing.T) {
	t.Run("standard effort list is intersected with runtime accounts", func(t *testing.T) {
		svc, _, catalog, scheduler, _, _ := newChatServiceBehaviorTest()
		catalog.models = []string{"gpt-5.6-terra"}
		scheduler.reasoningByOptions = map[WebChatReasoningOptions]bool{
			{Mode: WebChatReasoningModeStandard, Effort: "low"}:  true,
			{Mode: WebChatReasoningModeStandard, Effort: "high"}: true,
			{Mode: WebChatReasoningModeStandard, Effort: "max"}:  true,
			{Mode: WebChatReasoningModePro, Effort: "low"}:       true,
			{Mode: WebChatReasoningModePro, Effort: "high"}:      false,
			{Mode: WebChatReasoningModePro, Effort: "max"}:       true,
		}

		result, err := svc.ListModels(context.Background(), 7)
		require.NoError(t, err)
		require.Len(t, result.Models, 1)
		model := result.Models[0]
		require.Equal(t, []string{"low", "high"}, model.SupportedReasoningEfforts)
		require.NotContains(t, scheduler.reasoningOptions,
			WebChatReasoningOptions{Mode: WebChatReasoningModeStandard, Effort: "max"},
			"the Web Chat catalog must expose only efforts understood by its picker")
		require.True(t, model.SupportsReasoningSummary)
		require.False(t, model.SupportsReasoningProMode,
			"Pro must stay hidden when any picker-visible effort would be rejected")
	})

	t.Run("upstream billing applies the same max fail-closed rule as POST admission", func(t *testing.T) {
		svc, _, catalog, scheduler, _, _ := newChatServiceBehaviorTest()
		catalog.models = []string{"gpt-5.6-terra"}
		catalog.mapping = ChannelMappingResult{
			MappedModel:        "gpt-5.6-terra",
			BillingModelSource: BillingModelSourceUpstream,
		}
		scheduler.schedulable = true

		result, err := svc.ListModels(context.Background(), 7)
		require.NoError(t, err)
		require.Len(t, result.Models, 1)
		model := result.Models[0]
		require.Equal(t, []string{"low", "medium", "high", "xhigh"}, model.SupportedReasoningEfforts)
		require.NotContains(t, scheduler.reasoningOptions,
			WebChatReasoningOptions{Mode: WebChatReasoningModeStandard, Effort: "max"})
		require.NotContains(t, scheduler.reasoningOptions,
			WebChatReasoningOptions{Mode: WebChatReasoningModePro, Effort: "max"})
		require.True(t, model.SupportsReasoningProMode)
	})
}

func TestChatServiceReasoningEffortRejectsMaxForEffectiveGPT56Sol(t *testing.T) {
	for _, tt := range []struct {
		name      string
		requested string
		effective string
		mapped    bool
	}{
		{name: "exact Sol", requested: "gpt-5.6-sol"},
		{name: "bare GPT-5.6 alias", requested: "gpt-5.6"},
		{name: "channel alias mapped to Sol", requested: "gpt-safe-alias", effective: "openai/gpt-5.6-sol", mapped: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
			catalog.models = []string{tt.requested}
			if tt.effective != "" {
				catalog.mapping = ChannelMappingResult{
					MappedModel:        tt.effective,
					Mapped:             tt.mapped,
					BillingModelSource: BillingModelSourceChannelMapped,
				}
			}

			got, err := svc.NormalizeReasoningEffort(context.Background(), 7, tt.requested, "max")
			require.Empty(t, got)
			require.ErrorIs(t, err, ErrChatReasoningNotAllowed)
			require.True(t, infraerrors.IsBadRequest(err))
			require.Equal(t, "CHAT_REASONING_EFFORT_NOT_AVAILABLE", infraerrors.Reason(err))
		})
	}
}

func TestChatServiceReasoningEffortKeepsMaxForNonSolModel(t *testing.T) {
	svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-5.5"}

	got, err := svc.NormalizeReasoningEffort(context.Background(), 7, "gpt-5.5", "max")
	require.NoError(t, err)
	require.Equal(t, "max", got)
}

func TestChatServiceReasoningEffortRejectsMaxWhenUpstreamMappingCannotProveNonSol(t *testing.T) {
	svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-5.5"}
	catalog.mapping = ChannelMappingResult{
		MappedModel:        "gpt-5.5",
		BillingModelSource: BillingModelSourceUpstream,
	}

	got, err := svc.NormalizeReasoningEffort(context.Background(), 7, "gpt-5.5", "max")
	require.Empty(t, got)
	require.ErrorIs(t, err, ErrChatReasoningNotAllowed)
	require.Equal(t, "CHAT_REASONING_EFFORT_NOT_AVAILABLE", infraerrors.Reason(err))
}

func TestChatServiceReasoningEffortPreservesNonMaxLevelsForUpstreamMapping(t *testing.T) {
	for _, effort := range []string{"low", "medium", "high", "xhigh"} {
		t.Run(effort, func(t *testing.T) {
			svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
			catalog.models = []string{"gpt-5.5"}
			catalog.mapping = ChannelMappingResult{
				MappedModel:        "gpt-5.5",
				BillingModelSource: BillingModelSourceUpstream,
			}

			got, err := svc.NormalizeReasoningEffort(context.Background(), 7, "gpt-5.5", effort)
			require.NoError(t, err)
			require.Equal(t, effort, got)
		})
	}
}

func TestChatServiceReasoningEffortMappingFailureFailsClosed(t *testing.T) {
	svc, _, catalog, _, _, _ := newChatServiceBehaviorTest()
	catalog.models = []string{"gpt-safe-alias"}
	catalog.mappingErr = errors.New("channel cache unavailable")

	got, err := svc.NormalizeReasoningEffort(context.Background(), 7, "gpt-safe-alias", "max")
	require.Empty(t, got)
	require.Error(t, err)
	require.True(t, infraerrors.IsServiceUnavailable(err))
	require.Equal(t, "CHAT_CATALOG_UNAVAILABLE", infraerrors.Reason(err))
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
		require.Contains(t, modelIDs, "gpt-5.6-sol")
		require.NotContains(t, modelIDs, "gpt-5.4")
		require.NotContains(t, modelIDs, "gpt-image-1")
		for _, model := range result.Models {
			if model.ID == "gpt-5.6-sol" {
				require.Equal(t, "GPT-5.6 Sol", model.DisplayName)
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
