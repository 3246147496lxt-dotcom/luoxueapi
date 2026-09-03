package skillimport

import (
	"context"
	"crypto/sha1" // GitHub's tree API exposes Git object IDs, which are SHA-1 today.
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const GitHubAdapterType = "github"

var githubRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,199}$`)

type GitHubAdapter struct {
	fetcher HTTPFetcher
	token   string
	cacheMu sync.RWMutex
	cache   map[string]gitHubSnapshot
	flight  singleflight.Group
	now     func() time.Time
}

type gitHubSnapshot struct {
	commit   string
	tree     gitHubTreeResponse
	evidence []Evidence
	expires  time.Time
}

type gitHubConfig struct {
	Repository  string `json:"repository"`
	Path        string `json:"path"`
	Ref         string `json:"ref,omitempty"`
	SkillID     string `json:"skill_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type gitHubOpaque struct {
	Repository string `json:"repository"`
	Path       string `json:"path"`
	Ref        string `json:"ref"`
	SkillID    string `json:"skill_id"`
}

type gitHubTreeFile struct {
	fullPath string
	relative string
	sha      string
	size     int64
	mode     string
}

type gitHubCommitResponse struct {
	SHA string `json:"sha"`
}

type gitHubTreeResponse struct {
	SHA       string `json:"sha"`
	Truncated bool   `json:"truncated"`
	Tree      []struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		SHA  string `json:"sha"`
		Size int64  `json:"size"`
	} `json:"tree"`
}

func NewGitHubAdapter(fetcher HTTPFetcher, token string) *GitHubAdapter {
	return &GitHubAdapter{
		fetcher: fetcher, token: strings.TrimSpace(token), cache: make(map[string]gitHubSnapshot), now: time.Now,
	}
}

func (a *GitHubAdapter) Type() string    { return GitHubAdapterType }
func (a *GitHubAdapter) Version() string { return "1.1.0" }

func (a *GitHubAdapter) ValidateConfig(raw json.RawMessage) error {
	_, err := parseGitHubConfig(raw)
	return err
}

func parseGitHubConfig(raw json.RawMessage) (gitHubConfig, error) {
	config := gitHubConfig{Ref: "HEAD"}
	if err := decodeConfig(raw, &config); err != nil {
		return gitHubConfig{}, err
	}
	owner, repository, err := parseGitHubRepository(config.Repository)
	if err != nil {
		return gitHubConfig{}, err
	}
	config.Repository = owner + "/" + repository
	config.Path = strings.Trim(strings.TrimSpace(config.Path), "/")
	if config.Path != "" && (path.Clean(config.Path) != config.Path || strings.Contains(config.Path, "\\")) {
		return gitHubConfig{}, errors.New("path must be a canonical repository-relative directory")
	}
	for _, component := range strings.Split(config.Path, "/") {
		if config.Path == "" {
			break
		}
		if component == "" || component == "." || component == ".." {
			return gitHubConfig{}, errors.New("path contains an unsafe component")
		}
	}
	config.Ref = strings.TrimSpace(config.Ref)
	if !githubRefPattern.MatchString(config.Ref) || strings.Contains(config.Ref, "..") || strings.HasSuffix(config.Ref, "/") {
		return gitHubConfig{}, errors.New("ref contains unsupported characters")
	}
	config.SkillID = strings.TrimSpace(config.SkillID)
	if config.SkillID == "" && config.Path != "" {
		config.SkillID = path.Base(config.Path)
	}
	if config.SkillID == "" || len(config.SkillID) > 255 {
		return gitHubConfig{}, errors.New("skill_id is required when path is omitted")
	}
	config.Name = strings.TrimSpace(config.Name)
	if config.Name == "" {
		config.Name = config.SkillID
	}
	config.Description = strings.TrimSpace(config.Description)
	return config, nil
}

