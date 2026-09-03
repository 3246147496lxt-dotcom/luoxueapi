package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/stretchr/testify/require"
)

type chatSchedAccountRepo struct {
	AccountRepository
	accounts     []Account
	listErr      error
	parents      map[int64]*Account
	parentErr    error
	lastPlatform string
}

func (r *chatSchedAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]Account, error) {
	r.lastPlatform = platform
	return r.accounts, r.listErr
}

func (r *chatSchedAccountRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	if r.parentErr != nil {
		return nil, r.parentErr
	}
	if parent := r.parents[id]; parent != nil {
		return parent, nil
	}
	return nil, ErrAccountNotFound
}

type chatStrictChannelRepo struct {
	ChannelRepository
	channels       []Channel
	groupPlatforms map[int64]string
	listErr        error
	platformErr    error
	listCalls      int
}

func (r *chatStrictChannelRepo) ListAll(context.Context) ([]Channel, error) {
	r.listCalls++
	return r.channels, r.listErr
}

func (r *chatStrictChannelRepo) GetGroupPlatforms(context.Context, []int64) (map[int64]string, error) {
	return r.groupPlatforms, r.platformErr
}

func chatSchedulableAccount(id int64, platform string, mapping map[string]any) Account {
	credentials := map[string]any{"api_key": "sk-test"}
	if mapping != nil {
		credentials["model_mapping"] = mapping
	}
	return Account{
		ID:          id,
		Platform:    platform,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: credentials,
	}
}

func chatChannelServiceFromCache(source string, restrict bool, pricingModels []string, mapping map[string]string) *ChannelService {
	repo := &chatStrictChannelRepo{}
	svc := NewChannelService(repo, nil, nil, nil)
	channel := Channel{
		ID:                 71,
		Name:               "chat",
		Status:             StatusActive,
		BillingModelSource: source,
		RestrictModels:     restrict,
		GroupIDs:           []int64{1},
		ModelMapping:       map[string]map[string]string{PlatformOpenAI: mapping},
	}
	if len(pricingModels) > 0 {
		channel.ModelPricing = []ChannelModelPricing{{
			ChannelID: 71,
			Platform:  PlatformOpenAI,
			Models:    pricingModels,
		}}
	}
	svc.cache.Store(populateChannelCache([]Channel{channel}, map[int64]string{1: PlatformOpenAI}))
	return svc
}

func newChatSchedulabilityService(repo *chatSchedAccountRepo, channel *ChannelService) *OpenAIGatewayService {
	return &OpenAIGatewayService{accountRepo: repo, channelService: channel}
}

func TestHasSchedulableChatCompletionsAccountUsesOpenAIRequestEligibility(t *testing.T) {
	model := "gpt-5.4"
	base := chatSchedulableAccount(1, PlatformOpenAI, map[string]any{model: model})

	tests := []struct {
		name   string
		mutate func(*OpenAIGatewayService, *Account)
		ctx    context.Context
		want   bool
	}{
		{name: "eligible", want: true},
		{
			name: "Grok account cannot satisfy OpenAI group",
			mutate: func(_ *OpenAIGatewayService, account *Account) {
				account.Platform = PlatformGrok
			},
		},
		{
			name: "account model mapping",
			mutate: func(_ *OpenAIGatewayService, account *Account) {
				account.Credentials["model_mapping"] = map[string]any{"gpt-other": "gpt-other"}
			},
		},
		{
			name: "Chat Completions endpoint capability",
			mutate: func(_ *OpenAIGatewayService, account *Account) {
				account.Credentials[openAIEndpointCapabilitiesCredentialKey] = []any{string(OpenAIEndpointCapabilityEmbeddings)}
			},
		},
		{
			name: "quota auto-pause",
			mutate: func(_ *OpenAIGatewayService, account *Account) {
				account.Extra = map[string]any{
					"codex_5h_used_percent":   95.0,
					"auto_pause_5h_threshold": 0.95,
				}
			},
		},
		{
			name: "whole-account runtime block",
			mutate: func(svc *OpenAIGatewayService, account *Account) {
				svc.BlockAccountScheduling(account, time.Now().Add(time.Minute), "test")
			},
		},
		{
			name: "model runtime block",
			mutate: func(svc *OpenAIGatewayService, account *Account) {
				canonical := canonicalOpenAIAccountSchedulingModel(account, model)
				svc.recordOpenAIAccountModelTransientFailure(account, canonical, time.Now())
				svc.recordOpenAIAccountModelTransientFailure(account, canonical, time.Now())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := base
			account.Credentials = map[string]any{
				"api_key":       "sk-test",
				"model_mapping": map[string]any{model: model},
			}
			repo := &chatSchedAccountRepo{accounts: []Account{account}}
			svc := newChatSchedulabilityService(repo, chatChannelServiceFromCache(BillingModelSourceRequested, false, nil, nil))
			if tt.mutate != nil {
				tt.mutate(svc, &repo.accounts[0])
			}
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.Background()
			}

			got, err := svc.HasSchedulableChatCompletionsAccount(ctx, 1, model)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.Equal(t, PlatformOpenAI, repo.lastPlatform)
		})
	}
}

