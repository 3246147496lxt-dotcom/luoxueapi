package service

import "time"

const (
	SubscriptionWeeklyWindowDuration  = 7 * 24 * time.Hour
	SubscriptionMonthlyWindowDuration = 30 * 24 * time.Hour
)

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
	return s.IsExpiredAt(time.Now())
}

func (s *UserSubscription) IsExpiredAt(at time.Time) bool {
	return s == nil || !at.Before(s.ExpiresAt)
}

func (s *UserSubscription) DaysRemaining() int {
	if s.IsExpired() {
		return 0
	}
	return int(time.Until(s.ExpiresAt).Hours() / 24)
}

func (s *UserSubscription) IsWindowActivated() bool {
	return s != nil && s.WeeklyWindowStart != nil && s.MonthlyWindowStart != nil
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

// AnchoredMonthlyWindow returns the authoritative half-open 30x24h window
// [start, end) containing at. Like the weekly window, it is anchored to the
// exact subscription start and never to first use or a local calendar boundary.
func AnchoredMonthlyWindow(anchor, at time.Time) (start, end time.Time, ok bool) {
	if anchor.IsZero() || at.Before(anchor) {
		return time.Time{}, time.Time{}, false
	}
	windowIndex := at.Sub(anchor) / SubscriptionMonthlyWindowDuration
	start = anchor.Add(windowIndex * SubscriptionMonthlyWindowDuration)
	return start, start.Add(SubscriptionMonthlyWindowDuration), true
}

func (s *UserSubscription) WeeklyWindowAt(at time.Time) (start, end time.Time, ok bool) {
	if s == nil {
		return time.Time{}, time.Time{}, false
	}
	return AnchoredWeeklyWindow(s.StartsAt, at)
}

func (s *UserSubscription) MonthlyWindowAt(at time.Time) (start, end time.Time, ok bool) {
	if s == nil {
		return time.Time{}, time.Time{}, false
	}
	return AnchoredMonthlyWindow(s.StartsAt, at)
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
	return s.NeedsMonthlyResetAt(time.Now())
}

func (s *UserSubscription) NeedsMonthlyResetAt(now time.Time) bool {
	expectedStart, _, ok := s.MonthlyWindowAt(now)
	if !ok {
		return false
	}
	return s.MonthlyWindowStart == nil || !s.MonthlyWindowStart.Equal(expectedStart)
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
	_, end, ok := s.MonthlyWindowAt(time.Now())
	if !ok {
		return nil
	}
	return &end
}

func (s *UserSubscription) CheckDailyLimit(group *Group, additionalCost float64) bool {
	if !group.HasDailyLimit() {
		return true
	}
	return s.DailyUsageUSD+additionalCost <= *group.DailyLimitUSD
}

func (s *UserSubscription) CheckWeeklyLimit(group *Group, additionalCost float64) bool {
	return s.CheckWeeklyLimitAt(group, additionalCost, time.Now())
}

func (s *UserSubscription) CheckWeeklyLimitAt(group *Group, additionalCost float64, at time.Time) bool {
	if !group.HasWeeklyLimit() {
		return true
	}
	used := s.EffectiveWeeklyUsageAt(at)
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
	return s.CheckMonthlyLimitAt(group, additionalCost, time.Now())
}

func (s *UserSubscription) CheckMonthlyLimitAt(group *Group, additionalCost float64, at time.Time) bool {
	limit, ok := group.EffectiveMonthlyLimitUSD()
	if !ok {
		return true
	}
	used := s.EffectiveMonthlyUsageAt(at)
	if additionalCost <= 0 {
		return used < limit
	}
	return used+additionalCost <= limit
}

// EffectiveMonthlyUsageAt returns usage only when its persisted marker belongs
// to the starts_at-anchored 30-day period containing at.
func (s *UserSubscription) EffectiveMonthlyUsageAt(at time.Time) float64 {
	if s == nil || s.MonthlyWindowStart == nil {
		return 0
	}
	expectedStart, _, ok := s.MonthlyWindowAt(at)
	if !ok || !s.MonthlyWindowStart.Equal(expectedStart) {
		return 0
	}
	return s.MonthlyUsageUSD
}

func (s *UserSubscription) CheckAllLimits(group *Group, additionalCost float64) (daily, weekly, monthly bool) {
	// Daily remains a legacy compatibility field. Weekly and monthly are the
	// authoritative membership limits.
	daily = true
	weekly = s.CheckWeeklyLimit(group, additionalCost)
	monthly = s.CheckMonthlyLimit(group, additionalCost)
	return
}
