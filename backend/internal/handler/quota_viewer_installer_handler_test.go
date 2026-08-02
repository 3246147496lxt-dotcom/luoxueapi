//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestQuotaViewerInstallerHandlerIssuesAndRedirectsExactlyOnce(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { require.NoError(t, rdb.Close()) })
	svc := service.NewQuotaViewerInstallerDownloadService(repository.NewQuotaViewerInstallerTicketStore(rdb))
	h := NewQuotaViewerInstallerHandler(svc)

	issueRecorder := httptest.NewRecorder()
	issueContext, _ := gin.CreateTestContext(issueRecorder)
	issueContext.Request = httptest.NewRequest(http.MethodPost, "/api/v1/quota/releases/macos/latest/download", nil)
	issueContext.Params = gin.Params{{Key: "platform", Value: "macos"}}
	issueContext.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})

	h.IssueDownload(issueContext)

	require.Equal(t, http.StatusOK, issueRecorder.Code, issueRecorder.Body.String())
	require.Equal(t, "no-store", issueRecorder.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", issueRecorder.Header().Get("Referrer-Policy"))
	require.Equal(t, QuotaViewerInstallerDownloadAuditAction, issueContext.GetString("audit_action"))
	var issueResponse struct {
		Data service.QuotaViewerInstallerDownload `json:"data"`
	}
	require.NoError(t, json.Unmarshal(issueRecorder.Body.Bytes(), &issueResponse))
	require.Equal(t, "macos", issueResponse.Data.Platform)
	require.Contains(t, issueResponse.Data.DownloadPath, "/api/v1/quota/releases/macos/latest/download?code=")

	downloadRecorder := httptest.NewRecorder()
	downloadContext, _ := gin.CreateTestContext(downloadRecorder)
	downloadContext.Request = httptest.NewRequest(http.MethodGet, issueResponse.Data.DownloadPath, nil)
	downloadContext.Params = gin.Params{{Key: "platform", Value: "macos"}}
	h.Download(downloadContext)

	require.Equal(t, http.StatusTemporaryRedirect, downloadRecorder.Code)
	require.Equal(t, "no-store", downloadRecorder.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", downloadRecorder.Header().Get("Referrer-Policy"))
	require.Equal(
		t,
		"https://github.com/3246147496lxt-dotcom/luoxueapi/releases/download/quota-viewer-v2.0.0-rc.6/Luoxue-Quota-Viewer_2.0.0-rc.6_macOS-universal-UNNOTARIZED.dmg",
		downloadRecorder.Header().Get("Location"),
	)

	replayRecorder := httptest.NewRecorder()
	replayContext, _ := gin.CreateTestContext(replayRecorder)
	replayContext.Request = httptest.NewRequest(http.MethodGet, issueResponse.Data.DownloadPath, nil)
	replayContext.Params = gin.Params{{Key: "platform", Value: "macos"}}
	h.Download(replayContext)

	require.Equal(t, http.StatusUnauthorized, replayRecorder.Code)
	require.Contains(t, replayRecorder.Body.String(), "INVALID_QUOTA_VIEWER_INSTALLER_DOWNLOAD_CODE")
}

func TestQuotaViewerInstallerIssueRequiresAuthenticatedSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/quota/releases/windows/latest/download", nil)
	c.Params = gin.Params{{Key: "platform", Value: "windows"}}

	(&QuotaViewerInstallerHandler{}).IssueDownload(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, "no-referrer", recorder.Header().Get("Referrer-Policy"))
}
