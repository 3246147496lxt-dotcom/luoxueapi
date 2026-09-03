package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const officeFileChatRequest = `{
	"model":"test-model",
	"messages":[{"role":"user","content":[{
		"type":"file",
		"file":{
			"filename":"report.docx",
			"file_data":"data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,UEs="
		}
	}]}]
}`

const officeFileResponsesRequest = `{
	"model":"test-model",
	"input":[{"role":"user","content":[{
		"type":"input_file",
		"filename":"report.docx",
		"file_data":"data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,UEs="
	}]}]
}`

func providerFileTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	return ctx, recorder
}

func TestGatewayForwardAsChatCompletions_UnsupportedFileReturnsStable400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := providerFileTestContext()

	result, err := (&GatewayService{}).ForwardAsChatCompletions(
		context.Background(), ctx, &Account{}, []byte(officeFileChatRequest), nil,
	)
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), apicompat.ProviderFileUnsupportedCode)
	require.Contains(t, recorder.Body.String(), "report.docx")
}

func TestGeminiForwardAsChatCompletions_UnsupportedFileReturnsStable400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := providerFileTestContext()

	result, err := (&GeminiMessagesCompatService{}).ForwardAsChatCompletions(
		context.Background(), ctx, &Account{}, []byte(officeFileChatRequest),
	)
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), apicompat.ProviderFileUnsupportedCode)
	require.Contains(t, recorder.Body.String(), "report.docx")
}

func TestGatewayForwardAsResponses_UnsupportedFileReturnsStable400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, recorder := providerFileTestContext()

	result, err := (&GatewayService{}).ForwardAsResponses(
		context.Background(), ctx, &Account{}, []byte(officeFileResponsesRequest), nil,
	)
	require.Nil(t, result)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), apicompat.ProviderFileUnsupportedCode)
	require.Contains(t, recorder.Body.String(), "report.docx")
}
