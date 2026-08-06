package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSkillRepositoryStarsWorkerIsProductionWired(t *testing.T) {
	read := func(path string) string {
		source, err := os.ReadFile(path)
		require.NoError(t, err, path)
		return string(source)
	}

	generatedWire := read(filepath.Join("..", "..", "cmd", "server", "wire_gen.go"))
	runtimeSupervisor := read(filepath.Join("..", "..", "cmd", "server", "runtime_supervisor.go"))
	require.Contains(t, generatedWire, "service.NewSkillMarketService(skillMarketRepository, settingService)")
	require.Contains(t, generatedWire, "buildApplicationSupervisor(pricingService, settingService, skillMarketService,")
	require.Contains(t, runtimeSupervisor, `ComponentName: "skill-market-github-stars"`)
	require.Contains(t, runtimeSupervisor, "StartFunc:     skillMarket.Start")
	require.Contains(t, runtimeSupervisor, "StopFunc:      skillMarket.Stop")
}
