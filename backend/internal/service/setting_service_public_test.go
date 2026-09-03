//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingPublicRepoStub struct {
	values map[string]string
}

func (s *settingPublicRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingPublicRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *settingPublicRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingPublicRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingPublicRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingPublicRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingPublicRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingService_GetPublicSettings_ExposesRegistrationEmailSuffixWhitelist(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyRegistrationEnabled:              "true",
			SettingKeyEmailVerifyEnabled:               "true",
			SettingKeyRegistrationEmailSuffixWhitelist: `["@EXAMPLE.com"," @foo.bar ","*.EDU.CN","@invalid_domain",""]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar", "*.edu.cn"}, settings.RegistrationEmailSuffixWhitelist)
}

func TestSettingService_GetPublicSettings_DistinguishesCustomizedSiteSubtitle(t *testing.T) {
	t.Run("system default", func(t *testing.T) {
		svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{})

		settings, err := svc.GetPublicSettings(context.Background())
		require.NoError(t, err)
		require.Equal(t, "Subscription to API Conversion Platform", settings.SiteSubtitle)
		require.False(t, settings.SiteSubtitleCustomized)
	})

	t.Run("administrator configured", func(t *testing.T) {
		svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{
			SettingKeySiteSubtitle: "Subscription to API Conversion Platform",
		}}, &config.Config{})

		settings, err := svc.GetPublicSettings(context.Background())
		require.NoError(t, err)
		require.Equal(t, "Subscription to API Conversion Platform", settings.SiteSubtitle)
		require.True(t, settings.SiteSubtitleCustomized)
	})
}

func TestSettingService_GetPublicSettings_ExposesTablePreferences(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyTableDefaultPageSize: "50",
			SettingKeyTablePageSizeOptions: "[20,50,100]",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, 50, settings.TableDefaultPageSize)
	require.Equal(t, []int{20, 50, 100}, settings.TablePageSizeOptions)
}

func TestSettingService_GetPublicSettings_ExposesForceEmailOnThirdPartySignup(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyForceEmailOnThirdPartySignup: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.ForceEmailOnThirdPartySignup)
}

func TestSettingService_GetPublicSettings_ExposesAllowUserViewErrorRequests(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyAllowUserViewErrorRequests: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.AllowUserViewErrorRequests)
}

func TestSettingService_GetPublicSettingsForInjectionSanitizesCustomMenuURLs(t *testing.T) {
	rawMenuItems := `[
		{"id":"user-page","label":"User page","url":"https://admin:basic-secret@external.example/start?plan=pro&Token=secret&USER_ID=7&sRc_Url=https%3A%2F%2Fprivate.example%2Fpath&access_token=access-secret&api_key=api-secret&code=auth-code&s2a_launch_code=launch-secret&%74oken=encoded#token=fragment","visibility":"user","sort_order":1},
		{"id":"admin-page","label":"Admin page","url":"https://admin.example/start?token=admin-secret","visibility":"admin","sort_order":2}
	]`
	repo := &settingPublicRepoStub{values: map[string]string{
		SettingKeyCustomMenuItems: rawMenuItems,
	}}
	svc := NewSettingService(repo, &config.Config{})

	publicSettings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.NotContains(t, publicSettings.CustomMenuItems, "secret")
	require.NotContains(t, publicSettings.CustomMenuItems, "fragment")

	injected, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	payload, ok := injected.(*PublicSettingsInjectionPayload)
	require.True(t, ok)

	var items []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(payload.CustomMenuItems, &items))
	require.Len(t, items, 1)
	require.Equal(t, "user-page", items[0].ID)

	publicURL, err := url.Parse(items[0].URL)
	require.NoError(t, err)
	require.Nil(t, publicURL.User)
	require.Empty(t, publicURL.RawQuery)
	require.Empty(t, publicURL.Fragment)
	require.NotContains(t, items[0].URL, "secret")
	require.NotContains(t, items[0].URL, "auth-code")
	require.Equal(t, rawMenuItems, repo.values[SettingKeyCustomMenuItems], "public serialization must not mutate stored settings")
}

func TestSettingService_GetPublicSettingsForInjectionOmitsDisabledLoginAgreementDocuments(t *testing.T) {
	rawDocuments := `[{"id":"terms","title":"Terms","content_md":"A long agreement body"}]`

	t.Run("disabled keeps documents in public API but not SSR payload", func(t *testing.T) {
		repo := &settingPublicRepoStub{values: map[string]string{
			SettingKeyLoginAgreementEnabled:   "false",
			SettingKeyLoginAgreementDocuments: rawDocuments,
		}}
		svc := NewSettingService(repo, &config.Config{})

		publicSettings, err := svc.GetPublicSettings(context.Background())
		require.NoError(t, err)
		require.False(t, publicSettings.LoginAgreementEnabled)
		require.Len(t, publicSettings.LoginAgreementDocuments, 1)

		injected, err := svc.GetPublicSettingsForInjection(context.Background())
		require.NoError(t, err)
		payload, ok := injected.(*PublicSettingsInjectionPayload)
		require.True(t, ok)
		require.Empty(t, payload.LoginAgreementDocuments)

		encoded, err := json.Marshal(payload)
		require.NoError(t, err)
		require.Contains(t, string(encoded), `"login_agreement_documents":[]`)
	})

	t.Run("enabled keeps documents in SSR payload", func(t *testing.T) {
		repo := &settingPublicRepoStub{values: map[string]string{
			SettingKeyLoginAgreementEnabled:   "true",
			SettingKeyLoginAgreementDocuments: rawDocuments,
		}}
		svc := NewSettingService(repo, &config.Config{})

		injected, err := svc.GetPublicSettingsForInjection(context.Background())
		require.NoError(t, err)
		payload, ok := injected.(*PublicSettingsInjectionPayload)
		require.True(t, ok)
		require.True(t, payload.LoginAgreementEnabled)
		require.Len(t, payload.LoginAgreementDocuments, 1)
		require.Equal(t, "A long agreement body", payload.LoginAgreementDocuments[0].ContentMD)
	})
}

func TestSettingService_GetPublicSettings_HidesCatalogInBackendMode(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{
		SettingKeyPublicModelCatalogEnabled: "true",
		SettingKeyBackendModeEnabled:        "true",
	}}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.BackendModeEnabled)
	require.False(t, settings.PublicModelCatalogEnabled)
}

func TestSettingService_GetPublicSettings_ExposesWeChatOAuthModeCapabilities(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:             "true",
			SettingKeyWeChatConnectAppID:               "wx-mp-app",
			SettingKeyWeChatConnectAppSecret:           "wx-mp-secret",
			SettingKeyWeChatConnectMode:                "mp",
			SettingKeyWeChatConnectScopes:              "snsapi_base",
			SettingKeyWeChatConnectOpenEnabled:         "true",
			SettingKeyWeChatConnectMPEnabled:           "true",
			SettingKeyWeChatConnectRedirectURL:         "https://api.example.com/api/v1/auth/oauth/wechat/callback",
			SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.WeChatOAuthEnabled)
	require.True(t, settings.WeChatOAuthOpenEnabled)
	require.True(t, settings.WeChatOAuthMPEnabled)
}

func TestSettingService_GetPublicSettings_DoesNotExposeMobileOnlyWeChatAsWebOAuthAvailable(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:             "true",
			SettingKeyWeChatConnectMobileEnabled:       "true",
			SettingKeyWeChatConnectMode:                "mobile",
			SettingKeyWeChatConnectMobileAppID:         "wx-mobile-app",
			SettingKeyWeChatConnectMobileAppSecret:     "wx-mobile-secret",
			SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.WeChatOAuthEnabled)
	require.False(t, settings.WeChatOAuthOpenEnabled)
	require.False(t, settings.WeChatOAuthMPEnabled)
	require.True(t, settings.WeChatOAuthMobileEnabled)
}

func TestSettingService_GetPublicSettings_FallsBackToConfigForWeChatOAuthCapabilities(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Enabled:             true,
			OpenEnabled:         true,
			OpenAppID:           "wx-open-config",
			OpenAppSecret:       "wx-open-secret",
			FrontendRedirectURL: "/auth/wechat/config-callback",
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.WeChatOAuthEnabled)
	require.True(t, settings.WeChatOAuthOpenEnabled)
	require.False(t, settings.WeChatOAuthMPEnabled)
	require.False(t, settings.WeChatOAuthMobileEnabled)
}
