package skillimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const SkillsSHAdapterType = "skills_sh"

var skillsSHViewPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type SkillsSHAdapter struct {
	fetcher   HTTPFetcher
	github    *GitHubAdapter
	quotaMu   sync.Mutex
	quotaHour time.Time
	quotaUsed int
	now       func() time.Time
}

type skillsSHConfig struct {
	BaseURL              string            `json:"base_url,omitempty"`
	View                 string            `json:"view,omitempty"`
	AllowedHosts         []string          `json:"allowed_hosts,omitempty"`
	AllowedSourceHosts   []string          `json:"allowed_source_hosts,omitempty"`
	SourceBaseURLs       map[string]string `json:"source_base_urls,omitempty"`
	AcquisitionOrder     []string          `json:"acquisition_order,omitempty"`
	DownloadQuotaPerHour int               `json:"download_quota_per_hour,omitempty"`
}

type skillsSHCursor struct {
	Page     int      `json:"page"`
	Seen     int      `json:"seen"`
	Selected int      `json:"selected"`
	Keys     []string `json:"keys,omitempty"`
}

type skillsSHRankingResponse struct {
	Skills  []skillsSHRankingSkill `json:"skills"`
	Total   int                    `json:"total"`
	Page    int                    `json:"page"`
	HasMore bool                   `json:"hasMore"`
}

type skillsSHRankingSkill struct {
	Source   string `json:"source"`
	SkillID  string `json:"skillId"`
	Name     string `json:"name"`
	Installs int64  `json:"installs"`
}

type skillsSHOpaque struct {
	Source  string `json:"source"`
	SkillID string `json:"skill_id"`
}

type skillsSHDownloadResponse struct {
	Files []struct {
		Path     string `json:"path"`
		Contents string `json:"contents"`
	} `json:"files"`
	Hash string `json:"hash"`
}

func NewSkillsSHAdapter(fetcher HTTPFetcher, github ...*GitHubAdapter) *SkillsSHAdapter {
	githubAdapter := NewGitHubAdapter(fetcher, "")
	if len(github) > 0 && github[0] != nil {
		githubAdapter = github[0]
	}
	return &SkillsSHAdapter{fetcher: fetcher, github: githubAdapter, now: time.Now}
}

func (a *SkillsSHAdapter) Type() string    { return SkillsSHAdapterType }
func (a *SkillsSHAdapter) Version() string { return "1.1.0" }

func (a *SkillsSHAdapter) ValidateConfig(raw json.RawMessage) error {
	_, err := parseSkillsSHConfig(raw, "")
	return err
}

