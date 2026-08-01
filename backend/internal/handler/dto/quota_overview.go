package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type QuotaOverview struct {
	SchemaVersion   int                         `json:"schema_version"`
	RequestID       string                      `json:"request_id"`
	GeneratedAt     time.Time                   `json:"generated_at"`
	AsOf            time.Time                   `json:"as_of"`
	FreshUntil      time.Time                   `json:"fresh_until"`
	DisplayTimezone string                      `json:"display_timezone"`
	Freshness       string                      `json:"freshness"`
	Coverage        QuotaOverviewCoverage       `json:"coverage"`
	Account         QuotaOverviewAccount        `json:"account"`
	Wallet          QuotaOverviewWallet         `json:"wallet"`
	BillingGroups   []QuotaOverviewBillingGroup `json:"billing_groups"`
	Subscriptions   []QuotaOverviewSubscription `json:"subscriptions"`
	Actions         QuotaOverviewActions        `json:"actions"`
	Warnings        []string                    `json:"warnings"`
}

type QuotaOverviewCoverage struct {
	Included []string `json:"included"`
	Excluded []string `json:"excluded"`
}

type QuotaOverviewAccount struct {
	DisplayLabel      string              `json:"display_label"`
	DataScope         string              `json:"data_scope"`
	QuotaState        string              `json:"quota_state"`
	CanMakeRequest    *bool               `json:"can_make_request"`
	UsableGroupCount  int                 `json:"usable_group_count"`
	BlockedGroupCount int                 `json:"blocked_group_count"`
	UnknownGroupCount int                 `json:"unknown_group_count"`
	PrimaryIssue      *QuotaOverviewIssue `json:"primary_issue"`
}

type QuotaOverviewIssue struct {
	ScopeType         string     `json:"scope_type"`
	ScopeID           string     `json:"scope_id"`
	ReasonCode        string     `json:"reason_code"`
	RecommendedAction string     `json:"recommended_action"`
	RecoversAt        *time.Time `json:"recovers_at"`
}

type QuotaOverviewWallet struct {
	Unit                  string `json:"unit"`
	State                 string `json:"state"`
	Available             string `json:"available"`
	Reserved              string `json:"reserved"`
	TodaySpend            string `json:"today_spend"`
	MonthSpend            string `json:"month_spend"`
	BalanceBilledKeyCount int    `json:"balance_billed_key_count"`
}

type QuotaOverviewBillingGroup struct {
	ID                string                   `json:"id"`
	DisplayName       string                   `json:"display_name"`
	BillingMode       string                   `json:"billing_mode"`
	State             string                   `json:"state"`
	ReasonCode        *string                  `json:"reason_code"`
	RecommendedAction string                   `json:"recommended_action"`
	FallbackPolicy    string                   `json:"fallback_policy"`
	ResourceRef       QuotaOverviewResourceRef `json:"resource_ref"`
	Keys              []QuotaOverviewKey       `json:"keys"`
}

type QuotaOverviewResourceRef struct {
	Kind string  `json:"kind"`
	ID   *string `json:"id"`
}

type QuotaOverviewKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MaskedKey string `json:"masked_key"`
	State     string `json:"state"`
}

type QuotaOverviewSubscription struct {
	ID            string                     `json:"id"`
	GroupID       string                     `json:"group_id"`
	Name          string                     `json:"name"`
	Status        string                     `json:"status"`
	StartsAt      time.Time                  `json:"starts_at"`
	ExpiresAt     time.Time                  `json:"expires_at"`
	WeeklyWindow  QuotaOverviewWeeklyWindow  `json:"weekly_window"`
	MonthlyWindow QuotaOverviewMonthlyWindow `json:"monthly_window"`
	PeriodUsage   QuotaOverviewPeriodUsage   `json:"period_usage"`
	NextEvent     *QuotaOverviewNextEvent    `json:"next_event"`
}

type QuotaOverviewPeriodUsage struct {
	State         string                          `json:"state"`
	ObservedUntil *time.Time                      `json:"observed_until"`
	BucketKind    string                          `json:"bucket_kind"`
	TotalRequests *int64                          `json:"total_requests"`
	TotalTokens   *int64                          `json:"total_tokens"`
	Points        []QuotaOverviewPeriodUsagePoint `json:"points"`
}

type QuotaOverviewPeriodUsagePoint struct {
	Index           int       `json:"index"`
	StartAt         time.Time `json:"start_at"`
	EndAt           time.Time `json:"end_at"`
	State           string    `json:"state"`
	Requests        *int64    `json:"requests"`
	CacheHitTokens  *int64    `json:"cache_hit_tokens"`
	CacheMissTokens *int64    `json:"cache_miss_tokens"`
	OutputTokens    *int64    `json:"output_tokens"`
	TotalTokens     *int64    `json:"total_tokens"`
}

