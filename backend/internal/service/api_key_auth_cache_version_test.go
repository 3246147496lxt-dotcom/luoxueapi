package service

import "testing"

func TestAPIKeyService_RejectsV10AuthSnapshotWithoutModelsListConfig(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-legacy-models-list", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{
			Version:  10,
			APIKeyID: 1,
			UserID:   2,
			GroupID:  &groupID,
			Status:   StatusActive,
			User: APIKeyAuthUserSnapshot{
				ID:          2,
				Status:      StatusActive,
				Role:        RoleUser,
				Balance:     10,
				Concurrency: 3,
			},
			Group: &APIKeyAuthGroupSnapshot{
				ID:               groupID,
				Name:             "openai",
				Platform:         PlatformOpenAI,
				Status:           StatusActive,
				SubscriptionType: SubscriptionTypeStandard,
				RateMultiplier:   1,
			},
		},
	})

	if err != nil {
		t.Fatalf("expected stale snapshot to be ignored without error, got %v", err)
	}
	if ok {
		t.Fatalf("expected v10 auth snapshot to be rejected after models_list_config was added")
	}
	if apiKey != nil {
		t.Fatalf("expected no API key from stale snapshot, got %#v", apiKey)
	}
}

func TestAPIKeyService_AuthSnapshotV16CarriesServiceTierPreference(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}
	apiKey := &APIKey{
		ID:                    1,
		UserID:                2,
		Key:                   "k-priority",
		GroupID:               &groupID,
		Name:                  "priority key",
		Status:                StatusActive,
		Purpose:               APIKeyPurposeUser,
		ServiceTierPreference: ServiceTierPreferencePriority,
		User:                  &User{ID: 2, Status: StatusActive, Role: RoleUser, Balance: 10},
		Group:                 &Group{ID: groupID, Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard},
	}

	snapshot := svc.snapshotFromAPIKey(nil, apiKey)
	if snapshot == nil {
		t.Fatal("expected auth snapshot")
	}
	if snapshot.Version != 16 {
		t.Fatalf("expected v16 auth snapshot, got %d", snapshot.Version)
	}
	if snapshot.ServiceTierPreference != ServiceTierPreferencePriority {
		t.Fatalf("expected priority preference in snapshot, got %q", snapshot.ServiceTierPreference)
	}
	roundTrip, ok, err := svc.applyAuthCacheEntry(apiKey.Key, &APIKeyAuthCacheEntry{Snapshot: snapshot})
	if err != nil || !ok {
		t.Fatalf("expected v16 snapshot to apply, ok=%v err=%v", ok, err)
	}
	if roundTrip.ServiceTierPreference != ServiceTierPreferencePriority {
		t.Fatalf("expected priority preference after round-trip, got %q", roundTrip.ServiceTierPreference)
	}
	if roundTrip.Purpose != APIKeyPurposeUser {
		t.Fatalf("expected user purpose after round-trip, got %q", roundTrip.Purpose)
	}
}

func TestAPIKeyService_AuthSnapshotV15MissesAndReloads(t *testing.T) {
	svc := &APIKeyService{}
	apiKey, ok, err := svc.applyAuthCacheEntry("k-v15", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{
			Version:  15,
			APIKeyID: 1,
			UserID:   2,
			Status:   StatusActive,
			User:     APIKeyAuthUserSnapshot{ID: 2, Status: StatusActive, Role: RoleUser},
		},
	})
	if err != nil || ok || apiKey != nil {
		t.Fatalf("expected v15 snapshot miss, apiKey=%#v ok=%v err=%v", apiKey, ok, err)
	}
}

func TestAPIKeyService_AuthSnapshotForWebChatForcesStandardPreference(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}
	apiKey := &APIKey{
		ID:                    2,
		UserID:                3,
		Key:                   "k-web-chat",
		GroupID:               &groupID,
		Purpose:               APIKeyPurposeWebChat,
		Status:                StatusActive,
		ServiceTierPreference: ServiceTierPreferencePriority,
		User:                  &User{ID: 3, Status: StatusActive, Role: RoleUser},
	}
	snapshot := svc.snapshotFromAPIKey(nil, apiKey)
	if snapshot == nil {
		t.Fatal("expected auth snapshot")
	}
	if snapshot.ServiceTierPreference != ServiceTierPreferenceStandard {
		t.Fatalf("expected web chat snapshot to force standard, got %q", snapshot.ServiceTierPreference)
	}
	got := svc.snapshotToAPIKey(apiKey.Key, snapshot)
	if got.ServiceTierPreference != ServiceTierPreferenceStandard {
		t.Fatalf("expected web chat round-trip to force standard, got %q", got.ServiceTierPreference)
	}
}

func TestNormalizeServiceTierPreference(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: "", want: ServiceTierPreferenceStandard, ok: true},
		{input: " STANDARD ", want: ServiceTierPreferenceStandard, ok: true},
		{input: "priority", want: ServiceTierPreferencePriority, ok: true},
		{input: "fast", ok: false},
	}
	for _, tt := range tests {
		got, ok := NormalizeServiceTierPreference(tt.input)
		if got != tt.want || ok != tt.ok {
			t.Errorf("NormalizeServiceTierPreference(%q) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}
