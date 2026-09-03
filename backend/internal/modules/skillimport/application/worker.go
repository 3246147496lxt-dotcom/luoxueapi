package application

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/config"
	domain "github.com/Wei-Shaw/sub2api/internal/service"
	core "github.com/Wei-Shaw/sub2api/internal/skillimport"
)

const maxDiscoveryPages = 1000

const maxRunDiscoveryItems = 5000

// Bootstrap candidates include immutable ZIP bytes (up to five MiB each).
// Keep the page deliberately small so an existing large marketplace cannot
// make a worker materialize hundreds of packages in one database result.
const bootstrapCandidateBatchSize = 10

const (
	maxDiscoveryItemPayloadBytes  = 1024 * 1024
	maxDiscoveryTotalPayloadBytes = 32 * 1024 * 1024
)

var githubNamespacePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

var itemClaimSequence atomic.Uint64

type WorkerRuntime struct {
	service    *Service
	normalizer *core.Normalizer
	cfg        config.SkillImportConfig
	workerID   string

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewWorkerRuntime(service *Service, normalizer *core.Normalizer, cfg *config.Config) *WorkerRuntime {
	settings := config.SkillImportConfig{}
	if cfg != nil {
		settings = cfg.SkillImport
	}
	if normalizer == nil {
		normalizer = core.NewNormalizer()
	}
	return &WorkerRuntime{service: service, normalizer: normalizer, cfg: settings, workerID: newWorkerID()}
}

func (w *WorkerRuntime) Start(ctx context.Context) error {
	if w == nil || w.service == nil || !w.service.Enabled() || !w.cfg.WorkerEnabled {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}
	runtimeCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.wg.Add(1)
	go w.scheduleLoop(runtimeCtx)
	w.wg.Add(1)
	go w.eventCleanupLoop(runtimeCtx)
	// A runtime worker ID is the durable lease owner. Holding multiple runs under
	// the same owner would let sibling goroutines heartbeat or finish each
	// other's leases. Keep one run lease per process; WorkerConcurrency still
	// controls the parallel item workers inside that run.
	w.wg.Add(1)
	go w.runLoop(runtimeCtx)
	return nil
}

func (w *WorkerRuntime) Stop(ctx context.Context) error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	cancel := w.cancel
	w.cancel = nil
	w.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *WorkerRuntime) scheduleLoop(ctx context.Context) {
	defer w.wg.Done()
	w.poll(ctx, w.claimSchedule)
}

func (w *WorkerRuntime) runLoop(ctx context.Context) {
	defer w.wg.Done()
	w.poll(ctx, w.claimRun)
}

func (w *WorkerRuntime) eventCleanupLoop(ctx context.Context) {
	defer w.wg.Done()
	w.cleanupExpiredEvents(ctx)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.cleanupExpiredEvents(ctx)
		}
	}
}

func (w *WorkerRuntime) cleanupExpiredEvents(ctx context.Context) {
	days := w.cfg.EventsRetentionDays
	if days <= 0 {
		return
	}
	removed, err := w.service.repo.DeleteTerminalRunEventsBefore(ctx, time.Now().UTC().Add(-time.Duration(days)*24*time.Hour))
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("skill import event cleanup failed", "err", safeImportError(err))
		}
		return
	}
	if removed > 0 {
		slog.Info("skill import expired events removed", "count", removed)
	}
}

func (w *WorkerRuntime) poll(ctx context.Context, work func(context.Context) bool) {
	interval := time.Duration(w.cfg.PollIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 2 * time.Second
	}
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		worked := work(ctx)
		if worked {
			timer.Reset(10 * time.Millisecond)
		} else {
			timer.Reset(interval)
		}
	}
}

func (w *WorkerRuntime) claimSchedule(ctx context.Context) bool {
	now := time.Now().UTC()
	leaseUntil := now.Add(w.leaseTTL())
	schedule, err := w.service.repo.ClaimDueSchedule(ctx, w.workerID, now, leaseUntil)
	if err != nil {
		slog.Warn("skill import schedule claim failed", "err", safeImportError(err))
		return false
	}
	if schedule == nil {
		return false
	}
	scheduledFor := now
	if schedule.NextRunAt != nil {
		scheduledFor = schedule.NextRunAt.UTC()
	}
	next, nextErr := nextScheduleTime(schedule.CronExpression, schedule.Timezone, scheduledFor)
	if nextErr != nil {
		_ = w.service.repo.ReleaseScheduleClaim(ctx, schedule.ID, w.workerID, now.Add(time.Hour))
		return true
	}
	run, runErr := w.service.CreateScheduledRun(ctx, schedule, scheduledFor)
	if runErr != nil {
		_ = w.service.repo.ReleaseScheduleClaim(ctx, schedule.ID, w.workerID, *next)
		slog.Warn("skill import scheduled run creation failed", "schedule_id", schedule.ID, "err", safeImportError(runErr))
		return true
	}
	if err := w.service.repo.CompleteScheduleClaim(ctx, schedule.ID, w.workerID, &run.ID, now, *next); err != nil {
		slog.Warn("skill import schedule completion failed", "schedule_id", schedule.ID, "run_id", run.ID, "err", safeImportError(err))
	}
	return true
}

func (w *WorkerRuntime) claimRun(ctx context.Context) bool {
	now := time.Now().UTC()
	run, err := w.service.repo.ClaimNextRun(ctx, w.workerID, now, now.Add(w.leaseTTL()))
	if err != nil {
		slog.Warn("skill import run claim failed", "err", safeImportError(err))
		return false
	}
	if run == nil {
		return false
	}
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	go w.runHeartbeat(heartbeatCtx, run.ID)
	if err := w.processRun(ctx, run); err != nil && !errors.Is(err, context.Canceled) && ctx.Err() == nil {
		w.handleRunError(ctx, run, runProcessingError(err, w.leaseTTL()))
	}
	return true
}

