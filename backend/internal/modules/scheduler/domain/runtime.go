package domain

import "time"

type AccountSnapshot struct {
	ID          int64
	Platform    string
	Status      string
	Schedulable bool
	UpdatedAt   time.Time
}

type RuntimeStatus struct {
	Started                   bool
	InitialSnapshotDone       bool
	Degraded                  bool
	LastError                 string
	ShadowComparisons         uint64
	ShadowMismatches          uint64
	ShadowComparisonErrors    uint64
	LastShadowMismatchAt      time.Time
	LastShadowComparisonError string
}

type OutboxConsumeResult struct {
	Attempted bool
	Consumed  int
}