func (a *GitHubAdapter) Discover(_ context.Context, request DiscoverRequest) (DiscoveryPage, error) {
	config, err := parseGitHubConfig(request.Config)
	if err != nil {
		return DiscoveryPage{}, NewAdapterError(GitHubAdapterType, "discover", ErrorInvalidConfig, err)
	}
	selection, err := parseSelection(request.Selection)
	if err != nil {
		return DiscoveryPage{}, NewAdapterError(GitHubAdapterType, "discover", ErrorInvalidConfig, err)
	}
	if strings.TrimSpace(request.Cursor) != "" || selection.StartRank > 1 {
		return DiscoveryPage{Items: []DiscoveredSkill{}}, nil
	}
	canonical := fmt.Sprintf("https://github.com/%s/tree/%s", config.Repository, escapeURLPath(config.Ref))
	if config.Path != "" {
		canonical += "/" + escapeURLPath(config.Path)
	}
	return DiscoveryPage{Items: []DiscoveredSkill{{
		AdapterType: GitHubAdapterType, Namespace: config.Repository, ExternalID: config.SkillID,
		SuggestedName: config.Name, SuggestedSlug: config.SkillID, Description: config.Description,
		CanonicalURL: canonical, Revision: config.Ref, Rank: intPointer(1),
		Opaque: safeOpaque(gitHubOpaque{Repository: config.Repository, Path: config.Path, Ref: config.Ref, SkillID: config.SkillID}),
	}}}, nil
}

