package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type acquireConfig struct {
	OutputDir    string
	Mode         string
	NpxPath      string
	DownloadBase string
	HTTPClient   *http.Client
}

type downloadResponse struct {
	Files []struct {
		Path     string `json:"path"`
		Contents string `json:"contents"`
	} `json:"files"`
	Hash string `json:"hash"`
}

type skillsLock struct {
	Skills map[string]struct {
		Source    string `json:"source"`
		SkillPath string `json:"skillPath"`
	} `json:"skills"`
}

var gitCommitPattern = regexp.MustCompile(`^[0-9a-fA-F]{40,64}$`)

func acquireAll(ctx context.Context, config acquireConfig, snapshot snapshotFile, checkpoint *checkpointFile, saveCheckpoint func() error) map[string]acquiredSkill {
	result := make(map[string]acquiredSkill, len(snapshot.Skills))
	downloadAPIAvailable := true
	groups := make(map[string][]snapshotEntry)
	var sourceOrder []string
	for _, entry := range snapshot.Skills {
		key := skillKey(entry)
		if acquired, ok := acquiredFromCheckpoint(config.OutputDir, checkpoint.Skills[key]); ok {
			result[key] = acquired
			continue
		}
		if _, exists := groups[entry.Source]; !exists {
			sourceOrder = append(sourceOrder, entry.Source)
		}
		groups[entry.Source] = append(groups[entry.Source], entry)
	}

	for _, source := range sourceOrder {
		entries := groups[source]
		sourceCache := filepath.Join(config.OutputDir, "cache", "sources", sourceCacheName(source))
		_ = os.MkdirAll(sourceCache, 0o755)
		sourceState := checkpoint.Sources[source]
		if sourceState == nil {
			sourceState = &sourceCheckpoint{Source: source, SourceURL: sourceURL(source), CachePath: relativePath(config.OutputDir, sourceCache)}
			checkpoint.Sources[source] = sourceState
		}
		sourceState.LastAttemptAt = time.Now().UTC()
		var cliQueue []snapshotEntry
		for _, entry := range entries {
			key := skillKey(entry)
			state := ensureSkillCheckpoint(checkpoint, entry)
			state.Attempts++
			state.LastAttemptAt = time.Now().UTC()
			state.Status = "acquiring"
			state.LastError = ""

			if (config.Mode == "auto" || config.Mode == "api") && downloadAPIAvailable && isGitHubSource(source) {
				acquired, err := acquireFromDownloadAPI(ctx, config, sourceCache, entry)
				if err == nil {
					result[key] = acquired
					setAcquiredCheckpoint(config.OutputDir, state, acquired)
					_ = saveCheckpoint()
					continue
				}
				state.LastError = "skills.sh download API: " + err.Error()
				if strings.Contains(err.Error(), "HTTP 429") {
					downloadAPIAvailable = false
				}
				if config.Mode == "api" {
					state.Status = "failed"
					_ = saveCheckpoint()
					continue
				}
			}
			if config.Mode == "api" && isGitHubSource(source) && !downloadAPIAvailable {
				state.Status = "failed"
				state.LastError = "skills.sh download API disabled after HTTP 429; rerun to resume or use -acquire auto"
				_ = saveCheckpoint()
				continue
			}
			if config.Mode == "offline" {
				state.Status = "failed"
				state.LastError = "skill is not present in the resumable cache and acquisition mode is offline"
				_ = saveCheckpoint()
				continue
			}
			cliQueue = append(cliQueue, entry)
		}

		useCLI := config.Mode == "cli" || config.Mode == "auto" || (config.Mode == "api" && !isGitHubSource(source))
		if len(cliQueue) > 0 && useCLI {
			cliResults, err := acquireSourceWithCLI(ctx, config, sourceCache, source, cliQueue)
			if err != nil {
				sourceState.LastError = err.Error()
			} else {
				sourceState.LastError = ""
			}
			if isGitHubSource(source) && sourceState.HeadCommit == "" {
				sourceState.HeadCommit = resolveHeadCommit(ctx, sourceURL(source))
			}
			for _, entry := range cliQueue {
				key := skillKey(entry)
				state := ensureSkillCheckpoint(checkpoint, entry)
				if acquired, ok := cliResults[key]; ok {
					result[key] = acquired
					setAcquiredCheckpoint(config.OutputDir, state, acquired)
					continue
				}
				state.Status = "failed"
				if state.LastError == "" {
					state.LastError = sourceState.LastError
				}
				if state.LastError == "" {
					state.LastError = "official skills CLI did not install the requested skill"
				}
			}
			_ = saveCheckpoint()
		}
	}
	return result
}