type QuotaOverviewWeeklyWindow struct {
	Kind        string     `json:"kind"`
	State       string     `json:"state"`
	AnchorAt    time.Time  `json:"anchor_at"`
	PeriodStart *time.Time `json:"period_start"`
	PeriodEnd   *time.Time `json:"period_end"`
	ResetsAt    *time.Time `json:"resets_at"`
	Limit       *string    `json:"limit"`
	Used        *string    `json:"used"`
	Remaining   *string    `json:"remaining"`
	UsedPercent *float64   `json:"used_percent"`
}

type QuotaOverviewMonthlyWindow struct {
	Kind        string     `json:"kind"`
	State       string     `json:"state"`
	AnchorAt    time.Time  `json:"anchor_at"`
	PeriodStart *time.Time `json:"period_start"`
	PeriodEnd   *time.Time `json:"period_end"`
	ResetsAt    *time.Time `json:"resets_at"`
	Limit       *string    `json:"limit"`
	Used        *string    `json:"used"`
	Remaining   *string    `json:"remaining"`
	UsedPercent *float64   `json:"used_percent"`
}

type QuotaOverviewNextEvent struct {
	Kind string    `json:"kind"`
	At   time.Time `json:"at"`
}

type QuotaOverviewActions struct {
	RechargeURL            string `json:"recharge_url"`
	ManageKeysURL          string `json:"manage_keys_url"`
	ManageSubscriptionsURL string `json:"manage_subscriptions_url"`
}

