package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchedulerShadowComparisonIsProductionWired(t *testing.T) {
	read := func(path string) string {
		source, err := os.ReadFile(path)
		require.NoError(t, err, path)
		return string(source)
	}

	serviceWire := read("wire.go")
	require.Contains(t, serviceWire, "ProvideSchedulerModuleFacade")
	require.Contains(t, serviceWire, "ProvideGatewayScheduler")
	require.Contains(t, serviceWire, "ProvideAccountSnapshotReader")

	generatedWire := read(filepath.Join("..", "..", "cmd", "server", "wire_gen.go"))
	runtimeSupervisor := read(filepath.Join("..", "..", "cmd", "server", "runtime_supervisor.go"))
	require.Contains(t, generatedWire, "service.ProvideSchedulerModuleFacade(schedulerSnapshotService, accountRepository, cfg)")
	require.Contains(t, generatedWire, "service.ProvideGatewayScheduler(schedulerSnapshotService, applicationFacade)")
	require.Contains(t, generatedWire, "service.ProvideAccountSnapshotReader(gatewayScheduler)")
	require.Contains(t, generatedWire, "service.NewGatewayService(accountRepository, groupRepository, usageLogRepository, usageBillingRepository, userRepository, userSubscriptionRepository, userGroupRateRepository, gatewayCache, cfg, gatewayScheduler,")
	require.Contains(t, generatedWire, "service.NewOpenAIGatewayService(accountRepository, usageLogRepository, usageBillingRepository, userRepository, userSubscriptionRepository, userGroupRateRepository, gatewayCache, cfg, gatewayScheduler,")
	require.Contains(t, generatedWire, "service.NewGeminiMessagesCompatService(accountRepository, groupRepository, gatewayCache, gatewayScheduler,")
	require.Contains(t, generatedWire, "service.NewAntigravityGatewayService(accountRepository, gatewayCache, accountSnapshotReader,")
	require.Contains(t, generatedWire, "applicationFacade,")
	require.Contains(t, runtimeSupervisor, `ComponentName: "scheduler-shadow-comparison"`)
	require.Contains(t, runtimeSupervisor, "return schedulerModule.StartShadow(ctx)")
	require.Contains(t, runtimeSupervisor, "return schedulerModule.StopShadow(ctx)")
}