func (w *WorkerRuntime) processRun(ctx context.Context, run *domain.SkillImportRun) error {
	ctx, cancel := w.withRunDeadline(ctx, run)
	defer cancel()
	if run.TriggerType == domain.SkillImportTriggerBootstrap {
		return w.bootstrapExistingSkills(ctx, run)
	}
	var frozen FrozenRunConfig
	if err := json.Unmarshal(run.RequestConfig, &frozen); err != nil {
		return fmt.Errorf("decode frozen run configuration: %w", err)
	}
	if err := validateSkillImportRunConfig(frozen.RunConfig); err != nil {
		return fmt.Errorf("validate frozen run policy: %w", err)
	}
	adapter, ok := w.service.registry.Get(frozen.AdapterType)
	if !ok {
		return ErrAdapterNotFound
	}
	if adapter.Version() != frozen.AdapterVersion || frozen.NormalizerVersion != core.CoreVersion || frozen.ValidatorVersion != validatorRulesetVersion {
		return fmt.Errorf("frozen importer rules are unavailable for safe resume")
	}
	if run.Status == domain.SkillImportRunStatusDiscovering || run.SnapshotSHA256 == "" {
		if err := w.discoverRun(ctx, run, frozen, adapter); err != nil {
			return err
		}
		run.Status = domain.SkillImportRunStatusPreparing
	}
	if err := w.processAvailableItems(ctx, run, frozen, adapter); err != nil {
		return err
	}
	current, err := w.service.repo.GetRun(ctx, run.ID)
	if err != nil {
		return repositoryTemporary("get run", err)
	}
	if current.CancelRequestedAt != nil {
		return repositoryTemporary("cancel run", w.service.repo.UpdateRunStatus(ctx, run.ID, w.workerID, domain.SkillImportRunStatusCancelled, "CANCELLED", "Cancelled by an administrator", nil))
	}
	return w.finalizeRun(ctx, run, frozen)
}

func (w *WorkerRuntime) processAvailableItems(
	ctx context.Context,
	run *domain.SkillImportRun,
	frozen FrozenRunConfig,
	adapter core.SourceAdapter,
) error {
	concurrency := itemConcurrency(frozen.RunConfig, w.cfg.WorkerConcurrency)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error
	recordError := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
	}
	worker := func() {
		defer wg.Done()
		for ctx.Err() == nil {
			now := time.Now().UTC()
			itemLeaseOwner := newItemLeaseOwner(w.workerID)
			item, err := w.service.repo.ClaimNextRunItem(
				ctx, run.ID, w.workerID, itemLeaseOwner, now, now.Add(w.leaseTTL()),
			)
			if err != nil {
				recordError(repositoryTemporary("claim run item", err))
				return
			}
			if item == nil {
				return
			}
			if item.LeaseOwner == nil || *item.LeaseOwner != itemLeaseOwner {
				recordError(core.NewAdapterError("repository", "claim run item", core.ErrorIntegrity, errors.New("item claim returned a mismatched fencing token")))
				return
			}
			if err := w.processItemWithHeartbeat(ctx, run, frozen, adapter, item, itemLeaseOwner); err != nil {
				recordError(err)
				return
			}
		}
	}
	for index := 0; index < concurrency; index++ {
		wg.Add(1)
		go worker()
	}
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

func (w *WorkerRuntime) withRunDeadline(ctx context.Context, _ *domain.SkillImportRun) (context.Context, context.CancelFunc) {
	hours := w.cfg.MaxRunDurationHours
	if hours <= 0 {
		hours = 24
	}
	// This is an execution-attempt budget, not a wall clock tied to the first
	// attempt. Reusing StartedAt would make every resumed lease immediately
	// expire once an earlier worker reached the deadline.
	return context.WithTimeout(ctx, time.Duration(hours)*time.Hour)
}

func runProcessingError(err error, leaseTTL time.Duration) error {
	if !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return &core.AdapterError{
		Adapter: "worker", Operation: "process run", Kind: core.ErrorTemporary,
		RetryAfter: leaseTTL, Err: err,
	}
}

func (w *WorkerRuntime) bootstrapExistingSkills(ctx context.Context, run *domain.SkillImportRun) error {
	cursor := domain.SkillImportBootstrapCursor{Limit: bootstrapCandidateBatchSize}
	bound, skipped, failed := 0, 0, 0
	for {
		candidates, err := w.service.repo.ListBootstrapCandidates(ctx, cursor)
		if err != nil {
			return repositoryTemporary("list bootstrap candidates", err)
		}
		if len(candidates) == 0 {
			break
		}
		for _, candidate := range candidates {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			evidence, extractErr := ExtractSkillsSHBootstrapEvidence(run.SourceID, candidate)
			if extractErr != nil {
				skipped++
				_ = w.appendEvent(ctx, run.ID, nil, domain.SkillImportEventLevelWarn,
					"bootstrap_skipped", "Existing Skill had no unambiguous import evidence",
					map[string]any{"skill_id": candidate.SkillID, "slug": candidate.Slug})
				cursor.AfterSkillID = candidate.SkillID
				continue
			}
			_, bindErr := w.service.repo.BootstrapOrigin(ctx, domain.SkillImportBootstrapOriginInput{
				RunID: run.ID, RunWorkerID: w.workerID,
				StableKey: evidence.Key, SkillID: candidate.SkillID,
				VersionID: candidate.VersionID, MarketSlug: candidate.Slug,
				OriginURL: evidence.OriginURL, SourceRevision: evidence.SourceRevision,
				SourceContentSHA256: evidence.SourceContentSHA256, Provenance: evidence.Provenance,
			})
			if errors.Is(bindErr, domain.ErrSkillImportInvalidState) {
				current, currentErr := w.service.repo.GetRun(ctx, run.ID)
				if currentErr != nil {
					return repositoryTemporary("check bootstrap cancellation", currentErr)
				}
				if current.CancelRequestedAt != nil {
					return repositoryTemporary("cancel bootstrap run", w.service.repo.UpdateRunStatus(
						ctx, run.ID, w.workerID, domain.SkillImportRunStatusCancelled,
						"CANCELLED", "Cancelled by an administrator", nil,
					))
				}
			}
			if bindErr != nil && !errors.Is(bindErr, domain.ErrSkillImportConflict) && !errors.Is(bindErr, domain.ErrSkillImportInvalidState) {
				return repositoryTemporary("bootstrap origin", bindErr)
			}
			if bindErr != nil {
				failed++
				_ = w.appendEvent(ctx, run.ID, nil, domain.SkillImportEventLevelError,
					"bootstrap_conflict", "Existing Skill identity could not be bound safely",
					map[string]any{"skill_id": candidate.SkillID, "slug": candidate.Slug})
			} else {
				bound++
			}
			cursor.AfterSkillID = candidate.SkillID
		}
		if len(candidates) < cursor.Limit {
			break
		}
	}
	counts, err := w.service.repo.RefreshRunCounts(ctx, run.ID)
	if err != nil {
		return repositoryTemporary("refresh bootstrap counts", err)
	}
	status := domain.SkillImportRunStatusSucceeded
	if failed > 0 {
		if bound > 0 {
			status = domain.SkillImportRunStatusPartialSucceeded
		} else {
			status = domain.SkillImportRunStatusFailed
		}
	}
	_ = w.appendEvent(ctx, run.ID, nil, domain.SkillImportEventLevelInfo,
		"bootstrap_completed", "Existing Skill origin bootstrap completed",
		map[string]any{"bound": bound, "skipped": skipped, "failed": failed, "counts": counts})
	return repositoryTemporary("complete bootstrap run", w.service.repo.UpdateRunStatus(ctx, run.ID, w.workerID, status, "", "", nil))
}