func QuotaOverviewFromService(in *service.QuotaOverview) *QuotaOverview {
	if in == nil {
		return nil
	}
	out := &QuotaOverview{
		SchemaVersion:   in.SchemaVersion,
		RequestID:       in.RequestID,
		GeneratedAt:     in.GeneratedAt,
		AsOf:            in.AsOf,
		FreshUntil:      in.FreshUntil,
		DisplayTimezone: in.DisplayTimezone,
		Freshness:       in.Freshness,
		Coverage: QuotaOverviewCoverage{
			Included: append([]string(nil), in.Coverage.Included...),
			Excluded: append([]string(nil), in.Coverage.Excluded...),
		},
		Account: QuotaOverviewAccount{
			DisplayLabel:      in.Account.DisplayLabel,
			DataScope:         in.Account.DataScope,
			QuotaState:        in.Account.QuotaState,
			CanMakeRequest:    in.Account.CanMakeRequest,
			UsableGroupCount:  in.Account.UsableGroupCount,
			BlockedGroupCount: in.Account.BlockedGroupCount,
			UnknownGroupCount: in.Account.UnknownGroupCount,
			PrimaryIssue:      quotaOverviewIssueFromService(in.Account.PrimaryIssue),
		},
		Wallet: QuotaOverviewWallet{
			Unit:                  in.Wallet.Unit,
			State:                 in.Wallet.State,
			Available:             in.Wallet.Available,
			Reserved:              in.Wallet.Reserved,
			TodaySpend:            in.Wallet.TodaySpend,
			MonthSpend:            in.Wallet.MonthSpend,
			BalanceBilledKeyCount: in.Wallet.BalanceBilledKeyCount,
		},
		BillingGroups: make([]QuotaOverviewBillingGroup, 0, len(in.BillingGroups)),
		Subscriptions: make([]QuotaOverviewSubscription, 0, len(in.Subscriptions)),
		Actions: QuotaOverviewActions{
			RechargeURL:            in.Actions.RechargeURL,
			ManageKeysURL:          in.Actions.ManageKeysURL,
			ManageSubscriptionsURL: in.Actions.ManageSubscriptionsURL,
		},
		Warnings: append([]string(nil), in.Warnings...),
	}
	if out.Coverage.Included == nil {
		out.Coverage.Included = []string{}
	}
	if out.Coverage.Excluded == nil {
		out.Coverage.Excluded = []string{}
	}
	if out.Warnings == nil {
		out.Warnings = []string{}
	}
	for i := range in.BillingGroups {
		group := in.BillingGroups[i]
		mapped := QuotaOverviewBillingGroup{
			ID:                group.ID,
			DisplayName:       group.DisplayName,
			BillingMode:       group.BillingMode,
			State:             group.State,
			ReasonCode:        group.ReasonCode,
			RecommendedAction: group.RecommendedAction,
			FallbackPolicy:    group.FallbackPolicy,
			ResourceRef: QuotaOverviewResourceRef{
				Kind: group.ResourceRef.Kind,
				ID:   group.ResourceRef.ID,
			},
			Keys: make([]QuotaOverviewKey, 0, len(group.Keys)),
		}
		for j := range group.Keys {
			mapped.Keys = append(mapped.Keys, QuotaOverviewKey{
				ID:        group.Keys[j].ID,
				Name:      group.Keys[j].Name,
				MaskedKey: group.Keys[j].MaskedKey,
				State:     group.Keys[j].State,
			})
		}
		out.BillingGroups = append(out.BillingGroups, mapped)
	}
	for i := range in.Subscriptions {
		sub := in.Subscriptions[i]
		mapped := QuotaOverviewSubscription{
			ID:        sub.ID,
			GroupID:   sub.GroupID,
			Name:      sub.Name,
			Status:    sub.Status,
			StartsAt:  sub.StartsAt,
			ExpiresAt: sub.ExpiresAt,
			WeeklyWindow: QuotaOverviewWeeklyWindow{
				Kind:        sub.WeeklyWindow.Kind,
				State:       sub.WeeklyWindow.State,
				AnchorAt:    sub.WeeklyWindow.AnchorAt,
				PeriodStart: sub.WeeklyWindow.PeriodStart,
				PeriodEnd:   sub.WeeklyWindow.PeriodEnd,
				ResetsAt:    sub.WeeklyWindow.ResetsAt,
				Limit:       sub.WeeklyWindow.Limit,
				Used:        sub.WeeklyWindow.Used,
				Remaining:   sub.WeeklyWindow.Remaining,
				UsedPercent: sub.WeeklyWindow.UsedPercent,
			},
			MonthlyWindow: QuotaOverviewMonthlyWindow{
				Kind:        sub.MonthlyWindow.Kind,
				State:       sub.MonthlyWindow.State,
				AnchorAt:    sub.MonthlyWindow.AnchorAt,
				PeriodStart: sub.MonthlyWindow.PeriodStart,
				PeriodEnd:   sub.MonthlyWindow.PeriodEnd,
				ResetsAt:    sub.MonthlyWindow.ResetsAt,
				Limit:       sub.MonthlyWindow.Limit,
				Used:        sub.MonthlyWindow.Used,
				Remaining:   sub.MonthlyWindow.Remaining,
				UsedPercent: sub.MonthlyWindow.UsedPercent,
			},
			PeriodUsage: QuotaOverviewPeriodUsage{
				State:         sub.PeriodUsage.State,
				ObservedUntil: sub.PeriodUsage.ObservedUntil,
				BucketKind:    sub.PeriodUsage.BucketKind,
				TotalRequests: sub.PeriodUsage.TotalRequests,
				TotalTokens:   sub.PeriodUsage.TotalTokens,
				Points: make(
					[]QuotaOverviewPeriodUsagePoint,
					0,
					len(sub.PeriodUsage.Points),
				),
			},
		}
		if sub.PeriodUsage.Points == nil {
			mapped.PeriodUsage.Points = nil
		} else {
			for j := range sub.PeriodUsage.Points {
				point := sub.PeriodUsage.Points[j]
				mapped.PeriodUsage.Points = append(
					mapped.PeriodUsage.Points,
					QuotaOverviewPeriodUsagePoint{
						Index:           point.Index,
						StartAt:         point.StartAt,
						EndAt:           point.EndAt,
						State:           point.State,
						Requests:        point.Requests,
						CacheHitTokens:  point.CacheHitTokens,
						CacheMissTokens: point.CacheMissTokens,
						OutputTokens:    point.OutputTokens,
						TotalTokens:     point.TotalTokens,
					},
				)
			}
		}
		if sub.NextEvent != nil {
			mapped.NextEvent = &QuotaOverviewNextEvent{
				Kind: sub.NextEvent.Kind,
				At:   sub.NextEvent.At,
			}
		}
		out.Subscriptions = append(out.Subscriptions, mapped)
	}
	return out
}

func quotaOverviewIssueFromService(in *service.QuotaOverviewIssue) *QuotaOverviewIssue {
	if in == nil {
		return nil
	}
	return &QuotaOverviewIssue{
		ScopeType:         in.ScopeType,
		ScopeID:           in.ScopeID,
		ReasonCode:        in.ReasonCode,
		RecommendedAction: in.RecommendedAction,
		RecoversAt:        in.RecoversAt,
	}
}
