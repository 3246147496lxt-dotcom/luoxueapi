package admin

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modules/skillimport/application"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	domain "github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SkillImportHandler struct {
	service *application.Service
}

type skillImportUploadRequestConfig struct {
	Selection      json.RawMessage `json:"selection"`
	RunConfig      json.RawMessage `json:"run_config"`
	PublishPolicy  string          `json:"publish_policy"`
	MetadataPolicy string          `json:"metadata_policy"`
}

func NewSkillImportHandler(importService *application.Service) *SkillImportHandler {
	return &SkillImportHandler{service: importService}
}

func (h *SkillImportHandler) Adapters(c *gin.Context) {
	response.Success(c, gin.H{"items": h.service.AdapterDescriptors()})
}

func (h *SkillImportHandler) ListSources(c *gin.Context) {
	result, err := h.service.ListSources(c.Request.Context(), skillImportListFilter(c))
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (h *SkillImportHandler) GetSource(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.GetSource(c.Request.Context(), id)
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (h *SkillImportHandler) CreateSource(c *gin.Context) {
	var input application.SourceInput
	if !decodeSkillJSON(c, &input) {
		return
	}
	middleware2.SetAuditAction(c, "admin.skill_import.sources.create")
	executeAdminIdempotentJSON(c, "admin.skill_import.sources.create", input, domain.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.service.CreateSource(ctx, input, skillMarketActorID(c))
	})
}

func (h *SkillImportHandler) UpdateSource(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	var input application.SourceInput
	if !decodeSkillJSON(c, &input) {
		return
	}
	middleware2.SetAuditAction(c, "admin.skill_import.sources.update")
	result, err := h.service.UpdateSource(c.Request.Context(), id, input, skillMarketActorID(c))
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (h *SkillImportHandler) ListSchedules(c *gin.Context) {
	result, err := h.service.ListSchedules(c.Request.Context(), skillImportListFilter(c))
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (h *SkillImportHandler) GetSchedule(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.GetSchedule(c.Request.Context(), id)
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (h *SkillImportHandler) CreateSchedule(c *gin.Context) {
	var input application.ScheduleInput
	if !decodeSkillJSON(c, &input) {
		return
	}
	middleware2.SetAuditAction(c, "admin.skill_import.schedules.create")
	executeAdminIdempotentJSON(c, "admin.skill_import.schedules.create", input, domain.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.service.CreateSchedule(ctx, input, skillMarketActorID(c))
	})
}

func (h *SkillImportHandler) UpdateSchedule(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	var input application.ScheduleInput
	if !decodeSkillJSON(c, &input) {
		return
	}
	middleware2.SetAuditAction(c, "admin.skill_import.schedules.update")
	result, err := h.service.UpdateSchedule(c.Request.Context(), id, input, skillMarketActorID(c))
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (h *SkillImportHandler) RunSchedule(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	middleware2.SetAuditAction(c, "admin.skill_import.schedules.run")
	payload := gin.H{"schedule_id": id}
	executeAdminIdempotentJSON(c, "admin.skill_import.schedules.run", payload, domain.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.service.RunSchedule(ctx, id, skillMarketActorID(c), skillImportIdempotencyKey(c))
	})
}

func (h *SkillImportHandler) ListRuns(c *gin.Context) {
	result, err := h.service.ListRuns(c.Request.Context(), skillImportListFilter(c))
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (h *SkillImportHandler) CreateRun(c *gin.Context) {
	var input application.RunInput
	if !decodeSkillJSON(c, &input) {
		return
	}
	middleware2.SetAuditAction(c, "admin.skill_import.runs.create")
	executeAdminIdempotentJSON(c, "admin.skill_import.runs.create", input, domain.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.service.CreateRun(ctx, input, skillMarketActorID(c), skillImportIdempotencyKey(c))
	})
}

func (h *SkillImportHandler) UploadRun(c *gin.Context) {
	// Multipart framing plus the bounded JSON policy fields needs little room,
	// but one MiB lets a client send ordinary filenames and headers without
	// weakening the five-MiB artifact ceiling.
	const multipartOverhead = int64(1 << 20)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, domain.SkillArchiveMaxBytes+multipartOverhead)
	sourceID, err := strconv.ParseInt(strings.TrimSpace(c.PostForm("source_id")), 10, 64)
	if err != nil || sourceID <= 0 {
		response.BadRequest(c, "source_id is required")
		return
	}
	source, err := h.service.GetSource(c.Request.Context(), sourceID)
	if response.ErrorFrom(c, err) {
		return
	}
	if source.Adapter != "manifest" {
		response.BadRequest(c, "uploaded manifests require a manifest source")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil || fileHeader.Size <= 0 || fileHeader.Size > domain.SkillArchiveMaxBytes {
		response.BadRequest(c, "A JSON, CSV, or ZIP manifest up to 5 MiB is required")
		return
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileHeader.Filename)), ".")
	if format != "json" && format != "csv" && format != "zip" {
		response.BadRequest(c, "Manifest file must use .json, .csv, or .zip")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.BadRequest(c, "Cannot read manifest file")
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, domain.SkillArchiveMaxBytes+1))
	if err != nil || len(data) == 0 || int64(len(data)) > domain.SkillArchiveMaxBytes {
		response.BadRequest(c, "Cannot safely read manifest file")
		return
	}
	var requestConfig skillImportUploadRequestConfig
	if raw := strings.TrimSpace(c.PostForm("request_config")); raw != "" {
		if err := decodeSkillImportUploadRequestConfig(raw, &requestConfig); err != nil {
			response.BadRequest(c, "request_config must be a JSON object")
			return
		}
	}
	if raw := strings.TrimSpace(c.PostForm("selection")); raw != "" {
		requestConfig.Selection = json.RawMessage(raw)
	}
	if raw := strings.TrimSpace(c.PostForm("run_config")); raw != "" {
		requestConfig.RunConfig = json.RawMessage(raw)
	}
	if raw := strings.TrimSpace(c.PostForm("publish_policy")); raw != "" {
		requestConfig.PublishPolicy = raw
	}
	if raw := strings.TrimSpace(c.PostForm("metadata_policy")); raw != "" {
		requestConfig.MetadataPolicy = raw
	}
	adapterConfig, _ := json.Marshal(map[string]any{
		"inline_data_base64": base64.StdEncoding.EncodeToString(data),
		"format":             format, "default_namespace": source.Namespace,
	})
	input := application.RunInput{
		SourceID: sourceID, Mode: strings.TrimSpace(c.PostForm("mode")),
		Selection: requestConfig.Selection, RunConfig: requestConfig.RunConfig,
		PublishPolicy: requestConfig.PublishPolicy, MetadataPolicy: requestConfig.MetadataPolicy,
		AdapterConfigOverride: adapterConfig,
	}
	digest := sha256.Sum256(data)
	payload := gin.H{
		"source_id": sourceID, "mode": input.Mode, "format": format,
		"byte_size": len(data), "sha256": hex.EncodeToString(digest[:]),
		"selection": requestConfig.Selection, "run_config": requestConfig.RunConfig,
		"publish_policy": requestConfig.PublishPolicy, "metadata_policy": requestConfig.MetadataPolicy,
	}
	middleware2.SetAuditAction(c, "admin.skill_import.runs.upload")
	executeAdminIdempotentJSON(c, "admin.skill_import.runs.upload", payload, domain.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.service.CreateRun(ctx, input, skillMarketActorID(c), skillImportIdempotencyKey(c))
	})
}