func TestHasSchedulableWebChatReasoningAccountRequiresConfirmedResponsesAndModelMode(t *testing.T) {
	const model = "gpt-5.6-sol"
	newService := func(upstreamModel string, extra map[string]any) *OpenAIGatewayService {
		account := chatSchedulableAccount(1, PlatformOpenAI, map[string]any{upstreamModel: upstreamModel})
		account.Extra = extra
		repo := &chatSchedAccountRepo{accounts: []Account{account}}
		return newChatSchedulabilityService(
			repo,
			chatChannelServiceFromCache(BillingModelSourceRequested, false, nil, nil),
		)
	}

	standard := WebChatReasoningOptions{Mode: WebChatReasoningModeStandard, Effort: "medium"}
	pro := WebChatReasoningOptions{Mode: WebChatReasoningModePro, Effort: "medium"}

	unknown := newService(model, nil)
	got, err := unknown.HasSchedulableWebChatReasoningAccount(context.Background(), 1, model, standard)
	require.NoError(t, err)
	require.False(t, got, "unknown Responses support must fail closed for summaries")
	legacyGot, err := unknown.HasSchedulableChatCompletionsAccount(context.Background(), 1, model)
	require.NoError(t, err)
	require.True(t, legacyGot, "the existing Chat Completions admission contract remains unchanged")

	confirmedExtra := map[string]any{openai_compat.ExtraKeyResponsesSupported: true}
	confirmed := newService(model, confirmedExtra)
	got, err = confirmed.HasSchedulableWebChatReasoningAccount(context.Background(), 1, model, standard)
	require.NoError(t, err)
	require.True(t, got)
	for _, familyModel := range []string{"gpt-5.6-terra", "gpt-5.6-luna"} {
		familyService := newService(familyModel, confirmedExtra)
		got, err = familyService.HasSchedulableWebChatReasoningAccount(context.Background(), 1, familyModel, pro)
		require.NoError(t, err)
		require.True(t, got, familyModel)
	}
	got, err = confirmed.HasSchedulableWebChatReasoningAccount(context.Background(), 1, model, pro)
	require.NoError(t, err)
	require.True(t, got)

	got, err = confirmed.HasSchedulableWebChatReasoningAccount(context.Background(), 1, model, WebChatReasoningOptions{
		Mode: WebChatReasoningModePro, Effort: "low",
	})
	require.NoError(t, err)
	require.True(t, got, "Pro low must remain low and be scheduled independently")

	forcedChat := newService(model, map[string]any{
		openai_compat.ExtraKeyResponsesMode:      string(openai_compat.ResponsesSupportModeForceChatCompletions),
		openai_compat.ExtraKeyResponsesSupported: true,
	})
	got, err = forcedChat.HasSchedulableWebChatReasoningAccount(context.Background(), 1, model, pro)
	require.NoError(t, err)
	require.False(t, got)
}

func TestHasSchedulableChatCompletionsAccountRejectsDedicatedCapabilityModels(t *testing.T) {
	for _, model := range []string{
		"gpt-4o-search-preview",
		"gpt-image-2",
		"gpt-4o-audio-preview",
		"gpt-4o-realtime-preview",
		"gpt-4o-transcribe",
		"gpt-4o-mini-tts",
	} {
		t.Run(model, func(t *testing.T) {
			repo := &chatSchedAccountRepo{accounts: []Account{chatSchedulableAccount(1, PlatformOpenAI, nil)}}
			svc := newChatSchedulabilityService(repo, chatChannelServiceFromCache(BillingModelSourceRequested, false, nil, nil))

			got, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, model)
			require.NoError(t, err)
			require.False(t, got)
			require.Empty(t, repo.lastPlatform, "name-level rejection must happen before loading accounts")
		})
	}
}