type frozenSnapshot struct {
	SchemaVersion     int             `json:"schema_version"`
	AdapterType       string          `json:"adapter_type"`
	AdapterVersion    string          `json:"adapter_version"`
	NormalizerVersion string          `json:"normalizer_version"`
	ValidatorVersion  string          `json:"validator_version"`
	CapturedAt        time.Time       `json:"captured_at"`
	Evidence          []core.Evidence `json:"evidence"`
	ItemCount         int             `json:"item_count"`
}

func (w *WorkerRuntime) discoverRun(ctx context.Context, run *domain.SkillImportRun, frozen FrozenRunConfig, adapter core.SourceAdapter) error {
	cursor := ""
	items := make([]domain.SkillImportRunItem, 0)
	evidence := make([]core.Evidence, 0)
	seen := make(map[string]struct{})
	seenCursors := make(map[string]struct{})
	sourcePayloadBytes := 0
	for pageNumber := 0; pageNumber < maxDiscoveryPages; pageNumber++ {
		page, err := adapter.Discover(ctx, core.DiscoverRequest{
			Config: frozen.AdapterConfig, Selection: frozen.Selection,
			BaseURL: frozen.BaseURL, Cursor: cursor,
		})
		if err != nil {
			return err
		}
		evidence = append(evidence, page.Evidence...)
		for _, discovered := range page.Items {
			if discovered.AdapterType == "" {
				discovered.AdapterType = frozen.AdapterType
			}
			if err := validateDiscoveredSkill(discovered, frozen.AdapterType); err != nil {
				return core.NewAdapterError(frozen.AdapterType, "discover", core.ErrorIntegrity, err)
			}
			key := fmt.Sprintf("%d\x00%s\x00%s", run.SourceID, discovered.Namespace, discovered.ExternalID)
			if _, duplicate := seen[key]; duplicate {
				continue
			}
			if len(items) >= maxRunDiscoveryItems {
				return core.NewAdapterError(frozen.AdapterType, "discover", core.ErrorBlocked,
					fmt.Errorf("discovery exceeds %d unique items", maxRunDiscoveryItems))
			}
			seen[key] = struct{}{}
			payload, err := json.Marshal(discovered)
			if err != nil {
				return core.NewAdapterError(frozen.AdapterType, "discover", core.ErrorIntegrity, fmt.Errorf("encode discovered item: %w", err))
			}
			if bytes.Contains(payload, []byte(`\u0000`)) {
				return core.NewAdapterError(frozen.AdapterType, "discover", core.ErrorIntegrity, errors.New("adapter payload contains a PostgreSQL-incompatible NUL character"))
			}
			if len(payload) > maxDiscoveryItemPayloadBytes || sourcePayloadBytes+len(payload) > maxDiscoveryTotalPayloadBytes {
				return core.NewAdapterError(frozen.AdapterType, "discover", core.ErrorBlocked, errors.New("discovery payload exceeds the durable resource limit"))
			}
			sourcePayloadBytes += len(payload)
			items = append(items, domain.SkillImportRunItem{
				RunID:     run.ID,
				StableKey: domain.SkillImportStableKey{SourceID: run.SourceID, Namespace: discovered.Namespace, ExternalID: discovered.ExternalID},
				Rank:      discovered.Rank, MarketSlug: suggestedMarketSlug(discovered),
				Status: domain.SkillImportItemStatusQueued, UpstreamName: truncateRunes(discovered.SuggestedName, 240),
				OriginURL: discovered.CanonicalURL, SourceRevision: discovered.Revision,
				SourcePayload: payload, DesiredSkill: domain.SkillImportDesiredSkill{Tags: []string{}, ExamplePrompts: []string{}},
				ValidationReport: domain.SkillValidationReport{Valid: false, Errors: []domain.SkillValidationIssue{}, Warnings: []domain.SkillValidationIssue{}},
			})
		}
		if page.NextCursor == "" {
			break
		}
		if page.NextCursor == cursor {
			return core.NewAdapterError(frozen.AdapterType, "discover", core.ErrorIntegrity, errors.New("adapter returned a non-advancing discovery cursor"))
		}
		if _, repeated := seenCursors[page.NextCursor]; repeated {
			return core.NewAdapterError(frozen.AdapterType, "discover", core.ErrorIntegrity, errors.New("adapter returned a repeated discovery cursor"))
		}
		seenCursors[page.NextCursor] = struct{}{}
		cursor = page.NextCursor
		if pageNumber == maxDiscoveryPages-1 {
			return core.NewAdapterError(frozen.AdapterType, "discover", core.ErrorBlocked, fmt.Errorf("discovery exceeded %d pages", maxDiscoveryPages))
		}
	}
	snapshot := frozenSnapshot{
		SchemaVersion: 1, AdapterType: frozen.AdapterType, AdapterVersion: frozen.AdapterVersion,
		NormalizerVersion: frozen.NormalizerVersion, ValidatorVersion: frozen.ValidatorVersion,
		CapturedAt: time.Now().UTC(), Evidence: evidence, ItemCount: len(items),
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return core.NewAdapterError(frozen.AdapterType, "snapshot discovery", core.ErrorIntegrity, err)
	}
	digest := sha256.Sum256(snapshotJSON)
	if err := w.service.repo.CompleteDiscovery(ctx, run.ID, w.workerID, snapshotJSON, hex.EncodeToString(digest[:]), items); err != nil {
		return repositoryTemporary("complete discovery", err)
	}
	return w.appendEvent(ctx, run.ID, nil, domain.SkillImportEventLevelInfo, "discovery_completed", "Source catalog snapshot captured", map[string]any{"items": len(items)})
}

func validateDiscoveredSkill(skill core.DiscoveredSkill, adapterType string) error {
	if skill.AdapterType != adapterType {
		return errors.New("adapter returned a mismatched adapter type")
	}
	if !sourceNamespacePattern.MatchString(skill.Namespace) {
		return errors.New("adapter returned an invalid or oversized namespace")
	}
	if skill.ExternalID != strings.TrimSpace(skill.ExternalID) || skill.ExternalID == "" || utf8.RuneCountInString(skill.ExternalID) > 512 {
		return errors.New("adapter returned an invalid or oversized external ID")
	}
	for _, current := range skill.ExternalID {
		if unicode.IsControl(current) {
			return errors.New("adapter returned an external ID containing control characters")
		}
	}
	if utf8.RuneCountInString(skill.Revision) > 240 {
		return errors.New("adapter returned an oversized source revision")
	}
	if utf8.RuneCountInString(skill.SuggestedName) > 20000 || utf8.RuneCountInString(skill.SuggestedSlug) > 20000 || utf8.RuneCountInString(skill.Description) > 20000 {
		return errors.New("adapter returned oversized display metadata")
	}
	if len(skill.Metrics) > 128 {
		return errors.New("adapter returned too many metric fields")
	}
	for key := range skill.Metrics {
		if strings.TrimSpace(key) == "" || len(key) > 128 {
			return errors.New("adapter returned an invalid metric key")
		}
	}
	if skill.Rank != nil && *skill.Rank <= 0 {
		return errors.New("adapter returned a non-positive rank")
	}
	return validatePublicOriginURL(skill.CanonicalURL)
}

func validatePublicOriginURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if len(raw) > 2048 {
		return errors.New("adapter returned an oversized canonical URL")
	}
	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("adapter returned a canonical URL that is not safe public HTTPS metadata")
	}
	return nil
}

func (w *WorkerRuntime) processItemWithHeartbeat(ctx context.Context, run *domain.SkillImportRun, frozen FrozenRunConfig, adapter core.SourceAdapter, item *domain.SkillImportRunItem, itemLeaseOwner string) error {
	heartbeatCtx, stop := context.WithCancel(ctx)
	defer stop()
	go w.itemHeartbeat(heartbeatCtx, item.ID, itemLeaseOwner)
	return w.processItem(ctx, run, frozen, adapter, item, itemLeaseOwner)
}

func (w *WorkerRuntime) processItem(ctx context.Context, run *domain.SkillImportRun, frozen FrozenRunConfig, adapter core.SourceAdapter, item *domain.SkillImportRunItem, itemLeaseOwner string) error {
	var discovered core.DiscoveredSkill
	if err := json.Unmarshal(item.SourcePayload, &discovered); err != nil {
		return w.completeItemFailure(ctx, item, itemLeaseOwner, core.ErrorInvalidSource, "DISCOVERY_PAYLOAD_INVALID", err)
	}
	bundle, err := adapter.Acquire(ctx, core.AcquireRequest{Config: frozen.AdapterConfig, BaseURL: frozen.BaseURL, Skill: discovered})
	if err != nil {
		return w.completeAdapterFailure(ctx, item, itemLeaseOwner, err)
	}
	bundle.Revision = strings.TrimSpace(bundle.Revision)
	bundle.CanonicalURL = strings.TrimSpace(bundle.CanonicalURL)
	if utf8.RuneCountInString(bundle.Revision) > 240 {
		return w.completeItemFailure(ctx, item, itemLeaseOwner, core.ErrorInvalidSource, "SOURCE_REVISION_INVALID", errors.New("acquired source revision exceeds 240 characters"))
	}
	if err := validatePublicOriginURL(bundle.CanonicalURL); err != nil {
		return w.completeItemFailure(ctx, item, itemLeaseOwner, core.ErrorInvalidSource, "ORIGIN_URL_INVALID", err)
	}
	marketSlug := item.MarketSlug
	if origin, originErr := w.service.repo.GetOriginByStableKey(ctx, item.StableKey); originErr == nil && origin != nil {
		marketSlug = origin.MarketSlug
	} else if originErr != nil && !errors.Is(originErr, domain.ErrSkillImportOriginNotFound) {
		return repositoryTemporary("get stable origin", originErr)
	}
	normalized, err := w.normalizer.Normalize(core.NormalizeInput{Skill: discovered, Bundle: bundle, MarketSlug: marketSlug})
	if err != nil {
		return w.completeAdapterFailure(ctx, item, itemLeaseOwner, err)
	}
	desired := desiredSkillFromNormalized(discovered, bundle, normalized, frozen)
	provenance := normalizedProvenance(discovered, bundle, normalized, frozen)
	excludedPaths := make([]string, 0, len(normalized.ExcludedFiles))
	for _, excluded := range normalized.ExcludedFiles {
		excludedPaths = append(excludedPaths, excluded.Path)
	}
	validation := domain.SkillValidationReport{Valid: false, Errors: []domain.SkillValidationIssue{}, Warnings: normalized.Warnings}
	if normalized.ValidationReport != nil {
		validation = *normalized.ValidationReport
	}
	if normalized.Blocked {
		return w.completeRunItem(ctx, item.ID, itemLeaseOwner, domain.SkillImportRunItemPatch{
			Status: domain.SkillImportItemStatusBlocked, MarketSlug: marketSlug,
			SourceRevision: bundle.Revision, SourceContentSHA256: normalized.UpstreamContentHash,
			PackageSHA256: normalized.NormalizedSHA256, DesiredSkill: desired,
			ValidationReport: validation, Provenance: provenance,
			LicenseUnverified: normalized.LicenseUnverified, ExcludedFiles: excludedPaths,
			Warnings: normalized.Warnings, ErrorCode: "NORMALIZATION_BLOCKED",
			ErrorMessage: safeImportMessage(strings.Join(normalized.BlockedReasons, "; ")),
		})
	}
	gate := autoPublishGate(frozen.RunConfig)
	if run.Mode == domain.SkillImportModeAutoPublish && normalized.LicenseUnverified && !gate.AllowLicenseUnverified {
		return w.completeRunItem(ctx, item.ID, itemLeaseOwner, domain.SkillImportRunItemPatch{
			Status: domain.SkillImportItemStatusBlocked, MarketSlug: marketSlug,
			SourceRevision: bundle.Revision, SourceContentSHA256: normalized.UpstreamContentHash,
			PackageSHA256: normalized.NormalizedSHA256, DesiredSkill: desired,
			ValidationReport: validation, Provenance: provenance,
			LicenseUnverified: true, ExcludedFiles: excludedPaths,
			Warnings: normalized.Warnings, ErrorCode: "LICENSE_REVIEW_REQUIRED",
			ErrorMessage: "No package license evidence was present and this run disallows automatic publication",
		})
	}
	if shouldBlockNeedsReview(run.Mode, frozen.RunConfig, normalized.NeedsReview) {
		return w.completeRunItem(ctx, item.ID, itemLeaseOwner, domain.SkillImportRunItemPatch{
			Status: domain.SkillImportItemStatusBlocked, MarketSlug: marketSlug,
			SourceRevision: bundle.Revision, SourceContentSHA256: normalized.UpstreamContentHash,
			PackageSHA256: normalized.NormalizedSHA256, DesiredSkill: desired,
			ValidationReport: validation, Provenance: provenance,
			LicenseUnverified: normalized.LicenseUnverified, ExcludedFiles: excludedPaths,
			Warnings: normalized.Warnings, ErrorCode: "REVIEW_REQUIRED",
			ErrorMessage: safeImportMessage(strings.Join(normalized.ReviewReasons, "; ")),
		})
	}
	if run.Mode == domain.SkillImportModeDryRun {
		return w.completeRunItem(ctx, item.ID, itemLeaseOwner, domain.SkillImportRunItemPatch{
			Status: domain.SkillImportItemStatusSkipped, MarketSlug: marketSlug,
			SourceRevision: bundle.Revision, SourceContentSHA256: normalized.UpstreamContentHash,
			PackageSHA256: normalized.NormalizedSHA256, DesiredSkill: desired,
			ValidationReport: validation, Provenance: provenance,
			LicenseUnverified: normalized.LicenseUnverified, ExcludedFiles: excludedPaths,
			Warnings: normalized.Warnings, ErrorCode: "DRY_RUN", ErrorMessage: "Validated without staging",
		})
	}
	if normalized.ValidationReport == nil || !normalized.ValidationReport.Valid || len(normalized.PackageData) == 0 {
		return w.completeItemFailure(ctx, item, itemLeaseOwner, core.ErrorIntegrity, "NORMALIZED_PACKAGE_INVALID", errors.New("normalizer produced no publishable package"))
	}
	for attempt := 0; attempt < 3; attempt++ {
		result, stageErr := w.service.repo.StagePreparedItem(ctx, preparedStageInput(
			run, item, frozen, bundle, normalized, desired, provenance, excludedPaths, w.workerID, itemLeaseOwner,
		))
		if stageErr != nil {
			if errors.Is(stageErr, domain.ErrSkillImportVersionYanked) {
				return w.completeRunItem(ctx, item.ID, itemLeaseOwner, domain.SkillImportRunItemPatch{
					Status: domain.SkillImportItemStatusBlocked, MarketSlug: normalized.MarketSlug,
					SourceRevision: bundle.Revision, SourceContentSHA256: normalized.UpstreamContentHash,
					PackageSHA256: normalized.NormalizedSHA256, DesiredSkill: desired,
					ValidationReport: *normalized.ValidationReport, Provenance: provenance,
					LicenseUnverified: normalized.LicenseUnverified, ExcludedFiles: excludedPaths,
					Warnings: normalized.Warnings, ErrorCode: "VERSION_YANKED",
					ErrorMessage: "An administrator yanked the matching package version; provide changed upstream content or review that decision manually",
				})
			}
			if errors.Is(stageErr, domain.ErrSkillImportConflict) ||
				errors.Is(stageErr, domain.ErrSkillImportInvalidState) ||
				errors.Is(stageErr, domain.ErrSkillImportItemNotFound) ||
				errors.Is(stageErr, domain.ErrSkillImportPublishInvalid) {
				return w.completeItemFailure(ctx, item, itemLeaseOwner, core.ErrorInvalidSource, "STAGING_CONFLICT", stageErr)
			}
			return w.completeAdapterFailure(ctx, item, itemLeaseOwner, core.NewAdapterError("repository", "stage item", core.ErrorTemporary, stageErr))
		}
		if result.Action != domain.SkillImportStageActionRenormalize {
			return nil
		}
		if result.MarketSlug == "" || result.MarketSlug == normalized.MarketSlug {
			return w.completeItemFailure(ctx, item, itemLeaseOwner, core.ErrorInvalidSource, "SLUG_ALLOCATION_CONFLICT", domain.ErrSkillImportConflict)
		}
		normalized, err = w.normalizer.Normalize(core.NormalizeInput{
			Skill: discovered, Bundle: bundle, MarketSlug: result.MarketSlug,
		})
		if err != nil {
			return err
		}
		if normalized.Blocked || normalized.ValidationReport == nil || !normalized.ValidationReport.Valid || len(normalized.PackageData) == 0 {
			return w.completeItemFailure(ctx, item, itemLeaseOwner, core.ErrorIntegrity, "RENORMALIZATION_FAILED", errors.New("resolved market slug could not be packaged safely"))
		}
		desired = desiredSkillFromNormalized(discovered, bundle, normalized, frozen)
		provenance = normalizedProvenance(discovered, bundle, normalized, frozen)
		excludedPaths = excludedPaths[:0]
		for _, excluded := range normalized.ExcludedFiles {
			excludedPaths = append(excludedPaths, excluded.Path)
		}
		if shouldBlockNeedsReview(run.Mode, frozen.RunConfig, normalized.NeedsReview) {
			return w.completeRunItem(ctx, item.ID, itemLeaseOwner, domain.SkillImportRunItemPatch{
				Status: domain.SkillImportItemStatusBlocked, MarketSlug: normalized.MarketSlug,
				SourceRevision: bundle.Revision, SourceContentSHA256: normalized.UpstreamContentHash,
				PackageSHA256: normalized.NormalizedSHA256, DesiredSkill: desired,
				ValidationReport: *normalized.ValidationReport, Provenance: provenance,
				LicenseUnverified: normalized.LicenseUnverified, ExcludedFiles: excludedPaths,
				Warnings: normalized.Warnings, ErrorCode: "REVIEW_REQUIRED",
				ErrorMessage: safeImportMessage(strings.Join(normalized.ReviewReasons, "; ")),
			})
		}
	}
	return w.completeItemFailure(ctx, item, itemLeaseOwner, core.ErrorInvalidSource, "SLUG_ALLOCATION_CONFLICT",
		domain.ErrSkillImportConflict.WithMetadata(map[string]string{"slug": "could not allocate a stable market slug"}))
}

