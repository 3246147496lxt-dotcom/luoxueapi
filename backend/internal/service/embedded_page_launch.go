package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/time/rate"
)

const (
	CustomMenuAuthModeNone         = "none"
	CustomMenuAuthModeExchangeCode = "exchange_code"

	EmbeddedPageLaunchTTL = 60 * time.Second

	// Keep ordinary launch spikes well below the fuse while bounding each
	// process to a Redis-safe sustained exchange rate during anonymous floods.
	embeddedPageExchangeRequestsPerSecond = 100
	embeddedPageExchangeBurst             = 200
)

var (
	ErrInvalidEmbedLaunchCode = infraerrors.Unauthorized(
		"INVALID_EMBED_LAUNCH_CODE",
		"invalid or expired embedded page launch code",
	)
	ErrEmbeddedPageLaunchUnavailable = infraerrors.ServiceUnavailable(
		"EMBEDDED_PAGE_LAUNCH_UNAVAILABLE",
		"embedded page launch service is temporarily unavailable",
	)
	ErrEmbeddedPageExchangeRateLimited = infraerrors.TooManyRequests(
		"EMBEDDED_PAGE_EXCHANGE_RATE_LIMITED",
		"too many embedded page exchange requests",
	)
	ErrCustomPageNotFound = infraerrors.NotFound(
		"CUSTOM_PAGE_NOT_FOUND",
		"custom page not found",
	)
	ErrCustomPageForbidden = infraerrors.Forbidden(
		"CUSTOM_PAGE_FORBIDDEN",
		"custom page is not available to this user",
	)
	ErrCustomPageLaunchAuthMode = infraerrors.BadRequest(
		"CUSTOM_PAGE_AUTH_MODE_UNSUPPORTED",
		"custom page does not use exchange-code authentication",
	)
	ErrCustomPageHTTPSRequired = infraerrors.BadRequest(
		"CUSTOM_PAGE_HTTPS_REQUIRED",
		"exchange-code custom pages must use HTTPS",
	)
	ErrInvalidCustomPageLaunchOptions = infraerrors.BadRequest(
		"INVALID_CUSTOM_PAGE_LAUNCH_OPTIONS",
		"invalid custom page launch options",
	)

	embeddedLangPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,31}$`)
	embeddedUIModePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
)

type embeddedPageMenuSettings interface {
	GetCustomMenuItemsRaw(ctx context.Context) string
}

// EmbeddedPageLaunchStore persists hashed launch tickets. Consume must remove
// and return a ticket atomically; a missing or expired ticket is reported with
// found=false rather than leaking an infrastructure-specific sentinel error.
type EmbeddedPageLaunchStore interface {
	Put(ctx context.Context, digest string, payload []byte, ttl time.Duration) error
	Consume(ctx context.Context, digest string) (payload []byte, found bool, err error)
}

type embeddedPageExchangeAdmission interface {
	Allow() bool
}

// EmbeddedPageLaunchOptions contains non-sensitive UI context forwarded to an
// external custom page. Authentication data is deliberately not accepted here.
type EmbeddedPageLaunchOptions struct {
	Theme      string
	Lang       string
	UIMode     string
	SourceHost string
}

// EmbeddedPageLaunch is returned to the authenticated LuoxueAPI frontend.
// LaunchURL contains a short-lived, single-use code and must never be logged.
type EmbeddedPageLaunch struct {
	LaunchURL string `json:"launch_url"`
	AuthMode  string `json:"auth_mode"`
	ExpiresIn int    `json:"expires_in"`
}

// EmbeddedPageIdentity is the deliberately minimal exchange result.
type EmbeddedPageIdentity struct {
	UserID     int64  `json:"user_id"`
	MenuItemID string `json:"menu_item_id"`
}

type embeddedPageLaunchTicket struct {
	UserID       int64     `json:"user_id"`
	MenuItemID   string    `json:"menu_item_id"`
	TargetOrigin string    `json:"target_origin"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type embeddedPageMenuItem struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Visibility string `json:"visibility"`
	AuthMode   string `json:"auth_mode"`
}

// EmbeddedPageLaunchService issues and atomically consumes external-page
// launch tickets. Redis stores only a SHA-256 digest of the raw launch code.
type EmbeddedPageLaunchService struct {
	settings          embeddedPageMenuSettings
	store             EmbeddedPageLaunchStore
	exchangeAdmission embeddedPageExchangeAdmission
	now               func() time.Time
	random            io.Reader
}

func NewEmbeddedPageLaunchService(settings *SettingService, store EmbeddedPageLaunchStore) *EmbeddedPageLaunchService {
	return &EmbeddedPageLaunchService{
		settings: settings,
		store:    store,
		exchangeAdmission: rate.NewLimiter(
			rate.Limit(embeddedPageExchangeRequestsPerSecond),
			embeddedPageExchangeBurst,
		),
		now:    time.Now,
		random: rand.Reader,
	}
}

