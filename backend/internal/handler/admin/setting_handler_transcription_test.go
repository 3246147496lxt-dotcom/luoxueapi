package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpdateWebChatTranscriptionSettingsRequiresDailyAudioSeconds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/settings/transcription",
		strings.NewReader(`{"enabled":false,"model":"transcribe-v2","group_ids":[]}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")
	h := &SettingHandler{settingService: service.NewSettingService(nil, nil)}

	h.UpdateWebChatTranscriptionSettings(c)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	var got response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
	require.Equal(t, "INVALID_TRANSCRIPTION_SETTINGS", got.Reason)
	require.Equal(t, "user_daily_audio_seconds", got.Metadata["field"])
}
