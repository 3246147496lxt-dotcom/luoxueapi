package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SkillMarketHandler struct{ service *service.SkillMarketService }

const skillMarketGuardPassedKey = "public_skill_market_guard_passed"

func NewSkillMarketHandler(skillService *service.SkillMarketService) *SkillMarketHandler {
	return &SkillMarketHandler{service: skillService}
}

func (h *SkillMarketHandler) RequireEnabled(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if h.service == nil || !h.service.IsPublicEnabled(c.Request.Context()) {
		c.Header("Cache-Control", "no-store")
		response.NotFound(c, "Not found")
		c.Abort()
		return
	}
	c.Set(skillMarketGuardPassedKey, true)
	c.Next()
}

func (h *SkillMarketHandler) ensureEnabled(c *gin.Context) bool {
	c.Header("Cache-Control", "private, no-store")
	if passed, _ := c.Get(skillMarketGuardPassedKey); passed == true {
		return true
	}
	if h.service == nil || !h.service.IsPublicEnabled(c.Request.Context()) {
		c.Header("Cache-Control", "no-store")
		response.NotFound(c, "Not found")
		return false
	}
	return true
}

func (h *SkillMarketHandler) List(c *gin.Context) {
	if !h.ensureEnabled(c) {
		return
	}
	page, pageSize := response.ParsePagination(c)
	if pageSize > 100 {
		pageSize = 100
	}
	filter := service.SkillListFilter{
		Search: c.Query("search"), Category: c.Query("category"),
		Page: page, PageSize: pageSize,
	}
	if raw := strings.TrimSpace(c.Query("featured")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			response.BadRequest(c, "Invalid featured filter")
			return
		}
		filter.Featured = &value
	}
	result, err := h.service.ListPublic(c.Request.Context(), filter)
	if response.ErrorFrom(c, err) {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	response.Success(c, result)
}

func (h *SkillMarketHandler) Get(c *gin.Context) {
	if !h.ensureEnabled(c) {
		return
	}
	item, err := h.service.GetPublic(c.Request.Context(), c.Param("slug"))
	if response.ErrorFrom(c, err) {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	response.Success(c, item)
}

func (h *SkillMarketHandler) Versions(c *gin.Context) {
	if !h.ensureEnabled(c) {
		return
	}
	items, err := h.service.ListPublicVersions(c.Request.Context(), c.Param("slug"))
	if response.ErrorFrom(c, err) {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	response.Success(c, gin.H{"items": items, "total": len(items)})
}

func (h *SkillMarketHandler) DownloadVersion(c *gin.Context) {
	if !h.ensureEnabled(c) {
		return
	}
	artifact, err := h.service.DownloadVersion(c.Request.Context(), c.Param("slug"), c.Param("version"))
	if response.ErrorFrom(c, err) {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("ETag", fmt.Sprintf(`"%s"`, artifact.SHA256))
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.zip"`, artifact.Slug, artifact.Version))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Length", strconv.Itoa(len(artifact.Data)))
	c.Status(http.StatusOK)
	written, writeErr := c.Writer.Write(artifact.Data)
	if writeErr == nil && written == len(artifact.Data) {
		countCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 2*time.Second)
		defer cancel()
		h.service.RecordDownloadBestEffort(countCtx, artifact)
	}
}
