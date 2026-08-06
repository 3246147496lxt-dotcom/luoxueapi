package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	skillRepositoryStarsSweepInterval  = 15 * time.Minute
	skillRepositoryStarsBatchSize      = 20
	skillRepositoryStarsConcurrency    = 2
	skillRepositoryStarsRequestTimeout = 3 * time.Second
	skillRepositoryStarsSuccessTTL     = 24 * time.Hour
	skillRepositoryStarsFailureBackoff = time.Hour
	skillRepositoryStarsClaimTTL       = 15 * time.Minute

	githubRESTAPIVersion = "2026-03-10"
)

type skillGitHubStarsClient interface {
	RepositoryStars(ctx context.Context, repository string) (int64, error)
}

type githubRepositoryStarsClient struct {
	client *http.Client
	now    func() time.Time
}

func newGitHubRepositoryStarsClient(timeout time.Duration) *githubRepositoryStarsClient {
	return &githubRepositoryStarsClient{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		now: time.Now,
	}
}

type githubRateLimitError struct {
	StatusCode int
	RetryAt    time.Time
}

func (e *githubRateLimitError) Error() string {
	return fmt.Sprintf("github repository rate limited with status %d", e.StatusCode)
}

func (c *githubRepositoryStarsClient) RepositoryStars(ctx context.Context, repository string) (int64, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || !githubOwnerPattern.MatchString(parts[0]) || !githubRepositoryPattern.MatchString(parts[1]) {
		return 0, errors.New("invalid github repository")
	}
	endpoint := (&url.URL{
		Scheme: "https",
		Host:   "api.github.com",
		Path:   "/repos/" + parts[0] + "/" + parts[1],
	}).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("create github repository request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubRESTAPIVersion)
	req.Header.Set("User-Agent", "sub2api-skill-market")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch github repository: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		if retryAt, ok := githubRateLimitRetryAt(resp, c.now().UTC()); ok {
			return 0, &githubRateLimitError{StatusCode: resp.StatusCode, RetryAt: retryAt}
		}
		return 0, fmt.Errorf("github repository returned status %d", resp.StatusCode)
	}
	var payload struct {
		Stars *int64 `json:"stargazers_count"`
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 64<<10))
	if err := decoder.Decode(&payload); err != nil {
		return 0, fmt.Errorf("decode github repository: %w", err)
	}
	if payload.Stars == nil || *payload.Stars < 0 {
		return 0, errors.New("github repository returned invalid stars")
	}
	return *payload.Stars, nil
}

func githubRateLimitRetryAt(resp *http.Response, now time.Time) (time.Time, bool) {
	if resp == nil || (resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests) {
		return time.Time{}, false
	}
	if raw := strings.TrimSpace(resp.Header.Get("Retry-After")); raw != "" {
		if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil && seconds >= 0 {
			return now.Add(time.Duration(seconds) * time.Second), true
		}
		if retryAt, err := http.ParseTime(raw); err == nil {
			return retryAt.UTC(), true
		}
	}
	if raw := strings.TrimSpace(resp.Header.Get("X-RateLimit-Reset")); raw != "" {
		if unixSeconds, err := strconv.ParseInt(raw, 10, 64); err == nil && unixSeconds >= 0 {
			return time.Unix(unixSeconds, 0).UTC(), true
		}
	}
	return time.Time{}, false
}

type skillRepositoryStarsRuntime struct {
	client         skillGitHubStarsClient
	sweepInterval  time.Duration
	requestTimeout time.Duration
	successTTL     time.Duration
	failureBackoff time.Duration
	claimTTL       time.Duration
	batchSize      int
	concurrency    int
	now            func() time.Time

	lifecycleMu      sync.Mutex
	lifecycleCancel  context.CancelFunc
	lifecycleDone    chan struct{}
	lifecycleStarted bool
}

func newSkillRepositoryStarsRuntime() skillRepositoryStarsRuntime {
	return skillRepositoryStarsRuntime{
		client:         newGitHubRepositoryStarsClient(skillRepositoryStarsRequestTimeout),
		sweepInterval:  skillRepositoryStarsSweepInterval,
		requestTimeout: skillRepositoryStarsRequestTimeout,
		successTTL:     skillRepositoryStarsSuccessTTL,
		failureBackoff: skillRepositoryStarsFailureBackoff,
		claimTTL:       skillRepositoryStarsClaimTTL,
		batchSize:      skillRepositoryStarsBatchSize,
		concurrency:    skillRepositoryStarsConcurrency,
		now:            time.Now,
	}
}