func acquireFromDownloadAPI(ctx context.Context, config acquireConfig, sourceCache string, entry snapshotEntry) (acquiredSkill, error) {
	parts := strings.Split(entry.Source, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return acquiredSkill{}, errors.New("download API requires an owner/repository source")
	}
	itemDir := filepath.Join(sourceCache, "downloads", itemCacheName(entry))
	markerPath := filepath.Join(itemDir, ".skills-sh-download.json")
	if raw, err := os.ReadFile(markerPath); err == nil {
		var marker struct {
			Hash string `json:"hash"`
		}
		if json.Unmarshal(raw, &marker) == nil {
			if info, skillErr := os.Stat(filepath.Join(itemDir, "SKILL.md")); skillErr == nil && !info.IsDir() {
				return acquiredSkill{Path: itemDir, Acquisition: "skills.sh-download-api", DownloadHash: marker.Hash}, nil
			}
		}
	}

	endpoint := strings.TrimRight(config.DownloadBase, "/") + "/" +
		escapeURLSegment(parts[0]) + "/" + escapeURLSegment(parts[1]) + "/" + escapeURLSegment(entry.OriginalSlug)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return acquiredSkill{}, err
	}
	request.Header.Set("User-Agent", "sub2api-skill-market-prepare/1")
	response, err := config.HTTPClient.Do(request)
	if err != nil {
		return acquiredSkill{}, err
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 16*1024*1024))
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		return acquiredSkill{}, fmt.Errorf("read response: %v %v", readErr, closeErr)
	}
	if response.StatusCode != http.StatusOK {
		return acquiredSkill{}, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	var downloaded downloadResponse
	if err := json.Unmarshal(body, &downloaded); err != nil {
		return acquiredSkill{}, fmt.Errorf("decode response: %w", err)
	}
	if len(downloaded.Files) == 0 {
		return acquiredSkill{}, errors.New("download response contains no files")
	}
	parent := filepath.Dir(itemDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return acquiredSkill{}, err
	}
	temporary, err := os.MkdirTemp(parent, ".download-*")
	if err != nil {
		return acquiredSkill{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(temporary)
		}
	}()
	for _, file := range downloaded.Files {
		relative, pathErr := safeRelativePath(file.Path)
		if pathErr != nil {
			return acquiredSkill{}, pathErr
		}
		target := filepath.Join(temporary, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return acquiredSkill{}, err
		}
		if err := os.WriteFile(target, []byte(file.Contents), 0o644); err != nil {
			return acquiredSkill{}, err
		}
	}
	if _, err := os.Stat(filepath.Join(temporary, "SKILL.md")); err != nil {
		return acquiredSkill{}, errors.New("download response has no root SKILL.md")
	}
	marker, _ := json.MarshalIndent(struct {
		Hash       string    `json:"hash"`
		Downloaded time.Time `json:"downloaded_at"`
	}{downloaded.Hash, time.Now().UTC()}, "", "  ")
	if err := os.WriteFile(filepath.Join(temporary, ".skills-sh-download.json"), append(marker, '\n'), 0o644); err != nil {
		return acquiredSkill{}, err
	}
	if err := os.Rename(temporary, itemDir); err != nil {
		if _, statErr := os.Stat(itemDir); statErr == nil {
			return acquiredSkill{Path: itemDir, Acquisition: "skills.sh-download-api", DownloadHash: downloaded.Hash}, nil
		}
		return acquiredSkill{}, err
	}
	cleanup = false
	return acquiredSkill{Path: itemDir, Acquisition: "skills.sh-download-api", DownloadHash: downloaded.Hash}, nil
}

