package skillimport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestSecureFetcherReturnsTypedRetryAfter(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return response(request, http.StatusTooManyRequests, http.Header{"Retry-After": []string{"120"}}, "slow down"), nil
	})}
	fetcher := testSecureFetcher(client)
	_, err := fetcher.Get(context.Background(), "https://api.example.test/data", FetchOptions{
		Adapter: "fixture", Operation: "list", AllowedHosts: []string{"api.example.test"},
	})
	if ErrorKindOf(err) != ErrorRateLimited {
		t.Fatalf("ErrorKindOf() = %q, want %q (err=%v)", ErrorKindOf(err), ErrorRateLimited, err)
	}
	retryAfter, ok := RetryAfterOf(err)
	if !ok || retryAfter != 2*time.Minute {
		t.Fatalf("RetryAfterOf() = %s, %v; want 2m, true", retryAfter, ok)
	}
}

func TestRetryAfterIsBoundedAgainstIndefiniteHostCircuit(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	if got := parseRetryAfter("999999999999", now); got != maxRetryAfter {
		t.Fatalf("large numeric Retry-After = %s, want %s", got, maxRetryAfter)
	}
	if got := parseRetryAfter(now.Add(72*time.Hour).Format(http.TimeFormat), now); got != maxRetryAfter {
		t.Fatalf("large date Retry-After = %s, want %s", got, maxRetryAfter)
	}
}

func TestStatusErrorRecognizesGitHubRateLimitReset(t *testing.T) {
	now := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	request := &http.Request{URL: mustURL("https://api.github.com/repos/example/repo")}
	resetAt := now.Add(5 * time.Minute)
	headers := make(http.Header)
	headers.Set("X-RateLimit-Remaining", "0")
	headers.Set("X-RateLimit-Reset", fmt.Sprint(resetAt.Unix()))
	upstream := response(request, http.StatusForbidden, headers, "rate limited")
	err := statusError("github", "read tree", upstream, now)
	if ErrorKindOf(err) != ErrorRateLimited {
		t.Fatalf("kind = %q, want rate_limited (err=%v)", ErrorKindOf(err), err)
	}
	if retryAfter, ok := RetryAfterOf(err); !ok || retryAfter != 5*time.Minute {
		t.Fatalf("retry after = %s, %v; want 5m", retryAfter, ok)
	}
}

func TestSecureFetcherBlocksPrivateTargetsAndRedirects(t *testing.T) {
	fetcher := testSecureFetcher(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Hostname() == "public.example.test" {
			return response(request, http.StatusFound, http.Header{"Location": []string{"https://127.0.0.1/secret"}}, ""), nil
		}
		return nil, errors.New("private redirect must never reach the transport")
	})})

	_, err := fetcher.Get(context.Background(), "https://127.0.0.1/secret", FetchOptions{AllowedHosts: []string{"127.0.0.1"}})
	if ErrorKindOf(err) != ErrorUnsafe {
		t.Fatalf("literal private IP kind = %q, want unsafe (err=%v)", ErrorKindOf(err), err)
	}
	_, err = fetcher.Get(context.Background(), "https://public.example.test/start", FetchOptions{
		AllowedHosts: []string{"public.example.test", "127.0.0.1"},
	})
	if ErrorKindOf(err) != ErrorUnsafe {
		t.Fatalf("private redirect kind = %q, want unsafe (err=%v)", ErrorKindOf(err), err)
	}

	_, err = fetcher.Get(context.Background(), "https://public.example.test/start", FetchOptions{})
	if ErrorKindOf(err) != ErrorUnsafe {
		t.Fatalf("missing allowlist kind = %q, want unsafe (err=%v)", ErrorKindOf(err), err)
	}
}