func (a *GitHubAdapter) Acquire(ctx context.Context, request AcquireRequest) (SourceBundle, error) {
	if a == nil || a.fetcher == nil {
		return SourceBundle{}, NewAdapterError(GitHubAdapterType, "acquire", ErrorInvalidConfig, errors.New("HTTP fetcher is required"))
	}
	if _, err := parseGitHubConfig(request.Config); err != nil {
		return SourceBundle{}, NewAdapterError(GitHubAdapterType, "acquire", ErrorInvalidConfig, err)
	}
	var opaque gitHubOpaque
	if err := json.Unmarshal(request.Skill.Opaque, &opaque); err != nil {
		return SourceBundle{}, NewAdapterError(GitHubAdapterType, "acquire", ErrorInvalidSource, errors.New("skill discovery payload is invalid"))
	}
	owner, repository, err := parseGitHubRepository(opaque.Repository)
	if err != nil {
		return SourceBundle{}, NewAdapterError(GitHubAdapterType, "acquire", ErrorInvalidSource, err)
	}
	pathConfig, err := parseGitHubConfig(safeOpaque(gitHubConfig{Repository: opaque.Repository, Path: opaque.Path, Ref: opaque.Ref, SkillID: opaque.SkillID}))
	if err != nil {
		return SourceBundle{}, NewAdapterError(GitHubAdapterType, "acquire", ErrorInvalidSource, err)
	}
	snapshot, err := a.repositorySnapshot(ctx, owner, repository, pathConfig.Ref)
	if err != nil {
		return SourceBundle{}, err
	}
	if pathConfig.Path == "" {
		pathConfig.Path, err = locateGitHubSkillPath(snapshot.tree, pathConfig.SkillID)
		if err != nil {
			return SourceBundle{}, err
		}
	}
	prefix := pathConfig.Path + "/"
	rootSkill := pathConfig.Path == ""
	selected := make([]gitHubTreeFile, 0)
	rootLicenses := make([]gitHubTreeFile, 0)
	for _, entry := range snapshot.tree.Tree {
		if rootSkill {
			if entry.Path == "SKILL.md" {
				if entry.Type != "blob" || entry.Mode == "120000" {
					return SourceBundle{}, NewAdapterError(GitHubAdapterType, "read tree", ErrorUnsafe, errors.New("root SKILL.md is a link or non-file entry"))
				}
				selected = append(selected, gitHubTreeFile{fullPath: entry.Path, relative: "SKILL.md", sha: entry.SHA, size: entry.Size, mode: entry.Mode})
				continue
			}
			if entry.Type == "blob" && (entry.Mode == "100644" || entry.Mode == "100755") &&
				!strings.Contains(entry.Path, "/") && isLicenseBase(entry.Path) {
				rootLicenses = append(rootLicenses, gitHubTreeFile{fullPath: entry.Path, relative: entry.Path, sha: entry.SHA, size: entry.Size, mode: entry.Mode})
				continue
			}
			if isRootSkillCompanionPath(entry.Path) {
				if entry.Type == "commit" || entry.Mode == "120000" {
					return SourceBundle{}, NewAdapterError(GitHubAdapterType, "read tree", ErrorUnsafe, fmt.Errorf("root Skill companion path contains a link or submodule at %q", entry.Path))
				}
				if entry.Type == "blob" {
					selected = append(selected, gitHubTreeFile{fullPath: entry.Path, relative: entry.Path, sha: entry.SHA, size: entry.Size, mode: entry.Mode})
				}
			}
			continue
		}
		inside := strings.HasPrefix(entry.Path, prefix)
		if inside && (entry.Type == "commit" || entry.Mode == "120000") {
			return SourceBundle{}, NewAdapterError(GitHubAdapterType, "read tree", ErrorUnsafe, fmt.Errorf("skill path contains a link or submodule at %q", entry.Path))
		}
		if inside && entry.Type != "blob" {
			continue
		}
		if inside {
			relative := strings.TrimPrefix(entry.Path, prefix)
			if relative == "" {
				continue
			}
			selected = append(selected, gitHubTreeFile{fullPath: entry.Path, relative: relative, sha: entry.SHA, size: entry.Size, mode: entry.Mode})
			continue
		}
		if entry.Type == "blob" && (entry.Mode == "100644" || entry.Mode == "100755") &&
			!strings.Contains(entry.Path, "/") && isLicenseBase(entry.Path) {
			rootLicenses = append(rootLicenses, gitHubTreeFile{fullPath: entry.Path, relative: entry.Path, sha: entry.SHA, size: entry.Size, mode: entry.Mode})
		}
	}
	if len(selected) == 0 {
		return SourceBundle{}, NewAdapterError(GitHubAdapterType, "read tree", ErrorNotFound, errors.New("configured repository path contains no files"))
	}
	if !treeFilesHaveLicense(selected) && len(rootLicenses) > 0 {
		sort.Slice(rootLicenses, func(i, j int) bool { return rootLicenses[i].fullPath < rootLicenses[j].fullPath })
		selected = append(selected, rootLicenses[0])
	}
	if len(selected) > MaxSourceArchiveEntries {
		return SourceBundle{}, NewAdapterError(GitHubAdapterType, "read tree", ErrorBlocked, fmt.Errorf("skill path has more than %d files", MaxSourceArchiveEntries))
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].relative < selected[j].relative })
	files := make([]SourceFile, 0, len(selected))
	evidence := append([]Evidence(nil), snapshot.evidence...)
	var total int64
	for _, entry := range selected {
		if entry.size < 0 || entry.size > MaxSourceFileBytes {
			return SourceBundle{}, NewAdapterError(GitHubAdapterType, "read tree", ErrorBlocked, fmt.Errorf("source file %q exceeds the per-file limit", entry.fullPath))
		}
		total += entry.size
		if total > MaxSourceUnpackedBytes {
			return SourceBundle{}, NewAdapterError(GitHubAdapterType, "read tree", ErrorBlocked, errors.New("skill path exceeds the unpacked source limit"))
		}
		rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", url.PathEscape(owner), url.PathEscape(repository), snapshot.commit, escapeURLPath(entry.fullPath))
		result, fetchErr := a.fetcher.Get(ctx, rawURL, FetchOptions{
			Adapter: GitHubAdapterType, Operation: "fetch blob", AllowedHosts: []string{"raw.githubusercontent.com"},
			MaxBytes: MaxSourceFileBytes, Headers: http.Header{"Accept": []string{"application/octet-stream"}},
		})
		if fetchErr != nil {
			return SourceBundle{}, fetchErr
		}
		if int64(len(result.Body)) != entry.size {
			return SourceBundle{}, NewAdapterError(GitHubAdapterType, "verify blob", ErrorIntegrity, fmt.Errorf("GitHub size mismatch for %q", entry.fullPath))
		}
		if !strings.EqualFold(gitBlobObjectID(result.Body), entry.sha) {
			return SourceBundle{}, NewAdapterError(GitHubAdapterType, "verify blob", ErrorIntegrity, fmt.Errorf("git object ID mismatch for %q", entry.fullPath))
		}
		// Enforce the aggregate limit against bytes actually received as well as
		// tree metadata, so a malformed provider response cannot amplify memory.
		if total-int64(entry.size)+int64(len(result.Body)) > MaxSourceUnpackedBytes {
			return SourceBundle{}, NewAdapterError(GitHubAdapterType, "read tree", ErrorBlocked, errors.New("skill path exceeds the unpacked source limit"))
		}
		files = append(files, SourceFile{Path: entry.relative, Data: result.Body, Executable: entry.mode == "100755"})
		evidence = append(evidence, result.Evidence)
	}
	if err := validateSourceFiles(files); err != nil {
		return SourceBundle{}, NewAdapterError(GitHubAdapterType, "acquire", ErrorUnsafe, err)
	}
	canonical := fmt.Sprintf("https://github.com/%s/tree/%s/%s", pathConfig.Repository, snapshot.commit, escapeURLPath(pathConfig.Path))
	return SourceBundle{
		Files: files, Revision: snapshot.commit, UpstreamContentHash: canonicalFilesHash(files),
		CanonicalURL: canonical, LicenseEvidence: licenseEvidenceFromFiles(files), Evidence: evidence,
		IntegrityVerified: true,
	}, nil
}

