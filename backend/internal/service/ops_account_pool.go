package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	DefaultOpsAccountPoolAnomalyLimit = 5
	MaxOpsAccountPoolAnomalyLimit     = 20
	OpsAccountPoolLowRedundancyLimit  = int64(1)

	OpsAccountPoolCategoryActionable     = "actionable"
	OpsAccountPoolCategoryAutoRecovering = "auto_recovering"

	OpsAccountPoolReasonError             = "account_error"
	OpsAccountPoolReasonExpired           = "expired_auto_paused"
	OpsAccountPoolReasonQuotaExhausted    = "known_quota_exhausted"
	OpsAccountPoolReasonRateLimited       = "rate_limited"
	OpsAccountPoolReasonOverloaded        = "overloaded"
	OpsAccountPoolReasonTemporaryCooldown = "temporary_cooldown"
)

// OpsAccountPoolSummary is the operator-facing capacity summary. Quota coverage
// is reported separately because OAuth/model-scoped quota cannot be determined
// reliably and must not be mixed into an apparent health percentage.
type OpsAccountPoolSummary struct {
	TotalAccounts             int64 `json:"total_accounts"`
	BaseSchedulableCount      int64 `json:"base_schedulable_count"`
	ActionableCount           int64 `json:"actionable_count"`
	AutoRecoveringCount       int64 `json:"auto_recovering_count"`
	InactiveCount             int64 `json:"inactive_count"`
	ManualUnschedulableCount  int64 `json:"manual_unschedulable_count"`
	QuotaCoverageUnknownCount int64 `json:"quota_coverage_unknown_count"`
	ZeroCapacityGroupCount    int64 `json:"zero_capacity_group_count"`
	LowRedundancyGroupCount   int64 `json:"low_redundancy_group_count"`
	LowRedundancyThreshold    int64 `json:"low_redundancy_threshold"`
}

// OpsAccountPoolPlatformSummary describes account capacity for one platform.
type OpsAccountPoolPlatformSummary struct {
	Platform                  string `json:"platform"`
	TotalAccounts             int64  `json:"total_accounts"`
	BaseSchedulableCount      int64  `json:"base_schedulable_count"`
	ActionableCount           int64  `json:"actionable_count"`
	AutoRecoveringCount       int64  `json:"auto_recovering_count"`
	InactiveCount             int64  `json:"inactive_count"`
	ManualUnschedulableCount  int64  `json:"manual_unschedulable_count"`
	QuotaCoverageUnknownCount int64  `json:"quota_coverage_unknown_count"`
	ZeroCapacity              bool   `json:"zero_capacity"`
	LowRedundancy             bool   `json:"low_redundancy"`
}

// OpsAccountPoolGroupSummary describes account capacity for one group. An
// account may belong to multiple groups, so these counts are deliberately not
// additive across rows.
type OpsAccountPoolGroupSummary struct {
	GroupID                   int64  `json:"group_id"`
	GroupName                 string `json:"group_name"`
	Platform                  string `json:"platform"`
	TotalAccounts             int64  `json:"total_accounts"`
	BaseSchedulableCount      int64  `json:"base_schedulable_count"`
	ActionableCount           int64  `json:"actionable_count"`
	AutoRecoveringCount       int64  `json:"auto_recovering_count"`
	InactiveCount             int64  `json:"inactive_count"`
	ManualUnschedulableCount  int64  `json:"manual_unschedulable_count"`
	QuotaCoverageUnknownCount int64  `json:"quota_coverage_unknown_count"`
	ZeroCapacity              bool   `json:"zero_capacity"`
	LowRedundancy             bool   `json:"low_redundancy"`
}

// OpsAccountPoolAnomaly is intentionally credential-free. Detail is generated
// from a bounded reason taxonomy rather than copying upstream/database error
// strings that may contain secrets.
type OpsAccountPoolAnomaly struct {
	AccountID   int64    `json:"account_id"`
	AccountName string   `json:"account_name"`
	Platform    string   `json:"platform"`
	ProxyID     *int64   `json:"proxy_id,omitempty"`
	GroupIDs    []int64  `json:"group_ids"`
	GroupNames  []string `json:"group_names"`

	Status      string `json:"status"`
	Schedulable bool   `json:"schedulable"`
	Category    string `json:"category"`

	PrimaryReason string   `json:"primary_reason"`
	ReasonCodes   []string `json:"reason_codes"`
	Detail        string   `json:"detail"`

	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	RecoverAt        *time.Time `json:"recover_at,omitempty"`
	RemainingSeconds *int64     `json:"remaining_seconds,omitempty"`
}

