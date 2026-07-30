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
	LimitUSD        float64   `json:"limit_usd"`
	UsedUSD         float64   `json:"used_usd"`
	RemainingUSD    float64   `json:"remaining_usd"`
	Percentage      float64   `json:"percentage"`
	WindowStart     time.Time `json:"window_start"`
	ResetsAt        time.Time `json:"resets_at"`
	ResetsInSeconds int64     `json:"resets_in_seconds"`
}

// SubscriptionProgressInfo is the legacy user endpoint item shape:
// {subscription, progress}. It intentionally differs from the admin endpoint,
// which returns SubscriptionProgress directly.
type SubscriptionProgressInfo struct {
	Subscription *UserSubscription     `json:"subscription"`
	Progress     *SubscriptionProgress `json:"progress"`
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
		LimitUSD:        progress.LimitUSD,
		UsedUSD:         progress.UsedUSD,
		RemainingUSD:    progress.RemainingUSD,
		Percentage:      progress.Percentage,
		WindowStart:     progress.WindowStart,
		ResetsAt:        progress.ResetsAt,
		ResetsInSeconds: progress.ResetsInSeconds,
	}
}