// Issue creates a fresh launch code for one concrete iframe or new-tab launch.
func (s *EmbeddedPageLaunchService) Issue(
	ctx context.Context,
	userID int64,
	role string,
	menuItemID string,
	options EmbeddedPageLaunchOptions,
) (*EmbeddedPageLaunch, error) {
	if s == nil {
		return nil, ErrEmbeddedPageLaunchUnavailable
	}
	if userID <= 0 {
		return nil, ErrCustomPageForbidden
	}
	menuItem, ok := s.findMenuItem(ctx, strings.TrimSpace(menuItemID))
	if !ok {
		return nil, ErrCustomPageNotFound
	}
	if menuItem.Visibility != "user" && menuItem.Visibility != "admin" {
		return nil, ErrCustomPageNotFound
	}
	if menuItem.Visibility == "admin" && role != "admin" {
		return nil, ErrCustomPageForbidden
	}
	if normalizeCustomMenuAuthMode(menuItem.AuthMode) != CustomMenuAuthModeExchangeCode {
		return nil, ErrCustomPageLaunchAuthMode
	}

	target, targetOrigin, err := parseExchangeCodeTarget(menuItem.URL)
	if err != nil {
		return nil, err
	}
	normalizedOptions, err := normalizeEmbeddedPageLaunchOptions(options)
	if err != nil {
		return nil, err
	}

	rawCode := make([]byte, 32)
	if _, err := io.ReadFull(s.random, rawCode); err != nil {
		return nil, infraerrors.InternalServer("EMBEDDED_PAGE_LAUNCH_GENERATION_FAILED", "failed to generate launch code").WithCause(err)
	}
	code := base64.RawURLEncoding.EncodeToString(rawCode)
	digest := embeddedPageLaunchDigest(code)

	now := s.now().UTC()
	ticket := embeddedPageLaunchTicket{
		UserID:       userID,
		MenuItemID:   menuItem.ID,
		TargetOrigin: targetOrigin,
		IssuedAt:     now,
		ExpiresAt:    now.Add(EmbeddedPageLaunchTTL),
	}
	payload, err := json.Marshal(ticket)
	if err != nil {
		return nil, infraerrors.InternalServer("EMBEDDED_PAGE_LAUNCH_GENERATION_FAILED", "failed to generate launch ticket").WithCause(err)
	}

	launchURL := buildEmbeddedPageLaunchURL(target, menuItem.ID, code, normalizedOptions)
	if s.store == nil {
		return nil, ErrEmbeddedPageLaunchUnavailable
	}
	if err := s.store.Put(ctx, digest, payload, EmbeddedPageLaunchTTL); err != nil {
		return nil, ErrEmbeddedPageLaunchUnavailable.WithCause(fmt.Errorf("store embedded page launch ticket: %w", err))
	}

	return &EmbeddedPageLaunch{
		LaunchURL: launchURL,
		AuthMode:  CustomMenuAuthModeExchangeCode,
		ExpiresIn: int(EmbeddedPageLaunchTTL / time.Second),
	}, nil
}

// Exchange atomically consumes one launch code. Any invalid, expired, replayed,
// revoked, or menu-mismatched ticket is intentionally indistinguishable.
func (s *EmbeddedPageLaunchService) Exchange(ctx context.Context, menuItemID, code string) (*EmbeddedPageIdentity, error) {
	menuItemID = strings.TrimSpace(menuItemID)
	code = strings.TrimSpace(code)
	decoded, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil || len(decoded) != 32 || menuItemID == "" {
		return nil, ErrInvalidEmbedLaunchCode
	}

	if s == nil || s.store == nil {
		return nil, ErrEmbeddedPageLaunchUnavailable
	}
	// This process-local fuse bounds the rate at which anonymous, correctly
	// shaped guesses can reach Redis. It is deliberately global rather than
	// keyed by source IP so unrelated services behind one proxy do not share a
	// small bucket. Reject before GETDEL and leave a real ticket unconsumed so a
	// caller can retry after the short overload clears.
	if s.exchangeAdmission == nil {
		return nil, ErrEmbeddedPageLaunchUnavailable
	}
	if !s.exchangeAdmission.Allow() {
		return nil, ErrEmbeddedPageExchangeRateLimited
	}
	payload, found, err := s.store.Consume(ctx, embeddedPageLaunchDigest(code))
	if err != nil {
		return nil, ErrEmbeddedPageLaunchUnavailable.WithCause(fmt.Errorf("consume embedded page launch ticket: %w", err))
	}
	if !found {
		return nil, ErrInvalidEmbedLaunchCode
	}

	var ticket embeddedPageLaunchTicket
	if err := json.Unmarshal(payload, &ticket); err != nil {
		return nil, ErrInvalidEmbedLaunchCode
	}
	now := s.now().UTC()
	if ticket.UserID <= 0 || ticket.MenuItemID == "" || ticket.MenuItemID != menuItemID ||
		ticket.IssuedAt.After(now.Add(time.Second)) || !ticket.ExpiresAt.After(now) {
		return nil, ErrInvalidEmbedLaunchCode
	}

	menuItem, ok := s.findMenuItem(ctx, ticket.MenuItemID)
	if !ok || normalizeCustomMenuAuthMode(menuItem.AuthMode) != CustomMenuAuthModeExchangeCode {
		return nil, ErrInvalidEmbedLaunchCode
	}
	_, currentOrigin, err := parseExchangeCodeTarget(menuItem.URL)
	if err != nil || currentOrigin != ticket.TargetOrigin {
		return nil, ErrInvalidEmbedLaunchCode
	}

	return &EmbeddedPageIdentity{UserID: ticket.UserID, MenuItemID: ticket.MenuItemID}, nil
}

