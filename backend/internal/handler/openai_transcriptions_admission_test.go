package handler

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdmitTranscriptionRejectsInvalidUUIDBeforeGatewayLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/transcriptions", nil)
	c.Request.Header.Set("Idempotency-Key", "not-a-canonical-uuid")
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	h := &OpenAIGatewayHandler{cfg: &config.Config{
		RunMode: config.RunModeStandard,
		Transcription: config.TranscriptionConfig{
			Enabled: true,
		},
	}}

	release, admitted := h.AdmitTranscription(c)

	require.False(t, admitted)
	require.Nil(t, release)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "INVALID_IDEMPOTENCY_KEY")
}

type transcriptionAdmissionOrderChat struct {
	events     *[]string
	resolveErr error
}

func (s *transcriptionAdmissionOrderChat) ListModels(context.Context, int64) (*service.ChatModelsResult, error) {
	return nil, errors.New("not used")
}

func (s *transcriptionAdmissionOrderChat) ResolvePrincipal(context.Context, int64, string) (*service.ChatPrincipal, error) {
	return nil, errors.New("not used")
}

func (s *transcriptionAdmissionOrderChat) ResolveTranscriptionPrincipal(context.Context, int64) (*service.APIKey, error) {
	*s.events = append(*s.events, "resolve-principal")
	if s.resolveErr != nil {
		return nil, s.resolveErr
	}
	return nil, errors.New("stop after principal resolution")
}

type transcriptionAdmissionOrderGateway struct {
	events   *[]string
	admitted bool
}

func TestChatTranscriptionWorkflowDeadlineStopsBeforeDelegation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	events := make([]string, 0, 3)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/transcriptions", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	h := &ChatHandler{
		chat: &transcriptionAdmissionOrderChat{
			events:     &events,
			resolveErr: context.DeadlineExceeded,
		},
		gateway: &transcriptionAdmissionOrderGateway{events: &events, admitted: true},
	}

	h.Transcriptions(c)

	require.Equal(t, http.StatusGatewayTimeout, recorder.Code)
	require.Contains(t, recorder.Body.String(), "TRANSCRIPTION_TIMEOUT")
	require.Equal(t, []string{"admit", "resolve-principal", "release"}, events)
}

func (s *transcriptionAdmissionOrderGateway) ChatCompletions(*gin.Context) {}

func (s *transcriptionAdmissionOrderGateway) AdmitTranscription(c *gin.Context) (func(), bool) {
	*s.events = append(*s.events, "admit")
	if !s.admitted {
		transcriptionError(c, http.StatusTooManyRequests, "TRANSCRIPTION_BUSY", "Voice transcription is busy")
		return nil, false
	}
	return func() { *s.events = append(*s.events, "release") }, true
}

func (s *transcriptionAdmissionOrderGateway) Transcriptions(*gin.Context) {
	*s.events = append(*s.events, "delegate")
}

func TestChatTranscriptionAdmissionRunsBeforePrincipalResolution(t *testing.T) {
	for _, tc := range []struct {
		name       string
		admitted   bool
		wantEvents []string
	}{
		{name: "rejected ingress skips principal lookup", admitted: false, wantEvents: []string{"admit"}},
		{name: "admitted ingress holds lease through principal lookup", admitted: true, wantEvents: []string{"admit", "resolve-principal", "release"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			events := make([]string, 0, 3)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/transcriptions", nil)
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
			h := &ChatHandler{
				chat:    &transcriptionAdmissionOrderChat{events: &events},
				gateway: &transcriptionAdmissionOrderGateway{events: &events, admitted: tc.admitted},
			}

			h.Transcriptions(c)

			require.Equal(t, tc.wantEvents, events)
		})
	}
}

func TestChatTranscriptionRejectedAdmissionClosesSlowHTTP1BodyWithoutDrain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	events := make([]string, 0, 1)
	h := &ChatHandler{
		chat:    &transcriptionAdmissionOrderChat{events: &events},
		gateway: &transcriptionAdmissionOrderGateway{events: &events, admitted: false},
	}
	router := gin.New()
	router.POST("/api/v1/chat/transcriptions", func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
		h.Transcriptions(c)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	connection, err := net.Dial("tcp", server.Listener.Addr().String())
	require.NoError(t, err)
	defer func() { require.NoError(t, connection.Close()) }()
	require.NoError(t, connection.SetDeadline(time.Now().Add(2*time.Second)))
	_, err = io.WriteString(connection,
		"POST /api/v1/chat/transcriptions HTTP/1.1\r\n"+
			"Host: transcription.test\r\n"+
			"Content-Type: multipart/form-data; boundary=slow\r\n"+
			"Content-Length: 10485760\r\n\r\n",
	)
	require.NoError(t, err)

	responseMessage, err := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: http.MethodPost})
	require.NoError(t, err)
	defer func() { require.NoError(t, responseMessage.Body.Close()) }()
	_, err = io.Copy(io.Discard, responseMessage.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusTooManyRequests, responseMessage.StatusCode)
	require.True(t, responseMessage.Close)
	require.Equal(t, []string{"admit"}, events)

	oneByte := make([]byte, 1)
	_, err = connection.Read(oneByte)
	require.ErrorIs(t, err, io.EOF)
}

func TestTranscriptionConnectionReuseRestoredOnlyAfterServerForcedClose(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("server policy is reversible after the body reaches EOF", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/transcriptions", nil)

		forceCloseUntilTranscriptionBodyRead(c)
		require.True(t, c.Request.Close)
		require.Equal(t, "close", c.Writer.Header().Get("Connection"))

		allowConnectionReuseAfterTranscriptionBodyRead(c)
		require.False(t, c.Request.Close)
		require.Empty(t, c.Writer.Header().Get("Connection"))
	})

	t.Run("client requested close is never overridden", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/transcriptions", nil)
		c.Request.Close = true

		forceCloseUntilTranscriptionBodyRead(c)
		allowConnectionReuseAfterTranscriptionBodyRead(c)

		require.True(t, c.Request.Close)
		require.Equal(t, "close", c.Writer.Header().Get("Connection"))
	})
}