func acquireSourceWithCLI(ctx context.Context, config acquireConfig, sourceCache, source string, entries []snapshotEntry) (map[string]acquiredSkill, error) {
	results := make(map[string]acquiredSkill, len(entries))
	missing := make([]snapshotEntry, 0, len(entries))
	for _, entry := range entries {
		if directory := locateCLISkill(sourceCache, entry.OriginalSlug); directory != "" {
			results[skillKey(entry)] = acquiredSkill{Path: directory, Acquisition: "official-skills-cli"}
		} else {
			missing = append(missing, entry)
		}
	}
	if len(missing) == 0 {
		return results, nil
	}

	args := []string{"-y", "skills@latest", "add", sourceURL(source), "--skill"}
	for _, entry := range missing {
		args = append(args, entry.OriginalSlug)
	}
	args = append(args, "--agent", "codex", "--copy", "-y")
	groupErr := runSkillsCLI(ctx, config.NpxPath, sourceCache, args)
	for _, entry := range missing {
		if directory := locateCLISkill(sourceCache, entry.OriginalSlug); directory != "" {
			results[skillKey(entry)] = acquiredSkill{Path: directory, Acquisition: "official-skills-cli"}
		}
	}
	if len(results) == len(entries) {
		return results, nil
	}

	var individualErrors []error
	for _, entry := range missing {
		if _, ok := results[skillKey(entry)]; ok {
			continue
		}
		individualArgs := []string{"-y", "skills@latest", "add", sourceURL(source), "--skill", entry.OriginalSlug, "--agent", "codex", "--copy", "-y"}
		if err := runSkillsCLI(ctx, config.NpxPath, sourceCache, individualArgs); err != nil {
			individualErrors = append(individualErrors, fmt.Errorf("%s: %w", entry.OriginalSlug, err))
			continue
		}
		if directory := locateCLISkill(sourceCache, entry.OriginalSlug); directory != "" {
			results[skillKey(entry)] = acquiredSkill{Path: directory, Acquisition: "official-skills-cli"}
		} else {
			individualErrors = append(individualErrors, fmt.Errorf("%s: CLI completed but installed directory was not found", entry.OriginalSlug))
		}
	}
	if len(results) != len(entries) {
		if groupErr != nil {
			individualErrors = append([]error{fmt.Errorf("group install: %w", groupErr)}, individualErrors...)
		}
		return results, errors.Join(individualErrors...)
	}
	return results, nil
}

func runSkillsCLI(ctx context.Context, npxPath, directory string, args []string) error {
	logFile, err := os.OpenFile(filepath.Join(directory, "skills-cli.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()
	command := exec.CommandContext(ctx, npxPath, args...)
	command.Dir = directory
	command.Env = append(os.Environ(), "DISABLE_TELEMETRY=1", "NO_COLOR=1")
	command.Stdout = logFile
	command.Stderr = logFile
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", npxPath, strings.Join(args, " "), err)
	}
	return nil
}

func locateCLISkill(sourceCache, skillID string) string {
	for _, root := range []string{filepath.Join(sourceCache, ".agents", "skills"), filepath.Join(sourceCache, ".codex", "skills")} {
		candidate := filepath.Join(root, filepath.Base(skillID))
		if info, err := os.Stat(filepath.Join(candidate, "SKILL.md")); err == nil && !info.IsDir() {
			return candidate
		}
	}
	lock := readSkillsLock(filepath.Join(sourceCache, "skills-lock.json"))
	for key := range lock.Skills {
		if key != skillID && slugify(key) != slugify(skillID) {
			continue
		}
		for _, root := range []string{filepath.Join(sourceCache, ".agents", "skills"), filepath.Join(sourceCache, ".codex", "skills")} {
			candidate := filepath.Join(root, key)
			if info, err := os.Stat(filepath.Join(candidate, "SKILL.md")); err == nil && !info.IsDir() {
				return candidate
			}
		}
	}
	return ""
}

func readSkillsLock(path string) skillsLock {
	var lock skillsLock
	raw, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(raw, &lock)
	}
	if lock.Skills == nil {
		lock.Skills = make(map[string]struct {
			Source    string `json:"source"`
			SkillPath string `json:"skillPath"`
		})
	}
	return lock
}

