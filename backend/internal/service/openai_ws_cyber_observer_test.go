package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestObserveOpenAIWSCyberPolicyContract(t *testing.T) {
	setGinTestMode()

	tests := []struct {
		name           string
		payload        string
		upstreamStatus int
		usage          *OpenAIUsage
		wantHit        bool
		wantStatus     int
		wantInput      int
		wantOutput     int
		wantMessage    string
	}{
		{
			name:           "cyber uses supplied usage and defaults status",
			payload:        `{"type":"response.failed","response":{"error":{"code":"cyber_policy","message":"blocked by cyber policy"},"usage":{"input_tokens":1,"output_tokens":2}}}`,
			upstreamStatus: 0,
			usage:          &OpenAIUsage{InputTokens: 31, OutputTokens: 7},
			wantHit:        true,
			wantStatus:     http.StatusOK,
			wantInput:      31,
			wantOutput:     7,
			wantMessage:    "blocked by cyber policy",
		},
		{
			name:           "cyber parses usage when caller has none",
			payload:        `{"type":"response.failed","response":{"error":{"code":"cyber_policy","message":"bridge blocked"},"usage":{"input_tokens":23,"output_tokens":4}}}`,
			upstreamStatus: http.StatusForbidden,
			wantHit:        true,
			wantStatus:     http.StatusForbidden,
			wantInput:      23,
			wantOutput:     4,
			wantMessage:    "bridge blocked",
		},
		{
			name:           "ordinary failed response is not cyber",
			payload:        `{"type":"response.failed","response":{"error":{"code":"server_error","message":"ordinary upstream failure"},"usage":{"input_tokens":9,"output_tokens":1}}}`,
			upstreamStatus: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)

			hit := observeOpenAIWSCyberPolicy(
				c,
				[]byte(test.payload),
				test.upstreamStatus,
				test.usage,
			)

			require.Equal(t, test.wantHit, hit)
			mark := GetOpsCyberPolicy(c)
			if !test.wantHit {
				require.Nil(t, mark)
				return
			}
			require.NotNil(t, mark)
			require.Equal(t, "cyber_policy", mark.Code)
			require.Equal(t, test.wantMessage, mark.Message)
			require.Equal(t, test.wantStatus, mark.UpstreamStatus)
			require.Equal(t, test.wantInput, mark.UpstreamInTok)
			require.Equal(t, test.wantOutput, mark.UpstreamOutTok)
			require.Contains(t, mark.Body, `"code":"cyber_policy"`)
		})
	}
}

func TestProxyOpenAIWSHTTPBridgeTurnObservesCyberPolicy(t *testing.T) {
	setGinTestMode()

	tests := []struct {
		name       string
		errorCode  string
		errorMsg   string
		wantCyber  bool
		wantInput  int
		wantOutput int
	}{
		{
			name:       "cyber terminal",
			errorCode:  "cyber_policy",
			errorMsg:   "bridge blocked",
			wantCyber:  true,
			wantInput:  23,
			wantOutput: 4,
		},
		{
			name:       "ordinary failed terminal",
			errorCode:  "invalid_request_error",
			errorMsg:   "ordinary request failure",
			wantCyber:  false,
			wantInput:  9,
			wantOutput: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failedEvent := `{"type":"response.failed","response":{"id":"resp_bridge_failed","status":"failed","error":{"code":"` +
				test.errorCode + `","message":"` + test.errorMsg +
				`"},"usage":{"input_tokens":` + strconv.Itoa(test.wantInput) +
				`,"output_tokens":` + strconv.Itoa(test.wantOutput) + `}}}`
			upstream := &httpUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body: io.NopCloser(strings.NewReader(strings.Join([]string{
					"data: " + failedEvent,
					"",
				}, "\n"))),
			}}
			svc := &OpenAIGatewayService{
				cfg: &config.Config{
					Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize},
				},
				httpUpstream:  upstream,
				toolCorrector: NewCodexToolCorrector(),
			}
			account := &Account{
				ID:          8101,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Concurrency: 1,
				Status:      StatusActive,
			}
			payload := []byte(`{"type":"response.create","model":"gpt-5.5","stream":true,"input":"hello"}`)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/v1/responses", nil)
			var relayed [][]byte

			result, err := svc.proxyOpenAIWSHTTPBridgeTurn(
				context.Background(),
				c,
				account,
				"sk-test",
				payload,
				len(payload),
				"gpt-5.5",
				"",
				"",
				"",
				"",
				1,
				func(message []byte) error {
					relayed = append(relayed, append([]byte(nil), message...))
					return nil
				},
			)

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "response.failed", result.UpstreamTerminalEvent)
			require.Equal(t, test.wantInput, result.Usage.InputTokens)
			require.Equal(t, test.wantOutput, result.Usage.OutputTokens)
			require.Len(t, relayed, 1)
			require.Equal(t, test.wantCyber, GetOpsCyberPolicy(c) != nil)
			if test.wantCyber {
				mark := GetOpsCyberPolicy(c)
				require.Equal(t, test.wantInput, mark.UpstreamInTok)
				require.Equal(t, test.wantOutput, mark.UpstreamOutTok)
			}
		})
	}
}