// OpsAccountPoolResponse is the full read-only account-pool snapshot.
type OpsAccountPoolResponse struct {
	Summary OpsAccountPoolSummary `json:"summary"`

	Platforms []OpsAccountPoolPlatformSummary `json:"platforms"`
	Groups    []OpsAccountPoolGroupSummary    `json:"groups"`

	// Anomalies is the unified operator queue. The two category-specific lists
	// let the detailed account-pool view separate manual action from automatic
	// recovery without issuing another account scan.
	Anomalies               []OpsAccountPoolAnomaly `json:"anomalies"`
	ActionableAnomalies     []OpsAccountPoolAnomaly `json:"actionable_anomalies"`
	AutoRecoveringAnomalies []OpsAccountPoolAnomaly `json:"auto_recovering_anomalies"`

	GroupCountsAdditive bool      `json:"group_counts_additive"`
	CollectedAt         time.Time `json:"collected_at"`
}

type opsAccountPoolClassification struct {
	baseSchedulable bool
	actionable      bool
	autoRecovering  bool
	inactive        bool
	manualExcluded  bool
	quotaUnknown    bool
	reasons         []string
	primaryReason   string
	recoverAt       *time.Time
}

// GetAccountPool returns a bounded, read-only snapshot for the operations
// command center. It never invokes account tests, recovery, re-authentication,
// or scheduling mutations.
func (s *OpsService) GetAccountPool(
	ctx context.Context,
	platformFilter string,
	groupIDFilter *int64,
	limit int,
) (*OpsAccountPoolResponse, error) {
	if s == nil {
		return nil, errors.New("ops service is nil")
	}
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}

	accounts, err := s.listAllAccountsForOps(ctx, platformFilter, groupIDFilter)
	if err != nil {
		return nil, err
	}

	return buildOpsAccountPool(accounts, time.Now().UTC(), normalizeOpsAccountPoolLimit(limit)), nil
}

func normalizeOpsAccountPoolLimit(limit int) int {
	if limit <= 0 {
		return DefaultOpsAccountPoolAnomalyLimit
	}
	if limit > MaxOpsAccountPoolAnomalyLimit {
		return MaxOpsAccountPoolAnomalyLimit
	}
	return limit
}

