// Command skill-market-prepare freezes a skills.sh ranking snapshot and turns
// each selected skill into a package accepted by service.ValidateSkillArchive.
// It prepares local artifacts only; it never writes to the marketplace API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type runConfig struct {
	HTMLPath       string
	OutputDir      string
	Limit          int
	MinimumRecords int
	RankingAPI     string
	AcquireMode    string
	NpxPath        string
	DownloadBase   string
	Audit          bool
	AuditBase      string
	HTTPTimeout    time.Duration
	Version        string
}

func main() {
	config := runConfig{}
	flag.StringVar(&config.HTMLPath, "html", "", "saved skills.sh homepage HTML (preferred immutable input; existing snapshot.json takes precedence)")
	flag.StringVar(&config.OutputDir, "out", "skill-market-output", "artifact/checkpoint output directory")
	flag.IntVar(&config.Limit, "limit", 500, "number of top-ranked skills to freeze and prepare")
	flag.IntVar(&config.MinimumRecords, "min-records", 600, "minimum records required in the upstream ranking snapshot")
	flag.StringVar(&config.RankingAPI, "ranking-api", "https://skills.sh/api/skills/all-time", "fallback paginated ranking API used only when -html is empty")
	flag.StringVar(&config.AcquireMode, "acquire", "auto", "acquisition mode: auto, cli, api, or offline")
	flag.StringVar(&config.NpxPath, "npx", "npx", "npx executable for the official skills CLI")
	flag.StringVar(&config.DownloadBase, "download-api", "https://skills.sh/api/download", "skills.sh file snapshot endpoint used by auto/api mode")
	flag.BoolVar(&config.Audit, "audit", true, "cache non-blocking skills.sh provider audits and include them in risk_notes")
	flag.StringVar(&config.AuditBase, "audit-api", "https://skills.sh/api/v1/skills/audit", "skills.sh public audit endpoint")
	flag.DurationVar(&config.HTTPTimeout, "http-timeout", 30*time.Second, "timeout for each upstream HTTP request")
	flag.StringVar(&config.Version, "version", "1.0.0", "marketplace package version recorded in metadata")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, config, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "skill-market-prepare:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, config runConfig, output io.Writer) error {
	if err := validateRunConfig(config); err != nil {
		return err
	}
	absoluteOutput, err := filepath.Abs(config.OutputDir)
	if err != nil {
		return err
	}
	config.OutputDir = absoluteOutput
	for _, directory := range []string{
		config.OutputDir, filepath.Join(config.OutputDir, "packages"), filepath.Join(config.OutputDir, "metadata"),
		filepath.Join(config.OutputDir, "cache", "sources"), filepath.Join(config.OutputDir, "cache", "audits"),
	} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return err
		}
	}
	httpClient := &http.Client{Timeout: config.HTTPTimeout}

	snapshotPath := filepath.Join(config.OutputDir, "snapshot.json")
	snapshot, snapshotSHA, err := loadOrCreateSnapshot(ctx, config, httpClient, snapshotPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "snapshot: %d skills, sha256=%s\n", len(snapshot.Skills), snapshotSHA)

	checkpointPath := filepath.Join(config.OutputDir, "checkpoint.json")
	checkpoint, err := loadOrCreateCheckpoint(checkpointPath, snapshotSHA)
	if err != nil {
		return err
	}
	saveCheckpoint := func() error {
		checkpoint.UpdatedAt = time.Now().UTC()
		return writeJSONAtomic(checkpointPath, checkpoint)
	}
	if err := saveCheckpoint(); err != nil {
		return err
	}

	acquired := acquireAll(ctx, acquireConfig{
		OutputDir: config.OutputDir, Mode: config.AcquireMode, NpxPath: config.NpxPath,
		DownloadBase: config.DownloadBase, HTTPClient: httpClient,
	}, snapshot, checkpoint, saveCheckpoint)
	fmt.Fprintf(output, "acquired/cache-ready: %d/%d\n", len(acquired), len(snapshot.Skills))

	auditAvailable := config.Audit
	for _, entry := range snapshot.Skills {
		if err := ctx.Err(); err != nil {
			_ = saveCheckpoint()
			return err
		}
		state := ensureSkillCheckpoint(checkpoint, entry)
		packagePath := filepath.Join(config.OutputDir, "packages", entry.MarketSlug+".zip")
		metadataPath := filepath.Join(config.OutputDir, "metadata", entry.MarketSlug+".json")
		if state.Status == "completed" && existingCompletedArtifact(packagePath, metadataPath, state.PackageSHA256) {
			continue
		}
		item, ok := acquired[skillKey(entry)]
		if !ok {
			if state.Status != "failed" {
				state.Status = "failed"
				state.LastError = "acquisition produced no usable SKILL.md"
			}
			_ = saveCheckpoint()
			continue
		}
		state.Status = "packaging"
		state.LastAttemptAt = time.Now().UTC()
		state.LastError = ""
		if err := saveCheckpoint(); err != nil {
			return err
		}
		built, buildErr := buildValidatedPackage(item.Path, provenance{
			Source: entry.Source, OriginalSlug: entry.OriginalSlug, OriginalName: entry.OriginalName,
			MarketSlug: entry.MarketSlug, SnapshotSHA256: snapshotSHA, DownloadHash: item.DownloadHash, Rank: entry.Rank,
		})
		if buildErr != nil {
			state.Status = "failed"
			state.LastError = buildErr.Error()
			_ = saveCheckpoint()
			continue
		}
		if err := writeFileAtomic(packagePath, built.Validated.PackageData, 0o644); err != nil {
			state.Status = "failed"
			state.LastError = err.Error()
			_ = saveCheckpoint()
			continue
		}
		audits, auditErr := loadOrFetchAudit(ctx, httpClient, config.OutputDir, config.AuditBase, auditAvailable, entry)
		if auditErr != nil && strings.Contains(auditErr.Error(), "HTTP 429") {
			auditAvailable = false
		}
		metadata := makeMarketMetadata(config, snapshot, snapshotSHA, checkpoint, entry, item, built, audits, auditErr)
		if err := writeJSONAtomic(metadataPath, metadata); err != nil {
			state.Status = "failed"
			state.LastError = err.Error()
			_ = saveCheckpoint()
			continue
		}
		now := time.Now().UTC()
		state.Status = "completed"
		state.PackageSHA256 = built.Validated.SHA256
		state.LastError = ""
		state.CompletedAt = &now
		if err := saveCheckpoint(); err != nil {
			return err
		}
		fmt.Fprintf(output, "[%d/%d] %s -> %s\n", entry.Rank, len(snapshot.Skills), entry.OriginalSlug, entry.MarketSlug)
	}

	manifest, err := rebuildManifest(config.OutputDir, snapshot, snapshotSHA, checkpoint)
	if err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(config.OutputDir, "manifest.json"), manifest); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(config.OutputDir, "failures.json"), manifest.Failures); err != nil {
		return err
	}
	fmt.Fprintf(output, "manifest: completed=%d failed=%d path=%s\n", manifest.Completed, manifest.Failed, filepath.Join(config.OutputDir, "manifest.json"))
	if manifest.Failed > 0 {
		return fmt.Errorf("%d of %d skills failed; rerun the same command to retry failures", manifest.Failed, manifest.Requested)
	}
	return nil
}

