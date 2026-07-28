package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/desktop"
	"github.com/stretchr/testify/require"
)

type desktopCleanupCall struct {
	now    time.Time
	cutoff time.Time
	limit  int
}

type desktopCleanupRepoStub struct {
	calls chan desktopCleanupCall
}

func (r *desktopCleanupRepoStub) CleanupExpired(_ context.Context, now, cutoff time.Time, limit int) (*desktop.CleanupResult, error) {
	call := desktopCleanupCall{now: now, cutoff: cutoff, limit: limit}
	if r.calls != nil {
		r.calls <- call
	}
	return &desktop.CleanupResult{PendingDevicesDeleted: 2, SessionsDeleted: 3, DiagnosticsDeleted: 4}, nil
}

func TestDesktopCleanupServiceRunOnceAppliesRetentionAndBatch(t *testing.T) {
	repo := &desktopCleanupRepoStub{calls: make(chan desktopCleanupCall, 1)}
	svc := NewDesktopCleanupService(repo)
	now := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)

	result, err := svc.RunOnce(context.Background(), now)
	require.NoError(t, err)
	require.Equal(t, &desktop.CleanupResult{PendingDevicesDeleted: 2, SessionsDeleted: 3, DiagnosticsDeleted: 4}, result)
	call := <-repo.calls
	require.Equal(t, now, call.now)
	require.Equal(t, now.Add(-7*24*time.Hour), call.cutoff)
	require.Equal(t, 500, call.limit)
}

func TestDesktopCleanupServiceStartsWithImmediateRunAndStops(t *testing.T) {
	repo := &desktopCleanupRepoStub{calls: make(chan desktopCleanupCall, 1)}
	svc := NewDesktopCleanupService(repo)
	svc.Start()

	select {
	case <-repo.calls:
	case <-time.After(time.Second):
		t.Fatal("desktop cleanup did not run on startup")
	}
	svc.Stop()
}
