package domain

import "time"

type GroupBinding struct {
	GroupID  int64
	Priority int
}

type CreateAccountCommand struct {
	Name                string
	Notes               *string
	Platform            string
	Type                string
	Credentials         map[string]any
	Extra               map[string]any
	ProxyID             *int64
	Concurrency         int
	Priority            int
	RateMultiplier      *float64
	LoadFactor          *int
	Status              string
	Schedulable         bool
	ExpiresAt           *time.Time
	AutoPauseOnExpired  bool
	ParentAccountID     *int64
	QuotaDimension      string
	LastUsedAt          *time.Time
	RateLimitedAt       *time.Time
	RateLimitResetAt    *time.Time
	OverloadUntil       *time.Time
	SessionWindowStart  *time.Time
	SessionWindowEnd    *time.Time
	SessionWindowStatus string
	Groups              []GroupBinding
}

type Account struct {
	ID        int64
	Name      string
	Platform  string
	Type      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Groups    []GroupBinding
}