func validateRunConfig(config runConfig) error {
	if config.Limit <= 0 {
		return errors.New("-limit must be positive")
	}
	if config.MinimumRecords < config.Limit {
		return errors.New("-min-records must be greater than or equal to -limit")
	}
	if config.HTTPTimeout <= 0 {
		return errors.New("-http-timeout must be positive")
	}
	switch config.AcquireMode {
	case "auto", "cli", "api", "offline":
	default:
		return errors.New("-acquire must be auto, cli, api, or offline")
	}
	if strings.TrimSpace(config.Version) == "" {
		return errors.New("-version is required")
	}
	return nil
}

func loadOrCreateSnapshot(ctx context.Context, config runConfig, client *http.Client, path string) (snapshotFile, string, error) {
	var snapshot snapshotFile
	if raw, err := os.ReadFile(path); err == nil {
		if err := readJSON(path, &snapshot); err != nil {
			return snapshot, "", err
		}
		if err := validateSnapshot(snapshot, config.Limit); err != nil {
			return snapshot, "", err
		}
		return snapshot, bytesSHA256(raw), nil
	} else if !os.IsNotExist(err) {
		return snapshot, "", err
	}

	var records []rankingRecord
	dataSource := "skills.sh-homepage-hydration"
	upstreamHash := ""
	if config.HTMLPath != "" {
		raw, err := os.ReadFile(config.HTMLPath)
		if err != nil {
			return snapshot, "", err
		}
		records, err = parseSkillsHTML(raw, config.MinimumRecords)
		if err != nil {
			return snapshot, "", err
		}
		upstreamHash = bytesSHA256(raw)
	} else {
		if config.RankingAPI == "" {
			return snapshot, "", errors.New("provide -html or -ranking-api when snapshot.json does not exist")
		}
		var err error
		records, upstreamHash, err = fetchRankingAPI(ctx, client, config.RankingAPI, config.MinimumRecords)
		if err != nil {
			return snapshot, "", err
		}
		dataSource = "skills.sh-all-time-api"
	}
	if len(records) < config.Limit {
		return snapshot, "", fmt.Errorf("ranking has %d records, cannot select %d", len(records), config.Limit)
	}
	snapshot = snapshotFile{
		SchemaVersion: snapshotSchemaVersion, DataSource: dataSource, SourceURL: "https://skills.sh/",
		CapturedAt: time.Now().UTC(), UpstreamSHA256: upstreamHash, AvailableCount: len(records), Limit: config.Limit,
		Skills: allocateMarketSlugs(records[:config.Limit]),
	}
	if err := writeJSONAtomic(path, snapshot); err != nil {
		return snapshot, "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return snapshot, "", err
	}
	return snapshot, bytesSHA256(raw), nil
}

