package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type projectRequest struct {
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Color        string `json:"color"`
	Instructions string `json:"instructions"`
	MemoryMode   string `json:"memory_mode"`
}

// projectPatchRequest keeps omitted fields distinct from explicit empty
// values. In particular, an empty instructions string is how a user clears
// project instructions in the settings dialog.
type projectPatchRequest struct {
	Name         *string `json:"name"`
	Icon         *string `json:"icon"`
	Color        *string `json:"color"`
	Instructions *string `json:"instructions"`
	MemoryMode   *string `json:"memory_mode"`
}
type projectFileRequest struct {
	FileID string `json:"file_id"`
}
type moveConversationRequest struct {
	ProjectID *string `json:"project_id"`
}

func (h *ChatHandler) projectUser(c *gin.Context) (int64, bool) {
	c.Header("Cache-Control", "private, no-store")
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	if h == nil || h.projects == nil {
		response.InternalError(c, "Project service is unavailable")
		return 0, false
	}
	return subject.UserID, true
}
func decodeProjectJSON(c *gin.Context, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
func (h *ChatHandler) ListProjects(c *gin.Context) {
	uid, ok := h.projectUser(c)
	if !ok {
		return
	}
	v, e := h.projects.List(c.Request.Context(), uid)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"projects": v})
}
func (h *ChatHandler) CreateProject(c *gin.Context) {
	uid, ok := h.projectUser(c)
	if !ok {
		return
	}
	var req projectRequest
	if e := decodeProjectJSON(c, &req); e != nil {
		response.ErrorFrom(c, service.ErrProjectInvalid)
		return
	}
	p, e := h.projects.Create(c.Request.Context(), uid, service.ProjectInput{Name: req.Name, Icon: req.Icon, Color: req.Color, Instructions: req.Instructions, MemoryMode: req.MemoryMode})
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Created(c, p)
}
func (h *ChatHandler) GetProject(c *gin.Context) {
	uid, ok := h.projectUser(c)
	if !ok {
		return
	}
	p, e := h.projects.Get(c.Request.Context(), uid, c.Param("project_id"))
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, p)
}
func (h *ChatHandler) UpdateProject(c *gin.Context) {
	uid, ok := h.projectUser(c)
	if !ok {
		return
	}
	var req projectPatchRequest
	if e := decodeProjectJSON(c, &req); e != nil {
		response.ErrorFrom(c, service.ErrProjectInvalid)
		return
	}
	existing, e := h.projects.Get(c.Request.Context(), uid, c.Param("project_id"))
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	input := service.ProjectInput{
		Name: existing.Name, Icon: existing.Icon, Color: existing.Color,
		Instructions: existing.Instructions, MemoryMode: existing.MemoryMode,
	}
	if req.Name != nil {
		input.Name = *req.Name
	}
	if req.Icon != nil {
		input.Icon = *req.Icon
	}
	if req.Color != nil {
		input.Color = *req.Color
	}
	if req.Instructions != nil {
		input.Instructions = *req.Instructions
	}
	if req.MemoryMode != nil {
		input.MemoryMode = *req.MemoryMode
	}
	p, e := h.projects.Update(c.Request.Context(), uid, c.Param("project_id"), input)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, p)
}
func (h *ChatHandler) DeleteProject(c *gin.Context) {
	uid, ok := h.projectUser(c)
	if !ok {
		return
	}
	if e := h.projects.Delete(c.Request.Context(), uid, c.Param("project_id")); e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
func (h *ChatHandler) AddProjectFile(c *gin.Context) {
	uid, ok := h.projectUser(c)
	if !ok {
		return
	}
	var req projectFileRequest
	if e := decodeProjectJSON(c, &req); e != nil {
		response.ErrorFrom(c, service.ErrProjectInvalid)
		return
	}
	f, e := h.projects.AddFile(c.Request.Context(), uid, c.Param("project_id"), req.FileID)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Created(c, f)
}
func (h *ChatHandler) RemoveProjectFile(c *gin.Context) {
	uid, ok := h.projectUser(c)
	if !ok {
		return
	}
	if e := h.projects.RemoveFile(c.Request.Context(), uid, c.Param("project_id"), c.Param("file_id")); e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"removed": true})
}
func (h *ChatHandler) MoveConversationToProject(c *gin.Context) {
	uid, ok := h.projectUser(c)
	if !ok {
		return
	}
	var req moveConversationRequest
	if e := decodeProjectJSON(c, &req); e != nil {
		response.ErrorFrom(c, service.ErrProjectInvalid)
		return
	}
	if e := h.projects.MoveConversation(c.Request.Context(), uid, c.Param("conversation_id"), req.ProjectID); e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"conversation_id": c.Param("conversation_id"), "project_id": req.ProjectID})
}