func preparedStageInput(
	run *domain.SkillImportRun,
	item *domain.SkillImportRunItem,
	frozen FrozenRunConfig,
	bundle core.SourceBundle,
	normalized core.NormalizedSkill,
	desired domain.SkillImportDesiredSkill,
	provenance json.RawMessage,
	excludedPaths []string,
	runWorkerID string,
	itemLeaseOwner string,
) domain.SkillImportStagePreparedInput {
	return domain.SkillImportStagePreparedInput{
		RunID: run.ID, RunItemID: item.ID, RunWorkerID: runWorkerID, ItemLeaseOwner: itemLeaseOwner,
		StableKey: item.StableKey, Rank: item.Rank, DesiredSkill: desired,
		OriginURL: desired.OriginURL, SourceRevision: bundle.Revision,
		SourceContentSHA256: normalized.UpstreamContentHash,
		Artifact: domain.SkillImportPreparedArtifact{
			Changelog:    importChangelog(frozen.AdapterType, bundle.Revision),
			ManifestName: normalized.MarketSlug, ManifestDescription: normalized.Description,
			SkillMD:     normalizedSkillBody(normalized.PackageData, normalized.MarketSlug),
			PackageData: normalized.PackageData, PackageSHA256: normalized.NormalizedSHA256,
			ByteSize: int64(len(normalized.PackageData)), UnpackedSize: normalized.UnpackedSize,
			FileCount: normalized.FileCount, FileManifest: normalized.FileManifest,
			ValidationReport: *normalized.ValidationReport,
		},
		Provenance: provenance, Transformed: normalized.Transformed,
		LicenseUnverified: normalized.LicenseUnverified, ExcludedFiles: excludedPaths,
		Warnings: normalized.Warnings,
	}
}

