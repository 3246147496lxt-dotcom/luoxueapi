package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type opsAccountPoolRepoStub struct {
	service.AccountRepository
	accounts       []service.Account
	platformFilter string
	groupIDFilter  *int64
}

func (r *opsAccountPoolRepoStub) ListOpsAccountsForStats(_ context.Context, platformFilter string, groupIDFilter *int64) ([]service.Account, error) {
	r.platformFilter = platformFilter
	if groupIDFilter != nil {
		value := *groupIDFilter
		r.groupIDFilter = &value
	} else {
		r.groupIDFilter = nil
	}
	return append([]service.Account(nil), r.accounts...), nil
}

type opsAccountPoolSettingRepoStub struct {
	service.SettingRepository
	value string
}

func (r *opsAccountPoolSettingRepoStub) GetValue(context.Context, string) (string, error) {
	return r.value, nil
}

func newOpsAccountPoolHandler(accountRepo service.AccountRepository, settingRepo service.SettingRepository) *OpsHandler {
	serviceInstance := service.NewOpsService(nil, settingRepo, nil, accountRepo, nil, nil, nil, nil, nil, nil, nil)
	return NewOpsHandler(serviceInstance)
}

func performOpsAccountPoolRequest(handler *OpsHandler, target string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, target, nil)
	handler.GetAccountPool(ctx)
	return recorder
}

func TestOpsHandlerGetAccountPool_ForwardsScopeAndClampsLimit(t *testing.T) {
	accounts := make([]service.Account, 0, 25)
	for i := 1; i <= 25; i++ {
		accounts = append(accounts, service.Account{
			ID:          int64(i),
			Name:        "broken",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusError,
			Schedulable: true,
		})
	}
	repo := &opsAccountPoolRepoStub{accounts: accounts}
	handler := newOpsAccountPoolHandler(repo, nil)

	recorder := performOpsAccountPoolRequest(handler, "/api/v1/admin/ops/account-pool?platform=openai&group_id=42&limit=999")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if repo.platformFilter != "openai" || repo.groupIDFilter == nil || *repo.groupIDFilter != 42 {
		t.Fatalf("forwarded scope = platform %q group %v", repo.platformFilter, repo.groupIDFilter)
	}

	var envelope struct {
		Data service.OpsAccountPoolResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Data.Anomalies) != service.MaxOpsAccountPoolAnomalyLimit {
		t.Fatalf("anomaly len = %d, want max %d", len(envelope.Data.Anomalies), service.MaxOpsAccountPoolAnomalyLimit)
	}
}

func TestOpsHandlerGetAccountPool_RejectsInvalidQuery(t *testing.T) {
	handler := newOpsAccountPoolHandler(&opsAccountPoolRepoStub{}, nil)
	for _, target := range []string{
		"/api/v1/admin/ops/account-pool?group_id=bad",
		"/api/v1/admin/ops/account-pool?group_id=0",
		"/api/v1/admin/ops/account-pool?limit=bad",
		"/api/v1/admin/ops/account-pool?limit=0",
	} {
		t.Run(target, func(t *testing.T) {
			recorder := performOpsAccountPoolRequest(handler, target)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestOpsHandlerGetAccountPool_RespectsMonitoringDisabled(t *testing.T) {
	handler := newOpsAccountPoolHandler(
		&opsAccountPoolRepoStub{},
		&opsAccountPoolSettingRepoStub{value: "false"},
	)
	recorder := performOpsAccountPoolRequest(handler, "/api/v1/admin/ops/account-pool")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", recorder.Code, recorder.Body.String())
	}
}
