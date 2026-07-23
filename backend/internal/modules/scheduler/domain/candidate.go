package domain

import "sort"

// CandidateQuery is the stable input contract for scheduler candidate reads.
// GroupID nil means the ungrouped/global scheduling scope.
type CandidateQuery struct {
	GroupID       *int64
	Platform      string
	ForcePlatform bool
}

// CandidateSet intentionally contains identifiers rather than transport-layer
// account objects. Callers hydrate only the account snapshots they actually use.
type CandidateSet struct {
	AccountIDs      []int64
	MixedScheduling bool
}

// CandidateMismatch describes a shadow-read disagreement without changing the
// primary selector's result.
type CandidateMismatch struct {
	Query              CandidateQuery
	PrimaryAccountIDs  []int64
	ShadowAccountIDs   []int64
	PrimaryMixed       bool
	ShadowMixed        bool
	MissingFromShadow  []int64
	UnexpectedInShadow []int64
	ShadowError        string
}

func CompareCandidateSets(query CandidateQuery, primary, shadow CandidateSet) (CandidateMismatch, bool) {
	primaryIDs := normalizeIDs(primary.AccountIDs)
	shadowIDs := normalizeIDs(shadow.AccountIDs)
	mismatch := CandidateMismatch{
		Query:             query,
		PrimaryAccountIDs: primaryIDs,
		ShadowAccountIDs:  shadowIDs,
		PrimaryMixed:      primary.MixedScheduling,
		ShadowMixed:       shadow.MixedScheduling,
	}
	primarySet := make(map[int64]struct{}, len(primaryIDs))
	shadowSet := make(map[int64]struct{}, len(shadowIDs))
	for _, id := range primaryIDs {
		primarySet[id] = struct{}{}
	}
	for _, id := range shadowIDs {
		shadowSet[id] = struct{}{}
	}
	for _, id := range primaryIDs {
		if _, ok := shadowSet[id]; !ok {
			mismatch.MissingFromShadow = append(mismatch.MissingFromShadow, id)
		}
	}
	for _, id := range shadowIDs {
		if _, ok := primarySet[id]; !ok {
			mismatch.UnexpectedInShadow = append(mismatch.UnexpectedInShadow, id)
		}
	}
	different := primary.MixedScheduling != shadow.MixedScheduling ||
		len(mismatch.MissingFromShadow) > 0 || len(mismatch.UnexpectedInShadow) > 0
	return mismatch, different
}

func normalizeIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
