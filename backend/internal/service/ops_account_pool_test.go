package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildOpsAccountPool_ClassifiesAndAggregatesWithoutLeakingCredentials(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	rateReset := now.Add(10 * time.Minute)
	overloadUntil := now.Add(20 * time.Minute)
	tempUntil := now.Add(30 * time.Minute)

	groupOne := &Group{ID: 1, Name: "primary", Platform: PlatformOpenAI}
	groupTwo := &Group{ID: 2, Name: "backup", Platform: PlatformOpenAI}
	groupZero := &Group{ID: 3, Name: "zero", Platform: PlatformOpenAI}

	accounts := []Account{
		{
			ID: 1, Name: "healthy-oauth", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Status: StatusActive, Schedulable: true,
			Groups: []*Group{groupOne, groupTwo, groupOne},
		},
		{
			ID: 2, Name: "error", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Status: StatusError, Schedulable: true,
			ErrorMessage: `access_token=must-not-leak`,
			Groups:       []*Group{groupOne, groupZero},
		},
		{
			ID: 3, Name: "expired", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Status: StatusActive, Schedulable: false,
			AutoPauseOnExpired: true, ExpiresAt: &past,
			Groups: []*Group{groupOne},
		},
		{
			ID: 4, Name: "quota", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Status: StatusActive, Schedulable: true,
			Extra:  map[string]any{"quota_limit": 10.0, "quota_used": 10.0},
			Groups: []*Group{groupTwo},
		},
		{
			ID: 5, Name: "rate-and-overload", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Status: StatusActive, Schedulable: true,
			RateLimitResetAt: &rateReset, OverloadUntil: &overloadUntil,
			Groups: []*Group{groupOne},
		},
		{
			ID: 6, Name: "cooldown", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Status: StatusActive, Schedulable: true,
			TempUnschedulableUntil:  &tempUntil,
			TempUnschedulableReason: `proxy transport failed; authorization=Bearer-super-secret`,
			Groups:                  []*Group{groupTwo},
		},
		{
			ID: 7, Name: "disabled", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Status: StatusDisabled, Schedulable: false,
			Groups: []*Group{groupTwo},
		},
		{
			ID: 8, Name: "manual", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Status: StatusActive, Schedulable: false,
			Groups: []*Group{groupTwo},
		},
		{
			ID: 9, Name: "healthy-key", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Status: StatusActive, Schedulable: true,
		},
	}

	result := buildOpsAccountPool(accounts, now, DefaultOpsAccountPoolAnomalyLimit)
	if result.Summary.TotalAccounts != 9 {
		t.Fatalf("TotalAccounts = %d, want 9", result.Summary.TotalAccounts)
	}
	if result.Summary.BaseSchedulableCount != 2 {
		t.Fatalf("BaseSchedulableCount = %d, want 2", result.Summary.BaseSchedulableCount)
	}
	if result.Summary.ActionableCount != 3 {
		t.Fatalf("ActionableCount = %d, want 3", result.Summary.ActionableCount)
	}
	if result.Summary.AutoRecoveringCount != 2 {
		t.Fatalf("AutoRecoveringCount = %d, want 2", result.Summary.AutoRecoveringCount)
	}
	if result.Summary.InactiveCount != 1 || result.Summary.ManualUnschedulableCount != 1 {
		t.Fatalf("inactive/manual = %d/%d, want 1/1", result.Summary.InactiveCount, result.Summary.ManualUnschedulableCount)
	}
	if result.Summary.QuotaCoverageUnknownCount != 3 {
		t.Fatalf("QuotaCoverageUnknownCount = %d, want 3", result.Summary.QuotaCoverageUnknownCount)
	}
	if result.Summary.ZeroCapacityGroupCount != 1 || result.Summary.LowRedundancyGroupCount != 3 {
		t.Fatalf("zero/low redundancy groups = %d/%d, want 1/3", result.Summary.ZeroCapacityGroupCount, result.Summary.LowRedundancyGroupCount)
	}
	if result.GroupCountsAdditive {
		t.Fatal("group counts must explicitly be non-additive")
	}

	if len(result.Anomalies) != 5 {
		t.Fatalf("anomalies len = %d, want 5", len(result.Anomalies))
	}
	wantReasons := []string{
		OpsAccountPoolReasonError,
		OpsAccountPoolReasonExpired,
		OpsAccountPoolReasonQuotaExhausted,
		OpsAccountPoolReasonRateLimited,
		OpsAccountPoolReasonTemporaryCooldown,
	}
	for i, want := range wantReasons {
		if result.Anomalies[i].PrimaryReason != want {
			t.Fatalf("anomaly[%d].PrimaryReason = %q, want %q", i, result.Anomalies[i].PrimaryReason, want)
		}
	}

	rateAnomaly := result.Anomalies[3]
	if len(rateAnomaly.ReasonCodes) != 2 || rateAnomaly.ReasonCodes[0] != OpsAccountPoolReasonRateLimited || rateAnomaly.ReasonCodes[1] != OpsAccountPoolReasonOverloaded {
		t.Fatalf("rate anomaly reason order = %#v", rateAnomaly.ReasonCodes)
	}
	if rateAnomaly.RecoverAt == nil || !rateAnomaly.RecoverAt.Equal(overloadUntil) {
		t.Fatalf("recover_at = %v, want latest window %v", rateAnomaly.RecoverAt, overloadUntil)
	}

	errorAnomaly := result.Anomalies[0]
	if len(errorAnomaly.GroupIDs) != 2 || errorAnomaly.GroupIDs[0] != 1 || errorAnomaly.GroupIDs[1] != 3 {
		t.Fatalf("error group ids = %#v, want [1 3]", errorAnomaly.GroupIDs)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	body := string(encoded)
	for _, secret := range []string{"must-not-leak", "Bearer-super-secret"} {
		if strings.Contains(body, secret) {
			t.Fatalf("account-pool response leaked secret %q: %s", secret, body)
		}
	}
}

func TestClassifyOpsAccountPoolAccount_IntentionalExclusionsDoNotCreateNoise(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)

	tests := []struct {
		name    string
		account Account
	}{
		{
			name: "manual unscheduled with stale runtime flags",
			account: Account{ID: 1, Status: StatusActive, Schedulable: false,
				RateLimitResetAt: &future, OverloadUntil: &future, TempUnschedulableUntil: &future},
		},
		{
			name: "disabled with stale runtime flags",
			account: Account{ID: 2, Status: StatusDisabled, Schedulable: false,
				RateLimitResetAt: &future, OverloadUntil: &future, TempUnschedulableUntil: &future},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classification := classifyOpsAccountPoolAccount(&tt.account, now)
			if classification.actionable || classification.autoRecovering || len(classification.reasons) != 0 {
				t.Fatalf("intentional exclusion created anomaly: %#v", classification)
			}
		})
	}
}

func TestNormalizeOpsAccountPoolLimit(t *testing.T) {
	if got := normalizeOpsAccountPoolLimit(0); got != DefaultOpsAccountPoolAnomalyLimit {
		t.Fatalf("zero limit = %d, want default %d", got, DefaultOpsAccountPoolAnomalyLimit)
	}
	if got := normalizeOpsAccountPoolLimit(MaxOpsAccountPoolAnomalyLimit + 1); got != MaxOpsAccountPoolAnomalyLimit {
		t.Fatalf("oversized limit = %d, want max %d", got, MaxOpsAccountPoolAnomalyLimit)
	}
	if got := normalizeOpsAccountPoolLimit(7); got != 7 {
		t.Fatalf("valid limit = %d, want 7", got)
	}
}
