package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestReportOpenAIWSAccountScheduleFailureClassification(t *testing.T) {
	cfg := &config.Config{}
	settingRepo := &contentModerationHandlerSettingRepo{values: map[string]string{}}
	settingService := service.NewSettingService(settingRepo, cfg)
	setAdvancedScheduler := func(enabled bool) {
		require.NoError(t, settingService.UpdateSettings(context.Background(), &service.SystemSettings{
			OpenAIAdvancedSchedulerEnabled: enabled,
		}))
	}
	setAdvancedScheduler(true)
	t.Cleanup(func() {
		setAdvancedScheduler(false)
	})

	rateLimitService := service.NewRateLimitService(nil, nil, cfg, nil, nil)
	rateLimitService.SetSettingService(settingService)
	gatewayService := service.NewOpenAIGatewayService(
		nil, nil, nil, nil, nil, nil, nil, cfg, nil, nil, nil,
		rateLimitService, nil, nil, nil, nil, nil, nil, nil, nil,
		settingService, nil,
	)
	h := &OpenAIGatewayHandler{gatewayService: gatewayService}
	c := newTestGinContext()

	h.reportOpenAIWSAccountScheduleFailure(
		c,
		42,
		"gpt-5.5",
		service.NewOpenAIWSClientCloseError(
			coderws.StatusTryAgainLater,
			"account is busy, please retry later",
			errors.New("local concurrency backend unavailable"),
		),
	)
	require.Zero(
		t,
		gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount,
		"local client/concurrency errors must not create account runtime stats",
	)

	service.MarkOpsCyberPolicy(c, service.CyberPolicyMark{Message: "blocked", UpstreamStatus: 403})
	h.reportOpenAIWSAccountScheduleFailure(c, 42, "gpt-5.5", errors.New("upstream read failed"))
	require.Zero(
		t,
		gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount,
		"Cyber attribution guard must remain authoritative",
	)

	service.ClearOpsCyberPolicy(c)
	h.reportOpenAIWSAccountScheduleFailure(c, 42, "gpt-5.5", errors.New("upstream read failed"))
	require.Equal(
		t,
		1,
		gatewayService.SnapshotOpenAIAccountSchedulerMetrics().RuntimeStatsAccountCount,
		"real upstream errors must still update account runtime stats",
	)
}
