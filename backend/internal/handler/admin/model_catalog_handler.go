package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelCatalogHandler struct {
	catalogService *service.ModelCatalogService
}

func NewModelCatalogHandler(catalogService *service.ModelCatalogService) *ModelCatalogHandler {
	return &ModelCatalogHandler{catalogService: catalogService}
}

func (h *ModelCatalogHandler) Candidates(c *gin.Context) {
	items, err := h.catalogService.ListCandidates(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": len(items)})
}

func (h *ModelCatalogHandler) List(c *gin.Context) {
	items, err := h.catalogService.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	if status != "" && status != service.ModelCatalogStatusDraft && status != service.ModelCatalogStatusPublished && status != service.ModelCatalogStatusArchived {
		response.BadRequest(c, "Invalid model catalog status")
		return
	}
	items = filterModelCatalogAdminItems(items, status, c.Query("search"))
	response.Success(c, gin.H{"items": items, "total": len(items)})
}

func filterModelCatalogAdminItems(items []service.ModelCatalogAdminItem, status, search string) []service.ModelCatalogAdminItem {
	status = strings.ToLower(strings.TrimSpace(status))
	search = strings.ToLower(strings.TrimSpace(search))
	filtered := make([]service.ModelCatalogAdminItem, 0, len(items))
	for i := range items {
		item := items[i]
		if status != "" && item.Status != status {
			continue
		}
		if search != "" {
			haystack := strings.ToLower(strings.Join([]string{
				item.Model, item.DisplayNameZH, item.DisplayNameEN, item.Provider, item.Category, item.Platform,
			}, "\x00"))
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func (h *ModelCatalogHandler) Get(c *gin.Context) {
	id, ok := parseModelCatalogID(c)
	if !ok {
		return
	}
	item, err := h.catalogService.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *ModelCatalogHandler) Create(c *gin.Context) {
	var input service.ModelCatalogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.catalogService.Create(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, item)
}

func (h *ModelCatalogHandler) Update(c *gin.Context) {
	id, ok := parseModelCatalogID(c)
	if !ok {
		return
	}
	var input service.ModelCatalogInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.catalogService.Update(c.Request.Context(), id, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *ModelCatalogHandler) Publish(c *gin.Context) {
	id, ok := parseModelCatalogID(c)
	if !ok {
		return
	}
	item, err := h.catalogService.Publish(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *ModelCatalogHandler) Unpublish(c *gin.Context) {
	id, ok := parseModelCatalogID(c)
	if !ok {
		return
	}
	item, err := h.catalogService.Unpublish(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func parseModelCatalogID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid model catalog ID")
		return 0, false
	}
	return id, true
}
