package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordOpenAIProfitVetoBounded(t *testing.T) {
	failed := make(map[int64]struct{})
	count := 0

	for id := int64(1); id < int64(maxProfitVetoAttempts); id++ {
		require.True(t, recordOpenAIProfitVeto(failed, id, &count))
		require.Contains(t, failed, id)
	}
	require.False(t, recordOpenAIProfitVeto(failed, int64(maxProfitVetoAttempts), &count))
	require.Equal(t, maxProfitVetoAttempts, count)
	require.Len(t, failed, maxProfitVetoAttempts)
}

func TestProfitVetoBudgetSharedWithFailoverState(t *testing.T) {
	state := NewFailoverState(10, false)
	failed := make(map[int64]struct{})
	count := 0
	stateStoppedAt := 0
	openAIStoppedAt := 0

	for id := int64(1); id <= int64(maxProfitVetoAttempts)+5; id++ {
		if stateStoppedAt == 0 && state.RecordProfitVeto(id) == FailoverExhausted {
			stateStoppedAt = state.ProfitVetoCount()
		}
		if openAIStoppedAt == 0 && !recordOpenAIProfitVeto(failed, id, &count) {
			openAIStoppedAt = count
		}
	}

	require.Equal(t, maxProfitVetoAttempts, stateStoppedAt)
	require.Equal(t, stateStoppedAt, openAIStoppedAt)
}