func parseSkillsSHConfig(raw json.RawMessage, requestBaseURL string) (skillsSHConfig, error) {
	config := skillsSHConfig{
		BaseURL: "https://skills.sh", View: "all-time",
		AcquisitionOrder: []string{"github", "skills_sh_snapshot"}, DownloadQuotaPerHour: 60,
	}
	if err := decodeConfig(raw, &config); err != nil {
		return skillsSHConfig{}, err
	}
	if strings.TrimSpace(requestBaseURL) != "" {
		config.BaseURL = strings.TrimSpace(requestBaseURL)
	}
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.View = strings.ToLower(strings.TrimSpace(config.View))
	if !skillsSHViewPattern.MatchString(config.View) {
		return skillsSHConfig{}, errors.New("view must use lowercase kebab-case")
	}
	if _, err := explicitHostsForBase(config.AllowedHosts, config.BaseURL); err != nil {
		return skillsSHConfig{}, err
	}
	for _, host := range config.AllowedSourceHosts {
		if strings.TrimSpace(host) == "" || strings.ContainsAny(host, "/?#@") {
			return skillsSHConfig{}, fmt.Errorf("invalid allowed source host %q", host)
		}
	}
	normalizedSourceBases := make(map[string]string, len(config.SourceBaseURLs))
	for rawSource, rawBaseURL := range config.SourceBaseURLs {
		source := strings.TrimSpace(rawSource)
		baseURL := strings.TrimRight(strings.TrimSpace(rawBaseURL), "/")
		if source == "" {
			return skillsSHConfig{}, errors.New("source_base_urls keys must not be empty")
		}
		if baseURL == "" {
			return skillsSHConfig{}, fmt.Errorf("source_base_urls[%q] must not be empty", source)
		}
		if _, err := validateHTTPSImportTarget(baseURL, config.AllowedSourceHosts); err != nil {
			return skillsSHConfig{}, fmt.Errorf("source_base_urls[%q]: %w", source, err)
		}
		if _, duplicate := normalizedSourceBases[source]; duplicate {
			return skillsSHConfig{}, fmt.Errorf("duplicate normalized source_base_urls key %q", source)
		}
		normalizedSourceBases[source] = baseURL
	}
	config.SourceBaseURLs = normalizedSourceBases
	if len(config.AcquisitionOrder) == 0 {
		config.AcquisitionOrder = []string{"github", "skills_sh_snapshot"}
	}
	seenOrder := make(map[string]struct{}, len(config.AcquisitionOrder))
	for index := range config.AcquisitionOrder {
		config.AcquisitionOrder[index] = strings.ToLower(strings.TrimSpace(config.AcquisitionOrder[index]))
		switch config.AcquisitionOrder[index] {
		case "github", "skills_sh_snapshot":
		default:
			return skillsSHConfig{}, fmt.Errorf("unsupported acquisition_order entry %q", config.AcquisitionOrder[index])
		}
		if _, duplicate := seenOrder[config.AcquisitionOrder[index]]; duplicate {
			return skillsSHConfig{}, fmt.Errorf("duplicate acquisition_order entry %q", config.AcquisitionOrder[index])
		}
		seenOrder[config.AcquisitionOrder[index]] = struct{}{}
	}
	if config.DownloadQuotaPerHour < 0 || config.DownloadQuotaPerHour > 60 {
		return skillsSHConfig{}, errors.New("download_quota_per_hour must be between 0 and 60")
	}
	return config, nil
}

func (a *SkillsSHAdapter) Discover(ctx context.Context, request DiscoverRequest) (DiscoveryPage, error) {
	if a == nil || a.fetcher == nil {
		return DiscoveryPage{}, NewAdapterError(SkillsSHAdapterType, "discover", ErrorInvalidConfig, errors.New("HTTP fetcher is required"))
	}
	config, err := parseSkillsSHConfig(request.Config, request.BaseURL)
	if err != nil {
		return DiscoveryPage{}, NewAdapterError(SkillsSHAdapterType, "discover", ErrorInvalidConfig, err)
	}
	selection, err := parseSelection(request.Selection)
	if err != nil {
		return DiscoveryPage{}, NewAdapterError(SkillsSHAdapterType, "discover", ErrorInvalidConfig, err)
	}
	cursor := skillsSHCursor{}
	if err := decodeCursor(request.Cursor, &cursor); err != nil || cursor.Page < 0 || cursor.Seen < 0 || cursor.Selected < 0 || cursor.Selected > selection.Limit {
		if err == nil {
			err = errors.New("invalid skills.sh cursor state")
		}
		return DiscoveryPage{}, NewAdapterError(SkillsSHAdapterType, "discover", ErrorInvalidSource, err)
	}
	endpoint := fmt.Sprintf("%s/api/skills/%s/%d", config.BaseURL, url.PathEscape(config.View), cursor.Page)
	hosts, _ := explicitHostsForBase(config.AllowedHosts, config.BaseURL)
	result, err := a.fetcher.Get(ctx, endpoint, jsonFetchOptions(SkillsSHAdapterType, "ranking", hosts, 8*1024*1024, nil))
	if err != nil {
		return DiscoveryPage{}, err
	}
	var response skillsSHRankingResponse
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return DiscoveryPage{}, NewAdapterError(SkillsSHAdapterType, "decode ranking", ErrorInvalidSource, err)
	}
	if response.Page != 0 && response.Page != cursor.Page {
		return DiscoveryPage{}, NewAdapterError(SkillsSHAdapterType, "decode ranking", ErrorIntegrity, fmt.Errorf("wanted page %d, got %d", cursor.Page, response.Page))
	}
	if len(response.Skills) == 0 && response.HasMore {
		return DiscoveryPage{}, NewAdapterError(SkillsSHAdapterType, "decode ranking", ErrorIntegrity, errors.New("empty ranking page claims more results"))
	}
	seenKeys := make(map[string]struct{}, len(cursor.Keys)+len(response.Skills))
	for _, key := range cursor.Keys {
		seenKeys[key] = struct{}{}
	}
	items := make([]DiscoveredSkill, 0, len(response.Skills))
	for index, ranked := range response.Skills {
		rank := cursor.Seen + index + 1
		source := strings.TrimSpace(ranked.Source)
		skillID := strings.TrimSpace(ranked.SkillID)
		name := strings.TrimSpace(ranked.Name)
		if source == "" || skillID == "" || name == "" || ranked.Installs < 0 {
			return DiscoveryPage{}, NewAdapterError(SkillsSHAdapterType, "decode ranking", ErrorIntegrity, fmt.Errorf("ranking item %d is malformed", rank))
		}
		key := source + "\x00" + skillID
		if _, duplicate := seenKeys[key]; duplicate {
			continue
		}
		seenKeys[key] = struct{}{}
		if rank < selection.StartRank || cursor.Selected+len(items) >= selection.Limit {
			continue
		}
		canonical := config.BaseURL + "/" + escapeURLPath(source) + "/" + url.PathEscape(skillID)
		items = append(items, DiscoveredSkill{
			AdapterType: SkillsSHAdapterType, Namespace: source, ExternalID: skillID,
			SuggestedName: name, SuggestedSlug: skillID, CanonicalURL: canonical,
			Rank: intPointer(rank), Metrics: map[string]int64{"installs": ranked.Installs},
			Opaque: safeOpaque(skillsSHOpaque{Source: source, SkillID: skillID}),
		})
	}

	cursor.Seen += len(response.Skills)
	cursor.Selected += len(items)
	cursor.Page++
	cursor.Keys = cursor.Keys[:0]
	for key := range seenKeys {
		cursor.Keys = append(cursor.Keys, key)
	}
	sort.Strings(cursor.Keys)
	next := ""
	if response.HasMore && cursor.Selected < selection.Limit {
		next = encodeCursor(cursor)
	}
	return DiscoveryPage{Items: items, NextCursor: next, Evidence: []Evidence{result.Evidence}}, nil
}

