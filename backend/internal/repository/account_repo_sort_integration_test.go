//go:build integration

package repository

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (s *AccountRepoSuite) TestList_DefaultSortByNameAsc() {
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "z-account"})
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "a-account"})

	accounts, _, err := s.repo.List(s.ctx, pagination.PaginationParams{Page: 1, PageSize: 10})
	s.Require().NoError(err)
	s.Require().Len(accounts, 2)
	s.Require().Equal("a-account", accounts[0].Name)
	s.Require().Equal("z-account", accounts[1].Name)
}

func (s *AccountRepoSuite) TestListWithFilters_SortByPriorityDesc() {
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "low-priority", Priority: 10})
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "high-priority", Priority: 90})

	accounts, _, err := s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page:      1,
		PageSize:  10,
		SortBy:    "priority",
		SortOrder: "desc",
	}, "", "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Len(accounts, 2)
	s.Require().Equal("high-priority", accounts[0].Name)
	s.Require().Equal("low-priority", accounts[1].Name)
}

func (s *AccountRepoSuite) TestListWithFilters_SortByEffectiveStatusOpenAIOAuth() {
	now := time.Now().UTC()
	active := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "a-active",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"codex_5h_used_percent": 42.0,
		},
	})
	expiredSnapshot := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "b-reset-expired",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"codex_5h_used_percent": 100.0,
			"codex_5h_reset_at":     now.Add(-time.Minute).Format(time.RFC3339),
		},
	})
	futureSnapshot := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "c-quota-future",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"codex_5h_used_percent": 100.0,
			"codex_5h_reset_at":     now.Add(time.Hour).Format(time.RFC3339),
		},
	})
	unknownReset := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "d-quota-unknown",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"codex_7d_used_percent": 100.0,
		},
	})
	malformedPercent := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "e-string-percent-is-not-numeric",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"codex_5h_used_percent": "100",
		},
	})
	invalidReset := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "f-invalid-reset-is-unknown",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"codex_5h_used_percent": 100.0,
			"codex_5h_reset_at":     "2026-02-31T12:00:00Z",
		},
	})

	for _, account := range []*service.Account{active, expiredSnapshot, futureSnapshot, unknownReset, malformedPercent, invalidReset} {
		s.Require().Equal(service.StatusActive, account.Status)
	}

	accounts, _, err := s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page:      1,
		PageSize:  10,
		SortBy:    "effective_status",
		SortOrder: "asc",
	}, "", "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Equal(
		[]string{"a-active", "b-reset-expired", "e-string-percent-is-not-numeric", "c-quota-future", "d-quota-unknown", "f-invalid-reset-is-unknown"},
		accountNames(accounts),
	)

	accounts, _, err = s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page:      1,
		PageSize:  10,
		SortBy:    "effective_status",
		SortOrder: "desc",
	}, "", "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Equal(
		[]string{"f-invalid-reset-is-unknown", "d-quota-unknown", "c-quota-future", "e-string-percent-is-not-numeric", "b-reset-expired", "a-active"},
		accountNames(accounts),
	)

	// Pagination must happen after effective-status ordering. Page one in ASC is
	// healthy, while page one in DESC is quota-exhausted.
	ascFirst, _, err := s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page: 1, PageSize: 1, SortBy: "effective_status", SortOrder: "asc",
	}, "", "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Equal("a-active", ascFirst[0].Name)
	descFirst, _, err := s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page: 1, PageSize: 1, SortBy: "effective_status", SortOrder: "desc",
	}, "", "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Equal("f-invalid-reset-is-unknown", descFirst[0].Name)
}

