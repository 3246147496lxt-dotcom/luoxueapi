//go:build unit

package service

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func zhipuAccountForConnectionTest(id int64, protocol string, extras map[string]any) *Account {
	credentials := map[string]any{
		"api_key": "sk-zhipu-test",
	}
	if protocol != "" {
		credentials["api_protocol"] = protocol
	}
	for key, value := range extras {
		credentials[key] = value
	}
	return &Account{
		ID:          id,
		Platform:    PlatformZhipu,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Concurrency: 1,
		Credentials: credentials,
	}
}

func zhipuAccountTestService(account *Account, responses ...*http.Response) (*AccountTestService, *httpUpstreamRecorder) {
	repo := &mockAccountRepoForGemini{
		accountsByID: map[int64]*Account{account.ID: account},
	}
	upstream := &httpUpstreamRecorder{responses: responses}
	return &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
		cfg:          rawChatCompletionsTestConfig(),
	}, upstream
}

func zhipuChatConnectionResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(
			"data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: [DONE]\n\n",
		)),
	}
}

func zhipuAnthropicConnectionResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(
			"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"ok\"}}\n\n" +
				"data: {\"type\":\"message_stop\"}\n\n",
		)),
	}
}

func TestAccountTestService_ZhipuChatProtocolUsesCodingChatCompletions(t *testing.T) {
	account := zhipuAccountForConnectionTest(401, "", map[string]any{
		"account_mode": AccountModeCoding,
	})
	svc, upstream := zhipuAccountTestService(account, zhipuChatConnectionResponse())
	c, recorder := newTestContext()

	err := svc.TestAccountConnection(c, account.ID, "glm-4.7", "hello", AccountTestModeDefault)

	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://open.bigmodel.cn/api/coding/paas/v4/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer sk-zhipu-test", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, HTTPUpstreamProfileOpenAI, HTTPUpstreamProfileFromContext(upstream.lastReq.Context()))
	require.Equal(t, "glm-4.7", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "hello", gjson.GetBytes(upstream.lastBody, "messages.0.content").String())
	require.Contains(t, recorder.Body.String(), `"success":true`)
}

func TestAccountTestService_ZhipuResponsesProtocolFallsBackToChatCompletions(t *testing.T) {
	account := zhipuAccountForConnectionTest(402, APIProtocolResponses, map[string]any{
		"base_url": DefaultZhipuCodingBaseURL,
	})
	svc, upstream := zhipuAccountTestService(account, zhipuChatConnectionResponse())
	c, recorder := newTestContext()

	err := svc.TestAccountConnection(c, account.ID, "glm-4.7", "hello", AccountTestModeDefault)

	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	// Zhipu does not expose the native Responses contract in the account
	// adapter, so an explicit Responses selection remains a Chat bridge probe.
	require.Equal(t, "https://open.bigmodel.cn/api/coding/paas/v4/chat/completions", upstream.lastReq.URL.String())
	require.True(t, gjson.GetBytes(upstream.lastBody, "messages").Exists())
	require.False(t, gjson.GetBytes(upstream.lastBody, "input").Exists())
	require.Contains(t, recorder.Body.String(), `"success":true`)
}

func TestAccountTestService_ZhipuAnthropicProtocolUsesNativeMessages(t *testing.T) {
	account := zhipuAccountForConnectionTest(403, APIProtocolAnthropic, map[string]any{
		"base_url": DefaultZhipuAnthropicBaseURL,
	})
	svc, upstream := zhipuAccountTestService(account, zhipuAnthropicConnectionResponse())
	c, recorder := newTestContext()

	err := svc.TestAccountConnection(c, account.ID, "glm-4.7", "", AccountTestModeDefault)

	require.NoError(t, err)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "https://open.bigmodel.cn/api/anthropic/v1/messages", upstream.lastReq.URL.String())
	require.Empty(t, upstream.lastReq.URL.RawQuery)
	require.Equal(t, "sk-zhipu-test", upstream.lastReq.Header.Get("x-api-key"))
	require.Empty(t, upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "2023-06-01", upstream.lastReq.Header.Get("anthropic-version"))
	require.Equal(t, "glm-4.7", gjson.GetBytes(upstream.lastBody, "model").String())
	require.True(t, gjson.GetBytes(upstream.lastBody, "stream").Bool())
	require.Equal(t, "hi", gjson.GetBytes(upstream.lastBody, "messages.0.content").String())
	require.Contains(t, recorder.Body.String(), `"success":true`)
}
