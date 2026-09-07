package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type transcriptCyberTestCache struct {
	comboCacheAndStore
	scopes     map[string]bool
	findCalls  int
	scopeCalls int
	findErr    error
	scopeErr   error
	writeErr   error
	lastTTL    time.Duration
}

var _ CyberSessionTranscriptBlockStore = (*transcriptCyberTestCache)(nil)

func (c *transcriptCyberTestCache) SetCyberSessionBlockedKeys(ctx context.Context, scope string, keys []string, ttl time.Duration) error {
	if c.writeErr != nil {
		return c.writeErr
	}
	for _, key := range keys {
		_ = c.store.SetCyberSessionBlocked(ctx, key, ttl)
	}
	if c.scopes == nil {
		c.scopes = map[string]bool{}
	}
	if scope != "" {
		c.scopes[scope] = true
	}
	c.lastTTL = ttl
	return nil
}

func (c *transcriptCyberTestCache) IsCyberSessionScopeActive(_ context.Context, scope string) (bool, error) {
	c.scopeCalls++
	return c.scopes[scope], c.scopeErr
}

func (c *transcriptCyberTestCache) FindCyberSessionBlocked(_ context.Context, keys []string) (string, error) {
	c.findCalls++
	if c.findErr != nil {
		return "", c.findErr
	}
	for _, key := range keys {
		if c.store.blocked[key] {
			return key, nil
		}
	}
	return "", nil
}

func newTranscriptCyberTestService(enabled bool) (*OpenAIGatewayService, *transcriptCyberTestCache) {
	cache := &transcriptCyberTestCache{}
	settings := &SettingService{settingRepo: &fakeSettingRepo{vals: map[string]string{
		SettingKeyCyberSessionBlockEnabled:    strconv.FormatBool(enabled),
		SettingKeyCyberSessionBlockTTLSeconds: "60",
	}}}
	return &OpenAIGatewayService{cache: cache, settingService: settings}, cache
}

func TestCyberSessionBlockKeyClientHeadersAndWebSocketEnvelope(t *testing.T) {
	for _, header := range []string{"session-id", "session_id", "conversation_id", "X-Session-Affinity", "X-Session-Id", "X-OpenCode-Session", "X-Conversation-ID"} {
		t.Run(header, func(t *testing.T) {
			c, body := newCyberBlockTestCtx(map[string]string{header: " session-1 "}, `{"prompt_cache_key":"body-session"}`)
			expectedCtx, expectedBody := newCyberBlockTestCtx(map[string]string{"session-id": "session-1"}, `{}`)
			require.Equal(t, CyberSessionBlockKey(4, expectedCtx, expectedBody), CyberSessionBlockKey(4, c, body))
		})
	}
	wrapped := []byte(`{"type":"response.create","response":{"prompt_cache_key":"ws-session","input":[{"role":"user","content":"setup"},{"role":"assistant","content":"ready"},{"role":"user","content":"trigger"}]}}`)
	plain := []byte(`{"prompt_cache_key":"ws-session","input":[{"role":"user","content":"setup"},{"role":"assistant","content":"ready"},{"role":"user","content":"trigger"}]}`)
	c, _ := newCyberBlockTestCtx(nil, "")
	require.Equal(t, CyberSessionBlockKey(88, c, plain), CyberSessionBlockKey(88, c, wrapped))
	require.Equal(t, CyberSessionTranscriptBlockKeys(88, plain), CyberSessionTranscriptBlockKeys(88, wrapped))
	require.Len(t, CyberSessionTranscriptBlockKeys(88, wrapped), 2)
	require.Empty(t, CyberSessionBlockKey(88, c, []byte(`{"response":{"prompt_cache_key":"not-an-event"}}`)))
}