func (s *EmbeddedPageLaunchService) findMenuItem(ctx context.Context, id string) (embeddedPageMenuItem, bool) {
	if id == "" || s.settings == nil {
		return embeddedPageMenuItem{}, false
	}
	var items []embeddedPageMenuItem
	if err := json.Unmarshal([]byte(s.settings.GetCustomMenuItemsRaw(ctx)), &items); err != nil {
		return embeddedPageMenuItem{}, false
	}
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return embeddedPageMenuItem{}, false
}

func normalizeCustomMenuAuthMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return CustomMenuAuthModeNone
	}
	return mode
}

func parseExchangeCodeTarget(raw string) (*url.URL, string, error) {
	target, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !target.IsAbs() || target.Host == "" || target.User != nil ||
		!strings.EqualFold(target.Scheme, "https") {
		return nil, "", ErrCustomPageHTTPSRequired
	}
	target.Scheme = "https"
	target.Host = strings.ToLower(target.Host)
	// Administrator-provided query and fragment data are never forwarded. In
	// particular this strips legacy token fragments while keeping existing HTTPS
	// menu entries launchable after auth_mode is upgraded to exchange_code.
	target.RawQuery = ""
	target.ForceQuery = false
	target.Fragment = ""
	target.RawFragment = ""
	return target, target.Scheme + "://" + target.Host, nil
}

func normalizeEmbeddedPageLaunchOptions(options EmbeddedPageLaunchOptions) (EmbeddedPageLaunchOptions, error) {
	options.Theme = strings.TrimSpace(options.Theme)
	if options.Theme == "" {
		options.Theme = "light"
	}
	if options.Theme != "light" && options.Theme != "dark" {
		return EmbeddedPageLaunchOptions{}, ErrInvalidCustomPageLaunchOptions
	}

	options.Lang = strings.TrimSpace(options.Lang)
	if options.Lang != "" && !embeddedLangPattern.MatchString(options.Lang) {
		return EmbeddedPageLaunchOptions{}, ErrInvalidCustomPageLaunchOptions
	}

	options.UIMode = strings.TrimSpace(options.UIMode)
	if options.UIMode == "" {
		options.UIMode = "embedded"
	}
	if !embeddedUIModePattern.MatchString(options.UIMode) {
		return EmbeddedPageLaunchOptions{}, ErrInvalidCustomPageLaunchOptions
	}

	options.SourceHost = strings.TrimSpace(options.SourceHost)
	if options.SourceHost != "" {
		source, err := url.Parse(options.SourceHost)
		if err != nil || !source.IsAbs() || source.Host == "" || source.User != nil || source.RawQuery != "" ||
			source.Fragment != "" || (source.Path != "" && source.Path != "/") ||
			(!strings.EqualFold(source.Scheme, "http") && !strings.EqualFold(source.Scheme, "https")) {
			return EmbeddedPageLaunchOptions{}, ErrInvalidCustomPageLaunchOptions
		}
		options.SourceHost = strings.ToLower(source.Scheme) + "://" + strings.ToLower(source.Host)
	}

	return options, nil
}

func buildEmbeddedPageLaunchURL(target *url.URL, menuItemID, code string, options EmbeddedPageLaunchOptions) string {
	query := make(url.Values)
	query.Set("theme", options.Theme)
	if options.Lang != "" {
		query.Set("lang", options.Lang)
	} else {
		query.Del("lang")
	}
	query.Set("ui_mode", options.UIMode)
	if options.SourceHost != "" {
		query.Set("src_host", options.SourceHost)
	} else {
		query.Del("src_host")
	}
	query.Set("s2a_client_id", menuItemID)
	query.Set("s2a_launch_code", code)
	target.RawQuery = query.Encode()
	return target.String()
}

func embeddedPageLaunchDigest(code string) string {
	digest := sha256.Sum256([]byte(code))
	return hex.EncodeToString(digest[:])
}
