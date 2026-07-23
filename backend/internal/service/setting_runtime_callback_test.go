package service

import "testing"

func TestSettingUpdateRegistrationUnsubscribeIsOwnershipAware(t *testing.T) {
	settings := &SettingService{}
	firstCalls := 0
	secondCalls := 0

	unsubscribeFirst := settings.RegisterOnUpdateCallback(func() { firstCalls++ })
	unsubscribeSecond := settings.RegisterOnUpdateCallback(func() { secondCalls++ })
	unsubscribeFirst()
	settings.notifyUpdated()
	if firstCalls != 0 || secondCalls != 1 {
		t.Fatalf("after stale unsubscribe: first=%d second=%d", firstCalls, secondCalls)
	}

	unsubscribeSecond()
	unsubscribeSecond()
	settings.notifyUpdated()
	if secondCalls != 1 {
		t.Fatalf("callback invoked after unsubscribe: %d", secondCalls)
	}
}

func TestSettingLegacyUpdateCallbackRemainsCompatible(t *testing.T) {
	settings := &SettingService{}
	calls := 0
	settings.SetOnUpdateCallback(func() { calls++ })
	settings.notifyUpdated()
	if calls != 1 {
		t.Fatalf("legacy callback calls = %d, want 1", calls)
	}
	settings.SetOnUpdateCallback(nil)
	settings.notifyUpdated()
	if calls != 1 {
		t.Fatalf("legacy callback called after clear: %d", calls)
	}
}
