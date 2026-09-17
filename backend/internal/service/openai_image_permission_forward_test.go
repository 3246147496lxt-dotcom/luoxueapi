//go:build unit

package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIImagePermissionForward_AccountRemainsAvailableForText(t *testing.T) {
	setGinTestMode()
	for _, passthrough := range []bool{false, true} {
		name := "managed_responses"
		if passthrough {
			name = "api_key_passthrough"
		}
		t.Run(name, func(t *testing.T) {
			repo := &rateLimitAccountRepoStub{}
			counter := &openAI403CounterCacheStub{counts: []int64{1, 2, 3}}
			rateLimiter := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
			rateLimiter.SetOpenAI403CounterCache(counter)
			upstream := &httpUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(imagePermissionTestBody)),
			}}
			svc := &OpenAIGatewayService{
				cfg:              &config.Config{},
				httpUpstream:     upstream,
				rateLimitService: rateLimiter,
			}
			rateLimiter.SetAccountRuntimeBlocker(svc)
			account := &Account{
				ID: 148, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://api.example.test"},
				Extra: map[string]any{
					"openai_passthrough":         passthrough,
					"openai_responses_supported": true,
				},
				Status: StatusActive, Schedulable: true,
			}
			newContext := func() (*gin.Context, *httptest.ResponseRecorder) {
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
				return c, recorder
			}
			c, recorder := newContext()
			result, err := svc.Forward(context.Background(), c, account,
				[]byte(`{"model":"gpt-5.5","stream":false,"input":"hello","tools":[{"type":"image_generation"}]}`))
			require.Nil(t, result)
			require.Error(t, err)
			var failoverErr *UpstreamFailoverError
			if passthrough {
				require.False(t, errors.As(err, &failoverErr))
				require.Equal(t, http.StatusBadGateway, recorder.Code, "preserve the sanitized upstream error contract")
			} else {
				require.ErrorAs(t, err, &failoverErr)
				require.Equal(t, http.StatusForbidden, failoverErr.StatusCode)
				require.False(t, failoverErr.RetryableOnSameAccount)
				require.False(t, c.Writer.Written(), "another upstream may support the requested capability")
			}
			require.Zero(t, repo.tempCalls)
			require.Zero(t, repo.setErrorCalls)
			require.Equal(t, []int64{1, 2, 3}, counter.counts)
			require.True(t, account.IsSchedulable())
			require.False(t, svc.isOpenAIAccountRequestRuntimeBlocked(account, "gpt-5.5"))

			// A capability denial must not prevent the same account from serving
			// the next ordinary text request.
			upstream.resp = &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"id":"resp_text_ok","model":"gpt-5.5","usage":{"input_tokens":3,"output_tokens":2}}`)),
			}
			c, recorder = newContext()
			result, err = svc.Forward(context.Background(), c, account,
				[]byte(`{"model":"gpt-5.5","stream":false,"input":"hello"}`))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, 3, result.Usage.InputTokens)
			require.Equal(t, 2, result.Usage.OutputTokens)
		})
	}
}