func TestProxyResponsesWebSocketV2PassthroughObservesCyberPolicy(t *testing.T) {
	setGinTestMode()

	tests := []struct {
		name      string
		errorCode string
		errorMsg  string
		wantCyber bool
	}{
		{
			name:      "cyber terminal is marked before after turn",
			errorCode: "cyber_policy",
			errorMsg:  "passthrough blocked",
			wantCyber: true,
		},
		{
			name:      "ordinary failed terminal remains unmarked",
			errorCode: "invalid_request_error",
			errorMsg:  "ordinary request failure",
			wantCyber: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failedEvent := []byte(
				`{"type":"response.failed","response":{"id":"resp_passthrough_failed","status":"failed","error":{"code":"` +
					test.errorCode + `","message":"` + test.errorMsg +
					`"},"usage":{"input_tokens":17,"output_tokens":3}}}`,
			)
			upstreamConn := &openAIWSCaptureConn{events: [][]byte{failedEvent}}
			cfg := newOpenAIWSV2TestConfig()
			cfg.Security.URLAllowlist.Enabled = false
			cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
			cfg.Gateway.OpenAIWS.IngressModeDefault = OpenAIWSIngressModePassthrough
			cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
			cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 3
			cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3
			svc := &OpenAIGatewayService{
				cfg:                       cfg,
				openaiWSResolver:          NewOpenAIWSProtocolResolver(cfg),
				toolCorrector:             NewCodexToolCorrector(),
				openaiWSPassthroughDialer: &openAIWSCaptureDialer{conn: upstreamConn},
			}
			account := &Account{
				ID:          8201,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Credentials: map[string]any{"api_key": "sk-test"},
				Extra: map[string]any{
					"openai_apikey_responses_websockets_v2_mode": OpenAIWSIngressModePassthrough,
				},
			}

			type turnObservation struct {
				result  *OpenAIForwardResult
				turnErr error
				mark    *CyberPolicyMark
			}
			turnCh := make(chan turnObservation, 1)
			serverErrCh := make(chan error, 1)
			wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{
					CompressionMode: coderws.CompressionContextTakeover,
				})
				if err != nil {
					serverErrCh <- err
					return
				}
				defer func() { _ = conn.CloseNow() }()

				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = r.Clone(r.Context())

				readCtx, cancelRead := context.WithTimeout(r.Context(), 3*time.Second)
				_, firstMessage, readErr := conn.Read(readCtx)
				cancelRead()
				if readErr != nil {
					serverErrCh <- readErr
					return
				}
				hooks := &OpenAIWSIngressHooks{
					InitialRequestModel: "gpt-5.5",
					AfterTurn: func(_ int, result *OpenAIForwardResult, turnErr error) {
						var markSnapshot *CyberPolicyMark
						if mark := GetOpsCyberPolicy(c); mark != nil {
							snapshot := *mark
							markSnapshot = &snapshot
						}
						turnCh <- turnObservation{
							result:  result,
							turnErr: turnErr,
							mark:    markSnapshot,
						}
					},
				}
				serverErrCh <- svc.ProxyResponsesWebSocketFromClient(
					r.Context(),
					c,
					conn,
					account,
					"sk-test",
					firstMessage,
					hooks,
				)
			}))
			defer wsServer.Close()

			dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
			clientConn, _, err := coderws.Dial(
				dialCtx,
				"ws"+strings.TrimPrefix(wsServer.URL, "http"),
				nil,
			)
			cancelDial()
			require.NoError(t, err)
			defer func() { _ = clientConn.CloseNow() }()

			request := []byte(`{"type":"response.create","model":"gpt-5.5","stream":true,"input":"hello"}`)
			writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
			err = clientConn.Write(writeCtx, coderws.MessageText, request)
			cancelWrite()
			require.NoError(t, err)

			readCtx, cancelRead := context.WithTimeout(context.Background(), 3*time.Second)
			msgType, event, err := clientConn.Read(readCtx)
			cancelRead()
			require.NoError(t, err)
			require.Equal(t, coderws.MessageText, msgType)
			eventType, _, _ := parseOpenAIWSEventEnvelope(event)
			require.Equal(t, "response.failed", eventType)

			select {
			case observation := <-turnCh:
				require.NoError(t, observation.turnErr)
				require.NotNil(t, observation.result)
				require.Equal(t, "response.failed", observation.result.UpstreamTerminalEvent)
				require.Equal(t, test.wantCyber, observation.mark != nil)
				if test.wantCyber {
					require.Equal(t, 17, observation.mark.UpstreamInTok)
					require.Equal(t, 3, observation.mark.UpstreamOutTok)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("timed out waiting for passthrough AfterTurn")
			}

			require.NoError(t, clientConn.CloseNow())
			select {
			case proxyErr := <-serverErrCh:
				require.NoError(t, proxyErr)
			case <-time.After(3 * time.Second):
				t.Fatal("timed out waiting for passthrough proxy result")
			}
		})
	}
}