func TestCyberTranscriptBlockKeysRequireModelGeneratedHistory(t *testing.T) {
	first := []byte(`{"instructions":"shared","input":[{"role":"user","content":"fixed environment"},{"role":"user","content":"question one"}]}`)
	second := []byte(`{"instructions":"shared","input":[{"role":"user","content":"fixed environment"},{"role":"user","content":"question two"}]}`)
	firstKeys := CyberSessionTranscriptBlockKeys(77, first)
	secondKeys := CyberSessionTranscriptBlockKeys(77, second)
	require.Len(t, firstKeys, 1)
	require.Len(t, secondKeys, 1)
	require.NotEqual(t, firstKeys[0], secondKeys[0])
	require.NotContains(t, CyberSessionTranscriptLookupKeys(77, second), firstKeys[0])

	hit := []byte(`{"messages":[{"role":"user","content":"setup"},{"role":"assistant","content":"ready"},{"role":"user","content":"trigger"}]}`)
	continuation := []byte(`{"messages":[{"role":"user","content":"setup"},{"role":"assistant","content":"ready"},{"role":"user","content":"different trigger"},{"role":"assistant","content":"blocked"},{"role":"user","content":"continue"}]}`)
	hitKeys := CyberSessionTranscriptBlockKeys(77, hit)
	require.Len(t, hitKeys, 2)
	require.Contains(t, CyberSessionTranscriptLookupKeys(77, continuation), hitKeys[1])
	require.NotContains(t, CyberSessionTranscriptLookupKeys(78, continuation), hitKeys[1])
}

func TestCyberTranscriptIgnoresConfigurationButPreservesHistory(t *testing.T) {
	base := []byte(`{"model":"old","tools":[],"messages":[{"role":"user","content":"hello"}]}`)
	variant := []byte(`{"tools":[{"type":"function"}],"messages":[{"content":"hello", "role":"user"}],"model":"new"}`)
	require.Equal(t, CyberSessionTranscriptBlockKeys(9, base), CyberSessionTranscriptBlockKeys(9, variant))
	require.NotEqual(t, CyberSessionTranscriptBlockKeys(9, base), CyberSessionTranscriptBlockKeys(9, []byte(`{"messages":[{"role":"user","content":"different"}]}`)))
	require.NotEqual(t, CyberSessionTranscriptBlockKeys(9, base), CyberSessionTranscriptBlockKeys(9, []byte(`{"instructions":"different context","messages":[{"role":"user","content":"hello"}]}`)))
	for _, body := range []string{"", `[]`, `{}`, `{"model":"x"}`, `{"input":"  "}`} {
		require.Empty(t, CyberSessionTranscriptBlockKeys(9, []byte(body)))
	}
	require.Len(t, CyberSessionTranscriptBlockKeys(9, []byte(`{"input":"first question"}`)), 1)
}