func (a *SkillsSHAdapter) Acquire(ctx context.Context, request AcquireRequest) (SourceBundle, error) {
	if a == nil || a.fetcher == nil {
		return SourceBundle{}, NewAdapterError(SkillsSHAdapterType, "acquire", ErrorInvalidConfig, errors.New("HTTP fetcher is required"))
	}
	config, err := parseSkillsSHConfig(request.Config, request.BaseURL)
	if err != nil {
		return SourceBundle{}, NewAdapterError(SkillsSHAdapterType, "acquire", ErrorInvalidConfig, err)
	}
	var opaque skillsSHOpaque
	if err := json.Unmarshal(request.Skill.Opaque, &opaque); err != nil {
		return SourceBundle{}, NewAdapterError(SkillsSHAdapterType, "acquire", ErrorInvalidSource, errors.New("skill discovery payload is invalid"))
	}
	opaque.Source = strings.TrimSpace(opaque.Source)
	opaque.SkillID = strings.TrimSpace(opaque.SkillID)
	if sourceBase := strings.TrimRight(strings.TrimSpace(config.SourceBaseURLs[opaque.Source]), "/"); sourceBase != "" {
		return a.acquireMappedWellKnown(ctx, config, opaque, request.Skill, sourceBase)
	}
	owner, repository, repoErr := parseGitHubRepository(opaque.Source)
	if repoErr != nil {
		return SourceBundle{}, NewAdapterError(SkillsSHAdapterType, "acquire", ErrorInvalidConfig,
			fmt.Errorf("non-GitHub leaderboard source %q requires source_base_urls mapping", opaque.Source))
	}
	var acquisitionErrors []error
	for _, method := range config.AcquisitionOrder {
		switch method {
		case "github":
			gitConfig := gitHubConfig{Repository: owner + "/" + repository, Ref: "HEAD", SkillID: opaque.SkillID, Name: request.Skill.SuggestedName, Description: request.Skill.Description}
			bundle, acquireErr := a.github.Acquire(ctx, AcquireRequest{
				Config: safeOpaque(gitConfig), Skill: DiscoveredSkill{
					AdapterType: GitHubAdapterType, Namespace: owner + "/" + repository, ExternalID: opaque.SkillID,
					SuggestedName: request.Skill.SuggestedName, SuggestedSlug: opaque.SkillID,
					Description: request.Skill.Description,
					Opaque:      safeOpaque(gitHubOpaque{Repository: owner + "/" + repository, Ref: "HEAD", SkillID: opaque.SkillID}),
				},
			})
			if acquireErr == nil {
				// The imported catalog identity belongs to skills.sh. Keep GitHub's
				// immutable commit and HTTP observations as revision/evidence; the
				// public origin link must continue to point at the directory detail.
				bundle.CanonicalURL = skillsSHOriginURL(config, opaque, request.Skill.CanonicalURL)
				return bundle, nil
			}
			if terminalAcquisitionError(acquireErr) {
				return SourceBundle{}, acquireErr
			}
			acquisitionErrors = append(acquisitionErrors, acquireErr)
		case "skills_sh_snapshot":
			if retryAfter, allowed := a.reserveDownloadSlot(config.DownloadQuotaPerHour); !allowed {
				acquisitionErrors = append(acquisitionErrors, &AdapterError{
					Adapter: SkillsSHAdapterType, Operation: "download snapshot", Kind: ErrorRateLimited,
					RetryAfter: retryAfter, Err: errors.New("skills.sh snapshot hourly quota is exhausted"),
				})
				continue
			}
			bundle, acquireErr := a.acquireSnapshot(ctx, config, owner, repository, opaque, request.Skill)
			if acquireErr == nil {
				bundle.CanonicalURL = skillsSHOriginURL(config, opaque, request.Skill.CanonicalURL)
				return bundle, nil
			}
			if terminalAcquisitionError(acquireErr) {
				return SourceBundle{}, acquireErr
			}
			if ErrorKindOf(acquireErr) == ErrorRateLimited {
				a.exhaustDownloadQuota(config.DownloadQuotaPerHour)
			}
			acquisitionErrors = append(acquisitionErrors, acquireErr)
		}
	}
	return SourceBundle{}, combineAcquisitionErrors(acquisitionErrors)
}

