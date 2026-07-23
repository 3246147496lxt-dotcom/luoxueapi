package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type embeddedPageSettingsStub struct {
	mu  sync.RWMutex
	raw string
}

func (s *embeddedPageSettingsStub) GetCustomMenuItemsRaw(context.Context) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raw
}

func (s *embeddedPageSettingsStub) set(raw string) {
	s.mu.Lock()
	s.raw = raw
	s.mu.Unlock()
}

func embeddedPageMenus() string {
	return `[
		{"id":"pay","url":"https://PAY.example.com/checkout?keep=yes&token=legacy&user_id=99&src_url=https%3A%2F%2Fsource.invalid%2Fprivate#token=legacy-fragment","visibility":"user","auth_mode":"exchange_code"},
		{"id":"admin-pay","url":"https://admin-pay.example.com/","visibility":"admin","auth_mode":"exchange_code"},
		{"id":"plain","url":"https://plain.example.com/","visibility":"user","auth_mode":"none"},
		{"id":"insecure","url":"http://pay.example.com/","visibility":"user","auth_mode":"exchange_code"},
		{"id":"markdown","url":"md:terms","visibility":"user","auth_mode":"none"}
	]`
}

type fakeEmbeddedPageLaunchRecord struct {
	payload   []byte
	expiresAt time.Time
}

type fakeEmbeddedPageLaunchStore struct {
	mu         sync.Mutex
	now        func() time.Time
	records    map[string]fakeEmbeddedPageLaunchRecord
	putErr     error
	consumeErr error
}

func newFakeEmbeddedPageLaunchStore(now func() time.Time) *fakeEmbeddedPageLaunchStore {
	return &fakeEmbeddedPageLaunchStore{
		now:     now,
		records: make(map[string]fakeEmbeddedPageLaunchRecord),
	}
}

func (s *fakeEmbeddedPageLaunchStore) Put(_ context.Context, digest string, payload []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.putErr != nil {
		return s.putErr
	}
	payloadCopy := append([]byte(nil), payload...)
	s.records[digest] = fakeEmbeddedPageLaunchRecord{
		payload:   payloadCopy,
		expiresAt: s.now().Add(ttl),
	}
	return nil
}

func (s *fakeEmbeddedPageLaunchStore) Consume(_ context.Context, digest string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.consumeErr != nil {
		return nil, false, s.consumeErr
	}
	record, found := s.records[digest]
	if !found {
		return nil, false, nil
	}
	delete(s.records, digest)
	if !record.expiresAt.After(s.now()) {
		return nil, false, nil
	}
	return append([]byte(nil), record.payload...), true, nil
}

func (s *fakeEmbeddedPageLaunchStore) observation(digest string) ([]byte, time.Duration, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, found := s.records[digest]
	if !found {
		return nil, 0, false
	}
	return append([]byte(nil), record.payload...), record.expiresAt.Sub(s.now()), true
}

func (s *fakeEmbeddedPageLaunchStore) setErrors(putErr, consumeErr error) {
	s.mu.Lock()
	s.putErr = putErr
	s.consumeErr = consumeErr
	s.mu.Unlock()
}

func newEmbeddedPageLaunchTestService(t *testing.T) (*EmbeddedPageLaunchService, *embeddedPageSettingsStub, *fakeEmbeddedPageLaunchStore, *time.Time) {
	t.Helper()
	settings := &embeddedPageSettingsStub{raw: embeddedPageMenus()}
	now := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }
	store := newFakeEmbeddedPageLaunchStore(nowFn)
	svc := &EmbeddedPageLaunchService{
		settings: settings,
		store:    store,
		now:      nowFn,
		random:   rand.Reader,
	}
	return svc, settings, store, &now
}

func launchCodeFromURL(t *testing.T, launchURL string) string {
	t.Helper()
	parsed, err := url.Parse(launchURL)
	require.NoError(t, err)
	return parsed.Query().Get("s2a_launch_code")
}