func (w *WorkerRuntime) completeAdapterFailure(ctx context.Context, item *domain.SkillImportRunItem, itemLeaseOwner string, err error) error {
	kind := core.ErrorKindOf(err)
	if (kind == core.ErrorRateLimited || kind == core.ErrorTemporary) && item.AttemptCount < w.maxAttempts() {
		delay := retryDelay(item.AttemptCount)
		if requested, ok := core.RetryAfterOf(err); ok && requested > delay {
			delay = min(requested, time.Hour)
		}
		next := time.Now().UTC().Add(delay)
		if completeErr := w.service.repo.CompleteRunItem(ctx, item.ID, itemLeaseOwner, w.workerID, domain.SkillImportRunItemPatch{
			Status: domain.SkillImportItemStatusQueued, MarketSlug: item.MarketSlug,
			DesiredSkill: item.DesiredSkill, ValidationReport: item.ValidationReport,
			Provenance: item.Provenance, ExcludedFiles: item.ExcludedFiles, Warnings: item.Warnings,
			NextAttemptAt: &next, ErrorCode: strings.ToUpper(string(kind)), ErrorMessage: safeImportError(err),
		}); completeErr != nil {
			return core.NewAdapterError("repository", "queue item retry", core.ErrorTemporary, completeErr)
		}
		return nil
	}
	return w.completeItemFailure(ctx, item, itemLeaseOwner, kind, strings.ToUpper(string(kind)), err)
}

func (w *WorkerRuntime) completeRunItem(ctx context.Context, itemID int64, itemLeaseOwner string, patch domain.SkillImportRunItemPatch) error {
	return repositoryTemporary("complete run item", w.service.repo.CompleteRunItem(ctx, itemID, itemLeaseOwner, w.workerID, patch))
}

func (w *WorkerRuntime) completeItemFailure(ctx context.Context, item *domain.SkillImportRunItem, itemLeaseOwner string, kind core.ErrorKind, code string, err error) error {
	status := domain.SkillImportItemStatusFailed
	if kind == core.ErrorUnsafe || kind == core.ErrorIntegrity || kind == core.ErrorBlocked {
		status = domain.SkillImportItemStatusBlocked
	}
	if code == "" {
		code = "IMPORT_FAILED"
	}
	if completeErr := w.service.repo.CompleteRunItem(ctx, item.ID, itemLeaseOwner, w.workerID, domain.SkillImportRunItemPatch{
		Status: status, MarketSlug: item.MarketSlug, DesiredSkill: item.DesiredSkill,
		ValidationReport: item.ValidationReport, Provenance: item.Provenance,
		ExcludedFiles: item.ExcludedFiles, Warnings: item.Warnings,
		ErrorCode: code, ErrorMessage: safeImportError(err),
	}); completeErr != nil {
		return core.NewAdapterError("repository", "complete item failure", core.ErrorTemporary, completeErr)
	}
	return nil
}

func (w *WorkerRuntime) finalizeRun(ctx context.Context, run *domain.SkillImportRun, frozen FrozenRunConfig) error {
	counts, err := w.service.repo.RefreshRunCounts(ctx, run.ID)
	if err != nil {
		return repositoryTemporary("refresh run counts", err)
	}
	terminal := counts.Prepared + counts.Skipped + counts.Blocked + counts.Failed
	if terminal < counts.Discovered {
		next := w.nextItemRetryAt(ctx, run.ID)
		return repositoryTemporary("wait for item retry", w.service.repo.UpdateRunStatus(ctx, run.ID, w.workerID, domain.SkillImportRunStatusWaitingRetry, "ITEMS_WAITING_RETRY", "One or more items are waiting for a retry window", &next))
	}
	if run.Mode == domain.SkillImportModeDryRun {
		return w.completeWithoutPublication(ctx, run.ID, counts)
	}
	eligible, err := w.service.repo.ListEligibleItemIDs(ctx, run.ID)
	if err != nil {
		return repositoryTemporary("list eligible items", err)
	}
	if run.Mode == domain.SkillImportModeReview && len(eligible) > 0 {
		return repositoryTemporary("await review", w.service.repo.UpdateRunStatus(ctx, run.ID, w.workerID, domain.SkillImportRunStatusAwaitingReview, "", "", nil))
	}
	if run.Mode == domain.SkillImportModeAutoPublish && len(eligible) > 0 {
		gate := autoPublishGate(frozen.RunConfig)
		if gate.RequiresReview(counts) {
			return repositoryTemporary("hold automatic publication", w.service.repo.UpdateRunStatus(ctx, run.ID, w.workerID, domain.SkillImportRunStatusAwaitingReview,
				"AUTO_PUBLISH_GATE", "Automatic publication gate requires administrator review", nil))
		}
	}
	if len(eligible) > 0 {
		_, err = w.service.repo.PublishEligibleItems(ctx, run.ID, eligible, w.workerID, nil)
		return repositoryTemporary("publish eligible items", err)
	}
	return w.completeWithoutPublication(ctx, run.ID, counts)
}

func (w *WorkerRuntime) completeWithoutPublication(ctx context.Context, runID int64, counts domain.SkillImportRunCounts) error {
	status := domain.SkillImportRunStatusSucceeded
	if counts.Blocked > 0 || counts.Failed > 0 {
		if counts.Prepared+counts.Unchanged+counts.Skipped > 0 {
			status = domain.SkillImportRunStatusPartialSucceeded
		} else {
			status = domain.SkillImportRunStatusFailed
		}
	}
	return repositoryTemporary("complete run", w.service.repo.UpdateRunStatus(ctx, runID, w.workerID, status, "", "", nil))
}

func (w *WorkerRuntime) nextItemRetryAt(ctx context.Context, runID int64) time.Time {
	nextRetryAt, err := w.service.repo.GetNextRunItemRetryAt(ctx, runID)
	if err == nil && nextRetryAt != nil && !nextRetryAt.IsZero() {
		return nextRetryAt.UTC()
	}
	return time.Now().UTC().Add(time.Duration(max(2, w.cfg.PollIntervalSeconds)) * time.Second)
}

