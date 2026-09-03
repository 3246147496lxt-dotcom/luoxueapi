package skillimport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

const (
	defaultFetchTimeout       = 30 * time.Second
	defaultFetchBytes   int64 = 8 * 1024 * 1024
	maxRedirects              = 5
	maxRetryAfter             = time.Hour
)

type HTTPFetcher interface {
	Get(context.Context, string, FetchOptions) (FetchResult, error)
}

type FetchOptions struct {
	Adapter      string
	Operation    string
	AllowedHosts []string
	MaxBytes     int64
	ContentTypes []string
	Headers      http.Header
}

type FetchResult struct {
	Body     []byte
	Evidence Evidence
}

type SecureFetcherOptions struct {
	Timeout            time.Duration
	MaxResponseBytes   int64
	MaxConcurrent      int
	MaxConnsPerHost    int
	MinIntervalPerHost time.Duration
	UserAgent          string
}

type hostGate struct {
	semaphore chan struct{}
	mu        sync.Mutex
	nextAt    time.Time
}

type SecureFetcher struct {
	client           *http.Client
	timeout          time.Duration
	maxResponseBytes int64
	userAgent        string
	semaphore        chan struct{}
	maxPerHost       int
	minHostInterval  time.Duration
	hostMu           sync.Mutex
	hosts            map[string]*hostGate
	validateTarget   func(string, []string) (string, error)
	now              func() time.Time
}

func NewSecureFetcher(options SecureFetcherOptions) (*SecureFetcher, error) {
	if options.Timeout <= 0 {
		options.Timeout = defaultFetchTimeout
	}
	if options.MaxResponseBytes <= 0 {
		options.MaxResponseBytes = defaultFetchBytes
	}
	if options.MaxResponseBytes > 64*1024*1024 {
		return nil, errors.New("skill import fetch limit cannot exceed 64 MiB")
	}
	if options.MaxConcurrent <= 0 {
		options.MaxConcurrent = 8
	}
	if options.MaxConcurrent > 32 {
		return nil, errors.New("skill import fetch concurrency cannot exceed 32")
	}
	if options.MaxConnsPerHost <= 0 {
		options.MaxConnsPerHost = 4
	}
	if options.MaxConnsPerHost > options.MaxConcurrent || options.MaxConnsPerHost > 16 {
		return nil, errors.New("skill import per-host concurrency must not exceed global concurrency or 16")
	}
	if options.MinIntervalPerHost < 0 || options.MinIntervalPerHost > time.Minute {
		return nil, errors.New("skill import per-host minimum interval must be between 0 and 1 minute")
	}
	if strings.TrimSpace(options.UserAgent) == "" {
		options.UserAgent = "sub2api-skill-import/" + CoreVersion
	}
	client := newPinnedHTTPClient(options.Timeout, options.MaxConnsPerHost)
	return &SecureFetcher{
		client: client, timeout: options.Timeout, maxResponseBytes: options.MaxResponseBytes,
		userAgent: options.UserAgent, semaphore: make(chan struct{}, options.MaxConcurrent),
		maxPerHost: options.MaxConnsPerHost, minHostInterval: options.MinIntervalPerHost,
		hosts:          make(map[string]*hostGate),
		validateTarget: validateHTTPSImportTarget, now: time.Now,
	}, nil
}

// newPinnedHTTPClient resolves each destination once, rejects the whole DNS
// answer if it contains any private/special address, then dials an approved IP
// directly. This closes the validate-then-resolve-again DNS rebinding window.
func newPinnedHTTPClient(timeout time.Duration, maxConnsPerHost int) *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			addresses := make([]net.IP, 0, 4)
			if literal := net.ParseIP(host); literal != nil {
				addresses = append(addresses, literal)
			} else {
				resolved, resolveErr := net.DefaultResolver.LookupIP(ctx, "ip", host)
				if resolveErr != nil {
					return nil, fmt.Errorf("resolve %s: %w", host, resolveErr)
				}
				addresses = append(addresses, resolved...)
			}
			if len(addresses) == 0 {
				return nil, fmt.Errorf("resolve %s: no addresses", host)
			}
			for _, addressIP := range addresses {
				if isBlockedImportIP(addressIP) {
					return nil, fmt.Errorf("resolved IP %s is not allowed", addressIP.String())
				}
			}
			var dialErrors []error
			for _, addressIP := range addresses {
				connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(addressIP.String(), port))
				if dialErr == nil {
					return connection, nil
				}
				dialErrors = append(dialErrors, dialErr)
			}
			return nil, errors.Join(dialErrors...)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   maxConnsPerHost,
		MaxConnsPerHost:       maxConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: timeout,
	}
	return &http.Client{Transport: transport, Timeout: timeout}
}

func isBlockedImportIP(ip net.IP) bool {
	if ip == nil || ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return true
	}
	// Go intentionally does not classify carrier-grade NAT as private, but it
	// is never a valid public import destination and may expose provider-local
	// infrastructure.
	cgnat := &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}
	return cgnat.Contains(ip)
}