func TestEmbeddedPageLaunchIssueStoresOnlyDigestAndSanitizesURL(t *testing.T) {
	svc, _, store, _ := newEmbeddedPageLaunchTestService(t)

	result, err := svc.Issue(context.Background(), 123, "user", "pay", EmbeddedPageLaunchOptions{
		Theme:      "dark",
		Lang:       "zh-CN",
		UIMode:     "embedded",
		SourceHost: "https://API.Example.com/",
	})
	require.NoError(t, err)
	require.Equal(t, CustomMenuAuthModeExchangeCode, result.AuthMode)
	require.Equal(t, 60, result.ExpiresIn)

	parsed, err := url.Parse(result.LaunchURL)
	require.NoError(t, err)
	query := parsed.Query()
	require.Equal(t, "https", parsed.Scheme)
	require.Equal(t, "pay.example.com", parsed.Host)
	require.Empty(t, parsed.Fragment)
	require.Empty(t, query.Get("keep"))
	require.Empty(t, query.Get("token"))
	require.Empty(t, query.Get("user_id"))
	require.Empty(t, query.Get("src_url"))
	require.Equal(t, "dark", query.Get("theme"))
	require.Equal(t, "zh-CN", query.Get("lang"))
	require.Equal(t, "embedded", query.Get("ui_mode"))
	require.Equal(t, "https://api.example.com", query.Get("src_host"))
	require.Equal(t, "pay", query.Get("s2a_client_id"))
	require.ElementsMatch(t, []string{
		"theme", "lang", "ui_mode", "src_host", "s2a_client_id", "s2a_launch_code",
	}, queryKeys(query))

	code := query.Get("s2a_launch_code")
	decoded, err := base64.RawURLEncoding.DecodeString(code)
	require.NoError(t, err)
	require.Len(t, decoded, 32)

	digest := embeddedPageLaunchDigest(code)
	require.NotContains(t, digest, code)
	stored, ttl, found := store.observation(digest)
	require.True(t, found)
	require.Equal(t, EmbeddedPageLaunchTTL, ttl)
	require.NotContains(t, string(stored), code)
	require.NotContains(t, string(stored), "/checkout")

	var ticket embeddedPageLaunchTicket
	require.NoError(t, json.Unmarshal(stored, &ticket))
	require.Equal(t, int64(123), ticket.UserID)
	require.Equal(t, "pay", ticket.MenuItemID)
	require.Equal(t, "https://pay.example.com", ticket.TargetOrigin)
}

func queryKeys(values url.Values) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func TestEmbeddedPageLaunchExchangeSucceedsExactlyOnce(t *testing.T) {
	svc, _, _, _ := newEmbeddedPageLaunchTestService(t)
	launch, err := svc.Issue(context.Background(), 456, "user", "pay", EmbeddedPageLaunchOptions{})
	require.NoError(t, err)
	code := launchCodeFromURL(t, launch.LaunchURL)

	identity, err := svc.Exchange(context.Background(), "pay", code)
	require.NoError(t, err)
	require.Equal(t, &EmbeddedPageIdentity{UserID: 456, MenuItemID: "pay"}, identity)

	_, err = svc.Exchange(context.Background(), "pay", code)
	require.Equal(t, "INVALID_EMBED_LAUNCH_CODE", infraerrors.Reason(err))
}

func TestEmbeddedPageLaunchConcurrentExchangeHasSingleWinner(t *testing.T) {
	svc, _, _, _ := newEmbeddedPageLaunchTestService(t)
	launch, err := svc.Issue(context.Background(), 789, "user", "pay", EmbeddedPageLaunchOptions{})
	require.NoError(t, err)
	code := launchCodeFromURL(t, launch.LaunchURL)

	var successes atomic.Int32
	var invalid atomic.Int32
	var wg sync.WaitGroup
	for range 24 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, exchangeErr := svc.Exchange(context.Background(), "pay", code)
			switch infraerrors.Reason(exchangeErr) {
			case "":
				successes.Add(1)
			case "INVALID_EMBED_LAUNCH_CODE":
				invalid.Add(1)
			default:
				t.Errorf("unexpected exchange error: %v", exchangeErr)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int32(1), successes.Load())
	require.Equal(t, int32(23), invalid.Load())
}