func TestCyberTranscriptToolResultsDoNotStartUserTurns(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"setup"},{"role":"assistant","content":"ready"},{"role":"user","content":[{"type":"tool_result","content":"result"}]}]}`)
	require.Len(t, CyberSessionTranscriptBlockKeys(9, body), 1)
	withText := []byte(`{"messages":[{"role":"user","content":"setup"},{"role":"assistant","content":"ready"},{"role":"user","content":[{"type":"tool_result","content":"result"},{"type":"text","text":"next"}]}]}`)
	require.Len(t, CyberSessionTranscriptBlockKeys(9, withText), 2)
}

func TestFindCyberSessionBlockedForRequestTranscriptRoundTripAndIsolation(t *testing.T) {
	svc, cache := newTranscriptCyberTestService(true)
	ctx := context.Background()
	hitBody := []byte(`{"messages":[{"role":"user","content":"setup"},{"role":"assistant","content":"ready"},{"role":"user","content":"trigger"}]}`)
	nextBody := []byte(`{"messages":[{"role":"user","content":"setup"},{"role":"assistant","content":"ready"},{"role":"user","content":"rewritten trigger"},{"role":"assistant","content":"blocked"},{"role":"user","content":"continue"}]}`)
	c, _ := newCyberBlockTestCtx(nil, string(nextBody))
	const clientIP = "203.0.113.20"
	const userAgent = "Codex CLI 1.2.3"
	require.Empty(t, svc.FindCyberSessionBlockedForRequest(ctx, 9, c, nextBody, clientIP, userAgent))
	require.Zero(t, cache.findCalls, "inactive scopes must not query transcript keys")
	keys := CyberSessionTranscriptBlockKeys(9, hitBody)
	svc.MarkCyberSessionBlockedKeys(ctx, CyberSessionScopeKey(9, clientIP, userAgent), keys)
	require.Equal(t, time.Minute, cache.lastTTL)
	require.Equal(t, keys[1], svc.FindCyberSessionBlockedForRequest(ctx, 9, c, nextBody, clientIP, "Codex CLI 1.2.4"))
	require.Empty(t, svc.FindCyberSessionBlockedForRequest(ctx, 10, c, nextBody, clientIP, userAgent))
	require.Empty(t, svc.FindCyberSessionBlockedForRequest(ctx, 9, c, nextBody, "203.0.113.21", userAgent))
	require.Empty(t, svc.FindCyberSessionBlockedForRequest(ctx, 9, c, []byte(`{"input":"unrelated"}`), clientIP, userAgent))
}

func TestFindCyberSessionBlockedExplicitPrecedesScopeAndSupportsLegacyStore(t *testing.T) {
	ctx := context.Background()
	svc, cache := newTranscriptCyberTestService(true)
	c, body := newCyberBlockTestCtx(map[string]string{"X-Session-Affinity": "session"}, `{}`)
	key := CyberSessionBlockKey(9, c, body)
	svc.MarkCyberSessionBlockedKeys(ctx, "", []string{key})
	require.Equal(t, key, svc.FindCyberSessionBlockedForRequest(ctx, 9, c, body, "ip", "ua"))
	require.Zero(t, cache.scopeCalls)

	legacy := &comboCacheAndStore{}
	svc.cache = legacy
	svc.MarkCyberSessionBlockedKeys(ctx, "scope", []string{key})
	require.Equal(t, key, svc.FindCyberSessionBlockedForRequest(ctx, 9, c, body, "ip", "ua"))
}

func TestFindCyberSessionBlockedFailOpen(t *testing.T) {
	ctx := context.Background()
	c, body := newCyberBlockTestCtx(nil, `{"input":"trigger"}`)
	var nilSvc *OpenAIGatewayService
	require.Empty(t, nilSvc.FindCyberSessionBlockedForRequest(ctx, 9, c, body, "ip", "ua"))
	require.NotPanics(t, func() { nilSvc.MarkCyberSessionBlockedKeys(ctx, "scope", []string{"key"}) })

	for _, failure := range []string{"disabled", "missing_store", "write", "scope", "lookup", "explicit_lookup"} {
		t.Run(failure, func(t *testing.T) {
			svc, cache := newTranscriptCyberTestService(failure != "disabled")
			scope := CyberSessionScopeKey(9, "ip", "ua")
			keys := CyberSessionTranscriptBlockKeys(9, body)
			if failure == "write" {
				cache.writeErr = errors.New("redis unavailable")
			}
			svc.MarkCyberSessionBlockedKeys(ctx, scope, keys)
			if failure == "missing_store" {
				svc.cache = nil
			}
			if failure == "scope" {
				cache.scopeErr = errors.New("redis unavailable")
			}
			if failure == "lookup" || failure == "explicit_lookup" {
				cache.findErr = errors.New("redis unavailable")
			}
			testCtx, _ := newCyberBlockTestCtx(nil, string(body))
			if failure == "explicit_lookup" {
				testCtx.Request.Header.Set("session-id", "session")
			}
			require.Empty(t, svc.FindCyberSessionBlockedForRequest(ctx, 9, testCtx, body, "ip", "ua"))
		})
	}
}

func TestCyberTranscriptLongHistoryRetainsAllPrefixesWithoutScopeOnlyBlocking(t *testing.T) {
	messages := make([]map[string]string, 600)
	for i := range messages {
		messages[i] = map[string]string{"role": "user", "content": "message-" + strconv.Itoa(i)}
	}
	body, err := json.Marshal(map[string]any{"messages": messages})
	require.NoError(t, err)
	keys := CyberSessionTranscriptLookupKeys(77, body)
	require.Len(t, keys, len(messages))
	firstBody, err := json.Marshal(map[string]any{"messages": messages[:1]})
	require.NoError(t, err)
	firstKey := CyberSessionTranscriptBlockKeys(77, firstBody)[0]
	require.Equal(t, firstKey, keys[0], "padding must never discard the earliest prefix")
	require.Equal(t, CyberSessionTranscriptBlockKeys(77, body)[0], keys[len(keys)-1])

	svc, cache := newTranscriptCyberTestService(true)
	c, _ := newCyberBlockTestCtx(nil, string(body))
	ctx := context.Background()
	require.Empty(t, svc.FindCyberSessionBlockedForRequest(ctx, 77, c, body, "ip", "ua"))
	require.Zero(t, cache.findCalls)
	scope := CyberSessionScopeKey(77, "ip", "ua")
	svc.MarkCyberSessionBlockedKeys(ctx, scope, CyberSessionTranscriptBlockKeys(77, []byte(`{"input":"unrelated blocked conversation"}`)))
	require.Empty(t, svc.FindCyberSessionBlockedForRequest(ctx, 77, c, body, "ip", "ua"), "a long independent conversation sharing scope must be allowed")
	require.Equal(t, 1, cache.findCalls)

	// A block after the former 256-prefix cap must still be queried.
	svc.MarkCyberSessionBlockedKeys(ctx, scope, []string{keys[520]})
	require.Equal(t, keys[520], svc.FindCyberSessionBlockedForRequest(ctx, 77, c, body, "ip", "ua"))
	// An earlier blocked turn remains discoverable after hundreds of new turns.
	svc.MarkCyberSessionBlockedKeys(ctx, scope, []string{firstKey})
	require.Equal(t, firstKey, svc.FindCyberSessionBlockedForRequest(ctx, 77, c, body, "ip", "ua"))
}

func TestCyberTranscriptAnthropicSystemParticipatesInSessionIdentity(t *testing.T) {
	const messages = `,"messages":[{"role":"user","content":"setup"},{"role":"assistant","content":"ready"},{"role":"user","content":"trigger"}]}`
	for _, tc := range []struct{ name, firstSystem, sameSystem, otherSystem string }{
		{name: "string", firstSystem: `"system A"`, sameSystem: `"system A"`, otherSystem: `"system B"`},
		{name: "blocks", firstSystem: `[{"type":"text","text":"system A"}]`, sameSystem: `[ {"text":"system A", "type":"text"} ]`, otherSystem: `[{"type":"text","text":"system B"}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := []byte(`{"system":` + tc.firstSystem + messages)
			same := []byte(`{"system":` + tc.sameSystem + messages)
			other := []byte(`{"system":` + tc.otherSystem + messages)
			keys := CyberSessionTranscriptBlockKeys(9, first)
			require.Len(t, keys, 2)
			require.Equal(t, keys, CyberSessionTranscriptBlockKeys(9, same), "JSON object order and spacing must not affect system identity")
			for _, key := range keys {
				require.NotContains(t, CyberSessionTranscriptLookupKeys(9, other), key)
			}
			svc, _ := newTranscriptCyberTestService(true)
			ctx := context.Background()
			svc.MarkCyberSessionBlockedKeys(ctx, CyberSessionScopeKey(9, "ip", "ua"), keys)
			c, _ := newCyberBlockTestCtx(nil, string(other))
			require.Empty(t, svc.FindCyberSessionBlockedForRequest(ctx, 9, c, other, "ip", "ua"))
			require.NotEmpty(t, svc.FindCyberSessionBlockedForRequest(ctx, 9, c, same, "ip", "ua"))
		})
	}
	// Root text is quoted in the hash stream so field delimiters cannot make
	// one instructions string collide with separate instructions/system fields.
	joined := []byte(`{"instructions":"a|system=b","input":"hello"}`)
	separate := []byte(`{"instructions":"a","system":"b","input":"hello"}`)
	require.NotEqual(t, CyberSessionTranscriptBlockKeys(9, joined), CyberSessionTranscriptBlockKeys(9, separate))
}

func TestCyberSessionScopeKeyNormalizesUserAgentVersion(t *testing.T) {
	base := CyberSessionScopeKey(7, "203.0.113.10", "Codex CLI 1.2.3")
	require.NotEmpty(t, base)
	require.Equal(t, base, CyberSessionScopeKey(7, "203.0.113.10", "Codex CLI 1.2.4"))
	require.NotEqual(t, base, CyberSessionScopeKey(8, "203.0.113.10", "Codex CLI 1.2.3"))
	require.NotEqual(t, base, CyberSessionScopeKey(7, "203.0.113.11", "Codex CLI 1.2.3"))
	require.Empty(t, CyberSessionScopeKey(0, "ip", "ua"))
}