func validateHTTPSImportTarget(target string, allowedHosts []string) (string, error) {
	if len(allowedHosts) == 0 {
		return "", errors.New("an explicit outbound host allowlist is required")
	}
	normalized, err := urlvalidator.ValidateHTTPSURL(target, urlvalidator.ValidationOptions{
		AllowedHosts: allowedHosts, RequireAllowlist: true, AllowPrivate: false,
	})
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", err
	}
	if parsed.User != nil {
		return "", errors.New("URL userinfo is not allowed")
	}
	if parsed.Fragment != "" {
		return "", errors.New("URL fragments are not allowed")
	}
	return normalized, nil
}

func (f *SecureFetcher) Get(ctx context.Context, target string, options FetchOptions) (FetchResult, error) {
	if f == nil || f.client == nil || f.validateTarget == nil || f.semaphore == nil || f.maxPerHost <= 0 {
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, ErrorInvalidConfig, errors.New("secure fetcher is not configured"))
	}
	validated, err := f.validateTarget(target, options.AllowedHosts)
	if err != nil {
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, ErrorUnsafe, err)
	}
	parsedTarget, err := url.Parse(validated)
	if err != nil || parsedTarget.Hostname() == "" {
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, ErrorInvalidSource, errors.New("validated target has no host"))
	}
	hostname := strings.ToLower(parsedTarget.Hostname())
	limit := options.MaxBytes
	if limit <= 0 || limit > f.maxResponseBytes {
		limit = f.maxResponseBytes
	}
	// Take the process-wide slot before a host slot. The reverse order can
	// deadlock when one request redirects A→B while another request already
	// holds B's host slot waiting for the global slot.
	select {
	case f.semaphore <- struct{}{}:
		defer func() { <-f.semaphore }()
	case <-ctx.Done():
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, ErrorTemporary, ctx.Err())
	}
	releaseHost, err := f.acquireHost(ctx, hostname)
	if err != nil {
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, ErrorTemporary, err)
	}
	currentHost := hostname
	currentHostRelease := releaseHost
	defer func() {
		if currentHostRelease != nil {
			currentHostRelease()
		}
	}()
	requestCtx := ctx
	var cancel context.CancelFunc
	if f.timeout > 0 {
		requestCtx, cancel = context.WithTimeout(ctx, f.timeout)
		defer cancel()
	}
	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, validated, nil)
	if err != nil {
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, ErrorInvalidSource, err)
	}
	for name, values := range options.Headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	request.Header.Set("User-Agent", f.userAgent)
	request.Header.Set("Accept-Encoding", "identity")

	client := *f.client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return errors.New("too many redirects")
		}
		if req == nil || req.URL == nil {
			return errors.New("redirect has no URL")
		}
		if _, validateErr := f.validateTarget(req.URL.String(), options.AllowedHosts); validateErr != nil {
			return fmt.Errorf("unsafe redirect: %w", validateErr)
		}
		redirectHost := strings.ToLower(req.URL.Hostname())
		if redirectHost == "" {
			return errors.New("redirect has no host")
		}
		previousHost := parsedTarget.Host
		if len(via) > 0 && via[len(via)-1] != nil && via[len(via)-1].URL != nil {
			previousHost = via[len(via)-1].URL.Host
		}
		if !strings.EqualFold(req.URL.Host, previousHost) {
			// Never forward a source credential to a different host, even when an
			// administrator explicitly allowed that redirect destination.
			stripSensitiveRedirectHeaders(req.Header)
		}
		if redirectHost != currentHost {
			// Hold only the gate for the host currently being contacted. Retaining
			// every prior gate can deadlock concurrent A→B and B→A redirects.
			currentHostRelease()
			currentHostRelease = nil
			release, acquireErr := f.acquireHost(req.Context(), redirectHost)
			if acquireErr != nil {
				return fmt.Errorf("rate-limit redirect host: %w", acquireErr)
			}
			currentHost = redirectHost
			currentHostRelease = release
		}
		return nil
	}
	response, err := client.Do(request)
	if err != nil {
		kind := ErrorTemporary
		if strings.Contains(strings.ToLower(err.Error()), "redirect") || strings.Contains(strings.ToLower(err.Error()), "not allowed") {
			kind = ErrorUnsafe
		}
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, kind, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		statusErr := statusError(options.Adapter, options.Operation, response, f.now())
		if retryAfter, ok := RetryAfterOf(statusErr); ok {
			responseHost := hostname
			if response.Request != nil && response.Request.URL != nil && response.Request.URL.Hostname() != "" {
				responseHost = strings.ToLower(response.Request.URL.Hostname())
			}
			f.deferHost(responseHost, retryAfter)
		}
		return FetchResult{}, statusErr
	}
	if response.ContentLength > limit {
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, ErrorBlocked, fmt.Errorf("response exceeds %d bytes", limit))
	}
	contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
	if len(options.ContentTypes) > 0 && !allowedContentType(contentType, options.ContentTypes) {
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, ErrorInvalidSource, fmt.Errorf("unexpected content type %q", contentType))
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if readErr != nil {
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, ErrorTemporary, readErr)
	}
	if int64(len(body)) > limit {
		return FetchResult{}, NewAdapterError(options.Adapter, options.Operation, ErrorBlocked, fmt.Errorf("response exceeds %d bytes", limit))
	}
	now := time.Now().UTC()
	if f.now != nil {
		now = f.now().UTC()
	}
	return FetchResult{Body: body, Evidence: Evidence{
		URL: evidenceURL(response.Request.URL), CapturedAt: now,
		ETag: strings.TrimSpace(response.Header.Get("ETag")), LastModified: strings.TrimSpace(response.Header.Get("Last-Modified")),
		SHA256: digestBytes(body), ByteSize: int64(len(body)), ContentType: contentType,
	}}, nil
}