func validateSnapshot(snapshot snapshotFile, expectedLimit int) error {
	if snapshot.SchemaVersion != snapshotSchemaVersion {
		return fmt.Errorf("snapshot schema %d is unsupported", snapshot.SchemaVersion)
	}
	if snapshot.Limit != expectedLimit || len(snapshot.Skills) != expectedLimit {
		return fmt.Errorf("existing snapshot freezes limit=%d; rerun with -limit %d or choose a new -out directory", snapshot.Limit, snapshot.Limit)
	}
	seen := make(map[string]struct{}, len(snapshot.Skills))
	for index, entry := range snapshot.Skills {
		if entry.Rank != index+1 || entry.MarketSlug == "" || len(entry.MarketSlug) > 64 {
			return fmt.Errorf("snapshot entry %d is malformed", index+1)
		}
		if _, exists := seen[entry.MarketSlug]; exists {
			return fmt.Errorf("snapshot contains duplicate market slug %q", entry.MarketSlug)
		}
		seen[entry.MarketSlug] = struct{}{}
	}
	return nil
}

func loadOrCreateCheckpoint(path, snapshotSHA string) (*checkpointFile, error) {
	checkpoint := &checkpointFile{}
	if err := readJSON(path, checkpoint); err == nil {
		if checkpoint.SchemaVersion != checkpointSchemaVersion {
			return nil, fmt.Errorf("checkpoint schema %d is unsupported", checkpoint.SchemaVersion)
		}
		if checkpoint.SnapshotSHA256 != snapshotSHA {
			return nil, errors.New("checkpoint belongs to a different snapshot; choose a new -out directory")
		}
		if checkpoint.Sources == nil {
			checkpoint.Sources = make(map[string]*sourceCheckpoint)
		}
		if checkpoint.Skills == nil {
			checkpoint.Skills = make(map[string]*skillCheckpoint)
		}
		return checkpoint, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return &checkpointFile{
		SchemaVersion: checkpointSchemaVersion, SnapshotSHA256: snapshotSHA, UpdatedAt: time.Now().UTC(),
		Sources: make(map[string]*sourceCheckpoint), Skills: make(map[string]*skillCheckpoint),
	}, nil
}

func existingCompletedArtifact(packagePath, metadataPath, expectedSHA string) bool {
	packageData, err := os.ReadFile(packagePath)
	if err != nil || bytesSHA256(packageData) != expectedSHA {
		return false
	}
	var metadata marketMetadata
	return readJSON(metadataPath, &metadata) == nil && metadata.SHA256 == expectedSHA
}

func makeMarketMetadata(config runConfig, snapshot snapshotFile, snapshotSHA string, checkpoint *checkpointFile, entry snapshotEntry, acquired acquiredSkill, built *packageBuildResult, audits []auditRecord, auditErr error) marketMetadata {
	category, tags := classifySkill(entry, built.Description)
	description := built.Description
	if len(built.Excluded) > 0 {
		description += fmt.Sprintf("\n\n为兼容本市场的纯文本包与安全限制，已排除 %d 个不支持的文件或目录；完整上游内容请见来源仓库。", len(built.Excluded))
	}
	riskNotes := localRiskNotes(built.Validated.ValidationReport)
	riskNotes = append(riskNotes, auditRiskNotes(audits)...)
	if auditErr != nil {
		riskNotes = append(riskNotes, "skills.sh audit unavailable | "+auditErr.Error())
	}
	licenseFiles := archiveLicenseFiles(built.Validated.FileManifest)
	warnings := append([]service.SkillValidationIssue(nil), built.Validated.ValidationReport.Warnings...)
	if len(licenseFiles) == 0 {
		warnings = append(warnings, service.SkillValidationIssue{Code: "LICENSE_UNVERIFIED", Message: "no LICENSE, LICENCE, or COPYING file was present in the prepared snapshot"})
	}
	if len(built.Excluded) > 0 {
		warnings = append(warnings, service.SkillValidationIssue{Code: "FILES_EXCLUDED", Message: fmt.Sprintf("%d upstream files or directories were excluded to satisfy marketplace limits", len(built.Excluded))})
	}
	sourceState := checkpoint.Sources[entry.Source]
	commit := ""
	publicSourceURL := ""
	if isGitHubSource(entry.Source) {
		publicSourceURL = sourceURL(entry.Source)
		if acquired.Acquisition == "official-skills-cli" && sourceState != nil {
			commit = sourceState.HeadCommit
			cachePath := sourceState.CachePath
			if !filepath.IsAbs(cachePath) {
				cachePath = filepath.Join(config.OutputDir, filepath.FromSlash(cachePath))
			}
			publicSourceURL = fixedGitHubSkillURL(entry.Source, commit, cachePath, entry.OriginalSlug)
		}
	}
	changelog := fmt.Sprintf("Prepared from the skills.sh All Time snapshot captured %s at rank #%d (snapshot %s; acquisition %s", snapshot.CapturedAt.Format(time.RFC3339), entry.Rank, snapshotSHA[:12], acquired.Acquisition)
	if acquired.DownloadHash != "" {
		changelog += "; skills.sh download hash " + acquired.DownloadHash
	}
	changelog += "). The package was normalized by the marketplace archive validator"
	if len(built.Excluded) > 0 {
		changelog += fmt.Sprintf(" and excludes %d incompatible files/directories", len(built.Excluded))
	}
	changelog += "."
	snapshotHash := acquired.DownloadHash
	if snapshotHash == "" {
		snapshotHash = snapshotSHA
	}
	return marketMetadata{
		Rank: entry.Rank, Source: entry.Source, OriginalSlug: entry.OriginalSlug, MarketSlug: entry.MarketSlug,
		OriginalName: entry.OriginalName, Name: entry.Name, Installs: entry.Installs, SourceURL: publicSourceURL, SourceCommit: commit,
		Category: category, Tags: tags, Summary: summaryFromDescription(built.Description, entry.Name), Description: description,
		RiskNotes: riskNotes, Audits: audits, SortOrder: entry.Rank,
		PackagePath:  filepath.ToSlash(filepath.Join("packages", entry.MarketSlug+".zip")),
		MetadataPath: filepath.ToSlash(filepath.Join("metadata", entry.MarketSlug+".json")),
		Version:      config.Version, Changelog: changelog, ExcludedFiles: built.Excluded,
		SHA256: built.Validated.SHA256, ByteSize: int64(len(built.Validated.PackageData)), UnpackedSize: built.Validated.UnpackedSize,
		FileCount: len(built.Validated.FileManifest), FileManifest: built.Validated.FileManifest,
		ValidationReport: built.Validated.ValidationReport, Acquisition: acquired.Acquisition,
		DownloadHash: acquired.DownloadHash, SnapshotSHA256: snapshotSHA, CapturedAt: snapshot.CapturedAt,
		SnapshotHash: snapshotHash, LicenseUnverified: len(licenseFiles) == 0, LicenseFiles: licenseFiles, Warnings: warnings,
		RightsNotice: "Content remains attributable to its upstream author and is distributed subject to the upstream repository's license; this marketplace preparation does not claim authorship or relicense the content.",
	}
}

func rebuildManifest(outputDir string, snapshot snapshotFile, snapshotSHA string, checkpoint *checkpointFile) (manifestFile, error) {
	manifest := manifestFile{
		SchemaVersion: manifestSchemaVersion, Source: "https://skills.sh/", View: "all-time",
		Snapshot: snapshot.CapturedAt.Format("2006-01-02"), GeneratedAt: time.Now().UTC(), SnapshotSHA256: snapshotSHA,
		Requested: len(snapshot.Skills), Skills: []marketMetadata{}, Failures: []failureRecord{}, SlugMappings: []slugMapping{},
	}
	for _, entry := range snapshot.Skills {
		state := ensureSkillCheckpoint(checkpoint, entry)
		if entry.MarketSlug != entry.OriginalSlug {
			manifest.SlugMappings = append(manifest.SlugMappings, slugMapping{Rank: entry.Rank, Source: entry.Source, OriginalSlug: entry.OriginalSlug, MarketSlug: entry.MarketSlug})
		}
		if state.Status == "completed" {
			var metadata marketMetadata
			path := filepath.Join(outputDir, "metadata", entry.MarketSlug+".json")
			if err := readJSON(path, &metadata); err != nil {
				return manifest, err
			}
			manifest.Skills = append(manifest.Skills, metadata)
			continue
		}
		stage := state.Status
		if stage == "" {
			stage = "pending"
		}
		manifest.Failures = append(manifest.Failures, failureRecord{
			Rank: entry.Rank, Source: entry.Source, OriginalSlug: entry.OriginalSlug,
			MarketSlug: entry.MarketSlug, Stage: stage, Error: state.LastError,
		})
	}
	sort.Slice(manifest.Skills, func(i, j int) bool { return manifest.Skills[i].Rank < manifest.Skills[j].Rank })
	sort.Slice(manifest.Failures, func(i, j int) bool { return manifest.Failures[i].Rank < manifest.Failures[j].Rank })
	manifest.Completed = len(manifest.Skills)
	manifest.Count = manifest.Completed
	manifest.Failed = len(manifest.Failures)
	return manifest, nil
}
