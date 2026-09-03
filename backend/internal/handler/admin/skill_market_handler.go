package admin

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SkillMarketHandler struct{ service *service.SkillMarketService }

func NewSkillMarketHandler(skillService *service.SkillMarketService) *SkillMarketHandler {
	return &SkillMarketHandler{service: skillService}
}

func (h *SkillMarketHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > 100 {
		pageSize = 100
	}
	filter := service.SkillListFilter{
		Search: c.Query("search"), Status: c.Query("status"),
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
	result, err := h.service.ListAdmin(c.Request.Context(), filter)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *SkillMarketHandler) Get(c *gin.Context) {
	id, ok := parseSkillMarketID(c, "id")
	if !ok {
		return
	}
	item, err := h.service.GetAdmin(c.Request.Context(), id)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}

func (h *SkillMarketHandler) Create(c *gin.Context) {
	middleware2.SetAuditAction(c, "admin.skills.create")
	var input service.SkillInput
	if !decodeSkillJSON(c, &input) {
		return
	}
	item, err := h.service.Create(c.Request.Context(), input, skillMarketActorID(c))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, item)
}

func (h *SkillMarketHandler) Update(c *gin.Context) {
	id, ok := parseSkillMarketID(c, "id")
	if !ok {
		return
	}
	middleware2.SetAuditAction(c, "admin.skills.update")
	var input service.SkillInput
	if !decodeSkillJSON(c, &input) {
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, input, skillMarketActorID(c))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}

func (h *SkillMarketHandler) UploadVersion(c *gin.Context) {
	id, ok := parseSkillMarketID(c, "id")
	if !ok {
		return
	}
	middleware2.SetAuditAction(c, "admin.skills.versions.upload")
	const multipartOverhead = int64(1 << 20)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.SkillArchiveMaxBytes+multipartOverhead)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "A skill ZIP file is required")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > service.SkillArchiveMaxBytes {
		writeSkillArchiveIssue(c, "ARCHIVE_SIZE", "ZIP must be between 1 byte and 5 MiB")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		response.BadRequest(c, "Cannot read skill ZIP file")
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, service.SkillArchiveMaxBytes+1))
	if err != nil || len(data) == 0 || int64(len(data)) > service.SkillArchiveMaxBytes {
		writeSkillArchiveIssue(c, "ARCHIVE_SIZE", "ZIP must be between 1 byte and 5 MiB")
		return
	}
	item, err := h.service.UploadVersion(
		c.Request.Context(), id, c.PostForm("version"), c.PostForm("changelog"),
		data, skillMarketActorID(c),
	)
	if err != nil {
		var validationErr *service.SkillArchiveValidationError
		if errors.As(err, &validationErr) {
			writeSkillArchiveValidationError(c, err, &validationErr.Report)
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *SkillMarketHandler) Publish(c *gin.Context) {
	skillID, ok := parseSkillMarketID(c, "id")
	if !ok {
		return
	}
	var request struct {
		VersionID int64 `json:"version_id"`
	}
	if !decodeSkillJSON(c, &request) || request.VersionID <= 0 {
		if !c.IsAborted() && !c.Writer.Written() {
			response.BadRequest(c, "version_id is required")
		}
		return
	}
	middleware2.SetAuditAction(c, "admin.skills.publish")
	item, err := h.service.Publish(c.Request.Context(), skillID, request.VersionID, skillMarketActorID(c))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}

func (h *SkillMarketHandler) Activate(c *gin.Context) {
	skillID, versionID, ok := parseSkillAndVersionIDs(c)
	if !ok {
		return
	}
	middleware2.SetAuditAction(c, "admin.skills.versions.activate")
	item, err := h.service.Activate(c.Request.Context(), skillID, versionID, skillMarketActorID(c))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}

func (h *SkillMarketHandler) Yank(c *gin.Context) {
	skillID, versionID, ok := parseSkillAndVersionIDs(c)
	if !ok {
		return
	}
	middleware2.SetAuditAction(c, "admin.skills.versions.yank")
	item, err := h.service.Yank(c.Request.Context(), skillID, versionID, skillMarketActorID(c))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}

func (h *SkillMarketHandler) Archive(c *gin.Context) {
	skillID, ok := parseSkillMarketID(c, "id")
	if !ok {
		return
	}
	middleware2.SetAuditAction(c, "admin.skills.archive")
	item, err := h.service.Archive(c.Request.Context(), skillID, skillMarketActorID(c))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, item)
}

func (h *SkillMarketHandler) GetConfig(c *gin.Context) {
	config, err := h.service.GetConfig(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, config)
}

func (h *SkillMarketHandler) UpdateConfig(c *gin.Context) {
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if !decodeSkillJSON(c, &request) {
		return
	}
	if request.Enabled == nil {
		response.BadRequest(c, "enabled is required")
		return
	}
	middleware2.SetAuditAction(c, "admin.skills.config.update")
	config, err := h.service.UpdateConfig(c.Request.Context(), *request.Enabled)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, config)
}

func parseSkillMarketID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid skill ID")
		return 0, false
	}
	return id, true
}

func parseSkillAndVersionIDs(c *gin.Context) (int64, int64, bool) {
	skillID, ok := parseSkillMarketID(c, "id")
	if !ok {
		return 0, 0, false
	}
	versionID, err := strconv.ParseInt(strings.TrimSpace(c.Param("versionId")), 10, 64)
	if err != nil || versionID <= 0 {
		response.BadRequest(c, "Invalid skill version ID")
		return 0, 0, false
	}
	return skillID, versionID, true
}

func decodeSkillJSON(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		response.BadRequest(c, "Invalid request body")
		return false
	}
	return true
}

func skillMarketActorID(c *gin.Context) *int64 {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		return nil
	}
	id := subject.UserID
	return &id
}

func writeSkillArchiveValidationError(c *gin.Context, err error, report *service.SkillValidationReport) {
	status := infraerrors.FromError(err)
	data := gin.H{}
	if report != nil {
		data["validation_report"] = report
	}
	c.JSON(int(status.Code), response.Response{
		Code: int(status.Code), Message: status.Message, Reason: status.Reason,
		Metadata: status.Metadata, Data: data,
	})
}

func writeSkillArchiveIssue(c *gin.Context, code, message string) {
	report := service.SkillValidationReport{
		Valid:    false,
		Errors:   []service.SkillValidationIssue{{Code: code, Message: message}},
		Warnings: []service.SkillValidationIssue{},
	}
	encoded, _ := json.Marshal(report)
	err := service.ErrSkillArchiveInvalid.WithMetadata(map[string]string{
		"validation_code":   code,
		"validation_report": string(encoded),
	})
	writeSkillArchiveValidationError(c, err, &report)
}