func TestHasSchedulableChatCompletionsAccountRejectsUnsafeAccountMappingAcrossPool(t *testing.T) {
	const alias = "gpt-safe-alias"
	safe := chatSchedulableAccount(1, PlatformOpenAI, map[string]any{alias: "gpt-5.4"})
	unsafe := chatSchedulableAccount(2, PlatformOpenAI, map[string]any{alias: "gpt-4o-search-preview"})

	tests := []struct {
		name     string
		accounts []Account
		want     bool
	}{
		{name: "safe account", accounts: []Account{safe}, want: true},
		{name: "unsafe account", accounts: []Account{unsafe}},
		{name: "safe then unsafe account", accounts: []Account{safe, unsafe}},
		{name: "unsafe then safe account", accounts: []Account{unsafe, safe}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &chatSchedAccountRepo{accounts: tt.accounts}
			svc := newChatSchedulabilityService(repo, chatChannelServiceFromCache(BillingModelSourceRequested, false, nil, nil))

			got, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, alias)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHasSchedulableChatCompletionsAccountValidatesFinalModelAfterChannelAndAccountMappings(t *testing.T) {
	const (
		alias         = "gpt-safe-alias"
		channelTarget = "gpt-channel-text"
	)
	account := chatSchedulableAccount(1, PlatformOpenAI, map[string]any{
		alias:         alias,
		channelTarget: "gpt-4o-audio-preview",
	})
	repo := &chatSchedAccountRepo{accounts: []Account{account}}
	channel := chatChannelServiceFromCache(
		BillingModelSourceChannelMapped,
		false,
		nil,
		map[string]string{alias: channelTarget},
	)
	svc := newChatSchedulabilityService(repo, channel)

	got, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, alias)
	require.NoError(t, err)
	require.False(t, got)
}

func TestHasSchedulableChatCompletionsAccountRejectsRetiredGPT52AfterChannelAndAccountMappings(t *testing.T) {
	const (
		alias         = "gpt-safe-alias"
		channelTarget = "gpt-channel-text"
	)
	account := chatSchedulableAccount(1, PlatformOpenAI, map[string]any{
		alias:         alias,
		channelTarget: "gpt-5.2-codex",
	})
	repo := &chatSchedAccountRepo{accounts: []Account{account}}
	channel := chatChannelServiceFromCache(
		BillingModelSourceChannelMapped,
		false,
		nil,
		map[string]string{alias: channelTarget},
	)
	svc := newChatSchedulabilityService(repo, channel)

	got, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, alias)
	require.NoError(t, err)
	require.False(t, got)
}

func TestHasSchedulableChatCompletionsAccountUsesExactModelModeMetadata(t *testing.T) {
	const model = "gpt-private-special-purpose"
	account := chatSchedulableAccount(1, PlatformOpenAI, map[string]any{model: model})
	repo := &chatSchedAccountRepo{accounts: []Account{account}}
	svc := newChatSchedulabilityService(repo, chatChannelServiceFromCache(BillingModelSourceRequested, false, nil, nil))
	svc.billingService = &BillingService{pricingService: &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		model: {Mode: "audio_speech"},
	}}}

	got, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, model)
	require.NoError(t, err)
	require.False(t, got)
}

func TestHasSchedulableChatCompletionsAccountChecksShadowParentHealth(t *testing.T) {
	parentID := int64(90)
	shadow := Account{
		ID:              91,
		Platform:        PlatformOpenAI,
		Type:            AccountTypeOAuth,
		Status:          StatusActive,
		Schedulable:     true,
		Concurrency:     1,
		ParentAccountID: &parentID,
		QuotaDimension:  QuotaDimensionSpark,
	}
	parent := &Account{ID: parentID, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusDisabled}
	repo := &chatSchedAccountRepo{accounts: []Account{shadow}, parents: map[int64]*Account{parentID: parent}}
	svc := newChatSchedulabilityService(repo, chatChannelServiceFromCache(BillingModelSourceRequested, false, nil, nil))

	got, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, "gpt-5.3-codex-spark")
	require.NoError(t, err)
	require.False(t, got)

	parent.Status = StatusActive
	got, err = svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, "gpt-5.3-codex-spark")
	require.NoError(t, err)
	require.True(t, got)
}

func TestHasSchedulableChatCompletionsAccountPropagatesAccountAndParentErrors(t *testing.T) {
	t.Run("candidate repository", func(t *testing.T) {
		repo := &chatSchedAccountRepo{listErr: errors.New("account query failed")}
		svc := newChatSchedulabilityService(repo, chatChannelServiceFromCache(BillingModelSourceRequested, false, nil, nil))

		_, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, "gpt-5.4")
		require.ErrorIs(t, err, repo.listErr)
	})

	t.Run("shadow parent repository", func(t *testing.T) {
		parentID := int64(100)
		shadow := Account{
			ID:              101,
			Platform:        PlatformOpenAI,
			Type:            AccountTypeOAuth,
			Status:          StatusActive,
			Schedulable:     true,
			ParentAccountID: &parentID,
		}
		repo := &chatSchedAccountRepo{accounts: []Account{shadow}, parentErr: errors.New("parent query failed")}
		svc := newChatSchedulabilityService(repo, chatChannelServiceFromCache(BillingModelSourceRequested, false, nil, nil))

		_, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, "gpt-5.4")
		require.ErrorIs(t, err, repo.parentErr)
	})
}

