package main

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/lifecycle"
	"github.com/stretchr/testify/require"
)

func TestProvideServiceBuildInfo(t *testing.T) {
	in := handler.BuildInfo{
		Version:   "v-test",
		BuildType: "release",
	}
	out := provideServiceBuildInfo(in)
	require.Equal(t, in.Version, out.Version)
	require.Equal(t, in.BuildType, out.BuildType)
}

func TestCleanupApplicationInfrastructureWaitsForPendingComponents(t *testing.T) {
	allowStop := false
	supervisor := lifecycle.NewSupervisor(lifecycle.ComponentFuncs{
		ComponentName: "worker",
		StopFunc: func(context.Context) error {
			if !allowStop {
				return errors.New("still draining")
			}
			return nil
		},
	})
	require.NoError(t, supervisor.Start(context.Background()))
	require.Error(t, supervisor.Stop(context.Background()))
	cleanupCalls := 0
	app := &Application{
		Supervisor: supervisor,
		Cleanup:    func() { cleanupCalls++ },
	}

	cleanupApplicationInfrastructure(app)
	require.Zero(t, cleanupCalls)

	allowStop = true
	require.NoError(t, supervisor.Stop(context.Background()))
	cleanupApplicationInfrastructure(app)
	require.Equal(t, 1, cleanupCalls)
}

func TestProvideCleanup_WithMinimalDependencies_NoPanic(t *testing.T) {
	cleanup := provideCleanup(nil, nil)

	require.NotPanics(t, func() {
		cleanup()
	})
}
