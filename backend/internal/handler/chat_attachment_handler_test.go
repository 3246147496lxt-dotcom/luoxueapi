package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestReadSingleChatAttachmentAcceptsExactlyOneFile(t *testing.T) {
	ctx := attachmentMultipartContext(t, false)
	name, mimeType, data, err := readSingleChatAttachment(ctx, 1024)
	require.NoError(t, err)
	require.Equal(t, "photo.png", name)
	require.Equal(t, "image/png", mimeType)
	require.Equal(t, []byte("png-bytes"), data)
}

func TestReadSingleChatAttachmentRejectsAdditionalParts(t *testing.T) {
	ctx := attachmentMultipartContext(t, true)
	_, _, _, err := readSingleChatAttachment(ctx, 1024)
	require.Error(t, err)
}

func attachmentMultipartContext(t *testing.T, extra bool) *gin.Context {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="photo.png"`)
	header.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte("png-bytes"))
	require.NoError(t, err)
	if extra {
		require.NoError(t, writer.WriteField("extra", "forbidden"))
	}
	require.NoError(t, writer.Close())
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/chat/attachments", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c
}
