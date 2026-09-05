package service

import "testing"

func TestZhipuAnthropicBaseURLPreservesOfficialHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		base string
		want string
	}{
		{
			name: "international anthropic endpoint",
			base: "https://api.z.ai/api/anthropic",
			want: "https://api.z.ai/api/anthropic",
		},
		{
			name: "international coding endpoint is normalized",
			base: "https://api.z.ai/api/coding/paas/v4/",
			want: "https://api.z.ai/api/anthropic",
		},
		{
			name: "mainland coding endpoint is normalized",
			base: DefaultZhipuCodingBaseURL,
			want: DefaultZhipuAnthropicBaseURL,
		},
		{
			name: "custom relay remains opaque",
			base: "https://relay.example/glm/v1/messages",
			want: "https://relay.example/glm",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			account := &Account{
				Platform: PlatformZhipu,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"api_protocol": APIProtocolAnthropic,
					"base_url":     tc.base,
				},
			}
			if got := account.GetAnthropicProtocolBaseURL(); got != tc.want {
				t.Fatalf("GetAnthropicProtocolBaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestZhipuOpenAIFormatBaseURLPreservesOfficialHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		mode string
		base string
		want string
	}{
		{
			name: "international payg from anthropic",
			mode: AccountModePayG,
			base: "https://api.z.ai/api/anthropic",
			want: "https://api.z.ai/api/paas/v4",
		},
		{
			name: "international coding from anthropic",
			mode: AccountModeCoding,
			base: "https://api.z.ai/api/anthropic",
			want: "https://api.z.ai/api/coding/paas/v4",
		},
		{
			name: "international coding from mismatched openai endpoint",
			mode: AccountModeCoding,
			base: "https://api.z.ai/api/paas/v4",
			want: "https://api.z.ai/api/coding/paas/v4",
		},
		{
			name: "custom relay is never redirected to official host",
			mode: AccountModePayG,
			base: "https://relay.example/glm/anthropic",
			want: "https://relay.example/glm/anthropic",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			account := &Account{
				Platform: PlatformZhipu,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"account_mode": tc.mode,
					"api_protocol": APIProtocolAnthropic,
					"base_url":     tc.base,
				},
			}
			if got := account.GetOpenAIFormatBaseURL(); got != tc.want {
				t.Fatalf("GetOpenAIFormatBaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestZhipuCodingPlanProviderRequiresExactOfficialHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		baseURL string
		want    string
	}{
		{name: "mainland official", baseURL: DefaultZhipuCodingBaseURL, want: PlatformZhipu},
		{name: "international official", baseURL: "https://api.z.ai/api/coding/paas/v4", want: PlatformZhipu},
		{name: "host in path is not official", baseURL: "https://relay.example/forward/api.z.ai/api/coding/paas/v4", want: ""},
		{name: "lookalike suffix is not official", baseURL: "https://api.z.ai.evil.example/api/coding/paas/v4", want: ""},
		{name: "explicit port is rejected", baseURL: "https://api.z.ai:8443/api/coding/paas/v4", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			account := &Account{
				Platform: PlatformZhipu,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"account_mode": AccountModeCoding,
					"base_url":     tc.baseURL,
				},
			}
			if got := account.GetCodingPlanProvider(); got != tc.want {
				t.Fatalf("GetCodingPlanProvider() = %q, want %q", got, tc.want)
			}
		})
	}
}