func buildOpsAccountPool(accounts []Account, now time.Time, limit int) *OpsAccountPoolResponse {
	limit = normalizeOpsAccountPoolLimit(limit)
	result := &OpsAccountPoolResponse{
		Summary: OpsAccountPoolSummary{
			LowRedundancyThreshold: OpsAccountPoolLowRedundancyLimit,
		},
		Platforms:               []OpsAccountPoolPlatformSummary{},
		Groups:                  []OpsAccountPoolGroupSummary{},
		Anomalies:               []OpsAccountPoolAnomaly{},
		ActionableAnomalies:     []OpsAccountPoolAnomaly{},
		AutoRecoveringAnomalies: []OpsAccountPoolAnomaly{},
		GroupCountsAdditive:     false,
		CollectedAt:             now,
	}

	platforms := make(map[string]*OpsAccountPoolPlatformSummary)
	groups := make(map[int64]*OpsAccountPoolGroupSummary)
	actionable := make([]OpsAccountPoolAnomaly, 0)
	autoRecovering := make([]OpsAccountPoolAnomaly, 0)

	for i := range accounts {
		acc := &accounts[i]
		if acc.ID <= 0 {
			continue
		}

		classification := classifyOpsAccountPoolAccount(acc, now)
		result.Summary.TotalAccounts++
		applyOpsAccountPoolClassificationToSummary(&result.Summary, classification)

		platformName := strings.TrimSpace(acc.Platform)
		if platformName == "" {
			platformName = "unknown"
		}
		platform := platforms[platformName]
		if platform == nil {
			platform = &OpsAccountPoolPlatformSummary{Platform: platformName}
			platforms[platformName] = platform
		}
		platform.TotalAccounts++
		applyOpsAccountPoolClassificationToPlatform(platform, classification)

		seenGroups := make(map[int64]struct{}, len(acc.Groups))
		for _, grp := range acc.Groups {
			if grp == nil || grp.ID <= 0 {
				continue
			}
			if _, ok := seenGroups[grp.ID]; ok {
				continue
			}
			seenGroups[grp.ID] = struct{}{}

			group := groups[grp.ID]
			if group == nil {
				group = &OpsAccountPoolGroupSummary{
					GroupID:   grp.ID,
					GroupName: grp.Name,
					Platform:  grp.Platform,
				}
				groups[grp.ID] = group
			}
			group.TotalAccounts++
			applyOpsAccountPoolClassificationToGroup(group, classification)
		}

		if classification.actionable || classification.autoRecovering {
			anomaly := buildOpsAccountPoolAnomaly(acc, classification, now)
			if classification.actionable {
				actionable = append(actionable, anomaly)
			} else {
				autoRecovering = append(autoRecovering, anomaly)
			}
		}
	}

	for _, platform := range platforms {
		platform.ZeroCapacity = platform.TotalAccounts > 0 && platform.BaseSchedulableCount == 0
		platform.LowRedundancy = platform.TotalAccounts > 0 && platform.BaseSchedulableCount <= OpsAccountPoolLowRedundancyLimit
		result.Platforms = append(result.Platforms, *platform)
	}
	for _, group := range groups {
		group.ZeroCapacity = group.TotalAccounts > 0 && group.BaseSchedulableCount == 0
		group.LowRedundancy = group.TotalAccounts > 0 && group.BaseSchedulableCount <= OpsAccountPoolLowRedundancyLimit
		if group.ZeroCapacity {
			result.Summary.ZeroCapacityGroupCount++
		}
		if group.LowRedundancy {
			result.Summary.LowRedundancyGroupCount++
		}
		result.Groups = append(result.Groups, *group)
	}

	sort.Slice(result.Platforms, func(i, j int) bool {
		left, right := result.Platforms[i], result.Platforms[j]
		if left.BaseSchedulableCount != right.BaseSchedulableCount {
			return left.BaseSchedulableCount < right.BaseSchedulableCount
		}
		return left.Platform < right.Platform
	})
	sort.Slice(result.Groups, func(i, j int) bool {
		left, right := result.Groups[i], result.Groups[j]
		if left.BaseSchedulableCount != right.BaseSchedulableCount {
			return left.BaseSchedulableCount < right.BaseSchedulableCount
		}
		if left.TotalAccounts != right.TotalAccounts {
			return left.TotalAccounts > right.TotalAccounts
		}
		return left.GroupID < right.GroupID
	})

	sortOpsAccountPoolAnomalies(actionable)
	sortOpsAccountPoolAnomalies(autoRecovering)
	all := append(append(make([]OpsAccountPoolAnomaly, 0, len(actionable)+len(autoRecovering)), actionable...), autoRecovering...)
	sortOpsAccountPoolAnomalies(all)

	result.Anomalies = truncateOpsAccountPoolAnomalies(all, limit)
	result.ActionableAnomalies = truncateOpsAccountPoolAnomalies(actionable, limit)
	result.AutoRecoveringAnomalies = truncateOpsAccountPoolAnomalies(autoRecovering, limit)
	return result
}

