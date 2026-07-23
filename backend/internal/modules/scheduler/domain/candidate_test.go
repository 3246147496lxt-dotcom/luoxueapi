package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareCandidateSetsUsesNormalizedIDSets(t *testing.T) {
	query := CandidateQuery{Platform: "openai"}
	mismatch, different := CompareCandidateSets(
		query,
		CandidateSet{AccountIDs: []int64{9, 2, 9, 0}},
		CandidateSet{AccountIDs: []int64{2, 7}},
	)

	require.True(t, different)
	require.Equal(t, []int64{2, 9}, mismatch.PrimaryAccountIDs)
	require.Equal(t, []int64{2, 7}, mismatch.ShadowAccountIDs)
	require.Equal(t, []int64{9}, mismatch.MissingFromShadow)
	require.Equal(t, []int64{7}, mismatch.UnexpectedInShadow)
}

func TestCompareCandidateSetsIgnoresOrder(t *testing.T) {
	_, different := CompareCandidateSets(
		CandidateQuery{},
		CandidateSet{AccountIDs: []int64{3, 1, 2}, MixedScheduling: true},
		CandidateSet{AccountIDs: []int64{2, 3, 1}, MixedScheduling: true},
	)
	require.False(t, different)
}
