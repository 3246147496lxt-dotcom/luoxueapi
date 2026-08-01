package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// SubscriptionProgress is the explicit HTTP representation of subscription
// usage progress. Keep this DTO at the handler boundary so service structs are
// not serialized as an accidental public contract.
type SubscriptionProgress struct {
	ID            int64                `json:"id"`
	GroupName     string               `json:"group_name"`
	ExpiresAt     time.Time            `json:"expires_at"`
	ExpiresInDays int                  `json:"expires_in_days"`
	Daily         *UsageWindowProgress `json:"daily,omitempty"`
	Weekly        *UsageWindowProgress `json:"weekly,omitempty"`
	Monthly       *UsageWindowProgress `json:"monthly,omitempty"`
}

type UsageWindowProgress struct {
	State           string    `json:"state"`
	LimitUSD        float64   `json:"limit_usd"`
	UsedUSD         *float64  `json:"used_usd"`
	RemainingUSD    *float64  `json:"remaining_usd"`
	Percentage      *float64  `json:"percentage"`
	WindowStart     time.Time `json:"window_start"`
	ResetsAt        time.Time `json:"resets_at"`
	ResetsInSeconds int64     `json:"resets_in_seconds"`
}

// SubscriptionProgressSubscription preserves the legacy subscription object
// while allowing this endpoint to represent an unknown window counter as null.
// Other subscription endpoints keep their existing non-null usage contract.
type SubscriptionProgressSubscription struct {
	ID      int64 `json:"id"`
	UserID  int64 `json:"user_id"`
	GroupID int64 `json:"group_id"`

	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Status    string    `json:"status"`

	DailyWindowStart   *time.Time `json:"daily_window_start"`
	WeeklyWindowStart  *time.Time `json:"weekly_window_start"`
	MonthlyWindowStart *time.Time `json:"monthly_window_start"`

	DailyUsageUSD   *float64 `json:"daily_usage_usd"`
	WeeklyUsageUSD  *float64 `json:"weekly_usage_usd"`
	MonthlyUsageUSD *float64 `json:"monthly_usage_usd"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`

	User  *User  `json:"user,omitempty"`
	Group *Group `json:"group,omitempty"`
}

// SubscriptionProgressInfo is the legacy user endpoint item shape:
// {subscription, progress}. It intentionally differs from the admin endpoint,
// which returns SubscriptionProgress directly.
type SubscriptionProgressInfo struct {
	Subscription *SubscriptionProgressSubscription `json:"subscription"`
	Progress     *SubscriptionProgress             `json:"progress"`
}

func SubscriptionProgressInfoFromService(
	sub *service.UserSubscription,
	progress *service.SubscriptionProgress,
) SubscriptionProgressInfo {
	return SubscriptionProgressInfo{
		Subscription: subscriptionProgressSubscriptionFromService(sub, progress),
		Progress:     SubscriptionProgressFromService(progress),
	}
}

func subscriptionProgressSubscriptionFromService(
	sub *service.UserSubscription,
	progress *service.SubscriptionProgress,
) *SubscriptionProgressSubscription {
	base := UserSubscriptionFromService(sub)
	if base == nil {
		return nil
	}

	dailyUsage := base.DailyUsageUSD
	weeklyUsage := base.WeeklyUsageUSD
	monthlyUsage := base.MonthlyUsageUSD
	out := &SubscriptionProgressSubscription{
		ID:                 base.ID,
		UserID:             base.UserID,
		GroupID:            base.GroupID,
		StartsAt:           base.StartsAt,
		ExpiresAt:          base.ExpiresAt,
		Status:             base.Status,
		DailyWindowStart:   base.DailyWindowStart,
		WeeklyWindowStart:  base.WeeklyWindowStart,
		MonthlyWindowStart: base.MonthlyWindowStart,
		DailyUsageUSD:      &dailyUsage,
		WeeklyUsageUSD:     &weeklyUsage,
		MonthlyUsageUSD:    &monthlyUsage,
		CreatedAt:          base.CreatedAt,
		UpdatedAt:          base.UpdatedAt,
		RevokedAt:          base.RevokedAt,
		User:               base.User,
		Group:              base.Group,
	}
	if progress == nil {
		return out
	}
	if progress.Daily != nil {
		out.DailyWindowStart = &progress.Daily.WindowStart
		out.DailyUsageUSD = progress.Daily.UsedUSD
	}
	if progress.Weekly != nil {
		out.WeeklyWindowStart = &progress.Weekly.WindowStart
		out.WeeklyUsageUSD = progress.Weekly.UsedUSD
	}
	if progress.Monthly != nil {
		out.MonthlyWindowStart = &progress.Monthly.WindowStart
		out.MonthlyUsageUSD = progress.Monthly.UsedUSD
	}
	return out
}

func SubscriptionProgressFromService(progress *service.SubscriptionProgress) *SubscriptionProgress {
	if progress == nil {
		return nil
	}
	return &SubscriptionProgress{
		ID:            progress.ID,
		GroupName:     progress.GroupName,
		ExpiresAt:     progress.ExpiresAt,
		ExpiresInDays: progress.ExpiresInDays,
		Daily:         usageWindowProgressFromService(progress.Daily),
		Weekly:        usageWindowProgressFromService(progress.Weekly),
		Monthly:       usageWindowProgressFromService(progress.Monthly),
	}
}

func usageWindowProgressFromService(progress *service.UsageWindowProgress) *UsageWindowProgress {
	if progress == nil {
		return nil
	}
	return &UsageWindowProgress{
		State:           progress.State,
		LimitUSD:        progress.LimitUSD,
		UsedUSD:         progress.UsedUSD,
		RemainingUSD:    progress.RemainingUSD,
		Percentage:      progress.Percentage,
		WindowStart:     progress.WindowStart,
		ResetsAt:        progress.ResetsAt,
		ResetsInSeconds: progress.ResetsInSeconds,
	}
}
