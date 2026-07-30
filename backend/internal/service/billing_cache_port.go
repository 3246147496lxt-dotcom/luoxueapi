package service

import (
	"time"
)

// SubscriptionCacheData represents cached subscription data
type SubscriptionCacheData struct {
	SubscriptionID    int64
	Status            string
	StartsAt          time.Time
	ExpiresAt         time.Time
	WeeklyWindowStart *time.Time
	WeeklyWindowEnd   time.Time
	DailyUsage        float64
	WeeklyUsage       float64
	MonthlyUsage      float64
	Version           int64
}
