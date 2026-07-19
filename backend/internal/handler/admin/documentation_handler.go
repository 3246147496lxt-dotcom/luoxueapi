package admin

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/image/webp"
)

type DocumentationHandler struct {
	service *service.DocumentationService
}

func NewDocumentationHandler(documentationService *service.DocumentationService) *DocumentationHandler {
	return &DocumentationHandler{service: documentationService}
}

func (h *DocumentationHandler) Get(c *gin.Context) {
	state, err := h.service.GetAdminState(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, state)
}

func (h *DocumentationHandler) SaveDraft(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.DocumentationMaxJSONBytes+4096)
	var request struct {
		Content json.RawMessage `json:"content"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := ensureJSONEOF(decoder); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	state, err := h.service.SaveDraft(c.Request.Context(), request.Content, documentationActorID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, state)
}

func (h *DocumentationHandler) Publish(c *gin.Context) {
	state, err := h.service.Publish(c.Request.Context(), documentationActorID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, state)
}

func (h *DocumentationHandler) ListRevisions(c *gin.Context) {
	items, err := h.service.ListRevisions(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": len(items)})
}

func (h *DocumentationHandler) RestoreRevision(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid documentation revision ID")
		return
	}
	state, err := h.service.RestoreRevision(c.Request.Context(), id, documentationActorID(c))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, state)
}

func (h *DocumentationHandler) UploadAsset(c *gin.Context) {
	const multipartOverhead = 1 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.DocumentationMaxAssetBytes+multipartOverhead)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "A documentation image is required")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > service.DocumentationMaxAssetBytes {
		response.ErrorFrom(c, service.ErrDocumentationAsset)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.BadRequest(c, "Cannot read documentation image")
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, service.DocumentationMaxAssetBytes+1))
	if err != nil || len(data) == 0 || len(data) > service.DocumentationMaxAssetBytes {
		response.ErrorFrom(c, service.ErrDocumentationAsset)
		return
	}
	contentType := strings.TrimSpace(strings.Split(http.DetectContentType(data), ";")[0])
	config, err := decodeDocumentationImageConfig(contentType, data)
	if err != nil {
		response.ErrorFrom(c, service.ErrDocumentationAsset)
		return
	}
	digest := sha256.Sum256(data)
	asset := &service.DocumentationAsset{
		ID:          hex.EncodeToString(digest[:]),
		ContentType: contentType,
		ByteSize:    int64(len(data)),
		Width:       config.Width,
		Height:      config.Height,
		CreatedBy:   documentationActorID(c),
		Data:        data,
	}
	if err := h.service.SaveAsset(c.Request.Context(), asset); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, gin.H{
		"id":           asset.ID,
		"url":          asset.PublicURL(),
		"content_type": asset.ContentType,
		"byte_size":    asset.ByteSize,
		"width":        asset.Width,
		"height":       asset.Height,
		"sha256":       asset.ID,
	})
}

func decodeDocumentationImageConfig(contentType string, data []byte) (image.Config, error) {
	reader := bytes.NewReader(data)
	switch contentType {
	case "image/png":
		return png.DecodeConfig(reader)
	case "image/jpeg":
		return jpeg.DecodeConfig(reader)
	case "image/gif":
		return gif.DecodeConfig(reader)
	case "image/webp":
		return webp.DecodeConfig(reader)
	default:
		return image.Config{}, service.ErrDocumentationAsset
	}
}

func documentationActorID(c *gin.Context) *int64 {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		return nil
	}
	id := subject.UserID
	return &id
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return service.ErrDocumentationInvalid
		}
		return err
	}
	return nil
}
