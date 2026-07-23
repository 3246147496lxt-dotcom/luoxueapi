//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/model"
)

type blockingAuthCacheSubscriber struct {
	authCacheStub
	started chan struct{}
}

func (s *blockingAuthCacheSubscriber) SubscribeAuthCacheInvalidation(ctx context.Context, _ func(string)) error {
	close(s.started)
	<-ctx.Done()
	return ctx.Err()
}

func TestAPIKeyAuthCacheSubscriberIsCancellableAndWaitable(t *testing.T) {
	cache := &blockingAuthCacheSubscriber{started: make(chan struct{})}
	svc := NewAPIKeyService(nil, nil, nil, nil, nil, cache, &config.Config{
		APIKeyAuth: config.APIKeyAuthCacheConfig{L1Size: 10, L1TTLSeconds: 60},
	})
	if svc.authCacheL1 != nil {
		t.Fatal("API key constructor started the L1 runtime cache")
	}
	svc.StartAuthCacheInvalidationSubscriber(context.Background())

	select {
	case <-cache.started:
	case <-time.After(time.Second):
		t.Fatal("subscriber did not start")
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := svc.StopAuthCacheInvalidationSubscriber(stopCtx); err != nil {
		t.Fatalf("stop subscriber: %v", err)
	}
	if err := svc.StopAuthCacheInvalidationSubscriber(stopCtx); err != nil {
		t.Fatalf("second stop subscriber: %v", err)
	}
}

func TestContentModerationWireProviderIsConstructOnlyAndStopIsIdempotent(t *testing.T) {
	svc := ProvideContentModerationService(
		&contentModerationTestSettingRepo{values: map[string]string{}},
		&contentModerationTestRepo{},
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	if svc.lifecycleStarted {
		t.Fatal("Wire provider started content moderation during construction")
	}
	svc.Start()
	if !svc.lifecycleStarted {
		t.Fatal("content moderation did not start")
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := svc.Stop(stopCtx); err != nil {
		t.Fatalf("stop content moderation: %v", err)
	}
	if err := svc.Stop(stopCtx); err != nil {
		t.Fatalf("second stop content moderation: %v", err)
	}
}

func TestContentModerationStopCanContinueWaitingAfterDeadline(t *testing.T) {
	runtimeCtx, cancelRuntime := context.WithCancel(context.Background())
	releaseWorker := make(chan struct{})
	svc := &ContentModerationService{
		lifecycleCancel:  cancelRuntime,
		lifecycleStarted: true,
	}
	svc.lifecycleWG.Add(1)
	go func() {
		defer svc.lifecycleWG.Done()
		<-releaseWorker
	}()

	deadlineCtx, cancelDeadline := context.WithCancel(context.Background())
	cancelDeadline()
	if err := svc.Stop(deadlineCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("first Stop error = %v, want context.Canceled", err)
	}
	select {
	case <-runtimeCtx.Done():
	default:
		t.Fatal("first Stop did not cancel the worker runtime")
	}

	close(releaseWorker)
	stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
	defer cancelStop()
	if err := svc.Stop(stopCtx); err != nil {
		t.Fatalf("second Stop did not continue waiting for workers: %v", err)
	}
}

func TestIdempotencyWireProviderDoesNotInstallGlobalCoordinator(t *testing.T) {
	previous := DefaultIdempotencyCoordinator()
	SetDefaultIdempotencyCoordinator(nil)
	t.Cleanup(func() { SetDefaultIdempotencyCoordinator(previous) })

	coordinator := ProvideIdempotencyCoordinator(nil, &config.Config{})
	if coordinator == nil {
		t.Fatal("provider returned nil coordinator")
	}
	if DefaultIdempotencyCoordinator() != nil {
		t.Fatal("Wire provider installed a package-global coordinator")
	}
}

func TestSubscriptionConstructorDefersRuntimeCacheUntilStart(t *testing.T) {
	svc := NewSubscriptionService(nil, nil, nil, nil, &config.Config{
		SubscriptionCache: config.SubscriptionCacheConfig{L1Size: 16, L1TTLSeconds: 60},
	})
	if svc.subCacheL1 != nil {
		t.Fatal("constructor started the subscription runtime cache")
	}
	svc.Start()
	if svc.subCacheL1 == nil {
		t.Fatal("Start did not initialize the subscription runtime cache")
	}
	if err := svc.StopContext(context.Background()); err != nil {
		t.Fatalf("stop subscription service: %v", err)
	}
}

type lifecycleTLSProfileRepo struct{ listCalls int }

func (r *lifecycleTLSProfileRepo) List(context.Context) ([]*model.TLSFingerprintProfile, error) {
	r.listCalls++
	return nil, nil
}
func (*lifecycleTLSProfileRepo) GetByID(context.Context, int64) (*model.TLSFingerprintProfile, error) {
	return nil, nil
}
func (*lifecycleTLSProfileRepo) Create(context.Context, *model.TLSFingerprintProfile) (*model.TLSFingerprintProfile, error) {
	return nil, nil
}
func (*lifecycleTLSProfileRepo) Update(context.Context, *model.TLSFingerprintProfile) (*model.TLSFingerprintProfile, error) {
	return nil, nil
}
func (*lifecycleTLSProfileRepo) Delete(context.Context, int64) error { return nil }

type lifecycleTLSProfileCache struct {
	started chan struct{}
	release <-chan struct{}
}

func (*lifecycleTLSProfileCache) Get(context.Context) ([]*model.TLSFingerprintProfile, bool) {
	return nil, false
}
func (*lifecycleTLSProfileCache) Set(context.Context, []*model.TLSFingerprintProfile) error {
	return nil
}
func (*lifecycleTLSProfileCache) Invalidate(context.Context) error   { return nil }
func (*lifecycleTLSProfileCache) NotifyUpdate(context.Context) error { return nil }
func (c *lifecycleTLSProfileCache) SubscribeUpdates(ctx context.Context, _ func()) {
	close(c.started)
	<-ctx.Done()
	if c.release != nil {
		<-c.release
	}
}

func TestTLSFingerprintProfileConstructorIsPureAndSubscriberStops(t *testing.T) {
	repo := &lifecycleTLSProfileRepo{}
	cache := &lifecycleTLSProfileCache{started: make(chan struct{})}
	svc := NewTLSFingerprintProfileService(repo, cache)
	if repo.listCalls != 0 {
		t.Fatal("TLS profile constructor accessed the repository")
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("start TLS profile service: %v", err)
	}
	select {
	case <-cache.started:
	case <-time.After(time.Second):
		t.Fatal("TLS profile subscriber did not start")
	}
	if repo.listCalls != 1 {
		t.Fatalf("TLS profile Start list calls = %d, want 1", repo.listCalls)
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := svc.Stop(stopCtx); err != nil {
		t.Fatalf("stop TLS profile service: %v", err)
	}
	if err := svc.Stop(stopCtx); err != nil {
		t.Fatalf("second stop TLS profile service: %v", err)
	}
}

type lifecycleErrorRuleRepo struct{ listCalls int }

func (r *lifecycleErrorRuleRepo) List(context.Context) ([]*model.ErrorPassthroughRule, error) {
	r.listCalls++
	return nil, nil
}
func (*lifecycleErrorRuleRepo) GetByID(context.Context, int64) (*model.ErrorPassthroughRule, error) {
	return nil, nil
}
func (*lifecycleErrorRuleRepo) Create(context.Context, *model.ErrorPassthroughRule) (*model.ErrorPassthroughRule, error) {
	return nil, nil
}
func (*lifecycleErrorRuleRepo) Update(context.Context, *model.ErrorPassthroughRule) (*model.ErrorPassthroughRule, error) {
	return nil, nil
}
func (*lifecycleErrorRuleRepo) Delete(context.Context, int64) error { return nil }

type lifecycleErrorRuleCache struct {
	started chan struct{}
	release <-chan struct{}
}

func (*lifecycleErrorRuleCache) Get(context.Context) ([]*model.ErrorPassthroughRule, bool) {
	return nil, false
}
func (*lifecycleErrorRuleCache) Set(context.Context, []*model.ErrorPassthroughRule) error {
	return nil
}
func (*lifecycleErrorRuleCache) Invalidate(context.Context) error   { return nil }
func (*lifecycleErrorRuleCache) NotifyUpdate(context.Context) error { return nil }
func (c *lifecycleErrorRuleCache) SubscribeUpdates(ctx context.Context, _ func()) {
	close(c.started)
	<-ctx.Done()
	if c.release != nil {
		<-c.release
	}
}

func TestErrorPassthroughConstructorIsPureAndSubscriberStops(t *testing.T) {
	repo := &lifecycleErrorRuleRepo{}
	cache := &lifecycleErrorRuleCache{started: make(chan struct{})}
	svc := NewErrorPassthroughService(repo, cache)
	if repo.listCalls != 0 {
		t.Fatal("error passthrough constructor accessed the repository")
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("start error passthrough service: %v", err)
	}
	select {
	case <-cache.started:
	case <-time.After(time.Second):
		t.Fatal("error passthrough subscriber did not start")
	}
	if repo.listCalls != 1 {
		t.Fatalf("error passthrough Start list calls = %d, want 1", repo.listCalls)
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := svc.Stop(stopCtx); err != nil {
		t.Fatalf("stop error passthrough service: %v", err)
	}
	if err := svc.Stop(stopCtx); err != nil {
		t.Fatalf("second stop error passthrough service: %v", err)
	}
}

func TestCacheSubscriberStopCanContinueWaitingAfterDeadline(t *testing.T) {
	t.Run("tls fingerprint profiles", func(t *testing.T) {
		release := make(chan struct{})
		cache := &lifecycleTLSProfileCache{started: make(chan struct{}), release: release}
		svc := NewTLSFingerprintProfileService(&lifecycleTLSProfileRepo{}, cache)
		if err := svc.Start(context.Background()); err != nil {
			t.Fatalf("start TLS profile service: %v", err)
		}
		select {
		case <-cache.started:
		case <-time.After(time.Second):
			t.Fatal("TLS profile subscriber did not start")
		}

		deadlineCtx, cancelDeadline := context.WithCancel(context.Background())
		cancelDeadline()
		if err := svc.Stop(deadlineCtx); !errors.Is(err, context.Canceled) {
			t.Fatalf("first Stop error = %v, want context.Canceled", err)
		}
		close(release)
		if err := svc.Stop(context.Background()); err != nil {
			t.Fatalf("second Stop did not continue waiting for subscriber: %v", err)
		}
	})

	t.Run("error passthrough", func(t *testing.T) {
		release := make(chan struct{})
		cache := &lifecycleErrorRuleCache{started: make(chan struct{}), release: release}
		svc := NewErrorPassthroughService(&lifecycleErrorRuleRepo{}, cache)
		if err := svc.Start(context.Background()); err != nil {
			t.Fatalf("start error passthrough service: %v", err)
		}
		select {
		case <-cache.started:
		case <-time.After(time.Second):
			t.Fatal("error passthrough subscriber did not start")
		}

		deadlineCtx, cancelDeadline := context.WithCancel(context.Background())
		cancelDeadline()
		if err := svc.Stop(deadlineCtx); !errors.Is(err, context.Canceled) {
			t.Fatalf("first Stop error = %v, want context.Canceled", err)
		}
		close(release)
		if err := svc.Stop(context.Background()); err != nil {
			t.Fatalf("second Stop did not continue waiting for subscriber: %v", err)
		}
	})
}
