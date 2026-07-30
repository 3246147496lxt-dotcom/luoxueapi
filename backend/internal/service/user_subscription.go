package service

import "time"

const SubscriptionWeeklyWindowDuration = 7 * 24 * time.Hour

type UserSubscription struct {
	ID      int64
	UserID  int64
	GroupID int64

	StartsAt  time.Time
	ExpiresAt time.Time
	Status    string

	DailyWindowStart   *time.Time
	WeeklyWindowStart  *time.Time
	MonthlyWindowStart *time.Time

	DailyUsageUSD   float64
	WeeklyUsageUSD  float64
	MonthlyUsageUSD float64

	AssignedBy *int64
	AssignedAt time.Time
	Notes      string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	User           *User
	Group          *Group
	AssignedByUser *User
}

func (s *UserSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && time.Now().Before(s.ExpiresAt)
}

func (s *UserSubscription) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *UserSubscription) DaysRemaining() int {
	if s.IsExpired() {
		return 0
	}
	return int(time.Until(s.ExpiresAt).Hours() / 24)
}

func (s *UserSubscription) IsWindowActivated() bool {
	return s != nil && s.WeeklyWindowStart != nil
}

// AnchoredWeeklyWindow returns the authoritative half-open 7x24h window
// [start, end) containing at. The subscription start is the immutable anchor
// for one continuous subscription term.
func AnchoredWeeklyWindow(anchor, at time.Time) (start, end time.Time, ok bool) {
	if anchor.IsZero() || at.Before(anchor) {
		return time.Time{}, time.Time{}, false
	}
	windowIndex := at.Sub(anchor) / SubscriptionWeeklyWindowDuration
	start = anchor.Add(windowIndex * SubscriptionWeeklyWindowDuration)
	return start, start.Add(SubscriptionWeeklyWindowDuration), true
}

func (s *UserSubscription) WeeklyWindowAt(at time.Time) (start, end time.Time, ok bool) {
	if s == nil {
		return time.Time{}, time.Time{}, false
	}
	return AnchoredWeeklyWindow(s.StartsAt, at)
}

func (s *UserSubscription) HasOneTimeDailyQuota() bool {
	if s == nil || s.StartsAt.IsZero() || s.ExpiresAt.IsZero() {
		return false
	}
	return !s.ExpiresAt.After(s.StartsAt.AddDate(0, 0, 1))
}

func (s *UserSubscription) NeedsDailyReset() bool {
	return s.NeedsDailyResetAt(time.Now())
}

func (s *UserSubscription) NeedsDailyResetAt(now time.Time) bool {
	if s.DailyWindowStart == nil {
		return false
	}
	if s.HasOneTimeDailyQuota() {
		return false
	}
	return !now.Before(s.DailyWindowStart.Add(24 * time.Hour))
}

func (s *UserSubscription) NeedsWeeklyReset() bool {
	return s.NeedsWeeklyResetAt(time.Now())
}

func (s *UserSubscription) NeedsWeeklyResetAt(now time.Time) bool {
	expectedStart, _, ok := s.WeeklyWindowAt(now)
	if !ok {
		return false
	}
	return s.WeeklyWindowStart == nil || !s.WeeklyWindowStart.Equal(expectedStart)
}

func (s *UserSubscription) NeedsMonthlyReset() bool {
	if s.MonthlyWindowStart == nil {
		return false
	}
	return time.Since(*s.MonthlyWindowStart) >= 30*24*time.Hour
}

func (s *UserSubscription) DailyResetTime() *time.Time {
	if s.DailyWindowStart == nil {
		return nil
	}
	if s.HasOneTimeDailyQuota() {
		t := s.ExpiresAt
		return &t
	}
	t := s.DailyWindowStart.Add(24 * time.Hour)
	return &t
}

func (s *UserSubscription) WeeklyResetTime() *time.Time {
	_, end, ok := s.WeeklyWindowAt(time.Now())
	if !ok {
		return nil
	}
	return &end
}

func (s *UserSubscription) MonthlyResetTime() *time.Time {
	if s.MonthlyWindowStart == nil {
		return nil
	}
	t := s.MonthlyWindowStart.Add(30 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) CheckDailyLimit(group *Group, additionalCost float64) bool {
	if !group.HasDailyLimit() {
		return true
	}
	return s.DailyUsageUSD+additionalCost <= *group.DailyLimitUSD
}

func (s *UserSubscription) CheckWeeklyLimit(group *Group, additionalCost float64) bool {
	if !group.HasWeeklyLimit() {
		return true
	}
	used := s.EffectiveWeeklyUsageAt(time.Now())
	if additionalCost <= 0 {
		return used < *group.WeeklyLimitUSD
	}
	return used+additionalCost <= *group.WeeklyLimitUSD
}

// EffectiveWeeklyUsageAt returns usage only when its persisted marker belongs
// to the starts_at-anchored period containing at. A counter from any other
// period is stale and must not block the new period.
func (s *UserSubscription) EffectiveWeeklyUsageAt(at time.Time) float64 {
	if s == nil || s.WeeklyWindowStart == nil {
		return 0
	}
	expectedStart, _, ok := s.WeeklyWindowAt(at)
	if !ok || !s.WeeklyWindowStart.Equal(expectedStart) {
		return 0
	}
	return s.WeeklyUsageUSD
}

func (s *UserSubscription) CheckMonthlyLimit(group *Group, additionalCost float64) bool {
	if !group.HasMonthlyLimit() {
		return true
	}
	return s.MonthlyUsageUSD+additionalCost <= *group.MonthlyLimitUSD
}

func (s *UserSubscription) CheckAllLimits(group *Group, additionalCost float64) (daily, weekly, monthly bool) {
	// P0 subscriptions have one authoritative anchored 7-day quota. Legacy
	// daily/monthly columns remain readable during compatibility rollout but
	// must never block a subscription request.
	daily = true
	weekly = s.CheckWeeklyLimit(group, additionalCost)
	monthly = true
	return
}