func (h *SkillImportHandler) GetRun(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.GetRun(c.Request.Context(), id)
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (h *SkillImportHandler) ListRunItems(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.ListRunItems(c.Request.Context(), id, skillImportListFilter(c))
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (h *SkillImportHandler) ListRunEvents(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	result, err := h.service.ListRunEvents(c.Request.Context(), id, skillImportListFilter(c))
	if !response.ErrorFrom(c, err) {
		response.Success(c, result)
	}
}

func (h *SkillImportHandler) CancelRun(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	actorID, ok := requiredSkillImportActorID(c)
	if !ok {
		return
	}
	middleware2.SetAuditAction(c, "admin.skill_import.runs.cancel")
	payload := gin.H{"run_id": id}
	executeAdminIdempotentJSON(c, "admin.skill_import.runs.cancel", payload, domain.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.service.CancelRun(ctx, id, actorID)
	})
}

func (h *SkillImportHandler) RetryFailed(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	var request struct {
		ItemIDs []int64 `json:"item_ids"`
	}
	if !decodeSkillJSON(c, &request) {
		return
	}
	middleware2.SetAuditAction(c, "admin.skill_import.runs.retry_failed")
	payload := gin.H{"run_id": id, "item_ids": request.ItemIDs}
	executeAdminIdempotentJSON(c, "admin.skill_import.runs.retry_failed", payload, domain.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.service.RetryFailedItems(ctx, id, request.ItemIDs)
	})
}

func (h *SkillImportHandler) PublishRun(c *gin.Context) {
	id, ok := parseSkillImportID(c, "id")
	if !ok {
		return
	}
	middleware2.SetAuditAction(c, "admin.skill_import.runs.publish")
	var request struct {
		ItemIDs []int64 `json:"item_ids"`
	}
	if c.Request.ContentLength != 0 && !decodeSkillJSON(c, &request) {
		return
	}
	payload := gin.H{"run_id": id, "item_ids": request.ItemIDs}
	executeAdminIdempotentJSON(c, "admin.skill_import.runs.publish", payload, domain.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.service.PublishRun(ctx, id, request.ItemIDs, skillMarketActorID(c))
	})
}

func skillImportListFilter(c *gin.Context) domain.SkillImportListFilter {
	page, pageSize := response.ParsePagination(c)
	if pageSize > 100 {
		pageSize = 100
	}
	filter := domain.SkillImportListFilter{
		Search: c.Query("search"), Status: c.Query("status"), Page: page, PageSize: pageSize,
	}
	if raw := strings.TrimSpace(c.Query("source_id")); raw != "" {
		if value, err := strconv.ParseInt(raw, 10, 64); err == nil && value > 0 {
			filter.SourceID = &value
		}
	}
	return filter
}

func decodeSkillImportUploadRequestConfig(raw string, target *skillImportUploadRequestConfig) error {
	trimmed := strings.TrimSpace(raw)
	if target == nil || len(trimmed) < 2 || trimmed[0] != '{' {
		return io.ErrUnexpectedEOF
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func parseSkillImportID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid skill import ID")
		return 0, false
	}
	return id, true
}

func requiredSkillImportActorID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return 0, false
	}
	return subject.UserID, true
}

func skillImportIdempotencyKey(c *gin.Context) string {
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		key = strings.TrimSpace(c.GetHeader("X-Idempotency-Key"))
	}
	return key
}
