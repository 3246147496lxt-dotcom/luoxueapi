package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type skillGitHubRoundTripFunc func(*http.Request) (*http.Response, error)

func (f skillGitHubRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type skillGitHubStarsClientFunc func(context.Context, string) (int64, error)

func (f skillGitHubStarsClientFunc) RepositoryStars(ctx context.Context, repository string) (int64, error) {
	return f(ctx, repository)
}

type skillStarsCompletion struct {
	skillID      int64
	sourceURL    string
	stars        int64
	fetchedAt    time.Time
	refreshAfter time.Time
}

type skillStarsDeferral struct {
	skillID      int64
	sourceURL    string
	refreshAfter time.Time
}

type skillStarsRepositoryStub struct {
	SkillMarketRepository

	mu           sync.Mutex
	targets      []SkillRepositoryStarsTarget
	claimCalls   int
	claimLimit   int
	claimedUntil time.Time
	claimHook    func(int)
	completions  []skillStarsCompletion
	deferrals    []skillStarsDeferral
}

func (r *skillStarsRepositoryStub) ClaimRepositoryStarsRefresh(
	_ context.Context,
	limit int,
	claimedUntil time.Time,
) ([]SkillRepositoryStarsTarget, error) {
	r.mu.Lock()
	r.claimCalls++
	call := r.claimCalls
	r.claimLimit = limit
	r.claimedUntil = claimedUntil
	targets := append([]SkillRepositoryStarsTarget(nil), r.targets...)
	r.targets = nil
	hook := r.claimHook
	r.mu.Unlock()
	if hook != nil {
		hook(call)
	}
	return targets, nil
}

func (r *skillStarsRepositoryStub) CompleteRepositoryStarsRefresh(
	_ context.Context,
	skillID int64,
	sourceURL string,
	stars int64,
	fetchedAt time.Time,
	refreshAfter time.Time,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completions = append(r.completions, skillStarsCompletion{
		skillID: skillID, sourceURL: sourceURL, stars: stars,
		fetchedAt: fetchedAt, refreshAfter: refreshAfter,
	})
	return nil
}

func (r *skillStarsRepositoryStub) DeferRepositoryStarsRefresh(
	_ context.Context,
	skillID int64,
	sourceURL string,
	refreshAfter time.Time,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deferrals = append(r.deferrals, skillStarsDeferral{
		skillID: skillID, sourceURL: sourceURL, refreshAfter: refreshAfter,
	})
	return nil
}

func TestGitHubRepositoryStarsClientUsesFixedEndpointAndAllowsZeroStars(t *testing.T) {
	client := newGitHubRepositoryStarsClient(time.Second)
	client.client.Transport = skillGitHubRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "https", req.URL.Scheme)
		require.Equal(t, "api.github.com", req.URL.Host)
		require.Equal(t, "/repos/anthropics/skills", req.URL.Path)
		require.Equal(t, "application/vnd.github+json", req.Header.Get("Accept"))
		require.Equal(t, githubRESTAPIVersion, req.Header.Get("X-GitHub-Api-Version"))
		require.Equal(t, "sub2api-skill-market", req.Header.Get("User-Agent"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"stargazers_count":0}`)),
			Request:    req,
		}, nil
	})

	stars, err := client.RepositoryStars(context.Background(), "anthropics/skills")
	require.NoError(t, err)
	require.Zero(t, stars)
}

func TestGitHubRepositoryStarsClientDistinguishesNotFoundAndRateLimit(t *testing.T) {
	now := time.Date(2026, time.August, 5, 2, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		status      int
		headers     http.Header
		wantRetryAt time.Time
		wantLimited bool
	}{
		{name: "404 or private repository", status: http.StatusNotFound, headers: make(http.Header)},
		{
			name: "429 retry after", status: http.StatusTooManyRequests,
			headers:     http.Header{"Retry-After": []string{"120"}},
			wantRetryAt: now.Add(2 * time.Minute), wantLimited: true,
		},
		{
			name: "403 rate limit reset", status: http.StatusForbidden,
			headers: func() http.Header {
				headers := make(http.Header)
				headers.Set("X-RateLimit-Reset", "1785898800")
				return headers
			}(),
			wantRetryAt: time.Unix(1785898800, 0).UTC(), wantLimited: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newGitHubRepositoryStarsClient(time.Second)
			client.now = func() time.Time { return now }
			client.client.Transport = skillGitHubRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tt.status,
					Header:     tt.headers,
					Body:       io.NopCloser(strings.NewReader(`{"message":"unavailable"}`)),
					Request:    req,
				}, nil
			})

			_, err := client.RepositoryStars(context.Background(), "anthropics/skills")
			require.Error(t, err)
			var limited *githubRateLimitError
			if tt.wantLimited {
				require.ErrorAs(t, err, &limited)
				require.Equal(t, tt.wantRetryAt, limited.RetryAt)
			} else {
				require.False(t, errors.As(err, &limited))
			}
		})
	}
}

func TestSkillRepositoryStarsRefreshUsesSnapshotTTLsAndFailureBackoff(t *testing.T) {
	now := time.Date(2026, time.August, 5, 3, 0, 0, 0, time.UTC)
	rateLimitRetryAt := now.Add(7 * time.Hour)
	repo := &skillStarsRepositoryStub{targets: []SkillRepositoryStarsTarget{
		{ID: 1, SourceURL: "https://github.com/zero/repo", SourceRepository: "zero/repo"},
		{ID: 2, SourceURL: "https://github.com/private/repo", SourceRepository: "private/repo"},
		{ID: 3, SourceURL: "https://github.com/limited/repo", SourceRepository: "limited/repo"},
		{ID: 4, SourceURL: "https://github.com/timeout/repo", SourceRepository: "timeout/repo"},
	}}
	market := NewSkillMarketService(repo, nil)
	market.stars.now = func() time.Time { return now }
	market.stars.requestTimeout = 10 * time.Millisecond
	market.stars.client = skillGitHubStarsClientFunc(func(ctx context.Context, repository string) (int64, error) {
		switch repository {
		case "zero/repo":
			return 0, nil
		case "private/repo":
			return 0, errors.New("github repository returned status 404")
		case "limited/repo":
			return 0, &githubRateLimitError{StatusCode: http.StatusTooManyRequests, RetryAt: rateLimitRetryAt}
		case "timeout/repo":
			<-ctx.Done()
			return 0, ctx.Err()
		default:
			return 0, errors.New("unexpected repository")
		}
	})

	market.refreshRepositoryStars(context.Background())

	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Equal(t, skillRepositoryStarsBatchSize, repo.claimLimit)
	require.Equal(t, now.Add(skillRepositoryStarsClaimTTL), repo.claimedUntil)
	require.Equal(t, []skillStarsCompletion{{
		skillID: 1, sourceURL: "https://github.com/zero/repo", stars: 0,
		fetchedAt: now, refreshAfter: now.Add(skillRepositoryStarsSuccessTTL),
	}}, repo.completions)
	require.ElementsMatch(t, []skillStarsDeferral{
		{skillID: 2, sourceURL: "https://github.com/private/repo", refreshAfter: now.Add(skillRepositoryStarsFailureBackoff)},
		{skillID: 3, sourceURL: "https://github.com/limited/repo", refreshAfter: rateLimitRetryAt},
		{skillID: 4, sourceURL: "https://github.com/timeout/repo", refreshAfter: now.Add(skillRepositoryStarsFailureBackoff)},
	}, repo.deferrals)
}

func TestSkillRepositoryStarsRefreshLimitsConcurrency(t *testing.T) {
	targets := make([]SkillRepositoryStarsTarget, 4)
	for i := range targets {
		targets[i] = SkillRepositoryStarsTarget{
			ID: int64(i + 1), SourceURL: "https://github.com/example/repo",
			SourceRepository: "example/repo",
		}
	}
	repo := &skillStarsRepositoryStub{targets: targets}
	market := NewSkillMarketService(repo, nil)
	market.stars.requestTimeout = time.Second

	release := make(chan struct{})
	twoStarted := make(chan struct{})
	var mu sync.Mutex
	active, maxActive, started := 0, 0, 0
	market.stars.client = skillGitHubStarsClientFunc(func(context.Context, string) (int64, error) {
		mu.Lock()
		active++
		started++
		if active > maxActive {
			maxActive = active
		}
		if started == skillRepositoryStarsConcurrency {
			close(twoStarted)
		}
		mu.Unlock()
		<-release
		mu.Lock()
		active--
		mu.Unlock()
		return 1, nil
	})

	done := make(chan struct{})
	go func() {
		market.refreshRepositoryStars(context.Background())
		close(done)
	}()
	select {
	case <-twoStarted:
	case <-time.After(time.Second):
		t.Fatal("two refresh workers did not start")
	}
	mu.Lock()
	require.Equal(t, skillRepositoryStarsConcurrency, started)
	require.Equal(t, skillRepositoryStarsConcurrency, maxActive)
	mu.Unlock()
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("repository stars refresh did not finish")
	}
	mu.Lock()
	require.Equal(t, skillRepositoryStarsConcurrency, maxActive)
	mu.Unlock()
}

func TestSkillRepositoryStarsWorkerCanRestartAfterStop(t *testing.T) {
	claims := make(chan int, 2)
	repo := &skillStarsRepositoryStub{claimHook: func(call int) { claims <- call }}
	market := NewSkillMarketService(repo, nil)
	market.stars.sweepInterval = time.Hour

	waitForClaim := func(want int) {
		t.Helper()
		select {
		case got := <-claims:
			require.Equal(t, want, got)
		case <-time.After(time.Second):
			t.Fatalf("worker claim %d did not run", want)
		}
	}
	stop := func() {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, market.Stop(ctx))
	}

	require.NoError(t, market.Start(context.Background()))
	require.NoError(t, market.Start(context.Background()))
	waitForClaim(1)
	stop()

	require.NoError(t, market.Start(context.Background()))
	waitForClaim(2)
	stop()
	require.NoError(t, market.Stop(context.Background()))
}