func stripSensitiveRedirectHeaders(headers http.Header) {
	for name := range headers {
		lower := strings.ToLower(name)
		if lower == "authorization" || lower == "proxy-authorization" || lower == "cookie" || lower == "cookie2" ||
			strings.Contains(lower, "api-key") || strings.Contains(lower, "apikey") || strings.Contains(lower, "access-token") {
			headers.Del(name)
		}
	}
}

func (f *SecureFetcher) gateForHost(host string) *hostGate {
	f.hostMu.Lock()
	defer f.hostMu.Unlock()
	gate := f.hosts[host]
	if gate == nil {
		gate = &hostGate{semaphore: make(chan struct{}, f.maxPerHost)}
		f.hosts[host] = gate
	}
	return gate
}

func (f *SecureFetcher) acquireHost(ctx context.Context, host string) (func(), error) {
	gate := f.gateForHost(host)
	select {
	case gate.semaphore <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	release := func() { <-gate.semaphore }
	for {
		now := time.Now()
		if f.now != nil {
			now = f.now()
		}
		gate.mu.Lock()
		wait := gate.nextAt.Sub(now)
		if wait <= 0 {
			if f.minHostInterval > 0 {
				gate.nextAt = now.Add(f.minHostInterval)
			}
			gate.mu.Unlock()
			return release, nil
		}
		gate.mu.Unlock()
		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			release()
			return nil, ctx.Err()
		}
	}
}

func (f *SecureFetcher) deferHost(host string, delay time.Duration) {
	if delay <= 0 {
		return
	}
	if delay > maxRetryAfter {
		delay = maxRetryAfter
	}
	now := time.Now()
	if f.now != nil {
		now = f.now()
	}
	gate := f.gateForHost(host)
	until := now.Add(delay)
	gate.mu.Lock()
	if until.After(gate.nextAt) {
		gate.nextAt = until
	}
	gate.mu.Unlock()
}

func statusError(adapter, operation string, response *http.Response, now time.Time) error {
	status := 0
	if response != nil {
		status = response.StatusCode
	}
	kind := ErrorTemporary
	switch status {
	case http.StatusUnauthorized:
		kind = ErrorUnauthorized
	case http.StatusForbidden:
		kind = ErrorForbidden
	case http.StatusNotFound:
		kind = ErrorNotFound
	case http.StatusTooManyRequests:
		kind = ErrorRateLimited
	default:
		if status >= 400 && status < 500 {
			kind = ErrorInvalidSource
		}
	}
	typed := &AdapterError{
		Adapter: adapter, Operation: operation, Kind: kind, StatusCode: status,
		Err: fmt.Errorf("upstream returned HTTP %d", status),
	}
	if response != nil {
		typed.RetryAfter = parseRetryAfter(response.Header.Get("Retry-After"), now)
		if status == http.StatusForbidden && typed.RetryAfter <= 0 && strings.TrimSpace(response.Header.Get("X-RateLimit-Remaining")) == "0" {
			if resetUnix, err := strconv.ParseInt(strings.TrimSpace(response.Header.Get("X-RateLimit-Reset")), 10, 64); err == nil {
				resetAt := time.Unix(resetUnix, 0)
				if resetAt.After(now) {
					typed.RetryAfter = min(resetAt.Sub(now), maxRetryAfter)
				}
			}
		}
		if status == http.StatusForbidden && typed.RetryAfter > 0 {
			typed.Kind = ErrorRateLimited
		}
	}
	return typed
}

func parseRetryAfter(raw string, now time.Time) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil && seconds >= 0 {
		if seconds > int64(maxRetryAfter/time.Second) {
			return maxRetryAfter
		}
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(raw); err == nil && at.After(now) {
		return min(at.Sub(now), maxRetryAfter)
	}
	return 0
}

func allowedContentType(raw string, allowed []string) bool {
	mediaType, _, err := mime.ParseMediaType(raw)
	if err != nil {
		mediaType = strings.ToLower(strings.TrimSpace(strings.Split(raw, ";")[0]))
	}
	for _, candidate := range allowed {
		if strings.EqualFold(mediaType, strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func evidenceURL(value *url.URL) string {
	if value == nil {
		return ""
	}
	clone := *value
	clone.User = nil
	clone.RawQuery = ""
	clone.ForceQuery = false
	clone.Fragment = ""
	return clone.String()
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
