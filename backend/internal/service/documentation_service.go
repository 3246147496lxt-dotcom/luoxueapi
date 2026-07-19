package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	DocumentationSchemaVersion   = 1
	DocumentationMaxJSONBytes    = 1 << 20
	DocumentationMaxAssetBytes   = 5 << 20
	DocumentationMaxIconSVGBytes = 32 << 10
)

var (
	ErrDocumentationNotFound = infraerrors.New(http.StatusNotFound, "DOCUMENTATION_NOT_FOUND", "Not found")
	ErrDocumentationNoDraft  = infraerrors.New(http.StatusConflict, "DOCUMENTATION_DRAFT_REQUIRED", "Save a documentation draft before publishing")
	ErrDocumentationInvalid  = infraerrors.New(http.StatusBadRequest, "DOCUMENTATION_INVALID", "Invalid documentation content")
	ErrDocumentationAsset    = infraerrors.New(http.StatusBadRequest, "DOCUMENTATION_ASSET_INVALID", "Invalid documentation image")

	documentationIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
	htmlTagPattern         = regexp.MustCompile(`(?i)<\s*/?\s*[a-z][^>]*>`)
	assetIDPattern         = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

//go:embed default_documentation.json
var defaultDocumentationContent json.RawMessage

type DocumentationSiteConfig struct {
	BrandName      string `json:"brand_name,omitempty"`
	PageTitle      string `json:"page_title,omitempty"`
	SupportContact string `json:"support_contact,omitempty"`
	SupportValue   string `json:"support_value,omitempty"`
	FooterText     string `json:"footer_text,omitempty"`
}

type DocumentationImage struct {
	Src     string `json:"src"`
	Alt     string `json:"alt"`
	Caption string `json:"caption,omitempty"`
}

type DocumentationNote struct {
	Tone string `json:"tone,omitempty"`
	Text string `json:"text"`
}

type DocumentationLink struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type DocumentationCode struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type DocumentationStep struct {
	Title         string              `json:"title"`
	Description   string              `json:"description"`
	Image         *DocumentationImage `json:"image,omitempty"`
	Note          *DocumentationNote  `json:"note,omitempty"`
	NotePlacement string              `json:"note_placement,omitempty"`
	Link          *DocumentationLink  `json:"link,omitempty"`
	Code          *DocumentationCode  `json:"code,omitempty"`
}

type DocumentationTutorial struct {
	ID          string              `json:"id"`
	TabLabel    string              `json:"tab_label"`
	Icon        string              `json:"icon"`
	IconSVG     string              `json:"icon_svg,omitempty"`
	Description string              `json:"description"`
	Steps       []DocumentationStep `json:"steps"`
}

type DocumentationContent struct {
	SchemaVersion int                      `json:"schema_version"`
	SiteConfig    *DocumentationSiteConfig `json:"site_config,omitempty"`
	Tutorials     []DocumentationTutorial  `json:"tutorials"`
}

type DocumentationSnapshot struct {
	Content   json.RawMessage `json:"content"`
	Version   int64           `json:"version"`
	UpdatedAt time.Time       `json:"updated_at"`
	UpdatedBy *int64          `json:"updated_by,omitempty"`
}

type DocumentationAdminState struct {
	Draft     *DocumentationSnapshot `json:"draft"`
	Published *DocumentationSnapshot `json:"published"`
}

type DocumentationRevision struct {
	ID          int64     `json:"id"`
	Version     int64     `json:"version"`
	PublishedAt time.Time `json:"published_at"`
	PublishedBy *int64    `json:"published_by,omitempty"`
}

type DocumentationAsset struct {
	ID          string    `json:"id"`
	ContentType string    `json:"content_type"`
	ByteSize    int64     `json:"byte_size"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   *int64    `json:"created_by,omitempty"`
	Data        []byte    `json:"-"`
}

func (a DocumentationAsset) PublicURL() string {
	return "/api/v1/public/documentation/assets/" + a.ID
}

type DocumentationRepository interface {
	EnsureSeed(ctx context.Context, content json.RawMessage) error
	GetState(ctx context.Context) (*DocumentationAdminState, error)
	SaveDraft(ctx context.Context, content json.RawMessage, actorID *int64) (*DocumentationAdminState, error)
	Publish(ctx context.Context, actorID *int64) (*DocumentationAdminState, error)
	ListRevisions(ctx context.Context) ([]DocumentationRevision, error)
	RestoreRevision(ctx context.Context, revisionID int64, actorID *int64) (*DocumentationAdminState, error)
	GetPublished(ctx context.Context) (*DocumentationSnapshot, error)
	SaveAsset(ctx context.Context, asset *DocumentationAsset) error
	GetAsset(ctx context.Context, id string) (*DocumentationAsset, error)
}

type DocumentationService struct {
	repo        DocumentationRepository
	seedMu      sync.Mutex
	initialized atomic.Bool
}

func NewDocumentationService(repo DocumentationRepository) *DocumentationService {
	return &DocumentationService{repo: repo}
}

func (s *DocumentationService) GetAdminState(ctx context.Context) (*DocumentationAdminState, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, err
	}
	return s.repo.GetState(ctx)
}

func (s *DocumentationService) SaveDraft(ctx context.Context, raw json.RawMessage, actorID *int64) (*DocumentationAdminState, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, err
	}
	canonical, err := ValidateDocumentationDraft(raw)
	if err != nil {
		return nil, err
	}
	return s.repo.SaveDraft(ctx, canonical, actorID)
}

func (s *DocumentationService) Publish(ctx context.Context, actorID *int64) (*DocumentationAdminState, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, err
	}
	state, err := s.repo.GetState(ctx)
	if err != nil {
		return nil, err
	}
	if state.Draft == nil || len(state.Draft.Content) == 0 {
		return nil, ErrDocumentationNoDraft
	}
	if _, err := ValidateDocumentationContent(state.Draft.Content); err != nil {
		return nil, err
	}
	return s.repo.Publish(ctx, actorID)
}

func (s *DocumentationService) ListRevisions(ctx context.Context) ([]DocumentationRevision, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListRevisions(ctx)
}

func (s *DocumentationService) RestoreRevision(ctx context.Context, revisionID int64, actorID *int64) (*DocumentationAdminState, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, err
	}
	if revisionID <= 0 {
		return nil, ErrDocumentationNotFound
	}
	return s.repo.RestoreRevision(ctx, revisionID, actorID)
}

func (s *DocumentationService) PublicSnapshot(ctx context.Context) (*DocumentationSnapshot, string, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, "", err
	}
	snapshot, err := s.repo.GetPublished(ctx)
	if err != nil {
		return nil, "", err
	}
	hash := sha256.New()
	_, _ = hash.Write(snapshot.Content)
	_, _ = fmt.Fprintf(hash, "\n%d", snapshot.Version)
	return snapshot, `"` + hex.EncodeToString(hash.Sum(nil)) + `"`, nil
}

func (s *DocumentationService) SaveAsset(ctx context.Context, asset *DocumentationAsset) error {
	if err := s.ensureInitialized(ctx); err != nil {
		return err
	}
	if asset == nil || !assetIDPattern.MatchString(asset.ID) || asset.ByteSize <= 0 || asset.ByteSize > DocumentationMaxAssetBytes || int64(len(asset.Data)) != asset.ByteSize {
		return ErrDocumentationAsset
	}
	if !allowedDocumentationAssetType(asset.ContentType) || asset.Width <= 0 || asset.Height <= 0 || asset.Width > 10000 || asset.Height > 10000 || int64(asset.Width)*int64(asset.Height) > 40_000_000 {
		return ErrDocumentationAsset
	}
	return s.repo.SaveAsset(ctx, asset)
}

func (s *DocumentationService) GetAsset(ctx context.Context, id string) (*DocumentationAsset, string, error) {
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, "", err
	}
	id = strings.ToLower(strings.TrimSpace(id))
	if !assetIDPattern.MatchString(id) {
		return nil, "", ErrDocumentationNotFound
	}
	asset, err := s.repo.GetAsset(ctx, id)
	if err != nil {
		return nil, "", err
	}
	return asset, `"` + asset.ID + `"`, nil
}

func ValidateDocumentationContent(raw json.RawMessage) (json.RawMessage, error) {
	return validateAndCanonicalizeDocumentation(raw, true)
}

func ValidateDocumentationDraft(raw json.RawMessage) (json.RawMessage, error) {
	return validateAndCanonicalizeDocumentation(raw, false)
}

func validateAndCanonicalizeDocumentation(raw json.RawMessage, strict bool) (json.RawMessage, error) {
	if len(raw) == 0 || len(raw) > DocumentationMaxJSONBytes {
		return nil, invalidDocumentation("content must be between 1 byte and %d bytes", DocumentationMaxJSONBytes)
	}
	var content DocumentationContent
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&content); err != nil {
		return nil, invalidDocumentation("invalid JSON: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, invalidDocumentation("content must contain one JSON document")
	}
	if err := validateDocumentationContent(content, strict); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(content)
	if err != nil {
		return nil, invalidDocumentation("cannot encode content: %v", err)
	}
	return canonical, nil
}

func validateDocumentationContent(content DocumentationContent, strict bool) error {
	if content.SchemaVersion != DocumentationSchemaVersion {
		return invalidDocumentation("schema_version must be %d", DocumentationSchemaVersion)
	}
	if len(content.Tutorials) > 20 || (strict && len(content.Tutorials) == 0) {
		return invalidDocumentation("tutorials must contain between %d and 20 items", boolInt(strict))
	}
	if content.SiteConfig != nil {
		fields := []struct {
			name, value string
			max         int
		}{
			{"site_config.brand_name", content.SiteConfig.BrandName, 80},
			{"site_config.page_title", content.SiteConfig.PageTitle, 120},
			{"site_config.support_contact", content.SiteConfig.SupportContact, 240},
			{"site_config.support_value", content.SiteConfig.SupportValue, 160},
			{"site_config.footer_text", content.SiteConfig.FooterText, 300},
		}
		for _, field := range fields {
			if err := validateSafeText(field.name, field.value, field.max, false, false); err != nil {
				return err
			}
		}
	}
	allowedIcons := map[string]bool{"key": true, "client": true, "api": true, "wallet": true, "image": true, "book": true, "code": true, "terminal": true, "help": true}
	allowedTones := map[string]bool{"": true, "info": true, "warning": true}
	allowedPlacements := map[string]bool{"": true, "before-image": true, "after-image": true}
	seen := make(map[string]struct{}, len(content.Tutorials))
	totalSteps := 0
	for tutorialIndex, tutorial := range content.Tutorials {
		prefix := fmt.Sprintf("tutorials[%d]", tutorialIndex)
		if tutorial.ID != "" && !documentationIDPattern.MatchString(tutorial.ID) {
			return invalidDocumentation("%s.id must be a safe lowercase identifier", prefix)
		}
		if strict && tutorial.ID == "" {
			return invalidDocumentation("%s.id is required", prefix)
		}
		if _, ok := seen[tutorial.ID]; tutorial.ID != "" && ok {
			return invalidDocumentation("tutorial id %q is duplicated", tutorial.ID)
		}
		if tutorial.ID != "" {
			seen[tutorial.ID] = struct{}{}
		}
		if (strict || tutorial.Icon != "") && !allowedIcons[tutorial.Icon] {
			return invalidDocumentation("%s.icon is not allowed", prefix)
		}
		if err := validateDocumentationIconSVG(prefix+".icon_svg", tutorial.IconSVG); err != nil {
			return err
		}
		if err := validateSafeText(prefix+".tab_label", tutorial.TabLabel, 80, strict, strict); err != nil {
			return err
		}
		if err := validateSafeText(prefix+".description", tutorial.Description, 500, strict, strict); err != nil {
			return err
		}
		if len(tutorial.Steps) > 50 || (strict && len(tutorial.Steps) == 0) {
			return invalidDocumentation("%s.steps must contain between %d and 50 items", prefix, boolInt(strict))
		}
		totalSteps += len(tutorial.Steps)
		if totalSteps > 300 {
			return invalidDocumentation("documentation cannot contain more than 300 steps")
		}
		for stepIndex, step := range tutorial.Steps {
			stepPrefix := fmt.Sprintf("%s.steps[%d]", prefix, stepIndex)
			if err := validateSafeText(stepPrefix+".title", step.Title, 160, strict, strict); err != nil {
				return err
			}
			if err := validateSafeText(stepPrefix+".description", step.Description, 1200, strict, strict); err != nil {
				return err
			}
			if !allowedPlacements[step.NotePlacement] {
				return invalidDocumentation("%s.note_placement is not allowed", stepPrefix)
			}
			if step.NotePlacement != "" && step.Note == nil {
				return invalidDocumentation("%s.note_placement requires a note", stepPrefix)
			}
			if step.Image != nil {
				if (strict || strings.TrimSpace(step.Image.Src) != "") && !isSafeDocumentationURL(step.Image.Src) {
					return invalidDocumentation("%s.image.src must be an http(s) or root-relative URL", stepPrefix)
				}
				if err := validateSafeText(stepPrefix+".image.alt", step.Image.Alt, 300, strict, false); err != nil {
					return err
				}
				if err := validateSafeText(stepPrefix+".image.caption", step.Image.Caption, 500, false, false); err != nil {
					return err
				}
			}
			if step.Note != nil {
				if !allowedTones[step.Note.Tone] {
					return invalidDocumentation("%s.note.tone is not allowed", stepPrefix)
				}
				if err := validateSafeText(stepPrefix+".note.text", step.Note.Text, 1200, strict, false); err != nil {
					return err
				}
			}
			if step.Link != nil {
				if err := validateSafeText(stepPrefix+".link.label", step.Link.Label, 120, strict, false); err != nil {
					return err
				}
				if (strict || strings.TrimSpace(step.Link.Href) != "") && !isSafeDocumentationURL(step.Link.Href) {
					return invalidDocumentation("%s.link.href must be an http(s) or root-relative URL", stepPrefix)
				}
			}
			if step.Code != nil {
				if err := validateSafeText(stepPrefix+".code.label", step.Code.Label, 120, strict, false); err != nil {
					return err
				}
				if err := validateCodeText(stepPrefix+".code.value", step.Code.Value, 20_000, strict); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateSafeText(name, value string, maxRunes int, required, requireHan bool) error {
	trimmed := strings.TrimSpace(value)
	if required && trimmed == "" {
		return invalidDocumentation("%s is required", name)
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxRunes {
		return invalidDocumentation("%s exceeds its length limit", name)
	}
	if strings.IndexByte(value, 0) >= 0 || htmlTagPattern.MatchString(value) {
		return invalidDocumentation("%s cannot contain raw HTML", name)
	}
	if requireHan && !containsHan(value) {
		return invalidDocumentation("%s must contain Chinese text", name)
	}
	return nil
}

func containsHan(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func validateCodeText(name, value string, maxRunes int, required bool) error {
	if required && strings.TrimSpace(value) == "" {
		return invalidDocumentation("%s is required", name)
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxRunes {
		return invalidDocumentation("%s exceeds its length limit", name)
	}
	if strings.IndexByte(value, 0) >= 0 {
		return invalidDocumentation("%s contains an invalid character", name)
	}
	return nil
}

var blockedDocumentationSVGElements = map[string]struct{}{
	"a": {}, "animate": {}, "animatemotion": {}, "animatetransform": {},
	"audio": {}, "canvas": {}, "discard": {}, "embed": {}, "feimage": {},
	"foreignobject": {}, "iframe": {}, "image": {}, "link": {}, "meta": {},
	"mpath": {}, "object": {}, "script": {}, "set": {}, "style": {}, "video": {},
}

// validateDocumentationIconSVG accepts a deliberately small, self-contained
// SVG document. Icons are embedded into the documentation JSON, so external
// resources, active content, CSS and animation are never needed.
func validateDocumentationIconSVG(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if len(value) > DocumentationMaxIconSVGBytes || !utf8.ValidString(value) {
		return invalidDocumentation("%s must be a valid SVG no larger than %d bytes", name, DocumentationMaxIconSVGBytes)
	}

	decoder := xml.NewDecoder(strings.NewReader(value))
	decoder.Strict = true
	depth := 0
	nodes := 0
	rootSeen := false
	rootClosed := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return invalidDocumentation("%s must contain well-formed SVG XML", name)
		}

		switch typed := token.(type) {
		case xml.Directive:
			return invalidDocumentation("%s cannot contain XML directives", name)
		case xml.ProcInst:
			if rootSeen || !strings.EqualFold(typed.Target, "xml") {
				return invalidDocumentation("%s cannot contain processing instructions", name)
			}
		case xml.StartElement:
			if rootClosed {
				return invalidDocumentation("%s must contain exactly one SVG document", name)
			}
			depth++
			nodes++
			if depth > 64 || nodes > 4096 || len(typed.Attr) > 64 {
				return invalidDocumentation("%s is too complex", name)
			}

			elementName := strings.ToLower(typed.Name.Local)
			if !rootSeen {
				if elementName != "svg" {
					return invalidDocumentation("%s must use an svg root element", name)
				}
				rootSeen = true
			}
			if typed.Name.Space != "" && typed.Name.Space != "http://www.w3.org/2000/svg" {
				return invalidDocumentation("%s contains an unsupported XML namespace", name)
			}
			if _, blocked := blockedDocumentationSVGElements[elementName]; blocked {
				return invalidDocumentation("%s contains a blocked SVG element", name)
			}
			for _, attribute := range typed.Attr {
				if err := validateDocumentationSVGAttribute(name, attribute); err != nil {
					return err
				}
			}
		case xml.EndElement:
			depth--
			if depth < 0 {
				return invalidDocumentation("%s must contain well-formed SVG XML", name)
			}
			if depth == 0 {
				rootClosed = true
			}
		}
	}

	if !rootSeen || !rootClosed || depth != 0 {
		return invalidDocumentation("%s must contain one complete SVG document", name)
	}
	return nil
}

func validateDocumentationSVGAttribute(name string, attribute xml.Attr) error {
	attributeName := strings.ToLower(attribute.Name.Local)
	attributeValue := strings.TrimSpace(attribute.Value)
	if len(attributeValue) > 8192 || strings.IndexByte(attributeValue, 0) >= 0 {
		return invalidDocumentation("%s contains an invalid SVG attribute", name)
	}

	if attribute.Name.Space == "xmlns" || (attribute.Name.Space == "" && attributeName == "xmlns") {
		if attributeValue == "http://www.w3.org/2000/svg" || attributeValue == "http://www.w3.org/1999/xlink" {
			return nil
		}
		return invalidDocumentation("%s contains an unsupported XML namespace", name)
	}
	if strings.HasPrefix(attributeName, "on") || attributeName == "style" || attributeName == "src" || attributeName == "base" {
		return invalidDocumentation("%s contains an active SVG attribute", name)
	}
	if attributeName == "href" {
		if len(attributeValue) < 2 || !strings.HasPrefix(attributeValue, "#") {
			return invalidDocumentation("%s cannot reference external SVG resources", name)
		}
	}

	lowerValue := strings.ToLower(attributeValue)
	for _, scheme := range []string{"javascript:", "data:", "file:", "http:", "https:"} {
		if strings.Contains(lowerValue, scheme) {
			return invalidDocumentation("%s cannot reference external SVG resources", name)
		}
	}
	if strings.Contains(lowerValue, "//") || !svgURLsUseLocalFragments(attributeValue) {
		return invalidDocumentation("%s cannot reference external SVG resources", name)
	}
	return nil
}

func svgURLsUseLocalFragments(value string) bool {
	lowerValue := strings.ToLower(value)
	for {
		start := strings.Index(lowerValue, "url(")
		if start < 0 {
			return true
		}
		remainder := value[start+4:]
		end := strings.IndexByte(remainder, ')')
		if end < 0 {
			return false
		}
		target := strings.Trim(strings.TrimSpace(remainder[:end]), "\"'")
		if len(target) < 2 || !strings.HasPrefix(target, "#") {
			return false
		}
		value = remainder[end+1:]
		lowerValue = strings.ToLower(value)
	}
}

func isSafeDocumentationURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\x00\r\n\\") {
		return false
	}
	if strings.HasPrefix(value, "/") {
		return !strings.HasPrefix(value, "//")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func allowedDocumentationAssetType(contentType string) bool {
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func invalidDocumentation(format string, args ...any) error {
	return ErrDocumentationInvalid.WithMetadata(map[string]string{"detail": fmt.Sprintf(format, args...)})
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (s *DocumentationService) ensureInitialized(ctx context.Context) error {
	if s.initialized.Load() {
		return nil
	}
	s.seedMu.Lock()
	defer s.seedMu.Unlock()
	if s.initialized.Load() {
		return nil
	}
	canonical, err := ValidateDocumentationContent(defaultDocumentationContent)
	if err != nil {
		return fmt.Errorf("validate bundled documentation: %w", err)
	}
	if err := s.repo.EnsureSeed(ctx, canonical); err != nil {
		return err
	}
	s.initialized.Store(true)
	return nil
}