func (s *AccountRepoSuite) TestListWithFilters_SortByEffectiveStatusRestrictions() {
	now := time.Now().UTC()
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "active", Status: service.StatusActive, Schedulable: true})
	mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "quota-apikey",
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"quota_limit": 10.0,
			"quota_used":  10.0,
		},
	})
	mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "quota-codex-apikey",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra:       map[string]any{"codex_5h_used_percent": 100.0},
	})
	mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "quota-codex-setup-token",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeSetupToken,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra:       map[string]any{"codex_7d_used_percent": 100.0},
	})
	mustCreateAccount(s.T(), s.client, &service.Account{
		Name:          "overloaded",
		Status:        service.StatusActive,
		Schedulable:   true,
		OverloadUntil: timePointer(now.Add(time.Hour)),
	})
	mustCreateAccount(s.T(), s.client, &service.Account{
		Name:             "rate-limited",
		Status:           service.StatusActive,
		Schedulable:      true,
		RateLimitResetAt: timePointer(now.Add(time.Hour)),
	})
	temp := mustCreateAccount(s.T(), s.client, &service.Account{Name: "temp-unschedulable", Status: service.StatusActive, Schedulable: true})
	s.Require().NoError(s.client.Account.UpdateOneID(temp.ID).SetTempUnschedulableUntil(now.Add(time.Hour)).Exec(s.ctx))
	paused := mustCreateAccount(s.T(), s.client, &service.Account{Name: "paused", Status: service.StatusActive, Schedulable: true})
	s.Require().NoError(s.client.Account.UpdateOneID(paused.ID).SetSchedulable(false).Exec(s.ctx))
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "disabled", Status: service.StatusDisabled, Schedulable: true})
	mustCreateAccount(s.T(), s.client, &service.Account{Name: "error", Status: service.StatusError, Schedulable: true})

	accounts, _, err := s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page: 1, PageSize: 20, SortBy: "effective_status", SortOrder: "asc",
	}, "", "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Equal(
		[]string{"active", "quota-apikey", "quota-codex-apikey", "quota-codex-setup-token", "overloaded", "rate-limited", "temp-unschedulable", "paused", "disabled", "error"},
		accountNames(accounts),
	)
}

func (s *AccountRepoSuite) TestListWithFilters_SortByEffectiveStatusMalformedQuotaTimes() {
	mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "a-malformed-daily-start",
		Platform:    service.PlatformAnthropic,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"quota_daily_limit": 10.0,
			"quota_daily_used":  10.0,
			"quota_daily_start": "not-a-timestamp",
		},
	})
	mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "b-malformed-weekly-reset",
		Platform:    service.PlatformAnthropic,
		Type:        service.AccountTypeBedrock,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"quota_weekly_limit":      10.0,
			"quota_weekly_used":       10.0,
			"quota_weekly_reset_mode": "fixed",
			"quota_weekly_reset_at":   "2026-02-31T12:00:00Z",
		},
	})
	mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "c-valid-total-quota",
		Platform:    service.PlatformAnthropic,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"quota_limit": 10.0,
			"quota_used":  10.0,
		},
	})

	accounts, _, err := s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page: 1, PageSize: 10, SortBy: "effective_status", SortOrder: "asc",
	}, "", "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Equal(
		[]string{"a-malformed-daily-start", "b-malformed-weekly-reset", "c-valid-total-quota"},
		accountNames(accounts),
	)
}

func (s *AccountRepoSuite) TestListWithFilters_SortByEffectiveStatusStableTieBreakers() {
	first := mustCreateAccount(s.T(), s.client, &service.Account{
		Name: "quota-tie", Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true,
		Extra: map[string]any{"quota_limit": 1.0, "quota_used": 1.0},
	})
	second := mustCreateAccount(s.T(), s.client, &service.Account{
		Name: "quota-tie", Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true,
		Extra: map[string]any{"quota_limit": 1.0, "quota_used": 1.0},
	})

	accounts, _, err := s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page: 1, PageSize: 10, SortBy: "effective_status", SortOrder: "asc",
	}, "", "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Equal([]int64{first.ID, second.ID}, accountIDs(accounts))

	accounts, _, err = s.repo.ListWithFilters(s.ctx, pagination.PaginationParams{
		Page: 1, PageSize: 10, SortBy: "effective_status", SortOrder: "desc",
	}, "", "", "", "", 0, "")
	s.Require().NoError(err)
	s.Require().Equal([]int64{second.ID, first.ID}, accountIDs(accounts))
}

func accountNames(accounts []service.Account) []string {
	names := make([]string, len(accounts))
	for i := range accounts {
		names[i] = accounts[i].Name
	}
	return names
}

func accountIDs(accounts []service.Account) []int64 {
	ids := make([]int64, len(accounts))
	for i := range accounts {
		ids[i] = accounts[i].ID
	}
	return ids
}

func timePointer(value time.Time) *time.Time {
	return &value
}