func isRootSkillCompanionPath(value string) bool {
	for _, prefix := range []string{"scripts/", "references/", "assets/"} {
		if strings.HasPrefix(value, prefix) && len(value) > len(prefix) {
			return true
		}
	}
	return false
}

func (a *GitHubAdapter) apiHeaders() http.Header {
	headers := http.Header{
		"Accept":               []string{"application/vnd.github+json"},
		"X-GitHub-Api-Version": []string{"2022-11-28"},
	}
	if a != nil && strings.TrimSpace(a.token) != "" {
		headers.Set("Authorization", "Bearer "+strings.TrimSpace(a.token))
	}
	return headers
}

func (a *GitHubAdapter) repositorySnapshot(ctx context.Context, owner, repository, ref string) (gitHubSnapshot, error) {
	key := strings.ToLower(owner+"/"+repository) + "\x00" + ref
	now := time.Now()
	if a.now != nil {
		now = a.now()
	}
	a.cacheMu.RLock()
	cached, exists := a.cache[key]
	a.cacheMu.RUnlock()
	if exists && now.Before(cached.expires) {
		return cached, nil
	}
	value, err, _ := a.flight.Do(key, func() (any, error) {
		now := time.Now()
		if a.now != nil {
			now = a.now()
		}
		a.cacheMu.RLock()
		cached, exists := a.cache[key]
		a.cacheMu.RUnlock()
		if exists && now.Before(cached.expires) {
			return cached, nil
		}
		headers := a.apiHeaders()
		commitURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", url.PathEscape(owner), url.PathEscape(repository), escapeURLPath(ref))
		commitResult, fetchErr := a.fetcher.Get(ctx, commitURL, jsonFetchOptions(GitHubAdapterType, "resolve revision", []string{"api.github.com"}, 2*1024*1024, headers))
		if fetchErr != nil {
			return nil, fetchErr
		}
		var commit gitHubCommitResponse
		if decodeErr := json.Unmarshal(commitResult.Body, &commit); decodeErr != nil || !isGitHubObjectID(commit.SHA) {
			if decodeErr == nil {
				decodeErr = errors.New("GitHub returned an invalid commit object ID")
			}
			return nil, NewAdapterError(GitHubAdapterType, "resolve revision", ErrorIntegrity, decodeErr)
		}
		treeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", url.PathEscape(owner), url.PathEscape(repository), commit.SHA)
		treeResult, fetchErr := a.fetcher.Get(ctx, treeURL, jsonFetchOptions(GitHubAdapterType, "read tree", []string{"api.github.com"}, 8*1024*1024, headers))
		if fetchErr != nil {
			return nil, fetchErr
		}
		var tree gitHubTreeResponse
		if decodeErr := json.Unmarshal(treeResult.Body, &tree); decodeErr != nil {
			return nil, NewAdapterError(GitHubAdapterType, "read tree", ErrorInvalidSource, decodeErr)
		}
		if tree.Truncated {
			return nil, NewAdapterError(GitHubAdapterType, "read tree", ErrorIntegrity, errors.New("GitHub tree response was truncated"))
		}
		snapshot := gitHubSnapshot{
			commit: commit.SHA, tree: tree, evidence: []Evidence{commitResult.Evidence, treeResult.Evidence},
			expires: now.Add(5 * time.Minute),
		}
		a.cacheMu.Lock()
		if len(a.cache) >= 128 {
			for cacheKey, candidate := range a.cache {
				if now.After(candidate.expires) {
					delete(a.cache, cacheKey)
				}
			}
			if len(a.cache) >= 128 {
				for cacheKey := range a.cache {
					delete(a.cache, cacheKey)
					break
				}
			}
		}
		a.cache[key] = snapshot
		a.cacheMu.Unlock()
		return snapshot, nil
	})
	if err != nil {
		return gitHubSnapshot{}, err
	}
	snapshot, ok := value.(gitHubSnapshot)
	if !ok {
		return gitHubSnapshot{}, fmt.Errorf("resolve GitHub repository snapshot: unexpected result type %T", value)
	}
	return snapshot, nil
}