// Start owns the periodic GitHub projection worker. Refresh failures are
// deliberately degraded to logs: publication and public reads only consume the
// latest database snapshot and never wait on GitHub.
func (s *SkillMarketService) Start(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return nil
	}
	s.stars.lifecycleMu.Lock()
	if s.stars.lifecycleStarted {
		s.stars.lifecycleMu.Unlock()
		return nil
	}
	if s.stars.lifecycleDone != nil {
		select {
		case <-s.stars.lifecycleDone:
			s.stars.lifecycleCancel = nil
			s.stars.lifecycleDone = nil
		default:
			s.stars.lifecycleMu.Unlock()
			return errors.New("skill repository stars worker is still stopping")
		}
	}
	runtimeCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.stars.lifecycleCancel = cancel
	s.stars.lifecycleDone = done
	s.stars.lifecycleStarted = true
	s.stars.lifecycleMu.Unlock()

	go func() {
		defer close(done)
		s.refreshRepositoryStars(runtimeCtx)
		ticker := time.NewTicker(s.stars.sweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-runtimeCtx.Done():
				return
			case <-ticker.C:
				s.refreshRepositoryStars(runtimeCtx)
			}
		}
	}()
	return nil
}

// Stop cancels and joins the repository projection worker. It is idempotent.
func (s *SkillMarketService) Stop(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.stars.lifecycleMu.Lock()
	cancel := s.stars.lifecycleCancel
	done := s.stars.lifecycleDone
	s.stars.lifecycleStarted = false
	s.stars.lifecycleMu.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		s.stars.lifecycleMu.Lock()
		if s.stars.lifecycleDone == done {
			s.stars.lifecycleCancel = nil
			s.stars.lifecycleDone = nil
		}
		s.stars.lifecycleMu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *SkillMarketService) refreshRepositoryStars(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	now := s.stars.now().UTC()
	targets, err := s.repo.ClaimRepositoryStarsRefresh(ctx, s.stars.batchSize, now.Add(s.stars.claimTTL))
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("claim skill repository stars refresh failed", "err", err)
		}
		return
	}
	if len(targets) == 0 {
		return
	}

	workerCount := s.stars.concurrency
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(targets) {
		workerCount = len(targets)
	}
	jobs := make(chan SkillRepositoryStarsTarget)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for target := range jobs {
				s.refreshRepositoryStarsTarget(ctx, target)
			}
		}()
	}
	for _, target := range targets {
		select {
		case jobs <- target:
		case <-ctx.Done():
			close(jobs)
			workers.Wait()
			return
		}
	}
	close(jobs)
	workers.Wait()
}

func (s *SkillMarketService) refreshRepositoryStarsTarget(ctx context.Context, target SkillRepositoryStarsTarget) {
	requestCtx, cancel := context.WithTimeout(ctx, s.stars.requestTimeout)
	stars, err := s.stars.client.RepositoryStars(requestCtx, target.SourceRepository)
	cancel()
	now := s.stars.now().UTC()
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		refreshAfter := now.Add(s.stars.failureBackoff)
		var rateLimitErr *githubRateLimitError
		if errors.As(err, &rateLimitErr) && rateLimitErr.RetryAt.After(now) {
			refreshAfter = rateLimitErr.RetryAt
		}
		if deferErr := s.repo.DeferRepositoryStarsRefresh(ctx, target.ID, target.SourceURL, refreshAfter); deferErr != nil {
			slog.Warn("defer skill repository stars refresh failed", "skill_id", target.ID, "err", deferErr)
		}
		return
	}
	if err := s.repo.CompleteRepositoryStarsRefresh(
		ctx,
		target.ID,
		target.SourceURL,
		stars,
		now,
		now.Add(s.stars.successTTL),
	); err != nil && ctx.Err() == nil {
		slog.Warn("complete skill repository stars refresh failed", "skill_id", target.ID, "err", err)
	}
}