func classifyOpsAccountPoolAccount(acc *Account, now time.Time) opsAccountPoolClassification {
	classification := opsAccountPoolClassification{}
	if acc == nil {
		return classification
	}

	// The scheduler's account-level predicate is the source of truth. The
	// classifier below only explains why that predicate is false.
	classification.baseSchedulable = acc.IsSchedulable()
	expired := acc.AutoPauseOnExpired && acc.ExpiresAt != nil && !now.Before(*acc.ExpiresAt)
	classification.inactive = acc.Status != StatusActive && acc.Status != StatusError
	classification.manualExcluded = acc.Status == StatusActive && !acc.Schedulable && !expired
	classification.quotaUnknown = acc.Status == StatusActive && acc.Schedulable && !acc.IsAPIKeyOrBedrock()

	seen := make(map[string]struct{}, 6)
	addReason := func(reason string) {
		if _, ok := seen[reason]; ok {
			return
		}
		seen[reason] = struct{}{}
		classification.reasons = append(classification.reasons, reason)
	}

	if acc.Status == StatusError {
		addReason(OpsAccountPoolReasonError)
	}
	if expired {
		addReason(OpsAccountPoolReasonExpired)
	}

	// Runtime reasons on intentionally disabled/manual-unscheduled accounts are
	// suppressed as noise. Auto-expiry and explicit error states remain visible.
	runtimeCandidate := acc.Status == StatusActive && acc.Schedulable
	if runtimeCandidate && acc.IsAPIKeyOrBedrock() && acc.IsQuotaExceeded() {
		addReason(OpsAccountPoolReasonQuotaExhausted)
	}
	if runtimeCandidate && acc.RateLimitResetAt != nil && now.Before(*acc.RateLimitResetAt) {
		addReason(OpsAccountPoolReasonRateLimited)
		classification.recoverAt = laterTime(classification.recoverAt, acc.RateLimitResetAt)
	}
	if runtimeCandidate && acc.OverloadUntil != nil && now.Before(*acc.OverloadUntil) {
		addReason(OpsAccountPoolReasonOverloaded)
		classification.recoverAt = laterTime(classification.recoverAt, acc.OverloadUntil)
	}
	if runtimeCandidate && acc.TempUnschedulableUntil != nil && now.Before(*acc.TempUnschedulableUntil) {
		addReason(OpsAccountPoolReasonTemporaryCooldown)
		classification.recoverAt = laterTime(classification.recoverAt, acc.TempUnschedulableUntil)
	}

	for _, reason := range classification.reasons {
		switch reason {
		case OpsAccountPoolReasonError, OpsAccountPoolReasonExpired, OpsAccountPoolReasonQuotaExhausted:
			classification.actionable = true
		}
	}
	classification.autoRecovering = !classification.actionable && len(classification.reasons) > 0
	if len(classification.reasons) > 0 {
		classification.primaryReason = classification.reasons[0]
	}
	return classification
}

func laterTime(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.After(*current) {
		value := *candidate
		return &value
	}
	return current
}

func applyOpsAccountPoolClassificationToSummary(summary *OpsAccountPoolSummary, c opsAccountPoolClassification) {
	if c.baseSchedulable {
		summary.BaseSchedulableCount++
	}
	if c.actionable {
		summary.ActionableCount++
	}
	if c.autoRecovering {
		summary.AutoRecoveringCount++
	}
	if c.inactive {
		summary.InactiveCount++
	}
	if c.manualExcluded {
		summary.ManualUnschedulableCount++
	}
	if c.quotaUnknown {
		summary.QuotaCoverageUnknownCount++
	}
}

func applyOpsAccountPoolClassificationToPlatform(summary *OpsAccountPoolPlatformSummary, c opsAccountPoolClassification) {
	if c.baseSchedulable {
		summary.BaseSchedulableCount++
	}
	if c.actionable {
		summary.ActionableCount++
	}
	if c.autoRecovering {
		summary.AutoRecoveringCount++
	}
	if c.inactive {
		summary.InactiveCount++
	}
	if c.manualExcluded {
		summary.ManualUnschedulableCount++
	}
	if c.quotaUnknown {
		summary.QuotaCoverageUnknownCount++
	}
}

func applyOpsAccountPoolClassificationToGroup(summary *OpsAccountPoolGroupSummary, c opsAccountPoolClassification) {
	if c.baseSchedulable {
		summary.BaseSchedulableCount++
	}
	if c.actionable {
		summary.ActionableCount++
	}
	if c.autoRecovering {
		summary.AutoRecoveringCount++
	}
	if c.inactive {
		summary.InactiveCount++
	}
	if c.manualExcluded {
		summary.ManualUnschedulableCount++
	}
	if c.quotaUnknown {
		summary.QuotaCoverageUnknownCount++
	}
}