func TestEmbeddedPageLaunchInvalidCasesAreIndistinguishable(t *testing.T) {
	t.Run("forged", func(t *testing.T) {
		svc, _, _, _ := newEmbeddedPageLaunchTestService(t)
		forged := make([]byte, 32)
		_, err := rand.Read(forged)
		require.NoError(t, err)
		_, err = svc.Exchange(context.Background(), "pay", base64.RawURLEncoding.EncodeToString(forged))
		require.Equal(t, "INVALID_EMBED_LAUNCH_CODE", infraerrors.Reason(err))
	})

	t.Run("expired", func(t *testing.T) {
		svc, _, _, now := newEmbeddedPageLaunchTestService(t)
		launch, err := svc.Issue(context.Background(), 1, "user", "pay", EmbeddedPageLaunchOptions{})
		require.NoError(t, err)
		code := launchCodeFromURL(t, launch.LaunchURL)
		*now = now.Add(EmbeddedPageLaunchTTL + time.Second)
		_, err = svc.Exchange(context.Background(), "pay", code)
		require.Equal(t, "INVALID_EMBED_LAUNCH_CODE", infraerrors.Reason(err))
	})

	t.Run("menu mismatch consumes ticket", func(t *testing.T) {
		svc, _, _, _ := newEmbeddedPageLaunchTestService(t)
		launch, err := svc.Issue(context.Background(), 1, "user", "pay", EmbeddedPageLaunchOptions{})
		require.NoError(t, err)
		code := launchCodeFromURL(t, launch.LaunchURL)
		_, err = svc.Exchange(context.Background(), "other", code)
		require.Equal(t, "INVALID_EMBED_LAUNCH_CODE", infraerrors.Reason(err))
		_, err = svc.Exchange(context.Background(), "pay", code)
		require.Equal(t, "INVALID_EMBED_LAUNCH_CODE", infraerrors.Reason(err))
	})

	t.Run("menu origin changed", func(t *testing.T) {
		svc, settings, _, _ := newEmbeddedPageLaunchTestService(t)
		launch, err := svc.Issue(context.Background(), 1, "user", "pay", EmbeddedPageLaunchOptions{})
		require.NoError(t, err)
		settings.set(strings.Replace(embeddedPageMenus(), "https://PAY.example.com/checkout", "https://other.example.com/checkout", 1))
		_, err = svc.Exchange(context.Background(), "pay", launchCodeFromURL(t, launch.LaunchURL))
		require.Equal(t, "INVALID_EMBED_LAUNCH_CODE", infraerrors.Reason(err))
	})
}

func TestEmbeddedPageLaunchEnforcesMenuPolicy(t *testing.T) {
	svc, _, _, _ := newEmbeddedPageLaunchTestService(t)

	_, err := svc.Issue(context.Background(), 1, "user", "admin-pay", EmbeddedPageLaunchOptions{})
	require.Equal(t, "CUSTOM_PAGE_FORBIDDEN", infraerrors.Reason(err))
	_, err = svc.Issue(context.Background(), 1, "admin", "admin-pay", EmbeddedPageLaunchOptions{})
	require.NoError(t, err)

	_, err = svc.Issue(context.Background(), 1, "user", "plain", EmbeddedPageLaunchOptions{})
	require.Equal(t, "CUSTOM_PAGE_AUTH_MODE_UNSUPPORTED", infraerrors.Reason(err))
	_, err = svc.Issue(context.Background(), 1, "user", "insecure", EmbeddedPageLaunchOptions{})
	require.Equal(t, "CUSTOM_PAGE_HTTPS_REQUIRED", infraerrors.Reason(err))
	_, err = svc.Issue(context.Background(), 1, "user", "missing", EmbeddedPageLaunchOptions{})
	require.Equal(t, "CUSTOM_PAGE_NOT_FOUND", infraerrors.Reason(err))
}

func TestEmbeddedPageLaunchStoreFailuresReturnServiceUnavailable(t *testing.T) {
	svc, _, store, _ := newEmbeddedPageLaunchTestService(t)
	storeFailure := errors.New("store unavailable")
	store.setErrors(storeFailure, nil)

	_, err := svc.Issue(context.Background(), 1, "user", "pay", EmbeddedPageLaunchOptions{})
	require.Equal(t, "EMBEDDED_PAGE_LAUNCH_UNAVAILABLE", infraerrors.Reason(err))

	store.setErrors(nil, storeFailure)
	validShapedCode := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	_, err = svc.Exchange(context.Background(), "pay", validShapedCode)
	require.Equal(t, "EMBEDDED_PAGE_LAUNCH_UNAVAILABLE", infraerrors.Reason(err))
}

func TestEmbeddedPageLaunchNilStoreReturnsServiceUnavailable(t *testing.T) {
	settings := &embeddedPageSettingsStub{raw: embeddedPageMenus()}
	svc := NewEmbeddedPageLaunchService(nil, nil)
	svc.settings = settings

	_, err := svc.Issue(context.Background(), 1, "user", "pay", EmbeddedPageLaunchOptions{})
	require.Equal(t, "EMBEDDED_PAGE_LAUNCH_UNAVAILABLE", infraerrors.Reason(err))

	validShapedCode := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	_, err = svc.Exchange(context.Background(), "pay", validShapedCode)
	require.Equal(t, "EMBEDDED_PAGE_LAUNCH_UNAVAILABLE", infraerrors.Reason(err))
}