func (a *SkillsSHAdapter) acquireMappedWellKnown(
	ctx context.Context,
	config skillsSHConfig,
	opaque skillsSHOpaque,
	skill DiscoveredSkill,
	sourceBase string,
) (SourceBundle, error) {
	if _, validateErr := validateHTTPSImportTarget(sourceBase, config.AllowedSourceHosts); validateErr != nil {
		return SourceBundle{}, NewAdapterError(SkillsSHAdapterType, "acquire", ErrorUnsafe, validateErr)
	}
	wellKnownConfig, _ := json.Marshal(wellKnownConfig{AllowedHosts: config.AllowedSourceHosts})
	adapter := NewWellKnownAdapter(a.fetcher)
	discovered := skill
	discovered.AdapterType = WellKnownAdapterType
	discovered.Namespace = opaque.Source
	discovered.ExternalID = opaque.SkillID
	bundle, acquireErr := adapter.Acquire(ctx, AcquireRequest{Config: wellKnownConfig, BaseURL: sourceBase, Skill: discovered})
	if acquireErr != nil {
		return SourceBundle{}, acquireErr
	}
	bundle.CanonicalURL = skillsSHOriginURL(config, opaque, skill.CanonicalURL)
	return bundle, nil
}

func skillsSHOriginURL(config skillsSHConfig, skill skillsSHOpaque, discoveredURL string) string {
	if canonical := strings.TrimSpace(discoveredURL); canonical != "" {
		return canonical
	}
	return config.BaseURL + "/" + escapeURLPath(skill.Source) + "/" + url.PathEscape(skill.SkillID)
}

func terminalAcquisitionError(err error) bool {
	switch ErrorKindOf(err) {
	case ErrorUnsafe, ErrorIntegrity, ErrorBlocked:
		return true
	default:
		return false
	}
}