func (w *WorkerRuntime) handleRunError(ctx context.Context, run *domain.SkillImportRun, err error) {
	kind := core.ErrorKindOf(err)
	status := domain.SkillImportRunStatusFailed
	var next *time.Time
	if (kind == core.ErrorRateLimited || kind == core.ErrorTemporary) && run.AttemptCount < w.maxAttempts() {
		status = domain.SkillImportRunStatusWaitingRetry
		delay := retryDelay(run.AttemptCount)
		if requested, ok := core.RetryAfterOf(err); ok && requested > delay {
			delay = min(requested, time.Hour)
		}
		value := time.Now().UTC().Add(delay)
		next = &value
	}
	code := strings.ToUpper(string(kind))
	if code == "" {
		code = "RUN_FAILED"
	}
	message := safeImportError(err)
	if updateErr := w.service.repo.UpdateRunStatus(ctx, run.ID, w.workerID, status, code, message, next); updateErr != nil {
		slog.Warn("skill import run status update failed", "run_id", run.ID, "err", safeImportError(updateErr))
	}
	_ = w.appendEvent(ctx, run.ID, nil, domain.SkillImportEventLevelError, "run_error", message, map[string]any{"code": code, "retrying": status == domain.SkillImportRunStatusWaitingRetry})
}

func (w *WorkerRuntime) runHeartbeat(ctx context.Context, runID int64) {
	w.heartbeat(ctx, func(leaseUntil time.Time) error {
		return w.service.repo.HeartbeatRun(ctx, runID, w.workerID, leaseUntil)
	})
}

func (w *WorkerRuntime) itemHeartbeat(ctx context.Context, itemID int64, itemLeaseOwner string) {
	w.heartbeat(ctx, func(leaseUntil time.Time) error {
		return w.service.repo.HeartbeatRunItem(ctx, itemID, itemLeaseOwner, w.workerID, leaseUntil)
	})
}

func (w *WorkerRuntime) heartbeat(ctx context.Context, beat func(time.Time) error) {
	interval := w.leaseTTL() / 3
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := beat(time.Now().UTC().Add(w.leaseTTL())); err != nil {
				return
			}
		}
	}
}

func (w *WorkerRuntime) appendEvent(ctx context.Context, runID int64, itemID *int64, level, eventType, message string, payload any) error {
	raw, _ := json.Marshal(payload)
	return repositoryTemporary("append event", w.service.repo.AppendEvent(ctx, &domain.SkillImportEvent{
		RunID: runID, RunItemID: itemID, Level: level, EventType: eventType,
		Message: safeImportMessage(message), Payload: raw,
	}))
}

func repositoryTemporary(operation string, err error) error {
	if err == nil || core.ErrorKindOf(err) != "" {
		return err
	}
	return core.NewAdapterError("repository", operation, core.ErrorTemporary, err)
}

func (w *WorkerRuntime) leaseTTL() time.Duration {
	seconds := w.cfg.LeaseTTLSeconds
	if seconds <= 0 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}

func (w *WorkerRuntime) maxAttempts() int {
	if w.cfg.MaxAttempts > 0 {
		return w.cfg.MaxAttempts
	}
	return 5
}

func newWorkerID() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("skill-import-%d", time.Now().UnixNano())
	}
	return "skill-import-" + hex.EncodeToString(raw[:])
}

func newItemLeaseOwner(runWorkerID string) string {
	var raw [8]byte
	suffix := fmt.Sprintf("%x-%x", time.Now().UnixNano(), itemClaimSequence.Add(1))
	if _, err := rand.Read(raw[:]); err == nil {
		suffix = hex.EncodeToString(raw[:]) + "-" + suffix
	}
	return runWorkerID + ":item:" + suffix
}

func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 8 {
		attempt = 8
	}
	return min(time.Hour, 5*time.Second*time.Duration(1<<uint(attempt-1)))
}

func suggestedMarketSlug(skill core.DiscoveredSkill) string {
	raw := strings.ToLower(strings.TrimSpace(skill.SuggestedSlug))
	var output strings.Builder
	dash := false
	for _, current := range raw {
		if current >= 'a' && current <= 'z' || current >= '0' && current <= '9' {
			_, _ = output.WriteRune(current)
			dash = false
		} else if output.Len() > 0 && !dash {
			_ = output.WriteByte('-')
			dash = true
		}
	}
	result := strings.Trim(output.String(), "-")
	if result == "" {
		digest := sha256.Sum256([]byte(skill.StableKey()))
		result = "skill-" + hex.EncodeToString(digest[:4])
	}
	if len(result) > 64 {
		result = strings.Trim(result[:64], "-")
	}
	return result
}

func desiredSkillFromNormalized(discovered core.DiscoveredSkill, bundle core.SourceBundle, normalized core.NormalizedSkill, frozen FrozenRunConfig) domain.SkillImportDesiredSkill {
	category, tags := deterministicSkillClassification(discovered, normalized.Description, frozen.AdapterType)
	originURL := strings.TrimSpace(bundle.CanonicalURL)
	if originURL == "" {
		originURL = strings.TrimSpace(discovered.CanonicalURL)
	}
	sourceURL, sourceRepository := "", ""
	if isGitHubBackedSkill(discovered.Namespace, frozen) {
		sourceRepository = discovered.Namespace
		sourceURL = "https://github.com/" + discovered.Namespace
	}
	sortOrder := 0
	if discovered.Rank != nil {
		sortOrder = *discovered.Rank
	}
	return domain.SkillImportDesiredSkill{
		Slug: normalized.MarketSlug, DisplayName: truncateRunes(normalized.DisplayName, 120),
		Summary: summarizeSkill(normalized.Description, normalized.DisplayName), Description: normalized.Description,
		Category: category, Tags: tags, Icon: "", ExamplePrompts: []string{},
		RiskNotes: riskNotes(normalized.Warnings), OriginURL: originURL,
		SourceURL: sourceURL, SourceRepository: sourceRepository,
		Featured: false, SortOrder: sortOrder, CatalogSourcePriority: frozen.CatalogPriority,
		CatalogSourceRank: discovered.Rank,
	}
}

