package handler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type profitVetoLoopResult struct {
	outcome        string
	forwardedID    int64
	iterations     int
	backoffRetries int
}

// runProfitVetoLoop models only the selection/profit-veto/exhaustion state
// machine. A hard iteration budget turns any regression back to the old 503
// exclusion-reset livelock into a deterministic test failure.
func runProfitVetoLoop(t *testing.T, fs *FailoverState, pool []int64, vetoed map[int64]bool, maxIterations int) profitVetoLoopResult {
	t.Helper()
	res := profitVetoLoopResult{}
	for res.iterations = 1; res.iterations <= maxIterations; res.iterations++ {
		var picked int64
		for _, id := range pool {
			if _, excluded := fs.FailedAccountIDs[id]; !excluded {
				picked = id
				break
			}
		}
		if picked == 0 {
			if fs.HandleSelectionExhausted(context.Background()) == FailoverContinue {
				res.backoffRetries++
				continue
			}
			res.outcome = "exhausted"
			return res
		}
		if vetoed[picked] {
			if fs.RecordProfitVeto(picked) == FailoverExhausted {
				res.outcome = "exhausted"
				return res
			}
			continue
		}
		res.outcome = "forwarded"
		res.forwardedID = picked
		return res
	}
	res.outcome = "budget_exceeded"
	return res
}

func TestProfitVetoAfter503DoesNotLivelock(t *testing.T) {
	fs := NewFailoverState(10, false)
	fs.LastFailoverErr = newTestFailoverErr(503, false, false)
	fs.SwitchCount = 1
	fs.FailedAccountIDs[1] = struct{}{}

	start := time.Now()
	res := runProfitVetoLoop(t, fs, []int64{1}, map[int64]bool{1: true}, 50)

	require.Equal(t, "exhausted", res.outcome)
	require.LessOrEqual(t, res.backoffRetries, 1)
	require.Less(t, time.Since(start), 10*time.Second)
}

func TestProfitVetoKeepsBackoffUsefulForHealthyAccount(t *testing.T) {
	fs := NewFailoverState(10, false)
	fs.LastFailoverErr = newTestFailoverErr(503, false, false)
	fs.SwitchCount = 1
	fs.FailedAccountIDs[1] = struct{}{}

	res := runProfitVetoLoop(t, fs, []int64{2, 1}, map[int64]bool{2: true}, 50)

	require.Equal(t, "forwarded", res.outcome)
	require.Equal(t, int64(1), res.forwardedID)
	require.Equal(t, 1, res.backoffRetries)
	require.Contains(t, fs.FailedAccountIDs, int64(2))
}

func TestProfitVetoAttemptsCapped(t *testing.T) {
	fs := NewFailoverState(10, false)
	pool := make([]int64, 0, 64)
	vetoed := make(map[int64]bool, 64)
	for id := int64(1); id <= 64; id++ {
		pool = append(pool, id)
		vetoed[id] = true
	}

	res := runProfitVetoLoop(t, fs, pool, vetoed, 200)

	require.Equal(t, "exhausted", res.outcome)
	require.Equal(t, maxProfitVetoAttempts, fs.ProfitVetoCount())
	require.Equal(t, maxProfitVetoAttempts, res.iterations)
}

func TestRecordProfitVetoExcludesAccount(t *testing.T) {
	fs := NewFailoverState(10, false)
	require.Equal(t, FailoverContinue, fs.RecordProfitVeto(42))
	require.Contains(t, fs.FailedAccountIDs, int64(42))
	require.Equal(t, 1, fs.ProfitVetoCount())
}

func TestHandleSelectionExhaustedUnaffectedWithoutProfitVeto(t *testing.T) {
	fs := NewFailoverState(3, false)
	fs.LastFailoverErr = newTestFailoverErr(503, false, false)
	fs.SwitchCount = 1
	fs.FailedAccountIDs[100] = struct{}{}

	require.Equal(t, FailoverContinue, fs.HandleSelectionExhausted(context.Background()))
	require.Empty(t, fs.FailedAccountIDs)
}