func buildOpsAccountPoolAnomaly(acc *Account, c opsAccountPoolClassification, now time.Time) OpsAccountPoolAnomaly {
	groupIDs, groupNames := opsAccountPoolGroupIdentity(acc.Groups)
	category := OpsAccountPoolCategoryAutoRecovering
	if c.actionable {
		category = OpsAccountPoolCategoryActionable
	}

	anomaly := OpsAccountPoolAnomaly{
		AccountID:     acc.ID,
		AccountName:   acc.Name,
		Platform:      acc.Platform,
		ProxyID:       acc.ProxyID,
		GroupIDs:      groupIDs,
		GroupNames:    groupNames,
		Status:        acc.Status,
		Schedulable:   acc.Schedulable,
		Category:      category,
		PrimaryReason: c.primaryReason,
		ReasonCodes:   append([]string(nil), c.reasons...),
		Detail:        opsAccountPoolReasonDetail(c.primaryReason, acc.TempUnschedulableReason),
		RecoverAt:     c.recoverAt,
	}
	if c.primaryReason == OpsAccountPoolReasonExpired && acc.ExpiresAt != nil {
		expiresAt := acc.ExpiresAt.UTC()
		anomaly.ExpiresAt = &expiresAt
	}
	if c.recoverAt != nil {
		seconds := int64(c.recoverAt.Sub(now).Seconds())
		if seconds < 1 {
			seconds = 1
		}
		anomaly.RemainingSeconds = &seconds
	}
	return anomaly
}

func opsAccountPoolReasonDetail(reason, tempReason string) string {
	switch reason {
	case OpsAccountPoolReasonError:
		return "account is in an error state"
	case OpsAccountPoolReasonExpired:
		return "account expired and automatic pause is enabled"
	case OpsAccountPoolReasonQuotaExhausted:
		return "known API key or Bedrock quota is exhausted"
	case OpsAccountPoolReasonRateLimited:
		return "rate limit will recover automatically"
	case OpsAccountPoolReasonOverloaded:
		return "upstream overload window will recover automatically"
	case OpsAccountPoolReasonTemporaryCooldown:
		// Only derive a bounded taxonomy from the stored reason; never echo the
		// original value because it can contain an upstream response or token.
		normalized := strings.ToLower(tempReason)
		switch {
		case strings.Contains(normalized, "token refresh") || strings.Contains(normalized, "oauth"):
			return "credential refresh cooldown will recover automatically"
		case strings.Contains(normalized, "proxy") || strings.Contains(normalized, "transport") || strings.Contains(normalized, "network"):
			return "network or proxy cooldown will recover automatically"
		default:
			return "temporary cooldown will recover automatically"
		}
	default:
		return "account requires attention"
	}
}

func opsAccountPoolGroupIdentity(groups []*Group) ([]int64, []string) {
	type identity struct {
		id   int64
		name string
	}
	items := make([]identity, 0, len(groups))
	seen := make(map[int64]struct{}, len(groups))
	for _, group := range groups {
		if group == nil || group.ID <= 0 {
			continue
		}
		if _, ok := seen[group.ID]; ok {
			continue
		}
		seen[group.ID] = struct{}{}
		items = append(items, identity{id: group.ID, name: group.Name})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].id < items[j].id })
	ids := make([]int64, 0, len(items))
	names := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.id)
		names = append(names, item.name)
	}
	return ids, names
}

func sortOpsAccountPoolAnomalies(items []OpsAccountPoolAnomaly) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		leftCategory := opsAccountPoolCategoryRank(left.Category)
		rightCategory := opsAccountPoolCategoryRank(right.Category)
		if leftCategory != rightCategory {
			return leftCategory < rightCategory
		}
		leftReason := opsAccountPoolReasonRank(left.PrimaryReason)
		rightReason := opsAccountPoolReasonRank(right.PrimaryReason)
		if leftReason != rightReason {
			return leftReason < rightReason
		}
		if left.Platform != right.Platform {
			return left.Platform < right.Platform
		}
		return left.AccountID < right.AccountID
	})
}

func opsAccountPoolCategoryRank(category string) int {
	if category == OpsAccountPoolCategoryActionable {
		return 0
	}
	return 1
}

func opsAccountPoolReasonRank(reason string) int {
	switch reason {
	case OpsAccountPoolReasonError:
		return 0
	case OpsAccountPoolReasonExpired:
		return 1
	case OpsAccountPoolReasonQuotaExhausted:
		return 2
	case OpsAccountPoolReasonRateLimited:
		return 3
	case OpsAccountPoolReasonOverloaded:
		return 4
	case OpsAccountPoolReasonTemporaryCooldown:
		return 5
	default:
		return 6
	}
}

func truncateOpsAccountPoolAnomalies(items []OpsAccountPoolAnomaly, limit int) []OpsAccountPoolAnomaly {
	if len(items) == 0 {
		return []OpsAccountPoolAnomaly{}
	}
	if limit > len(items) {
		limit = len(items)
	}
	return append([]OpsAccountPoolAnomaly(nil), items[:limit]...)
}
