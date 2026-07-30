package handler

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/modules/quotaauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type quotaOverviewUsecase interface {
	GetOverview(ctx context.Context, userID int64, displayTimezone string) (*service.QuotaOverview, error)
}

type QuotaOverviewHandler struct {
	service quotaOverviewUsecase
}

func NewQuotaOverviewHandler(service *service.QuotaOverviewService) *QuotaOverviewHandler {
	return &QuotaOverviewHandler{service: service}
}

// GetOverview serves only the independently authenticated quota-viewer
// subject. The route must additionally be protected by RequireQuotaRead.
func (h *QuotaOverviewHandler) GetOverview(c *gin.Context) {
	setQuotaOverviewHeaders(c)
	subject, ok := middleware.GetQuotaAuthSubjectFromContext(c)
	if !ok {
		response.ErrorFrom(c, quotaauth.ErrAuthRequired)
		return
	}
	timezoneName, err := parseQuotaOverviewTimezone(c)
	if err != nil {
		response.ErrorFrom(c, quotaauth.ErrInvalidAuthorizationRequest.WithCause(err))
		return
	}
	if h == nil || h.service == nil {
		response.ErrorFrom(c, service.ErrQuotaOverviewUnavailable)
		return
	}
	overview, err := h.service.GetOverview(c.Request.Context(), subject.UserID, timezoneName)
	if response.ErrorFrom(c, err) {
		return
	}
	if overview == nil {
		response.ErrorFrom(c, service.ErrQuotaOverviewUnavailable)
		return
	}
	requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string)
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	overview.RequestID = requestID
	response.Success(c, dto.QuotaOverviewFromService(overview))
}

func parseQuotaOverviewTimezone(c *gin.Context) (string, error) {
	values, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil {
		return "", err
	}
	for name := range values {
		if name != "timezone" {
			return "", &quotaOverviewQueryError{parameter: name}
		}
	}
	timezones, exists := values["timezone"]
	if !exists {
		return "UTC", nil
	}
	if len(timezones) != 1 || strings.TrimSpace(timezones[0]) == "" {
		return "", &quotaOverviewQueryError{parameter: "timezone"}
	}
	timezoneName := strings.TrimSpace(timezones[0])
	if _, err := time.LoadLocation(timezoneName); err != nil {
		return "", err
	}
	return timezoneName, nil
}

type quotaOverviewQueryError struct {
	parameter string
}

func (e *quotaOverviewQueryError) Error() string {
	return "unsupported or invalid query parameter: " + e.parameter
}

func setQuotaOverviewHeaders(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Vary", "Authorization")
}