func locateGitHubSkillPath(tree gitHubTreeResponse, skillID string) (string, error) {
	wanted := strings.ToLower(strings.TrimSpace(skillID))
	type candidate struct {
		path  string
		score int
	}
	candidates := make([]candidate, 0)
	rootSkill := false
	for _, entry := range tree.Tree {
		if entry.Type != "blob" || !strings.EqualFold(path.Base(entry.Path), "SKILL.md") {
			continue
		}
		directory := path.Dir(entry.Path)
		if directory == "." {
			rootSkill = true
			directory = ""
		}
		base := strings.ToLower(path.Base(directory))
		if base != wanted {
			continue
		}
		score := 100 + strings.Count(directory, "/")
		switch strings.ToLower(directory) {
		case wanted:
			score = 0
		case "skills/" + wanted:
			score = 1
		case ".agents/skills/" + wanted:
			score = 2
		case ".claude/skills/" + wanted:
			score = 3
		case ".codex/skills/" + wanted:
			score = 4
		}
		candidates = append(candidates, candidate{path: directory, score: score})
	}
	if len(candidates) == 0 {
		if rootSkill {
			return "", nil
		}
		return "", NewAdapterError(GitHubAdapterType, "locate skill", ErrorNotFound, fmt.Errorf("repository does not contain a unique SKILL.md directory for %q", skillID))
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].score < candidates[j].score
	})
	if len(candidates) > 1 && candidates[0].score == candidates[1].score {
		return "", NewAdapterError(GitHubAdapterType, "locate skill", ErrorIntegrity, fmt.Errorf("repository contains ambiguous SKILL.md directories for %q", skillID))
	}
	return candidates[0].path, nil
}

func isGitHubObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func gitBlobObjectID(data []byte) string {
	hasher := sha1.New()
	_, _ = fmt.Fprintf(hasher, "blob %d%c", len(data), 0)
	_, _ = hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func isLicenseBase(value string) bool {
	base := strings.ToLower(path.Base(value))
	stem := strings.TrimSuffix(base, path.Ext(base))
	return stem == "license" || stem == "licence" || stem == "copying" || stem == "notice"
}

func treeFilesHaveLicense(files []gitHubTreeFile) bool {
	for _, file := range files {
		if isLicenseBase(file.relative) {
			return true
		}
	}
	return false
}
