package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *ChatHandler) UploadAttachment(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	forceCloseUntilTranscriptionBodyRead(c)
	userID, ok := chatAttachmentUser(c, h)
	if !ok {
		return
	}
	release, err := h.attachments.AdmitUpload(userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer release()
	filename, declaredMIME, data, err := readSingleChatAttachment(c, h.attachments.MaxUploadBytes())
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || errors.Is(err, service.ErrChatAttachmentTooLarge) {
			response.ErrorWithDetails(c, http.StatusRequestEntityTooLarge, "Chat attachment is too large", "CHAT_ATTACHMENT_TOO_LARGE", nil)
			return
		}
		response.ErrorFrom(c, service.ErrChatAttachmentInvalid.WithCause(err))
		return
	}
	attachment, err := h.attachments.UploadAdmitted(c.Request.Context(), userID, service.ChatAttachmentUpload{
		Filename: filename, DeclaredMIME: declaredMIME, Data: data,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, attachment)
}

func (h *ChatHandler) DeleteAttachment(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	userID, ok := chatAttachmentUser(c, h)
	if !ok {
		return
	}
	if err := h.attachments.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *ChatHandler) AttachmentContent(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	userID, ok := chatAttachmentUser(c, h)
	if !ok {
		return
	}
	content, err := h.attachments.GetImageContent(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Content-Length", strconv.Itoa(len(content.Data)))
	c.Header("Content-Disposition", "inline")
	c.Data(http.StatusOK, content.Attachment.MIMEType, content.Data)
}

func chatAttachmentUser(c *gin.Context, h *ChatHandler) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	if h == nil || h.attachments == nil {
		response.ErrorFrom(c, service.ErrChatAttachmentUnavailable)
		return 0, false
	}
	return subject.UserID, true
}

func readSingleChatAttachment(c *gin.Context, maxBytes int64) (string, string, []byte, error) {
	if c == nil || c.Request == nil || !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		return "", "", nil, service.ErrChatAttachmentInvalid
	}
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return "", "", nil, err
	}
	part, err := reader.NextPart()
	if err != nil {
		return "", "", nil, err
	}
	defer func() { _ = part.Close() }()
	if part.FormName() != "file" || strings.TrimSpace(part.FileName()) == "" {
		return "", "", nil, service.ErrChatAttachmentInvalid
	}
	if maxBytes <= 0 {
		maxBytes = 20 << 20
	}
	data, err := io.ReadAll(io.LimitReader(part, maxBytes+1))
	if err != nil {
		return "", "", nil, err
	}
	if int64(len(data)) > maxBytes {
		return "", "", nil, service.ErrChatAttachmentTooLarge
	}
	if err = part.Close(); err != nil {
		return "", "", nil, err
	}
	next, nextErr := reader.NextPart()
	if nextErr != io.EOF {
		if next != nil {
			_ = next.Close()
		}
		if nextErr != nil {
			return "", "", nil, nextErr
		}
		return "", "", nil, service.ErrChatAttachmentInvalid
	}
	allowConnectionReuseAfterTranscriptionBodyRead(c)
	return part.FileName(), part.Header.Get("Content-Type"), data, nil
}
