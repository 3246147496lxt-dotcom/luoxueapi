package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type DocumentationHandler struct {
	service *service.DocumentationService
}

func NewDocumentationHandler(documentationService *service.DocumentationService) *DocumentationHandler {
	return &DocumentationHandler{service: documentationService}
}

func (h *DocumentationHandler) Get(c *gin.Context) {
	snapshot, etag, err := h.service.PublicSnapshot(c.Request.Context())
	if err != nil {
		if errors.Is(err, service.ErrDocumentationNotFound) {
			c.Header("Cache-Control", "no-store")
		}
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=30, must-revalidate")
	c.Header("ETag", etag)
	if requestETagMatches(c.GetHeader("If-None-Match"), etag) {
		c.Status(http.StatusNotModified)
		return
	}
	response.Success(c, gin.H{
		"content":      snapshot.Content,
		"version":      snapshot.Version,
		"published_at": snapshot.UpdatedAt,
	})
}

func (h *DocumentationHandler) GetAsset(c *gin.Context) {
	asset, etag, err := h.service.GetAsset(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, service.ErrDocumentationNotFound) {
			c.Header("Cache-Control", "no-store")
		}
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("ETag", etag)
	c.Header("X-Content-Type-Options", "nosniff")
	if requestETagMatches(c.GetHeader("If-None-Match"), etag) {
		c.Status(http.StatusNotModified)
		return
	}
	c.Header("Content-Length", strconv.FormatInt(asset.ByteSize, 10))
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, asset.ID))
	c.Data(http.StatusOK, asset.ContentType, asset.Data)
}