func (a *SkillsSHAdapter) acquireSnapshot(ctx context.Context, config skillsSHConfig, owner, repository string, opaque skillsSHOpaque, skill DiscoveredSkill) (SourceBundle, error) {
	endpoint := fmt.Sprintf("%s/api/download/%s/%s/%s", config.BaseURL, url.PathEscape(owner), url.PathEscape(repository), url.PathEscape(opaque.SkillID))
	hosts, _ := explicitHostsForBase(config.AllowedHosts, config.BaseURL)
	result, err := a.fetcher.Get(ctx, endpoint, jsonFetchOptions(SkillsSHAdapterType, "download", hosts, MaxSourceArchiveBytes, http.Header{"Accept": []string{"application/json"}}))
	if err != nil {
		return SourceBundle{}, err
	}
	var response skillsSHDownloadResponse
	if err := json.Unmarshal(result.Body, &response); err != nil {
		return SourceBundle{}, NewAdapterError(SkillsSHAdapterType, "decode download", ErrorInvalidSource, err)
	}
	files := make([]SourceFile, 0, len(response.Files))
	for _, file := range response.Files {
		files = append(files, SourceFile{Path: file.Path, Data: []byte(file.Contents)})
	}
	if err := validateSourceFiles(files); err != nil {
		return SourceBundle{}, NewAdapterError(SkillsSHAdapterType, "decode download", ErrorUnsafe, err)
	}
	contentHash := canonicalFilesHash(files)
	// skills.sh documents this value as its snapshot/cache revision, not as a
	// canonical digest over the returned file set. Preserve it as an opaque
	// revision and compute our own logical content hash, but never claim that
	// the provider cryptographically verified those bytes. Automatic imports
	// therefore stop at the normal safe review gate when this fallback channel
	// is used.
	const snapshotReviewReason = "skills.sh download hash is an opaque snapshot revision, not a verified file-set digest"
	canonical := skill.CanonicalURL
	if canonical == "" {
		canonical = config.BaseURL + "/" + escapeURLPath(opaque.Source) + "/" + url.PathEscape(opaque.SkillID)
	}
	return SourceBundle{
		Files: files, Revision: strings.TrimSpace(response.Hash), UpstreamContentHash: contentHash,
		CanonicalURL: canonical, LicenseEvidence: licenseEvidenceFromFiles(files), Evidence: []Evidence{result.Evidence},
		IntegrityVerified: false, NeedsReview: true, ReviewReasons: []string{snapshotReviewReason},
	}, nil
}

func (a *SkillsSHAdapter) reserveDownloadSlot(limit int) (time.Duration, bool) {
	if limit <= 0 {
		return time.Hour, false
	}
	now := time.Now().UTC()
	if a.now != nil {
		now = a.now().UTC()
	}
	hour := now.Truncate(time.Hour)
	a.quotaMu.Lock()
	defer a.quotaMu.Unlock()
	if !a.quotaHour.Equal(hour) {
		a.quotaHour = hour
		a.quotaUsed = 0
	}
	if a.quotaUsed >= limit {
		return hour.Add(time.Hour).Sub(now), false
	}
	a.quotaUsed++
	return 0, true
}

func (a *SkillsSHAdapter) exhaustDownloadQuota(limit int) {
	a.quotaMu.Lock()
	if a.quotaUsed < limit {
		a.quotaUsed = limit
	}
	a.quotaMu.Unlock()
}

func combineAcquisitionErrors(values []error) error {
	joined := errors.Join(values...)
	if joined == nil {
		return NewAdapterError(SkillsSHAdapterType, "acquire", ErrorInvalidConfig, errors.New("acquisition_order contains no usable method"))
	}
	for _, wanted := range []ErrorKind{ErrorRateLimited, ErrorTemporary} {
		for _, value := range values {
			if ErrorKindOf(value) != wanted {
				continue
			}
			retryAfter, _ := RetryAfterOf(value)
			return &AdapterError{
				Adapter: SkillsSHAdapterType, Operation: "acquire", Kind: wanted,
				RetryAfter: retryAfter, Err: joined,
			}
		}
	}
	return joined
}

func escapeURLPath(value string) string {
	parts := strings.Split(strings.Trim(value, "/"), "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}