func TestSecureFetcherStripsAuthorizationAndMovesHostGateOnRedirect(t *testing.T) {
	var redirectedAuthorization, redirectedAPIKey, redirectedCookie string
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Hostname() {
		case "source.example.test":
			return response(request, http.StatusFound, http.Header{"Location": []string{"https://artifact.example.test/file"}}, ""), nil
		case "artifact.example.test":
			redirectedAuthorization = request.Header.Get("Authorization")
			redirectedAPIKey = request.Header.Get("X-Api-Key")
			redirectedCookie = request.Header.Get("Cookie")
			return response(request, http.StatusOK, http.Header{"Content-Type": []string{"text/plain"}}, "skill"), nil
		default:
			return nil, fmt.Errorf("unexpected host %s", request.URL.Hostname())
		}
	})}
	fetcher := testSecureFetcher(client)
	_, err := fetcher.Get(context.Background(), "https://source.example.test/start", FetchOptions{
		AllowedHosts: []string{"source.example.test", "artifact.example.test"},
		Headers: http.Header{
			"Authorization": []string{"Bearer secret"}, "X-Api-Key": []string{"secret"}, "Cookie": []string{"session=secret"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if redirectedAuthorization != "" {
		t.Fatalf("cross-host redirect leaked Authorization %q", redirectedAuthorization)
	}
	if redirectedAPIKey != "" || redirectedCookie != "" {
		t.Fatalf("cross-host redirect leaked sensitive headers api_key=%q cookie=%q", redirectedAPIKey, redirectedCookie)
	}
	for _, host := range []string{"source.example.test", "artifact.example.test"} {
		if inUse := len(fetcher.gateForHost(host).semaphore); inUse != 0 {
			t.Fatalf("host gate %s still has %d slots in use", host, inUse)
		}
	}
}

func TestSecureFetcherCrossHostRedirectCannotDeadlockGlobalAndHostGates(t *testing.T) {
	started := make(chan struct{})
	releaseRedirect := make(chan struct{})
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Hostname() == "a.example.test" {
			select {
			case <-started:
			default:
				close(started)
			}
			<-releaseRedirect
			return response(request, http.StatusFound, http.Header{"Location": []string{"https://b.example.test/redirect-target"}}, ""), nil
		}
		return response(request, http.StatusOK, http.Header{"Content-Type": []string{"text/plain"}}, "ok"), nil
	})}
	fetcher := testSecureFetcher(client)
	fetcher.semaphore = make(chan struct{}, 1)
	fetcher.maxPerHost = 1

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	errorsSeen := make(chan error, 2)
	go func() {
		_, err := fetcher.Get(ctx, "https://a.example.test/start", FetchOptions{AllowedHosts: []string{"a.example.test", "b.example.test"}})
		errorsSeen <- err
	}()
	<-started
	go func() {
		_, err := fetcher.Get(ctx, "https://b.example.test/independent", FetchOptions{AllowedHosts: []string{"b.example.test"}})
		errorsSeen <- err
	}()
	time.Sleep(20 * time.Millisecond)
	close(releaseRedirect)
	for range 2 {
		if err := <-errorsSeen; err != nil {
			t.Fatalf("cross-host request failed: %v", err)
		}
	}
}

func TestSecureFetcherRejectsOversizedBody(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return response(request, http.StatusOK, http.Header{"Content-Type": []string{"text/plain"}}, "12345"), nil
	})}
	fetcher := testSecureFetcher(client)
	_, err := fetcher.Get(context.Background(), "https://files.example.test/value", FetchOptions{
		AllowedHosts: []string{"files.example.test"}, MaxBytes: 4,
	})
	if ErrorKindOf(err) != ErrorBlocked {
		t.Fatalf("oversized body kind = %q, want blocked (err=%v)", ErrorKindOf(err), err)
	}
}

func TestSecureFetcherEnforcesPerHostConcurrency(t *testing.T) {
	fetcher := testSecureFetcher(&http.Client{})
	fetcher.maxPerHost = 1
	firstRelease, err := fetcher.acquireHost(context.Background(), "api.example.test")
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan func(), 1)
	go func() {
		release, acquireErr := fetcher.acquireHost(context.Background(), "api.example.test")
		if acquireErr == nil {
			acquired <- release
		}
	}()
	select {
	case release := <-acquired:
		release()
		t.Fatal("second same-host slot acquired before the first was released")
	case <-time.After(20 * time.Millisecond):
	}
	firstRelease()
	select {
	case release := <-acquired:
		release()
	case <-time.After(time.Second):
		t.Fatal("second same-host slot did not acquire after release")
	}
}

func TestPinnedDialerBlocksPrivateAndCarrierGradeNATAddresses(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "::1", "fc00::1"} {
		if !isBlockedImportIP(net.ParseIP(raw)) {
			t.Errorf("isBlockedImportIP(%s) = false, want true", raw)
		}
	}
	if isBlockedImportIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public address was incorrectly blocked")
	}
}