func TestHasSchedulableChatCompletionsAccountChecksEveryChannelBillingSource(t *testing.T) {
	t.Run("requested restricted", func(t *testing.T) {
		account := chatSchedulableAccount(1, PlatformOpenAI, map[string]any{"gpt-5.4": "gpt-upstream"})
		repo := &chatSchedAccountRepo{accounts: []Account{account}}
		svc := newChatSchedulabilityService(repo, chatChannelServiceFromCache(BillingModelSourceRequested, true, []string{"gpt-other"}, nil))

		got, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, "gpt-5.4")
		require.NoError(t, err)
		require.False(t, got)
	})

	t.Run("channel mapped allowed", func(t *testing.T) {
		account := chatSchedulableAccount(1, PlatformOpenAI, map[string]any{"gpt-alias": "gpt-upstream"})
		repo := &chatSchedAccountRepo{accounts: []Account{account}}
		channel := chatChannelServiceFromCache(
			BillingModelSourceChannelMapped,
			true,
			[]string{"gpt-billing"},
			map[string]string{"gpt-alias": "gpt-billing"},
		)
		svc := newChatSchedulabilityService(repo, channel)

		got, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, "gpt-alias")
		require.NoError(t, err)
		require.True(t, got)
	})

	t.Run("upstream account mapping restricted", func(t *testing.T) {
		account := chatSchedulableAccount(1, PlatformOpenAI, map[string]any{"gpt-alias": "gpt-upstream"})
		repo := &chatSchedAccountRepo{accounts: []Account{account}}
		svc := newChatSchedulabilityService(repo, chatChannelServiceFromCache(BillingModelSourceUpstream, true, []string{"gpt-other"}, nil))

		got, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, "gpt-alias")
		require.NoError(t, err)
		require.False(t, got)
	})

	t.Run("upstream account mapping allowed", func(t *testing.T) {
		account := chatSchedulableAccount(1, PlatformOpenAI, map[string]any{"gpt-alias": "gpt-upstream"})
		repo := &chatSchedAccountRepo{accounts: []Account{account}}
		svc := newChatSchedulabilityService(repo, chatChannelServiceFromCache(BillingModelSourceUpstream, true, []string{"gpt-upstream"}, nil))

		got, err := svc.HasSchedulableChatCompletionsAccount(context.Background(), 1, "gpt-alias")
		require.NoError(t, err)
		require.True(t, got)
	})
}

func TestStrictChannelLookupKeepsCachedRepositoryFailureFailClosed(t *testing.T) {
	repo := &chatStrictChannelRepo{listErr: errors.New("channel database unavailable")}
	svc := NewChannelService(repo, nil, nil, nil)

	_, firstErr := svc.ResolveChannelMappingStrict(context.Background(), 1, "gpt-5.4")
	_, secondErr := svc.IsModelRestrictedStrict(context.Background(), 1, "gpt-5.4")
	require.ErrorIs(t, firstErr, repo.listErr)
	require.ErrorIs(t, secondErr, repo.listErr)
	require.Equal(t, 1, repo.listCalls, "short-lived error cache must preserve the failure without retrying or failing open")
}

func TestGatewayStrictCatalogPropagatesCandidateRepositoryFailure(t *testing.T) {
	repo := &chatSchedAccountRepo{listErr: errors.New("account database unavailable")}
	svc := &GatewayService{accountRepo: repo}
	groupID := int64(1)

	_, err := svc.GetAvailableModelsStrict(context.Background(), &groupID, PlatformOpenAI)
	require.ErrorIs(t, err, repo.listErr)
}

func TestGatewayStrictCatalogUsesOpenAIDefaultsWhenAccountMappingsAreEmpty(t *testing.T) {
	repo := &chatSchedAccountRepo{accounts: []Account{
		chatSchedulableAccount(1, PlatformOpenAI, nil),
		chatSchedulableAccount(2, PlatformOpenAI, map[string]any{}),
	}}
	svc := &GatewayService{accountRepo: repo}
	groupID := int64(1)

	models, err := svc.GetAvailableModelsStrict(context.Background(), &groupID, PlatformOpenAI)
	require.NoError(t, err)
	require.Equal(t, openai.DefaultModelIDs(), models)
}

func TestGatewayStrictChannelMethodsFailClosedWithoutChannelService(t *testing.T) {
	svc := &GatewayService{}

	_, mappingErr := svc.ResolveChannelMappingStrict(context.Background(), 1, "gpt-5.4")
	_, restrictionErr := svc.IsModelRestrictedStrict(context.Background(), 1, "gpt-5.4")
	require.Error(t, mappingErr)
	require.Error(t, restrictionErr)
}
