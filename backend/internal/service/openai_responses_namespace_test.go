package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestShouldFlattenOpenAIResponsesNamespaces(t *testing.T) {
	oauth := &Account{Type: AccountTypeOAuth}
	apiKey := &Account{Type: AccountTypeAPIKey}

	tests := []struct {
		name               string
		account            *Account
		transport          OpenAIUpstreamTransport
		passthroughEnabled bool
		want               bool
	}{
		{name: "oauth_http", account: oauth, transport: OpenAIUpstreamTransportHTTPSSE, want: true},
		{name: "oauth_http_passthrough", account: oauth, transport: OpenAIUpstreamTransportHTTPSSE, passthroughEnabled: true, want: true},
		// WSv2 出口原样转发上游事件、不做回程还原，摊平会让客户端收到无法匹配的平名。
		{name: "oauth_wsv2", account: oauth, transport: OpenAIUpstreamTransportResponsesWebsocketV2, want: false},
		// 透传账号先于 WSv2 分支经 HTTP 转发返回，仍需摊平。
		{name: "oauth_wsv2_passthrough", account: oauth, transport: OpenAIUpstreamTransportResponsesWebsocketV2, passthroughEnabled: true, want: true},
		{name: "apikey_http", account: apiKey, transport: OpenAIUpstreamTransportHTTPSSE, want: false},
		{name: "nil_account", account: nil, transport: OpenAIUpstreamTransportHTTPSSE, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, shouldFlattenOpenAIResponsesNamespaces(tt.account, tt.transport, tt.passthroughEnabled))
		})
	}
}

func TestFlattenOpenAIResponsesNamespacesPreservesLargeNumbers(t *testing.T) {
	body := []byte(`{"model":"gpt-5","sequence":900719925474099312345,"tools":[{"type":"namespace","name":"collaboration","tools":[{"type":"function","name":"spawn_agent"}]}]}`)

	flattened, err := flattenOpenAIResponsesNamespaces(nil, body)

	require.NoError(t, err)
	require.Equal(t, "900719925474099312345", gjson.GetBytes(flattened, "sequence").Raw)
	require.True(t, gjson.GetBytes(flattened, `tools.#(type=="function")`).Exists())
}