func resolveHeadCommit(ctx context.Context, repositoryURL string) string {
	command := exec.CommandContext(ctx, "git", "ls-remote", repositoryURL, "HEAD")
	output, err := command.Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(output))
	if len(fields) < 1 || !gitCommitPattern.MatchString(fields[0]) {
		return ""
	}
	return strings.ToLower(fields[0])
}

func fixedGitHubSkillURL(source, commit, sourceCache, originalSlug string) string {
	root := sourceURL(source)
	if commit == "" || !isGitHubSource(source) {
		return root
	}
	lock := readSkillsLock(filepath.Join(sourceCache, "skills-lock.json"))
	item, ok := lock.Skills[originalSlug]
	if !ok {
		for key, candidate := range lock.Skills {
			if slugify(key) == slugify(originalSlug) {
				item = candidate
				ok = true
				break
			}
		}
	}
	if !ok || item.SkillPath == "" {
		return root
	}
	directory := filepath.ToSlash(filepath.Dir(item.SkillPath))
	if directory == "." {
		directory = ""
	}
	result := root + "/tree/" + commit
	for _, component := range strings.Split(directory, "/") {
		if component != "" {
			result += "/" + escapeURLSegment(component)
		}
	}
	return result
}

func safeRelativePath(raw string) (string, error) {
	if raw == "" || strings.Contains(raw, "\\") || strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("unsafe downloaded path %q", raw)
	}
	clean := filepath.ToSlash(filepath.Clean(raw))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != raw {
		return "", fmt.Errorf("unsafe downloaded path %q", raw)
	}
	return clean, nil
}

func sourceURL(source string) string {
	trimmed := strings.TrimSpace(source)
	if strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "http://") {
		return strings.TrimRight(trimmed, "/")
	}
	if isGitHubSource(trimmed) {
		return "https://github.com/" + strings.Trim(trimmed, "/")
	}
	return "https://" + strings.Trim(trimmed, "/")
}

func isGitHubSource(source string) bool {
	if strings.HasPrefix(source, "https://github.com/") {
		return true
	}
	parts := strings.Split(strings.Trim(source, "/"), "/")
	return len(parts) == 2 && !strings.Contains(parts[0], ".") && parts[0] != "" && parts[1] != ""
}

func sourceCacheName(source string) string {
	return truncateSlug(slugify(strings.ReplaceAll(source, "/", "-")), 44) + "-" + bytesSHA256([]byte(source))[:12]
}

func itemCacheName(entry snapshotEntry) string {
	return fmt.Sprintf("%04d-%s-%s", entry.Rank, truncateSlug(entry.MarketSlug, 32), bytesSHA256([]byte(entry.Source + "\x00" + entry.OriginalSlug))[:10])
}

func skillKey(entry snapshotEntry) string {
	return fmt.Sprintf("%04d:%s", entry.Rank, bytesSHA256([]byte(entry.Source + "\x00" + entry.OriginalSlug))[:16])
}

func ensureSkillCheckpoint(checkpoint *checkpointFile, entry snapshotEntry) *skillCheckpoint {
	key := skillKey(entry)
	state := checkpoint.Skills[key]
	if state == nil {
		state = &skillCheckpoint{Rank: entry.Rank, Source: entry.Source, OriginalSlug: entry.OriginalSlug, MarketSlug: entry.MarketSlug, Status: "pending"}
		checkpoint.Skills[key] = state
	}
	return state
}

func setAcquiredCheckpoint(outputDir string, state *skillCheckpoint, acquired acquiredSkill) {
	state.Status = "acquired"
	state.Acquisition = acquired.Acquisition
	state.AcquiredPath = relativePath(outputDir, acquired.Path)
	state.DownloadHash = acquired.DownloadHash
	state.LastError = ""
}

func acquiredFromCheckpoint(outputDir string, state *skillCheckpoint) (acquiredSkill, bool) {
	if state == nil || state.AcquiredPath == "" {
		return acquiredSkill{}, false
	}
	path := state.AcquiredPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(outputDir, filepath.FromSlash(path))
	}
	info, err := os.Stat(filepath.Join(path, "SKILL.md"))
	if err != nil || info.IsDir() {
		return acquiredSkill{}, false
	}
	return acquiredSkill{Path: path, Acquisition: state.Acquisition, DownloadHash: state.DownloadHash}, true
}

func relativePath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}