func isGitHubBackedSkill(namespace string, frozen FrozenRunConfig) bool {
	namespace = strings.TrimSpace(namespace)
	if !githubNamespacePattern.MatchString(namespace) {
		return false
	}
	switch frozen.AdapterType {
	case core.GitHubAdapterType:
		return true
	case core.SkillsSHAdapterType:
		var config struct {
			SourceBaseURLs map[string]string `json:"source_base_urls"`
		}
		if json.Unmarshal(frozen.AdapterConfig, &config) != nil {
			return false
		}
		for source := range config.SourceBaseURLs {
			if strings.TrimSpace(source) == namespace {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func normalizedProvenance(discovered core.DiscoveredSkill, bundle core.SourceBundle, normalized core.NormalizedSkill, frozen FrozenRunConfig) json.RawMessage {
	value := map[string]any{
		"adapter_type": frozen.AdapterType, "adapter_version": frozen.AdapterVersion,
		"normalizer_version": frozen.NormalizerVersion, "validator_version": frozen.ValidatorVersion,
		"namespace": discovered.Namespace, "external_id": discovered.ExternalID,
		"revision": bundle.Revision, "upstream_content_sha256": normalized.UpstreamContentHash,
		"normalized_package_sha256": normalized.NormalizedSHA256, "canonical_url": bundle.CanonicalURL,
		"integrity_verified": bundle.IntegrityVerified, "transformed": normalized.Transformed,
		"needs_review": normalized.NeedsReview, "review_reasons": normalized.ReviewReasons,
		"excluded_files": normalized.ExcludedFiles, "license_evidence": normalized.LicenseEvidence,
		"evidence": bundle.Evidence, "metrics": discovered.Metrics,
	}
	raw, _ := json.Marshal(value)
	return raw
}

func deterministicSkillClassification(skill core.DiscoveredSkill, description, adapterType string) (string, []string) {
	haystack := strings.ToLower(skill.Namespace + " " + skill.ExternalID + " " + skill.SuggestedName + " " + description)
	category := "开发工具"
	cases := []struct {
		category string
		terms    []string
	}{
		{"安全与研究", []string{"security", "secure", "audit", "osint", "vulnerability", "pentest"}},
		{"营销与增长", []string{"marketing", "seo", "social", "content", "campaign", "sales", "cro"}},
		{"设计与前端", []string{"design", "frontend", "css", "tailwind", " ui", "ux", "animation"}},
		{"测试与质量", []string{"test", "playwright", " qa", "debug", "review", "lint"}},
		{"文档与办公", []string{"document", "docs", "pdf", "spreadsheet", "slide", "presentation", "notion", "obsidian"}},
		{"数据与数据库", []string{"database", "postgres", "sql", "data", "analytics", "supabase", "firebase"}},
		{"AI 与自动化", []string{"agent", "prompt", "model", "llm", "automation", "workflow", "mcp"}},
		{"研发协作", []string{"github", "git", "commit", "release", "deploy", "ci", "repository"}},
	}
	for _, candidate := range cases {
		matched := false
		for _, term := range candidate.terms {
			if strings.Contains(haystack, term) {
				category, matched = candidate.category, true
				break
			}
		}
		if matched {
			break
		}
	}
	tagSet := map[string]struct{}{strings.ReplaceAll(adapterType, "_", "-"): {}}
	for _, source := range []string{strings.Split(skill.Namespace, "/")[0], skill.ExternalID} {
		for _, token := range strings.FieldsFunc(strings.ToLower(source), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		}) {
			if count := utf8.RuneCountInString(token); count >= 2 && count <= 40 {
				tagSet[token] = struct{}{}
			}
		}
	}
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	if len(tags) > 7 {
		tags = tags[:7]
	}
	return category, tags
}

func summarizeSkill(description, fallback string) string {
	value := strings.Join(strings.Fields(description), " ")
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	return truncateRunes(value, 280)
}

func riskNotes(warnings []domain.SkillValidationIssue) string {
	lines := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		line := warning.Code + ": " + warning.Message
		if warning.Path != "" {
			line += " (" + warning.Path + ")"
		}
		lines = append(lines, line)
	}
	return truncateRunes(strings.Join(lines, "\n"), 10000)
}

func safeGateEnabled(raw json.RawMessage) bool {
	value, err := parseSkillImportRunConfig(raw)
	if err != nil || value.SafeGate == nil {
		return true
	}
	return *value.SafeGate
}

func shouldBlockNeedsReview(mode string, runConfig json.RawMessage, needsReview bool) bool {
	return needsReview && mode == domain.SkillImportModeAutoPublish && safeGateEnabled(runConfig)
}

type parsedAutoPublishGate struct {
	RequireAllValid        bool
	AllowLicenseUnverified bool
	MaxBlockedItems        *int
	MaxFailedItems         *int
}

func autoPublishGate(raw json.RawMessage) parsedAutoPublishGate {
	result := parsedAutoPublishGate{AllowLicenseUnverified: true}
	value, err := parseSkillImportRunConfig(raw)
	if err != nil {
		// Creation rejects malformed policy, but a corrupt or legacy frozen run
		// must still fail closed if it reaches an individual policy helper.
		return parsedAutoPublishGate{RequireAllValid: true, AllowLicenseUnverified: false}
	}
	if value.AutoPublishGate == nil {
		return result
	}
	if value.AutoPublishGate.RequireAllValid != nil {
		result.RequireAllValid = *value.AutoPublishGate.RequireAllValid
	}
	if value.AutoPublishGate.AllowLicenseUnverified != nil {
		result.AllowLicenseUnverified = *value.AutoPublishGate.AllowLicenseUnverified
	}
	result.MaxBlockedItems = value.AutoPublishGate.MaxBlockedItems
	result.MaxFailedItems = value.AutoPublishGate.MaxFailedItems
	return result
}

func (g parsedAutoPublishGate) RequiresReview(counts domain.SkillImportRunCounts) bool {
	if g.RequireAllValid && (counts.Blocked > 0 || counts.Failed > 0) {
		return true
	}
	if g.MaxBlockedItems != nil && counts.Blocked > *g.MaxBlockedItems {
		return true
	}
	return g.MaxFailedItems != nil && counts.Failed > *g.MaxFailedItems
}

func itemConcurrency(raw json.RawMessage, processLimit int) int {
	parsed, err := parseSkillImportRunConfig(raw)
	value := 0
	if err == nil && parsed.Concurrency != nil {
		value = *parsed.Concurrency
	}
	if value <= 0 {
		value = processLimit
	}
	if value <= 0 {
		value = 1
	}
	if processLimit > 0 && value > processLimit {
		value = processLimit
	}
	if value > 16 {
		value = 16
	}
	return value
}

func importChangelog(adapterType, revision string) string {
	value := "Imported from " + adapterType
	if strings.TrimSpace(revision) != "" {
		value += " at revision " + strings.TrimSpace(revision)
	}
	return truncateRunes(value+".", 2000)
}

func normalizedSkillBody(packageData []byte, slug string) string {
	validated, err := domain.ValidateSkillArchive(packageData, slug)
	if err != nil || validated == nil {
		return ""
	}
	return validated.SkillMD
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}

func safeImportError(err error) string {
	if err == nil {
		return ""
	}
	return safeImportMessage(err.Error())
}

func safeImportMessage(value string) string {
	value = regexp.MustCompile(`(?i)(https://[^?\s]+)\?[^\s]+`).ReplaceAllString(value, `$1?<redacted>`)
	return truncateRunes(strings.TrimSpace(value), 1000)
}
