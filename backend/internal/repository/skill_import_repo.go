package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"golang.org/x/mod/semver"
)

// skillImportRepository stores the complete state of a resumable import. All
// workers coordinate through PostgreSQL leases; no correctness decision relies
// on process-local state.
type skillImportRepository struct{ db *sql.DB }

func NewSkillImportRepository(db *sql.DB) service.SkillImportRepository {
	return &skillImportRepository{db: db}
}

type skillImportScanner interface{ Scan(...any) error }

const skillImportSourceColumns = `
id, name, adapter, namespace, base_url, source_config, catalog_priority, enabled,
created_by, updated_by, created_at, updated_at`

func scanSkillImportSource(scanner skillImportScanner) (*service.SkillImportSource, error) {
	var source service.SkillImportSource
	var config []byte
	var createdBy, updatedBy sql.NullInt64
	if err := scanner.Scan(
		&source.ID, &source.Name, &source.Adapter, &source.Namespace, &source.BaseURL,
		&config, &source.CatalogPriority, &source.Enabled, &createdBy, &updatedBy,
		&source.CreatedAt, &source.UpdatedAt,
	); err != nil {
		return nil, err
	}
	source.SourceConfig = cloneImportJSON(config, `{}`)
	source.CreatedBy = nullInt64Pointer(createdBy)
	source.UpdatedBy = nullInt64Pointer(updatedBy)
	return &source, nil
}

func (r *skillImportRepository) CreateSource(ctx context.Context, source *service.SkillImportSource) error {
	config := normalizedImportJSON(source.SourceConfig, `{}`)
	err := r.db.QueryRowContext(ctx, `
INSERT INTO skill_import_sources (
  name, adapter, namespace, base_url, source_config, catalog_priority, enabled,
  created_by, updated_by
) VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7,$8,$8)
RETURNING id, created_at, updated_at`,
		source.Name, source.Adapter, source.Namespace, source.BaseURL, config,
		source.CatalogPriority, source.Enabled, source.CreatedBy,
	).Scan(&source.ID, &source.CreatedAt, &source.UpdatedAt)
	if err != nil {
		return mapSkillImportWriteError(err, "create skill import source")
	}
	source.SourceConfig = json.RawMessage(config)
	source.UpdatedBy = source.CreatedBy
	return nil
}

func (r *skillImportRepository) UpdateSource(ctx context.Context, source *service.SkillImportSource) error {
	config := normalizedImportJSON(source.SourceConfig, `{}`)
	result, err := r.db.ExecContext(ctx, `
UPDATE skill_import_sources SET
  name=$2, adapter=$3, namespace=$4, base_url=$5, source_config=$6::jsonb,
  catalog_priority=$7, enabled=$8, updated_by=$9, updated_at=NOW()
WHERE id=$1`, source.ID, source.Name, source.Adapter, source.Namespace,
		source.BaseURL, config, source.CatalogPriority, source.Enabled, source.UpdatedBy)
	if err != nil {
		return mapSkillImportWriteError(err, "update skill import source")
	}
	if err := requireSkillImportAffected(result, service.ErrSkillImportSourceNotFound); err != nil {
		return err
	}
	updated, err := r.GetSource(ctx, source.ID)
	if err != nil {
		return err
	}
	*source = *updated
	return nil
}

func (r *skillImportRepository) GetSource(ctx context.Context, id int64) (*service.SkillImportSource, error) {
	source, err := scanSkillImportSource(r.db.QueryRowContext(ctx,
		`SELECT `+skillImportSourceColumns+` FROM skill_import_sources WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSkillImportSourceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get skill import source: %w", err)
	}
	return source, nil
}

func (r *skillImportRepository) ListSources(ctx context.Context, filter service.SkillImportListFilter) ([]service.SkillImportSource, int64, error) {
	where, args := buildSkillImportFilter(filter, "name", "enabled")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM skill_import_sources`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count skill import sources: %w", err)
	}
	limit, offset := importPage(filter)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `SELECT `+skillImportSourceColumns+`
FROM skill_import_sources`+where+fmt.Sprintf(` ORDER BY id ASC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list skill import sources: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.SkillImportSource, 0)
	for rows.Next() {
		item, scanErr := scanSkillImportSource(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan skill import source: %w", scanErr)
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

const skillImportScheduleColumns = `
s.id, s.source_id, s.name, s.enabled, s.cron_expression, s.timezone,
s.selection, s.run_config, s.publish_policy, s.metadata_policy, s.next_run_at,
s.last_run_at, s.last_run_id, s.lease_owner, s.lease_expires_at,
s.created_by, s.updated_by, s.created_at, s.updated_at`

func scanSkillImportSchedule(scanner skillImportScanner) (*service.SkillImportSchedule, error) {
	var schedule service.SkillImportSchedule
	var selection, runConfig []byte
	var nextRunAt, lastRunAt, leaseExpiresAt sql.NullTime
	var lastRunID, createdBy, updatedBy sql.NullInt64
	var leaseOwner sql.NullString
	if err := scanner.Scan(
		&schedule.ID, &schedule.SourceID, &schedule.Name, &schedule.Enabled,
		&schedule.CronExpression, &schedule.Timezone, &selection, &runConfig,
		&schedule.PublishPolicy, &schedule.MetadataPolicy, &nextRunAt, &lastRunAt,
		&lastRunID, &leaseOwner, &leaseExpiresAt, &createdBy, &updatedBy,
		&schedule.CreatedAt, &schedule.UpdatedAt,
	); err != nil {
		return nil, err
	}
	schedule.Selection = cloneImportJSON(selection, `{}`)
	schedule.RunConfig = cloneImportJSON(runConfig, `{}`)
	schedule.NextRunAt = nullTimePointer(nextRunAt)
	schedule.LastRunAt = nullTimePointer(lastRunAt)
	schedule.LastRunID = nullInt64Pointer(lastRunID)
	schedule.LeaseOwner = nullStringPointer(leaseOwner)
	schedule.LeaseExpiresAt = nullTimePointer(leaseExpiresAt)
	schedule.CreatedBy = nullInt64Pointer(createdBy)
	schedule.UpdatedBy = nullInt64Pointer(updatedBy)
	return &schedule, nil
}

func (r *skillImportRepository) CreateSchedule(ctx context.Context, schedule *service.SkillImportSchedule) error {
	selection := normalizedImportJSON(schedule.Selection, `{}`)
	runConfig := normalizedImportJSON(schedule.RunConfig, `{}`)
	err := r.db.QueryRowContext(ctx, `
INSERT INTO skill_import_schedules (
  source_id, name, enabled, cron_expression, timezone, selection, run_config,
  publish_policy, metadata_policy, next_run_at, created_by, updated_by
) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11,$11)
RETURNING id, created_at, updated_at`, schedule.SourceID, schedule.Name,
		schedule.Enabled, schedule.CronExpression, schedule.Timezone, selection,
		runConfig, schedule.PublishPolicy, schedule.MetadataPolicy,
		schedule.NextRunAt, schedule.CreatedBy,
	).Scan(&schedule.ID, &schedule.CreatedAt, &schedule.UpdatedAt)
	if err != nil {
		return mapSkillImportWriteError(err, "create skill import schedule")
	}
	schedule.Selection = json.RawMessage(selection)
	schedule.RunConfig = json.RawMessage(runConfig)
	schedule.UpdatedBy = schedule.CreatedBy
	return nil
}

func (r *skillImportRepository) UpdateSchedule(ctx context.Context, schedule *service.SkillImportSchedule) error {
	selection := normalizedImportJSON(schedule.Selection, `{}`)
	runConfig := normalizedImportJSON(schedule.RunConfig, `{}`)
	result, err := r.db.ExecContext(ctx, `
UPDATE skill_import_schedules SET
  source_id=$2, name=$3, enabled=$4, cron_expression=$5, timezone=$6,
  selection=$7::jsonb, run_config=$8::jsonb, publish_policy=$9,
  metadata_policy=$10, next_run_at=$11, updated_by=$12, updated_at=NOW()
WHERE id=$1 AND (lease_owner IS NULL OR lease_expires_at <= NOW())`, schedule.ID,
		schedule.SourceID, schedule.Name, schedule.Enabled, schedule.CronExpression,
		schedule.Timezone, selection, runConfig, schedule.PublishPolicy,
		schedule.MetadataPolicy, schedule.NextRunAt, schedule.UpdatedBy)
	if err != nil {
		return mapSkillImportWriteError(err, "update skill import schedule")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect skill import schedule update: %w", err)
	}
	if affected == 0 {
		var exists bool
		if lookupErr := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM skill_import_schedules WHERE id=$1)`, schedule.ID).Scan(&exists); lookupErr != nil {
			return fmt.Errorf("inspect skill import schedule: %w", lookupErr)
		}
		if exists {
			return service.ErrSkillImportConflict
		}
		return service.ErrSkillImportScheduleNotFound
	}
	updated, err := r.GetSchedule(ctx, schedule.ID)
	if err != nil {
		return err
	}
	*schedule = *updated
	return nil
}

func (r *skillImportRepository) GetSchedule(ctx context.Context, id int64) (*service.SkillImportSchedule, error) {
	schedule, err := scanSkillImportSchedule(r.db.QueryRowContext(ctx,
		`SELECT `+skillImportScheduleColumns+` FROM skill_import_schedules s WHERE s.id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSkillImportScheduleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get skill import schedule: %w", err)
	}
	return schedule, nil
}

func (r *skillImportRepository) ListSchedules(ctx context.Context, filter service.SkillImportListFilter) ([]service.SkillImportSchedule, int64, error) {
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 5)
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions, fmt.Sprintf(`s.name ILIKE $%d`, len(args)))
	}
	if filter.Status != "" {
		enabled := filter.Status == "enabled" || filter.Status == "true"
		args = append(args, enabled)
		conditions = append(conditions, fmt.Sprintf(`s.enabled=$%d`, len(args)))
	}
	if filter.SourceID != nil {
		args = append(args, *filter.SourceID)
		conditions = append(conditions, fmt.Sprintf(`s.source_id=$%d`, len(args)))
	}
	where := importWhere(conditions)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM skill_import_schedules s`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count skill import schedules: %w", err)
	}
	limit, offset := importPage(filter)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `SELECT `+skillImportScheduleColumns+`
FROM skill_import_schedules s`+where+fmt.Sprintf(` ORDER BY s.id ASC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list skill import schedules: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.SkillImportSchedule, 0)
	for rows.Next() {
		item, scanErr := scanSkillImportSchedule(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan skill import schedule: %w", scanErr)
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *skillImportRepository) ClaimDueSchedule(ctx context.Context, workerID string, now, leaseUntil time.Time) (*service.SkillImportSchedule, error) {
	schedule, err := scanSkillImportSchedule(r.db.QueryRowContext(ctx, `
WITH candidate AS (
  SELECT id FROM skill_import_schedules
  WHERE enabled=TRUE AND next_run_at IS NOT NULL AND next_run_at <= $1
    AND (lease_owner IS NULL OR lease_expires_at <= $1)
  ORDER BY next_run_at ASC, id ASC
  FOR UPDATE SKIP LOCKED LIMIT 1
), claimed AS (
  UPDATE skill_import_schedules s SET
    lease_owner=$2, lease_expires_at=$3, updated_at=NOW()
  FROM candidate c WHERE s.id=c.id
  RETURNING s.*
)
SELECT `+skillImportScheduleColumns+` FROM claimed s`, now, workerID, leaseUntil))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim due skill import schedule: %w", err)
	}
	return schedule, nil
}

func (r *skillImportRepository) CompleteScheduleClaim(ctx context.Context, scheduleID int64, workerID string, runID *int64, lastRunAt, nextRunAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE skill_import_schedules SET last_run_id=$3, last_run_at=$4, next_run_at=$5,
  lease_owner=NULL, lease_expires_at=NULL, updated_at=NOW()
WHERE id=$1 AND lease_owner=$2`, scheduleID, workerID, runID, lastRunAt, nextRunAt)
	if err != nil {
		return fmt.Errorf("complete skill import schedule claim: %w", err)
	}
	return requireSkillImportLease(result)
}

func (r *skillImportRepository) ReleaseScheduleClaim(ctx context.Context, scheduleID int64, workerID string, nextRunAt time.Time) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE skill_import_schedules SET next_run_at=$3, lease_owner=NULL,
  lease_expires_at=NULL, updated_at=NOW()
WHERE id=$1 AND lease_owner=$2`, scheduleID, workerID, nextRunAt)
	if err != nil {
		return fmt.Errorf("release skill import schedule claim: %w", err)
	}
	return requireSkillImportLease(result)
}

const skillImportRunColumns = `
r.id, r.source_id, r.schedule_id, r.parent_run_id, r.trigger_type, r.mode,
r.status, r.request_config, r.snapshot, r.snapshot_sha256, r.scheduled_for,
r.requested_count, r.discovered_count, r.prepared_count, r.created_count,
r.updated_count, r.unchanged_count, r.skipped_count, r.blocked_count,
r.failed_count, r.published_count, r.cancel_requested_at,
r.cancel_requested_by, r.attempt_count, r.next_attempt_at, r.last_error_code,
r.last_error_message, r.lease_owner, r.lease_expires_at, r.heartbeat_at,
r.created_by, r.started_at, r.finished_at, r.created_at, r.updated_at`

// List responses must never materialize large inline upload payloads. GetRun
// deliberately keeps the original request_config for the worker; admin read
// redaction is an additional application-layer defense.
const skillImportRunListColumns = `
r.id, r.source_id, r.schedule_id, r.parent_run_id, r.trigger_type, r.mode,
r.status, (r.request_config #- '{adapter_config,inline_data_base64}'),
r.snapshot, r.snapshot_sha256, r.scheduled_for,
r.requested_count, r.discovered_count, r.prepared_count, r.created_count,
r.updated_count, r.unchanged_count, r.skipped_count, r.blocked_count,
r.failed_count, r.published_count, r.cancel_requested_at,
r.cancel_requested_by, r.attempt_count, r.next_attempt_at, r.last_error_code,
r.last_error_message, r.lease_owner, r.lease_expires_at, r.heartbeat_at,
r.created_by, r.started_at, r.finished_at, r.created_at, r.updated_at`

func scanSkillImportRun(scanner skillImportScanner) (*service.SkillImportRun, error) {
	var run service.SkillImportRun
	var requestConfig, snapshot []byte
	var scheduleID, parentRunID, cancelBy, createdBy sql.NullInt64
	var scheduledFor, cancelAt, nextAttemptAt, leaseExpiresAt, heartbeatAt sql.NullTime
	var startedAt, finishedAt sql.NullTime
	var leaseOwner sql.NullString
	if err := scanner.Scan(
		&run.ID, &run.SourceID, &scheduleID, &parentRunID, &run.TriggerType,
		&run.Mode, &run.Status, &requestConfig, &snapshot, &run.SnapshotSHA256,
		&scheduledFor, &run.Counts.Requested, &run.Counts.Discovered,
		&run.Counts.Prepared, &run.Counts.Created, &run.Counts.Updated,
		&run.Counts.Unchanged, &run.Counts.Skipped, &run.Counts.Blocked,
		&run.Counts.Failed, &run.Counts.Published, &cancelAt, &cancelBy,
		&run.AttemptCount, &nextAttemptAt, &run.LastErrorCode,
		&run.LastErrorMessage, &leaseOwner, &leaseExpiresAt, &heartbeatAt,
		&createdBy, &startedAt, &finishedAt, &run.CreatedAt, &run.UpdatedAt,
	); err != nil {
		return nil, err
	}
	run.ScheduleID = nullInt64Pointer(scheduleID)
	run.ParentRunID = nullInt64Pointer(parentRunID)
	run.RequestConfig = cloneImportJSON(requestConfig, `{}`)
	run.Snapshot = cloneImportJSON(snapshot, `{}`)
	run.ScheduledFor = nullTimePointer(scheduledFor)
	run.CancelRequestedAt = nullTimePointer(cancelAt)
	run.CancelRequestedBy = nullInt64Pointer(cancelBy)
	run.NextAttemptAt = nullTimePointer(nextAttemptAt)
	run.LeaseOwner = nullStringPointer(leaseOwner)
	run.LeaseExpiresAt = nullTimePointer(leaseExpiresAt)
	run.HeartbeatAt = nullTimePointer(heartbeatAt)
	run.CreatedBy = nullInt64Pointer(createdBy)
	run.StartedAt = nullTimePointer(startedAt)
	run.FinishedAt = nullTimePointer(finishedAt)
	return &run, nil
}

func (r *skillImportRepository) CreateRun(ctx context.Context, run *service.SkillImportRun, idempotencyKeyHash string) error {
	requestConfig := normalizedImportJSON(run.RequestConfig, `{}`)
	snapshot := normalizedImportJSON(run.Snapshot, `{}`)
	err := r.db.QueryRowContext(ctx, `
INSERT INTO skill_import_runs (
  source_id, schedule_id, parent_run_id, trigger_type, mode, status,
  request_config, snapshot, snapshot_sha256, idempotency_key_hash,
  scheduled_for, requested_count, created_by
) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9,$10,$11,$12,$13)
RETURNING id, created_at, updated_at`, run.SourceID, run.ScheduleID,
		run.ParentRunID, run.TriggerType, run.Mode, run.Status, requestConfig,
		snapshot, run.SnapshotSHA256, idempotencyKeyHash, run.ScheduledFor,
		run.Counts.Requested, run.CreatedBy,
	).Scan(&run.ID, &run.CreatedAt, &run.UpdatedAt)
	if err != nil {
		return mapSkillImportWriteError(err, "create skill import run")
	}
	run.RequestConfig = json.RawMessage(requestConfig)
	run.Snapshot = json.RawMessage(snapshot)
	return nil
}

func (r *skillImportRepository) GetRun(ctx context.Context, id int64) (*service.SkillImportRun, error) {
	run, err := scanSkillImportRun(r.db.QueryRowContext(ctx,
		`SELECT `+skillImportRunColumns+` FROM skill_import_runs r WHERE r.id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSkillImportRunNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get skill import run: %w", err)
	}
	return run, nil
}

func (r *skillImportRepository) ListRuns(ctx context.Context, filter service.SkillImportListFilter) ([]service.SkillImportRun, int64, error) {
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 5)
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions, fmt.Sprintf(`CAST(r.id AS TEXT) ILIKE $%d`, len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, fmt.Sprintf(`r.status=$%d`, len(args)))
	}
	if filter.SourceID != nil {
		args = append(args, *filter.SourceID)
		conditions = append(conditions, fmt.Sprintf(`r.source_id=$%d`, len(args)))
	}
	where := importWhere(conditions)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM skill_import_runs r`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count skill import runs: %w", err)
	}
	limit, offset := importPage(filter)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `SELECT `+skillImportRunListColumns+`
FROM skill_import_runs r`+where+fmt.Sprintf(` ORDER BY r.created_at DESC, r.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list skill import runs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.SkillImportRun, 0)
	for rows.Next() {
		item, scanErr := scanSkillImportRun(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan skill import run: %w", scanErr)
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *skillImportRepository) ClaimNextRun(ctx context.Context, workerID string, now, leaseUntil time.Time) (*service.SkillImportRun, error) {
	run, err := scanSkillImportRun(r.db.QueryRowContext(ctx, `
WITH candidate AS (
  SELECT id FROM skill_import_runs
  WHERE cancel_requested_at IS NULL
    AND (
      (status='queued' AND (next_attempt_at IS NULL OR next_attempt_at <= $1))
      OR (status='waiting_retry' AND next_attempt_at <= $1)
      OR (status IN ('discovering','preparing') AND lease_expires_at <= $1)
    )
    AND (lease_owner IS NULL OR lease_expires_at <= $1)
  ORDER BY COALESCE(next_attempt_at, created_at), id
  FOR UPDATE SKIP LOCKED LIMIT 1
), claimed AS (
  UPDATE skill_import_runs r SET
    status=CASE
      WHEN r.status IN ('queued','waiting_retry') AND r.snapshot_sha256<>'' THEN 'preparing'
      WHEN r.status IN ('queued','waiting_retry') THEN 'discovering'
      ELSE r.status
    END,
    lease_owner=$2, lease_expires_at=$3, heartbeat_at=$1,
    attempt_count=r.attempt_count+1, started_at=COALESCE(r.started_at,$1),
    next_attempt_at=NULL, updated_at=NOW()
  FROM candidate c WHERE r.id=c.id
  RETURNING r.*
)
SELECT `+skillImportRunColumns+` FROM claimed r`, now, workerID, leaseUntil))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim skill import run: %w", err)
	}
	return run, nil
}

func (r *skillImportRepository) HeartbeatRun(ctx context.Context, runID int64, workerID string, leaseUntil time.Time) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE skill_import_runs SET lease_expires_at=$3, heartbeat_at=NOW(), updated_at=NOW()
WHERE id=$1 AND lease_owner=$2
  AND status IN ('discovering','preparing','publishing')`, runID, workerID, leaseUntil)
	if err != nil {
		return fmt.Errorf("heartbeat skill import run: %w", err)
	}
	return requireSkillImportLease(result)
}

func (r *skillImportRepository) UpdateRunStatus(ctx context.Context, runID int64, workerID, status, errorCode, errorMessage string, nextAttemptAt *time.Time) error {
	terminal := isSkillImportRunTerminal(status)
	release := terminal || status == service.SkillImportRunStatusWaitingRetry ||
		status == service.SkillImportRunStatusReady || status == service.SkillImportRunStatusAwaitingReview
	if status != service.SkillImportRunStatusCancelled {
		tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
		if err != nil {
			return fmt.Errorf("begin skill import run status update: %w", err)
		}
		defer func() { _ = tx.Rollback() }()
		var cancelRequestedAt sql.NullTime
		if err := tx.QueryRowContext(ctx, `
SELECT cancel_requested_at FROM skill_import_runs
WHERE id=$1 AND lease_owner=$2 FOR UPDATE`, runID, workerID).Scan(&cancelRequestedAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return service.ErrSkillImportLeaseLost
			}
			return fmt.Errorf("lock skill import run status update: %w", err)
		}
		if cancelRequestedAt.Valid {
			if err := finalizeCancelledSkillImportRunTx(ctx, tx, runID, workerID, "CANCELLED", "Cancelled by administrator"); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("commit cancelled skill import run update: %w", err)
			}
			return nil
		}
		// A partial result remains retryable. Keep the bounded inline manifest
		// until every retry succeeds or an administrator explicitly cancels the
		// run; otherwise ZIP/JSON/CSV imports lose their only acquisition source.
		purgeInlineUpload := purgeInlineSkillImportUpload(status)
		result, err := tx.ExecContext(ctx, `
UPDATE skill_import_runs SET status=$3, last_error_code=$4, last_error_message=$5,
  next_attempt_at=$6,
	  request_config=CASE WHEN $9
	    THEN request_config #- '{adapter_config,inline_data_base64}'
	    ELSE request_config END,
  lease_owner=CASE WHEN $7 THEN NULL ELSE lease_owner END,
  lease_expires_at=CASE WHEN $7 THEN NULL ELSE lease_expires_at END,
  finished_at=CASE WHEN $8 THEN NOW() ELSE NULL END,
  updated_at=NOW()
WHERE id=$1 AND lease_owner=$2`, runID, workerID, status, errorCode,
			errorMessage, nextAttemptAt, release, terminal, purgeInlineUpload)
		if err != nil {
			return mapSkillImportWriteError(err, "update skill import run status")
		}
		if err := requireSkillImportLease(result); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit skill import run status update: %w", err)
		}
		return nil
	}

	// Worker-observed cancellation must release the same durable staging bytes
	// as an idle run cancelled directly by the admin endpoint. A transaction is
	// required so a crash cannot leave a terminal parent retaining full ZIPs.
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin cancelled skill import run update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := finalizeCancelledSkillImportRunTx(ctx, tx, runID, workerID, errorCode, errorMessage); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit cancelled skill import run update: %w", err)
	}
	return nil
}

func finalizeCancelledSkillImportRunTx(ctx context.Context, tx *sql.Tx, runID int64, workerID, errorCode, errorMessage string) error {
	result, err := tx.ExecContext(ctx, `
UPDATE skill_import_runs SET status='cancelled', last_error_code=$3,
  last_error_message=$4, next_attempt_at=NULL, lease_owner=NULL,
  lease_expires_at=NULL, heartbeat_at=NULL, finished_at=NOW(), updated_at=NOW()
	, request_config=request_config #- '{adapter_config,inline_data_base64}'
WHERE id=$1 AND lease_owner=$2`, runID, workerID, errorCode, errorMessage)
	if err != nil {
		return mapSkillImportWriteError(err, "cancel skill import run")
	}
	if err := requireSkillImportLease(result); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE skill_import_run_items SET status='cancelled', lease_owner=NULL,
  lease_expires_at=NULL, heartbeat_at=NULL, next_attempt_at=NULL,
  completed_at=COALESCE(completed_at,NOW()), updated_at=NOW()
WHERE run_id=$1 AND status NOT IN ('published','unchanged','skipped','cancelled')`, runID); err != nil {
		return fmt.Errorf("cancel skill import run items: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE skill_import_run_items SET
  staged_package_data=NULL,
  staged_artifact=staged_artifact - 'skill_md' - 'file_manifest',
  updated_at=NOW()
WHERE run_id=$1 AND (staged_package_data IS NOT NULL OR staged_artifact ?| ARRAY['skill_md','file_manifest'])`, runID); err != nil {
		return fmt.Errorf("scrub cancelled skill import staging: %w", err)
	}
	return nil
}

func (r *skillImportRepository) RequestRunCancellation(ctx context.Context, runID, actorID int64) (bool, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return false, fmt.Errorf("begin skill import run cancellation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	if err := tx.QueryRowContext(ctx, `
SELECT status FROM skill_import_runs WHERE id=$1 FOR UPDATE`, runID).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, service.ErrSkillImportRunNotFound
		}
		return false, fmt.Errorf("lock skill import run for cancellation: %w", err)
	}
	if isSkillImportRunTerminal(status) {
		return false, nil
	}
	result, err := tx.ExecContext(ctx, `
UPDATE skill_import_runs SET
  cancel_requested_at=COALESCE(cancel_requested_at,NOW()),
  cancel_requested_by=COALESCE(cancel_requested_by,$2),
  status='cancelled', lease_owner=NULL, lease_expires_at=NULL,
  heartbeat_at=NULL, next_attempt_at=NULL, finished_at=NOW(),
  request_config=request_config #- '{adapter_config,inline_data_base64}',
  updated_at=NOW()
WHERE id=$1 AND status=$3`, runID, actorID, status)
	if err != nil {
		return false, fmt.Errorf("request skill import run cancellation: %w", err)
	}
	if err := requireSkillImportAffected(result, service.ErrSkillImportInvalidState); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE skill_import_run_items SET status='cancelled', lease_owner=NULL,
  lease_expires_at=NULL, heartbeat_at=NULL, next_attempt_at=NULL,
  completed_at=COALESCE(completed_at,NOW()), updated_at=NOW()
WHERE run_id=$1 AND status NOT IN ('published','unchanged','skipped','cancelled')`, runID); err != nil {
		return false, fmt.Errorf("cancel skill import run items: %w", err)
	}
	// A terminally cancelled run no longer needs a retryable ZIP. Keep the
	// compact validation/provenance/hash evidence, but remove the two fields
	// that can dominate storage. This is deliberately in the same transaction
	// as the terminal state transition.
	if _, err := tx.ExecContext(ctx, `
UPDATE skill_import_run_items SET
	  staged_package_data=NULL,
	  staged_artifact=staged_artifact - 'skill_md' - 'file_manifest',
	  updated_at=NOW()
WHERE run_id=$1 AND (staged_package_data IS NOT NULL OR staged_artifact ?| ARRAY['skill_md','file_manifest'])`, runID); err != nil {
		return false, fmt.Errorf("scrub cancelled skill import staging: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit skill import run cancellation: %w", err)
	}
	return true, nil
}

func (r *skillImportRepository) RefreshRunCounts(ctx context.Context, runID int64) (service.SkillImportRunCounts, error) {
	return refreshSkillImportCountsTx(ctx, r.db, runID)
}

func (r *skillImportRepository) CompleteDiscovery(ctx context.Context, runID int64, workerID string, snapshot json.RawMessage, snapshotSHA256 string, items []service.SkillImportRunItem) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin complete skill import discovery: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var sourceID int64
	var status, leaseOwner string
	if err := tx.QueryRowContext(ctx, `
SELECT source_id, status, COALESCE(lease_owner,'')
FROM skill_import_runs WHERE id=$1 FOR UPDATE`, runID).Scan(&sourceID, &status, &leaseOwner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrSkillImportRunNotFound
		}
		return fmt.Errorf("lock skill import run for discovery: %w", err)
	}
	if status != service.SkillImportRunStatusDiscovering || leaseOwner != workerID {
		return service.ErrSkillImportLeaseLost
	}
	for i := range items {
		if items[i].RunID != 0 && items[i].RunID != runID {
			return service.ErrSkillImportConflict
		}
		if items[i].StableKey.SourceID != 0 && items[i].StableKey.SourceID != sourceID {
			return service.ErrSkillImportConflict
		}
		items[i].RunID = runID
		items[i].StableKey.SourceID = sourceID
		if items[i].Status == "" {
			items[i].Status = service.SkillImportItemStatusQueued
		}
		if err := insertSkillImportRunItem(ctx, tx, &items[i]); err != nil {
			return err
		}
	}
	snapshotValue := normalizedImportJSON(snapshot, `{}`)
	result, err := tx.ExecContext(ctx, `
UPDATE skill_import_runs SET snapshot=$3::jsonb, snapshot_sha256=$4,
  status='preparing', discovered_count=$5,
  requested_count=CASE WHEN requested_count=0 THEN $5 ELSE requested_count END,
  last_error_code='', last_error_message='', updated_at=NOW()
WHERE id=$1 AND lease_owner=$2 AND status='discovering'`, runID, workerID,
		snapshotValue, snapshotSHA256, len(items))
	if err != nil {
		return mapSkillImportWriteError(err, "complete skill import discovery")
	}
	if err := requireSkillImportLease(result); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit skill import discovery: %w", err)
	}
	return nil
}

func (r *skillImportRepository) CreateRunItems(ctx context.Context, items []service.SkillImportRunItem) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create skill import items: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for i := range items {
		if err := insertSkillImportRunItem(ctx, tx, &items[i]); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit skill import items: %w", err)
	}
	return nil
}

type skillImportDBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func insertSkillImportRunItem(ctx context.Context, db skillImportDBTX, item *service.SkillImportRunItem) error {
	desired, err := json.Marshal(item.DesiredSkill)
	if err != nil {
		return fmt.Errorf("encode desired skill: %w", err)
	}
	validation, err := json.Marshal(item.ValidationReport)
	if err != nil {
		return fmt.Errorf("encode skill validation report: %w", err)
	}
	excluded, err := json.Marshal(nonNilStrings(item.ExcludedFiles))
	if err != nil {
		return fmt.Errorf("encode excluded files: %w", err)
	}
	warnings, err := json.Marshal(nonNilIssues(item.Warnings))
	if err != nil {
		return fmt.Errorf("encode skill import warnings: %w", err)
	}
	status := item.Status
	if status == "" {
		status = service.SkillImportItemStatusQueued
	}
	err = db.QueryRowContext(ctx, `
INSERT INTO skill_import_run_items (
  run_id, source_id, namespace, external_id, rank, market_slug, status,
  upstream_name, origin_url, source_revision, source_content_sha256,
  package_sha256, desired_skill, source_payload, validation_report,
  provenance, license_unverified, excluded_files, warnings, skill_id,
  version_id, next_attempt_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14::jsonb,
  $15::jsonb,$16::jsonb,$17,$18::jsonb,$19::jsonb,$20,$21,$22)
RETURNING id, created_at, updated_at`, item.RunID, item.StableKey.SourceID,
		item.StableKey.Namespace, item.StableKey.ExternalID, item.Rank,
		item.MarketSlug, status, item.UpstreamName, item.OriginURL,
		item.SourceRevision, item.SourceContentSHA256, item.PackageSHA256,
		string(desired), normalizedImportJSON(item.SourcePayload, `{}`),
		string(validation), normalizedImportJSON(item.Provenance, `{}`),
		item.LicenseUnverified, string(excluded), string(warnings), item.SkillID,
		item.VersionID, item.NextAttemptAt,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return mapSkillImportWriteError(err, "create skill import item")
	}
	item.Status = status
	return nil
}

const skillImportItemColumns = `
i.id, i.run_id, i.source_id, i.namespace, i.external_id, i.rank,
i.market_slug, i.status, i.upstream_name, i.origin_url, i.source_revision,
i.source_content_sha256, i.package_sha256, i.stage_action, i.desired_skill, i.source_payload,
i.validation_report, i.provenance, i.license_unverified, i.excluded_files,
i.warnings, i.skill_id, i.version_id, i.attempt_count, i.next_attempt_at,
i.error_code, i.error_message, i.lease_owner, i.lease_expires_at,
i.heartbeat_at, i.started_at, i.completed_at, i.created_at, i.updated_at`

func scanSkillImportRunItem(scanner skillImportScanner) (*service.SkillImportRunItem, error) {
	var item service.SkillImportRunItem
	var desired, sourcePayload, validation, provenance, excluded, warnings []byte
	var rank, skillID, versionID sql.NullInt64
	var nextAttemptAt, leaseExpiresAt, heartbeatAt, startedAt, completedAt sql.NullTime
	var leaseOwner sql.NullString
	if err := scanner.Scan(
		&item.ID, &item.RunID, &item.StableKey.SourceID,
		&item.StableKey.Namespace, &item.StableKey.ExternalID, &rank,
		&item.MarketSlug, &item.Status, &item.UpstreamName, &item.OriginURL,
		&item.SourceRevision, &item.SourceContentSHA256, &item.PackageSHA256,
		&item.StageAction, &desired, &sourcePayload, &validation, &provenance,
		&item.LicenseUnverified, &excluded, &warnings, &skillID, &versionID,
		&item.AttemptCount, &nextAttemptAt, &item.ErrorCode, &item.ErrorMessage,
		&leaseOwner, &leaseExpiresAt, &heartbeatAt, &startedAt, &completedAt,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(desired, &item.DesiredSkill); err != nil {
		return nil, fmt.Errorf("decode desired skill: %w", err)
	}
	if err := json.Unmarshal(validation, &item.ValidationReport); err != nil {
		return nil, fmt.Errorf("decode skill validation report: %w", err)
	}
	if err := json.Unmarshal(excluded, &item.ExcludedFiles); err != nil {
		return nil, fmt.Errorf("decode excluded files: %w", err)
	}
	if err := json.Unmarshal(warnings, &item.Warnings); err != nil {
		return nil, fmt.Errorf("decode skill import warnings: %w", err)
	}
	item.SourcePayload = cloneImportJSON(sourcePayload, `{}`)
	item.Provenance = cloneImportJSON(provenance, `{}`)
	item.Rank = nullIntPointer(rank)
	item.SkillID = nullInt64Pointer(skillID)
	item.VersionID = nullInt64Pointer(versionID)
	item.NextAttemptAt = nullTimePointer(nextAttemptAt)
	item.LeaseOwner = nullStringPointer(leaseOwner)
	item.LeaseExpiresAt = nullTimePointer(leaseExpiresAt)
	item.HeartbeatAt = nullTimePointer(heartbeatAt)
	item.StartedAt = nullTimePointer(startedAt)
	item.CompletedAt = nullTimePointer(completedAt)
	if item.ExcludedFiles == nil {
		item.ExcludedFiles = []string{}
	}
	if item.Warnings == nil {
		item.Warnings = []service.SkillValidationIssue{}
	}
	return &item, nil
}

func (r *skillImportRepository) GetRunItem(ctx context.Context, id int64) (*service.SkillImportRunItem, error) {
	item, err := scanSkillImportRunItem(r.db.QueryRowContext(ctx,
		`SELECT `+skillImportItemColumns+` FROM skill_import_run_items i WHERE i.id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSkillImportItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get skill import item: %w", err)
	}
	return item, nil
}

func (r *skillImportRepository) ListRunItems(ctx context.Context, filter service.SkillImportListFilter) ([]service.SkillImportRunItem, int64, error) {
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 5)
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions, fmt.Sprintf(`(i.market_slug ILIKE $%d OR i.upstream_name ILIKE $%d OR i.external_id ILIKE $%d)`, len(args), len(args), len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, fmt.Sprintf(`i.status=$%d`, len(args)))
	}
	if filter.RunID != nil {
		args = append(args, *filter.RunID)
		conditions = append(conditions, fmt.Sprintf(`i.run_id=$%d`, len(args)))
	}
	if filter.SourceID != nil {
		args = append(args, *filter.SourceID)
		conditions = append(conditions, fmt.Sprintf(`i.source_id=$%d`, len(args)))
	}
	where := importWhere(conditions)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM skill_import_run_items i`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count skill import items: %w", err)
	}
	limit, offset := importPage(filter)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `SELECT `+skillImportItemColumns+`
FROM skill_import_run_items i`+where+fmt.Sprintf(` ORDER BY i.rank ASC NULLS LAST, i.id ASC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list skill import items: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.SkillImportRunItem, 0)
	for rows.Next() {
		item, scanErr := scanSkillImportRunItem(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan skill import item: %w", scanErr)
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *skillImportRepository) GetNextRunItemRetryAt(ctx context.Context, runID int64) (*time.Time, error) {
	var nextRetryAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
SELECT MIN(CASE
  WHEN status='queued' THEN next_attempt_at
  WHEN status='processing' THEN lease_expires_at
END)
FROM skill_import_run_items
WHERE run_id=$1 AND status IN ('queued','processing')`, runID).Scan(&nextRetryAt)
	if err != nil {
		return nil, fmt.Errorf("get next skill import item retry time: %w", err)
	}
	if !nextRetryAt.Valid {
		return nil, nil
	}
	value := nextRetryAt.Time.UTC()
	return &value, nil
}

func (r *skillImportRepository) ClaimNextRunItem(ctx context.Context, runID int64, runWorkerID, itemLeaseOwner string, now, leaseUntil time.Time) (*service.SkillImportRunItem, error) {
	if strings.TrimSpace(runWorkerID) == "" || strings.TrimSpace(itemLeaseOwner) == "" || runWorkerID == itemLeaseOwner {
		return nil, service.ErrSkillImportInvalidState
	}
	item, err := scanSkillImportRunItem(r.db.QueryRowContext(ctx, `
WITH candidate AS (
  SELECT i.id FROM skill_import_run_items i
  JOIN skill_import_runs r ON r.id=i.run_id
	  WHERE i.run_id=$1 AND r.status='preparing' AND r.cancel_requested_at IS NULL
	    AND r.lease_owner=$2 AND r.lease_expires_at >= $4
    AND (
	      (i.status='queued' AND (i.next_attempt_at IS NULL OR i.next_attempt_at <= $4))
	      OR (i.status='processing' AND i.lease_expires_at <= $4)
    )
	    AND (i.lease_owner IS NULL OR i.lease_expires_at <= $4)
  ORDER BY i.rank ASC NULLS LAST, i.id ASC
  FOR UPDATE OF i SKIP LOCKED LIMIT 1
), claimed AS (
	  UPDATE skill_import_run_items i SET status='processing', lease_owner=$3,
	    lease_expires_at=$5, heartbeat_at=$4, attempt_count=i.attempt_count+1,
	    next_attempt_at=NULL, started_at=COALESCE(i.started_at,$4),
	    completed_at=NULL, updated_at=NOW()
  FROM candidate c WHERE i.id=c.id
  RETURNING i.*
)
SELECT `+skillImportItemColumns+` FROM claimed i`, runID, runWorkerID, itemLeaseOwner, now, leaseUntil))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim skill import item: %w", err)
	}
	return item, nil
}

func (r *skillImportRepository) HeartbeatRunItem(ctx context.Context, itemID int64, itemLeaseOwner, runWorkerID string, leaseUntil time.Time) error {
	result, err := r.db.ExecContext(ctx, `
UPDATE skill_import_run_items i SET lease_expires_at=$4, heartbeat_at=NOW(), updated_at=NOW()
FROM skill_import_runs r
WHERE i.id=$1 AND i.lease_owner=$2 AND i.status='processing'
  AND r.id=i.run_id AND r.status='preparing' AND r.cancel_requested_at IS NULL
  AND r.lease_owner=$3 AND r.lease_expires_at > NOW()`, itemID, itemLeaseOwner, runWorkerID, leaseUntil)
	if err != nil {
		return fmt.Errorf("heartbeat skill import item: %w", err)
	}
	return requireSkillImportLease(result)
}

func (r *skillImportRepository) CompleteRunItem(ctx context.Context, itemID int64, itemLeaseOwner, runWorkerID string, patch service.SkillImportRunItemPatch) error {
	desired, err := json.Marshal(patch.DesiredSkill)
	if err != nil {
		return fmt.Errorf("encode desired skill: %w", err)
	}
	validation, err := json.Marshal(patch.ValidationReport)
	if err != nil {
		return fmt.Errorf("encode skill validation report: %w", err)
	}
	provenance := normalizedImportJSON(patch.Provenance, `{}`)
	excluded, err := json.Marshal(nonNilStrings(patch.ExcludedFiles))
	if err != nil {
		return fmt.Errorf("encode excluded files: %w", err)
	}
	warnings, err := json.Marshal(nonNilIssues(patch.Warnings))
	if err != nil {
		return fmt.Errorf("encode skill import warnings: %w", err)
	}
	terminal := patch.Status != service.SkillImportItemStatusQueued && patch.Status != service.SkillImportItemStatusProcessing
	result, err := r.db.ExecContext(ctx, `
UPDATE skill_import_run_items i SET status=$3::text, market_slug=$4,
  source_revision=$5, source_content_sha256=$6, package_sha256=$7,
  desired_skill=$8::jsonb, validation_report=$9::jsonb,
  provenance=$10::jsonb, license_unverified=$11, excluded_files=$12::jsonb,
  warnings=$13::jsonb, skill_id=$14, version_id=$15, error_code=$16,
  error_message=$17, next_attempt_at=CASE
    WHEN $3::text='queued' THEN $18::timestamptz ELSE NULL::timestamptz
  END,
  lease_owner=NULL, lease_expires_at=NULL,
	  completed_at=CASE WHEN $19 THEN NOW() ELSE NULL END, updated_at=NOW()
FROM skill_import_runs r
WHERE i.id=$1 AND i.lease_owner=$2 AND i.status='processing'
  AND r.id=i.run_id AND r.status='preparing' AND r.cancel_requested_at IS NULL
  AND r.lease_owner=$20 AND r.lease_expires_at > NOW()`, itemID, itemLeaseOwner,
		patch.Status, patch.MarketSlug, patch.SourceRevision,
		patch.SourceContentSHA256, patch.PackageSHA256, string(desired),
		string(validation), provenance, patch.LicenseUnverified, string(excluded),
		string(warnings), patch.SkillID, patch.VersionID, patch.ErrorCode,
		patch.ErrorMessage, patch.NextAttemptAt, terminal, runWorkerID)
	if err != nil {
		return mapSkillImportWriteError(err, "complete skill import item")
	}
	return requireSkillImportLease(result)
}

// ResetFailedItems and the parent run transition intentionally share one
// transaction. Otherwise a retried item can become queued while its terminal
// run remains permanently unclaimable.
func (r *skillImportRepository) ResetFailedItems(ctx context.Context, runID int64, itemIDs []int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin retry skill import items: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var status string
	var runLeaseOwner sql.NullString
	var runLeaseExpiresAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `
SELECT status, lease_owner, lease_expires_at
FROM skill_import_runs WHERE id=$1 FOR UPDATE`, runID).Scan(&status, &runLeaseOwner, &runLeaseExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, service.ErrSkillImportRunNotFound
		}
		return 0, fmt.Errorf("lock skill import run for retry: %w", err)
	}
	if status != service.SkillImportRunStatusFailed && status != service.SkillImportRunStatusPartialSucceeded &&
		status != service.SkillImportRunStatusSucceeded && status != service.SkillImportRunStatusCancelled &&
		status != service.SkillImportRunStatusAwaitingReview && status != service.SkillImportRunStatusReady {
		return 0, service.ErrSkillImportInvalidState
	}
	now := time.Now().UTC()
	if runLeaseOwner.Valid && (!runLeaseExpiresAt.Valid || runLeaseExpiresAt.Time.After(now)) {
		return 0, service.ErrSkillImportInvalidState
	}

	var result sql.Result
	if len(itemIDs) == 0 {
		result, err = tx.ExecContext(ctx, `
UPDATE skill_import_run_items SET status='queued', error_code='', error_message='',
  next_attempt_at=NULL, lease_owner=NULL, lease_expires_at=NULL,
  heartbeat_at=NULL, completed_at=NULL, updated_at=NOW()
WHERE run_id=$1 AND (
  status='failed'
  OR (status='processing' AND (lease_expires_at IS NULL OR lease_expires_at <= $2))
)`, runID, now)
	} else {
		result, err = tx.ExecContext(ctx, `
UPDATE skill_import_run_items SET status='queued', error_code='', error_message='',
  next_attempt_at=NULL, lease_owner=NULL, lease_expires_at=NULL,
  heartbeat_at=NULL, completed_at=NULL, updated_at=NOW()
WHERE run_id=$1 AND id=ANY($2) AND (
  status IN ('failed','blocked')
  OR (status='processing' AND (lease_expires_at IS NULL OR lease_expires_at <= $3))
)`, runID, pq.Array(itemIDs), now)
	}
	if err != nil {
		return 0, fmt.Errorf("reset failed skill import items: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("inspect reset skill import items: %w", err)
	}
	// A failed parent can be the only persisted evidence of a transient final
	// refresh/publication error: every item may already be ready/unchanged. In
	// that case an unfiltered retry intentionally requeues only the parent.
	if affected == 0 && (len(itemIDs) != 0 || status != service.SkillImportRunStatusFailed) {
		return 0, service.ErrSkillImportInvalidState
	}
	if len(itemIDs) > 0 && affected != int64(len(uniqueImportIDs(itemIDs))) {
		return 0, service.ErrSkillImportInvalidState
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE skill_import_runs SET status='queued', cancel_requested_at=NULL,
  cancel_requested_by=NULL, next_attempt_at=NULL, last_error_code='',
  last_error_message='', lease_owner=NULL, lease_expires_at=NULL,
  heartbeat_at=NULL, finished_at=NULL, updated_at=NOW()
WHERE id=$1`, runID); err != nil {
		return 0, fmt.Errorf("requeue skill import run: %w", err)
	}
	if _, err := refreshSkillImportCountsTx(ctx, tx, runID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit retry skill import items: %w", err)
	}
	return affected, nil
}

func (r *skillImportRepository) ListEligibleItemIDs(ctx context.Context, runID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id FROM skill_import_run_items
WHERE run_id=$1 AND status IN ('ready','unchanged')
	  AND (
	    (stage_action IN ('create','new_version') AND skill_id IS NULL
	      AND version_id IS NULL AND staged_package_data IS NOT NULL)
	    OR (stage_action='unchanged' AND skill_id IS NOT NULL AND version_id IS NOT NULL)
	  )
	  AND COALESCE((validation_report->>'valid')::boolean,FALSE)=TRUE
ORDER BY rank ASC NULLS LAST, id ASC`, runID)
	if err != nil {
		return nil, fmt.Errorf("list eligible skill import items: %w", err)
	}
	defer func() { _ = rows.Close() }()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan eligible skill import item: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *skillImportRepository) StagePreparedItem(ctx context.Context, input service.SkillImportStagePreparedInput) (*service.SkillImportStagePreparedResult, error) {
	if !input.Artifact.ValidationReport.Valid || input.Artifact.PackageSHA256 == "" || len(input.Artifact.PackageData) == 0 {
		return nil, service.ErrSkillImportPublishInvalid
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin stage skill import item: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Lock parent before child everywhere (publication and cancellation use the
	// same order). Besides avoiding a cancellation/staging deadlock, this makes
	// it impossible to commit a new staging reservation after the run has
	// already become terminal.
	var runSourceID int64
	var sourcePriority int
	var runStatus, runLeaseOwner string
	var cancelRequestedAt sql.NullTime
	var runLeaseExpiresAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `
SELECT r.source_id, r.status, r.cancel_requested_at, src.catalog_priority,
  COALESCE(r.lease_owner,''), r.lease_expires_at
FROM skill_import_runs r
JOIN skill_import_sources src ON src.id=r.source_id
WHERE r.id=$1 FOR UPDATE OF r`, input.RunID).Scan(
		&runSourceID, &runStatus, &cancelRequestedAt, &sourcePriority,
		&runLeaseOwner, &runLeaseExpiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSkillImportRunNotFound
		}
		return nil, fmt.Errorf("lock skill import run for staging: %w", err)
	}
	if runStatus != service.SkillImportRunStatusPreparing || cancelRequestedAt.Valid ||
		runLeaseOwner != input.RunWorkerID || !runLeaseExpiresAt.Valid ||
		!runLeaseExpiresAt.Time.After(time.Now().UTC()) {
		return nil, service.ErrSkillImportInvalidState
	}

	var namespace, externalID, status, leaseOwner string
	if err := tx.QueryRowContext(ctx, `
SELECT i.source_id, i.namespace, i.external_id, i.status,
  COALESCE(i.lease_owner,'')
FROM skill_import_run_items i
WHERE i.id=$1 AND i.run_id=$2
FOR UPDATE OF i`, input.RunItemID, input.RunID).Scan(&runSourceID, &namespace,
		&externalID, &status, &leaseOwner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSkillImportItemNotFound
		}
		return nil, fmt.Errorf("lock skill import item for staging: %w", err)
	}
	if status != service.SkillImportItemStatusProcessing || leaseOwner != input.ItemLeaseOwner {
		return nil, service.ErrSkillImportLeaseLost
	}
	if input.StableKey.SourceID != runSourceID || input.StableKey.Namespace != namespace ||
		input.StableKey.ExternalID != externalID {
		return nil, service.ErrSkillImportConflict
	}
	// Serialize allocation for the same suggested public slug without making
	// unrelated item staging contend on the parent run.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, input.DesiredSkill.Slug); err != nil {
		return nil, fmt.Errorf("lock imported skill slug allocation: %w", err)
	}
	originURL := strings.TrimSpace(input.OriginURL)
	if originURL == "" {
		originURL = strings.TrimSpace(input.DesiredSkill.OriginURL)
	}
	input.DesiredSkill.OriginURL = originURL
	input.DesiredSkill.CatalogSourcePriority = sourcePriority
	input.DesiredSkill.CatalogSourceRank = input.Rank
	input.Artifact.Transformed = input.Transformed

	var skillID int64
	var originSkillID sql.NullInt64
	var originMarketSlug, existingSkillSlug, existingSkillStatus string
	lookupErr := tx.QueryRowContext(ctx, `
SELECT o.skill_id, o.market_slug, s.slug, s.status
FROM skill_origins o
JOIN skills s ON s.id=o.skill_id
WHERE o.source_id=$1 AND o.namespace=$2 AND o.external_id=$3
FOR UPDATE OF o, s`, runSourceID, namespace, externalID).Scan(
		&originSkillID, &originMarketSlug, &existingSkillSlug, &existingSkillStatus,
	)
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return nil, fmt.Errorf("lock existing skill origin: %w", lookupErr)
	}
	marketSlug := input.DesiredSkill.Slug
	result := &service.SkillImportStagePreparedResult{MarketSlug: marketSlug}
	if originSkillID.Valid {
		skillID = originSkillID.Int64
		marketSlug = originMarketSlug
		result.MarketSlug = marketSlug
		if existingSkillSlug != marketSlug || existingSkillStatus == service.SkillStatusArchived {
			return nil, service.ErrSkillImportConflict
		}
	} else {
		marketSlug, err = resolveSkillImportMarketSlug(ctx, tx, input.DesiredSkill.Slug, input.StableKey, input.RunItemID)
		if err != nil {
			return nil, err
		}
		result.MarketSlug = marketSlug
		if marketSlug != input.DesiredSkill.Slug || marketSlug != input.Artifact.ManifestName {
			return &service.SkillImportStagePreparedResult{
				Action: service.SkillImportStageActionRenormalize, MarketSlug: marketSlug,
			}, nil
		}
	}
	if marketSlug != input.DesiredSkill.Slug || marketSlug != input.Artifact.ManifestName {
		return &service.SkillImportStagePreparedResult{
			Action: service.SkillImportStageActionRenormalize, MarketSlug: marketSlug,
		}, nil
	}
	input.DesiredSkill.Slug = marketSlug
	var stagedPackageData []byte
	var stagedSkillID, stagedVersionID any
	if !originSkillID.Valid {
		result.Action = service.SkillImportStageActionCreate
		result.Version = "1.0.0"
		stagedPackageData = input.Artifact.PackageData
	} else {
		var existingVersionID int64
		var existingVersion string
		var existingVersionYankedAt sql.NullTime
		err = tx.QueryRowContext(ctx, `
SELECT id, version, yanked_at FROM skill_versions
WHERE skill_id=$1 AND sha256=$2`, skillID, input.Artifact.PackageSHA256).Scan(
			&existingVersionID, &existingVersion, &existingVersionYankedAt,
		)
		if err == nil {
			if existingVersionYankedAt.Valid {
				return nil, service.ErrSkillImportVersionYanked
			}
			result.Action = service.SkillImportStageActionUnchanged
			result.SkillID = skillID
			result.VersionID = existingVersionID
			result.Version = existingVersion
			stagedSkillID = skillID
			stagedVersionID = existingVersionID
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("find matching imported skill version: %w", err)
		} else {
			version, versionErr := nextSkillImportVersion(ctx, tx, skillID)
			if versionErr != nil {
				return nil, versionErr
			}
			result.Action = service.SkillImportStageActionNewVersion
			result.Version = version
			stagedPackageData = input.Artifact.PackageData
		}
	}

	stagedArtifact, err := json.Marshal(input.Artifact)
	if err != nil {
		return nil, fmt.Errorf("encode staged skill artifact: %w", err)
	}
	provenance := normalizedImportJSON(input.Provenance, `{}`)
	desired, err := json.Marshal(input.DesiredSkill)
	if err != nil {
		return nil, fmt.Errorf("encode staged desired skill: %w", err)
	}
	validation, err := json.Marshal(input.Artifact.ValidationReport)
	if err != nil {
		return nil, fmt.Errorf("encode staged validation report: %w", err)
	}
	excluded, err := json.Marshal(nonNilStrings(input.ExcludedFiles))
	if err != nil {
		return nil, fmt.Errorf("encode staged exclusions: %w", err)
	}
	warnings, err := json.Marshal(nonNilIssues(input.Warnings))
	if err != nil {
		return nil, fmt.Errorf("encode staged warnings: %w", err)
	}
	itemStatus := service.SkillImportItemStatusReady
	if result.Action == service.SkillImportStageActionUnchanged {
		itemStatus = service.SkillImportItemStatusUnchanged
	}
	updateResult, err := tx.ExecContext(ctx, `
UPDATE skill_import_run_items SET status=$3, rank=$4, desired_skill=$5::jsonb,
  origin_url=$6, source_revision=$7, source_content_sha256=$8,
  package_sha256=$9, validation_report=$10::jsonb, provenance=$11::jsonb,
  license_unverified=$12, excluded_files=$13::jsonb, warnings=$14::jsonb,
  market_slug=$15, stage_action=$16, staged_artifact=$17::jsonb,
  staged_package_data=$18, skill_id=$19, version_id=$20,
  error_code='', error_message='',
  lease_owner=NULL, lease_expires_at=NULL, completed_at=NOW(), updated_at=NOW()
WHERE id=$1 AND lease_owner=$2 AND status='processing'`, input.RunItemID,
		input.ItemLeaseOwner, itemStatus, input.Rank, string(desired), originURL,
		input.SourceRevision, input.SourceContentSHA256, input.Artifact.PackageSHA256,
		string(validation), provenance, input.LicenseUnverified, string(excluded),
		string(warnings), marketSlug, result.Action, string(stagedArtifact),
		stagedPackageData, stagedSkillID, stagedVersionID)
	if err != nil {
		return nil, mapSkillImportWriteError(err, "stage skill import item")
	}
	if err := requireSkillImportLease(updateResult); err != nil {
		return nil, err
	}
	if _, err := refreshSkillImportCountsTx(ctx, tx, input.RunID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit staged skill import item: %w", err)
	}
	return result, nil
}

type lockedPublishItem struct {
	id, sourceID              int64
	namespace, externalID     string
	rank                      *int
	marketSlug, status        string
	stageAction               string
	desired                   service.SkillImportDesiredSkill
	validationValid           bool
	skillID, versionID        *int64
	originURL, sourceRevision string
	sourceContentSHA256       string
	packageSHA256             string
	artifact                  service.SkillImportPreparedArtifact
	packageData               []byte
	provenance                json.RawMessage
}

func (r *skillImportRepository) PublishEligibleItems(ctx context.Context, runID int64, itemIDs []int64, workerID string, actorID *int64) (*service.SkillImportPublishResult, error) {
	ids := uniqueImportIDs(itemIDs)
	if len(ids) == 0 || len(ids) != len(itemIDs) {
		return nil, service.ErrSkillImportPublishInvalid
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin publish skill import run: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var runStatus string
	var requestConfig []byte
	var cancelRequestedAt sql.NullTime
	var runLeaseOwner sql.NullString
	var runLeaseExpiresAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `
SELECT status, request_config, cancel_requested_at, lease_owner, lease_expires_at
FROM skill_import_runs WHERE id=$1 FOR UPDATE`, runID).Scan(
		&runStatus, &requestConfig, &cancelRequestedAt, &runLeaseOwner, &runLeaseExpiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSkillImportRunNotFound
		}
		return nil, fmt.Errorf("lock skill import run for publication: %w", err)
	}
	adminPublish := actorID != nil
	validRunState := (adminPublish && (runStatus == service.SkillImportRunStatusReady ||
		runStatus == service.SkillImportRunStatusAwaitingReview)) ||
		(!adminPublish && runStatus == service.SkillImportRunStatusPreparing &&
			runLeaseOwner.Valid && runLeaseOwner.String == workerID &&
			runLeaseExpiresAt.Valid && runLeaseExpiresAt.Time.After(time.Now().UTC()))
	if !validRunState {
		return nil, service.ErrSkillImportInvalidState
	}
	if cancelRequestedAt.Valid {
		return nil, service.ErrSkillImportInvalidState
	}
	var unfinished int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_import_run_items
WHERE run_id=$1 AND status IN ('queued','processing')`, runID).Scan(&unfinished); err != nil {
		return nil, fmt.Errorf("check unfinished skill import items: %w", err)
	}
	if unfinished != 0 {
		return nil, service.ErrSkillImportPublishInvalid
	}

	rows, err := tx.QueryContext(ctx, `
SELECT id, source_id, namespace, external_id, rank, market_slug, status,
  stage_action, desired_skill,
  COALESCE((validation_report->>'valid')::boolean,FALSE),
  skill_id, version_id, origin_url, source_revision, source_content_sha256,
  package_sha256, staged_artifact, staged_package_data, provenance
FROM skill_import_run_items
WHERE run_id=$1 AND id=ANY($2)
ORDER BY id FOR UPDATE`, runID, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("lock eligible skill import items: %w", err)
	}
	locked := make([]lockedPublishItem, 0, len(ids))
	for rows.Next() {
		var item lockedPublishItem
		var desired, stagedArtifact, provenance []byte
		var rank, skillID, versionID sql.NullInt64
		if err := rows.Scan(
			&item.id, &item.sourceID, &item.namespace, &item.externalID, &rank,
			&item.marketSlug, &item.status, &item.stageAction, &desired,
			&item.validationValid, &skillID, &versionID, &item.originURL,
			&item.sourceRevision, &item.sourceContentSHA256, &item.packageSHA256,
			&stagedArtifact, &item.packageData, &provenance,
		); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan eligible skill import item: %w", err)
		}
		if err := json.Unmarshal(desired, &item.desired); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("decode publish metadata: %w", err)
		}
		if err := json.Unmarshal(stagedArtifact, &item.artifact); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("decode staged skill artifact: %w", err)
		}
		item.rank = nullIntPointer(rank)
		item.skillID = nullInt64Pointer(skillID)
		item.versionID = nullInt64Pointer(versionID)
		item.provenance = cloneImportJSON(provenance, `{}`)
		locked = append(locked, item)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close eligible skill import rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eligible skill import items: %w", err)
	}
	if len(locked) != len(ids) {
		return nil, service.ErrSkillImportPublishInvalid
	}
	for _, item := range locked {
		if item.status != service.SkillImportItemStatusReady && item.status != service.SkillImportItemStatusUnchanged {
			return nil, service.ErrSkillImportPublishInvalid
		}
		if !item.validationValid || !item.artifact.ValidationReport.Valid ||
			item.marketSlug == "" || item.marketSlug != item.desired.Slug ||
			item.marketSlug != item.artifact.ManifestName ||
			item.packageSHA256 == "" || item.packageSHA256 != item.artifact.PackageSHA256 {
			return nil, service.ErrSkillImportPublishInvalid
		}
		switch item.stageAction {
		case service.SkillImportStageActionCreate, service.SkillImportStageActionNewVersion:
			if item.status != service.SkillImportItemStatusReady || item.skillID != nil || item.versionID != nil ||
				len(item.packageData) == 0 || int64(len(item.packageData)) != item.artifact.ByteSize {
				return nil, service.ErrSkillImportPublishInvalid
			}
		case service.SkillImportStageActionUnchanged:
			if item.status != service.SkillImportItemStatusUnchanged || item.skillID == nil || item.versionID == nil ||
				len(item.packageData) != 0 {
				return nil, service.ErrSkillImportPublishInvalid
			}
		default:
			return nil, service.ErrSkillImportPublishInvalid
		}
	}
	var omittedEligible int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM skill_import_run_items
WHERE run_id=$1 AND status IN ('ready','unchanged')
  AND NOT (id=ANY($2))`, runID, pq.Array(ids)).Scan(&omittedEligible); err != nil {
		return nil, fmt.Errorf("check omitted eligible skill import items: %w", err)
	}
	if omittedEligible != 0 {
		return nil, service.ErrSkillImportPublishInvalid
	}
	// All public slugs are locked in deterministic order before marketplace
	// rows are touched. This complements the staging reservation and keeps two
	// cohorts from deadlocking when they contain overlapping slugs.
	slugs := make([]string, 0, len(locked))
	seenSlugs := make(map[string]struct{}, len(locked))
	for _, item := range locked {
		if _, seen := seenSlugs[item.marketSlug]; !seen {
			seenSlugs[item.marketSlug] = struct{}{}
			slugs = append(slugs, item.marketSlug)
		}
	}
	sort.Strings(slugs)
	for _, slug := range slugs {
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, slug); err != nil {
			return nil, fmt.Errorf("lock imported skill publication slug: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE skill_import_runs SET status='publishing', updated_at=NOW() WHERE id=$1`, runID); err != nil {
		return nil, fmt.Errorf("mark skill import run publishing: %w", err)
	}

	metadataPolicy := importMetadataPolicy(requestConfig)
	published := make([]service.SkillImportPublishItemResult, 0, len(locked))
	for _, item := range locked {
		skillID, versionID, originID, err := publishStagedSkillImportItem(
			ctx, tx, runID, item, metadataPolicy, actorID,
		)
		if err != nil {
			return nil, err
		}
		_ = originID
		if _, err := tx.ExecContext(ctx, `
UPDATE skill_versions SET released_at=COALESCE(released_at,NOW()),
  released_by=COALESCE(released_by,$3)
WHERE id=$1 AND skill_id=$2 AND yanked_at IS NULL`, versionID, skillID, actorID); err != nil {
			return nil, fmt.Errorf("release imported skill version: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
UPDATE skills SET status='published', current_version_id=$2,
  published_at=COALESCE(published_at,NOW()), archived_at=NULL,
  updated_by=$3, updated_at=NOW()
WHERE id=$1 AND status<>'archived'`, skillID, versionID, actorID)
		if err != nil {
			return nil, mapSkillImportWriteError(err, "publish imported skill")
		}
		if err := requireSkillImportAffected(result, service.ErrSkillImportPublishInvalid); err != nil {
			return nil, err
		}
		itemStatus := service.SkillImportItemStatusPublished
		if item.stageAction == service.SkillImportStageActionUnchanged {
			itemStatus = service.SkillImportItemStatusUnchanged
		}
		itemResult, err := tx.ExecContext(ctx, `
UPDATE skill_import_run_items SET status=$2, skill_id=$3, version_id=$4,
	  staged_package_data=NULL,
	  staged_artifact=staged_artifact - 'skill_md' - 'file_manifest',
	  completed_at=NOW(), updated_at=NOW()
WHERE id=$1 AND status IN ('ready','unchanged')`, item.id, itemStatus, skillID, versionID)
		if err != nil {
			return nil, fmt.Errorf("mark skill import item published: %w", err)
		}
		if err := requireSkillImportAffected(itemResult, service.ErrSkillImportPublishInvalid); err != nil {
			return nil, err
		}
		published = append(published, service.SkillImportPublishItemResult{
			RunItemID: item.id, SkillID: skillID, VersionID: versionID,
		})
	}
	counts, err := refreshSkillImportCountsTx(ctx, tx, runID)
	if err != nil {
		return nil, err
	}
	finalStatus := service.SkillImportRunStatusSucceeded
	if counts.Blocked > 0 || counts.Failed > 0 {
		finalStatus = service.SkillImportRunStatusPartialSucceeded
	}
	purgeInlineUpload := purgeInlineSkillImportUpload(finalStatus)
	if _, err := tx.ExecContext(ctx, `
UPDATE skill_import_runs SET status=$2, lease_owner=NULL, lease_expires_at=NULL,
  heartbeat_at=NULL, finished_at=NOW(), last_error_code='',
	  last_error_message='',
	  request_config=CASE WHEN $3
	    THEN request_config #- '{adapter_config,inline_data_base64}'
	    ELSE request_config END,
	  updated_at=NOW()
WHERE id=$1`, runID, finalStatus, purgeInlineUpload); err != nil {
		return nil, fmt.Errorf("complete skill import publication: %w", err)
	}
	publishedAt := time.Now().UTC()
	eventPayload, _ := json.Marshal(map[string]any{"published": len(published), "status": finalStatus})
	if _, err := tx.ExecContext(ctx, `
INSERT INTO skill_import_events (run_id, level, event_type, message, payload)
VALUES ($1,'info','publication_completed','Eligible Skill cohort published',$2::jsonb)`, runID, string(eventPayload)); err != nil {
		return nil, fmt.Errorf("record skill import publication event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit skill import publication: %w", err)
	}
	return &service.SkillImportPublishResult{
		RunID: runID, Status: finalStatus, Items: published,
		Counts: counts, PublishedAt: publishedAt,
	}, nil
}

func purgeInlineSkillImportUpload(status string) bool {
	// Failed and partially successful runs can be requeued. Their bounded
	// upload is therefore still live recovery state, not disposable history.
	return status == service.SkillImportRunStatusSucceeded
}

// publishStagedSkillImportItem is intentionally called only from the cohort
// transaction above. No marketplace row is durable unless every selected item
// validates, publishes, and the parent run completes in that same transaction.
func publishStagedSkillImportItem(
	ctx context.Context,
	tx *sql.Tx,
	runID int64,
	item lockedPublishItem,
	metadataPolicy string,
	actorID *int64,
) (int64, int64, int64, error) {
	var skillID, versionID int64
	switch item.stageAction {
	case service.SkillImportStageActionCreate:
		var occupied bool
		if err := tx.QueryRowContext(ctx, `
SELECT EXISTS(
  SELECT 1 FROM skill_origins
  WHERE source_id=$1 AND namespace=$2 AND external_id=$3
) OR EXISTS(SELECT 1 FROM skills WHERE slug=$4)`, item.sourceID, item.namespace,
			item.externalID, item.marketSlug).Scan(&occupied); err != nil {
			return 0, 0, 0, fmt.Errorf("verify staged skill identity: %w", err)
		}
		if occupied {
			return 0, 0, 0, service.ErrSkillImportConflict
		}
		var err error
		skillID, err = insertPublishedSkillImportSkill(ctx, tx, item.desired, actorID)
		if err != nil {
			return 0, 0, 0, err
		}
		versionID, err = insertPublishedSkillImportVersion(
			ctx, tx, skillID, "1.0.0", item.artifact, item.packageData, actorID,
		)
		if err != nil {
			return 0, 0, 0, err
		}

	case service.SkillImportStageActionNewVersion:
		var skillSlug, skillStatus string
		if err := tx.QueryRowContext(ctx, `
SELECT o.skill_id, s.slug, s.status
FROM skill_origins o
JOIN skills s ON s.id=o.skill_id
WHERE o.source_id=$1 AND o.namespace=$2 AND o.external_id=$3
FOR UPDATE OF o, s`, item.sourceID, item.namespace, item.externalID).Scan(
			&skillID, &skillSlug, &skillStatus,
		); err != nil {
			return 0, 0, 0, service.ErrSkillImportPublishInvalid
		}
		if skillSlug != item.marketSlug || skillStatus == service.SkillStatusArchived {
			return 0, 0, 0, service.ErrSkillImportPublishInvalid
		}
		var existingVersion string
		err := tx.QueryRowContext(ctx, `
SELECT id, version FROM skill_versions
WHERE skill_id=$1 AND sha256=$2 AND yanked_at IS NULL`, skillID,
			item.packageSHA256).Scan(&versionID, &existingVersion)
		if errors.Is(err, sql.ErrNoRows) {
			version, versionErr := nextSkillImportVersion(ctx, tx, skillID)
			if versionErr != nil {
				return 0, 0, 0, versionErr
			}
			versionID, err = insertPublishedSkillImportVersion(
				ctx, tx, skillID, version, item.artifact, item.packageData, actorID,
			)
		}
		if err != nil {
			return 0, 0, 0, fmt.Errorf("resolve staged skill version: %w", err)
		}
		if metadataPolicy == service.SkillImportMetadataPolicyRefresh {
			if err := updatePublishedSkillMetadata(ctx, tx, skillID, item.desired, actorID); err != nil {
				return 0, 0, 0, err
			}
		}

	case service.SkillImportStageActionUnchanged:
		if item.skillID == nil || item.versionID == nil {
			return 0, 0, 0, service.ErrSkillImportPublishInvalid
		}
		skillID, versionID = *item.skillID, *item.versionID
		var originSkillID, versionSkillID int64
		var skillSlug, skillStatus, versionSHA string
		var yankedAt sql.NullTime
		if err := tx.QueryRowContext(ctx, `
SELECT o.skill_id, s.slug, s.status, v.skill_id, v.sha256, v.yanked_at
FROM skill_origins o
JOIN skills s ON s.id=o.skill_id
JOIN skill_versions v ON v.id=$4
WHERE o.source_id=$1 AND o.namespace=$2 AND o.external_id=$3
FOR UPDATE OF o, s, v`, item.sourceID, item.namespace, item.externalID,
			versionID).Scan(&originSkillID, &skillSlug, &skillStatus,
			&versionSkillID, &versionSHA, &yankedAt); err != nil {
			return 0, 0, 0, service.ErrSkillImportPublishInvalid
		}
		if originSkillID != skillID || versionSkillID != skillID ||
			skillSlug != item.marketSlug || versionSHA != item.packageSHA256 ||
			skillStatus == service.SkillStatusArchived || yankedAt.Valid {
			return 0, 0, 0, service.ErrSkillImportPublishInvalid
		}
		if metadataPolicy == service.SkillImportMetadataPolicyRefresh {
			if err := updatePublishedSkillMetadata(ctx, tx, skillID, item.desired, actorID); err != nil {
				return 0, 0, 0, err
			}
		}

	default:
		return 0, 0, 0, service.ErrSkillImportPublishInvalid
	}

	originID, err := upsertPublishedSkillImportOrigin(ctx, tx, runID, skillID, item)
	if err != nil {
		return 0, 0, 0, err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO skill_version_origins (
  version_id, origin_id, run_item_id, source_revision,
  source_content_sha256, transformed, provenance
) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb)
ON CONFLICT (run_item_id, version_id) DO NOTHING`, versionID, originID, item.id,
		item.sourceRevision, item.sourceContentSHA256, item.artifact.Transformed,
		normalizedImportJSON(item.provenance, `{}`)); err != nil {
		return 0, 0, 0, fmt.Errorf("create imported skill version origin: %w", err)
	}
	return skillID, versionID, originID, nil
}

func insertPublishedSkillImportSkill(ctx context.Context, tx *sql.Tx, desired service.SkillImportDesiredSkill, actorID *int64) (int64, error) {
	tags, prompts, err := encodeDesiredSkillLists(desired)
	if err != nil {
		return 0, err
	}
	var skillID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO skills (
  slug, display_name, summary, description, category, tags, icon,
  example_prompts, risk_notes, source_url, source_repository,
  repository_stars_refresh_after, origin_url, catalog_source_priority,
  catalog_source_rank, status, featured, sort_order, created_by, updated_by
) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8::jsonb,$9,$10,$11,
  CASE WHEN $10='' THEN NULL ELSE NOW() END,$12,$13,$14,'draft',$15,$16,$17,$17)
RETURNING id`, desired.Slug, desired.DisplayName, desired.Summary,
		desired.Description, desired.Category, tags, desired.Icon, prompts,
		desired.RiskNotes, desired.SourceURL, desired.SourceRepository,
		desired.OriginURL, desired.CatalogSourcePriority, desired.CatalogSourceRank,
		desired.Featured, desired.SortOrder, actorID).Scan(&skillID)
	if err != nil {
		return 0, mapSkillImportWriteError(err, "create imported skill")
	}
	return skillID, nil
}

func insertPublishedSkillImportVersion(
	ctx context.Context,
	tx *sql.Tx,
	skillID int64,
	version string,
	artifact service.SkillImportPreparedArtifact,
	packageData []byte,
	actorID *int64,
) (int64, error) {
	fileManifest, err := json.Marshal(artifact.FileManifest)
	if err != nil {
		return 0, fmt.Errorf("encode imported skill file manifest: %w", err)
	}
	validation, err := json.Marshal(artifact.ValidationReport)
	if err != nil {
		return 0, fmt.Errorf("encode imported skill validation report: %w", err)
	}
	var versionID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO skill_versions (
  skill_id, version, changelog, manifest_name, manifest_description,
  skill_md, package_data, sha256, byte_size, unpacked_size, file_count,
  file_manifest, validation_report, created_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13::jsonb,$14)
RETURNING id`, skillID, version, artifact.Changelog, artifact.ManifestName,
		artifact.ManifestDescription, artifact.SkillMD, packageData,
		artifact.PackageSHA256, artifact.ByteSize, artifact.UnpackedSize,
		artifact.FileCount, string(fileManifest), string(validation), actorID).Scan(&versionID)
	if err != nil {
		return 0, mapSkillImportWriteError(err, "create imported skill version")
	}
	return versionID, nil
}

func upsertPublishedSkillImportOrigin(
	ctx context.Context,
	tx *sql.Tx,
	runID, skillID int64,
	item lockedPublishItem,
) (int64, error) {
	var originID int64
	err := tx.QueryRowContext(ctx, `
INSERT INTO skill_origins (
  source_id, namespace, external_id, skill_id, market_slug, origin_url,
  source_revision, source_content_sha256, first_seen_run_id, last_seen_run_id,
  last_rank, active
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9,$10,TRUE)
ON CONFLICT (source_id, namespace, external_id) DO UPDATE SET
  origin_url=EXCLUDED.origin_url,
  source_revision=EXCLUDED.source_revision,
  source_content_sha256=EXCLUDED.source_content_sha256,
  last_seen_run_id=EXCLUDED.last_seen_run_id, last_seen_at=NOW(),
  last_rank=EXCLUDED.last_rank, active=TRUE, updated_at=NOW()
WHERE skill_origins.skill_id=EXCLUDED.skill_id
  AND skill_origins.market_slug=EXCLUDED.market_slug
RETURNING id`, item.sourceID, item.namespace, item.externalID, skillID,
		item.marketSlug, item.originURL, item.sourceRevision,
		item.sourceContentSHA256, runID, item.rank).Scan(&originID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrSkillImportConflict
	}
	if err != nil {
		return 0, mapSkillImportWriteError(err, "upsert imported skill origin")
	}
	return originID, nil
}

func (r *skillImportRepository) AppendEvent(ctx context.Context, event *service.SkillImportEvent) error {
	payload := normalizedImportJSON(event.Payload, `{}`)
	err := r.db.QueryRowContext(ctx, `
INSERT INTO skill_import_events (run_id, run_item_id, level, event_type, message, payload)
VALUES ($1,$2,$3,$4,$5,$6::jsonb)
RETURNING id, created_at`, event.RunID, event.RunItemID, event.Level,
		event.EventType, event.Message, payload).Scan(&event.ID, &event.CreatedAt)
	if err != nil {
		return mapSkillImportWriteError(err, "append skill import event")
	}
	event.Payload = json.RawMessage(payload)
	return nil
}

func (r *skillImportRepository) ListEvents(ctx context.Context, filter service.SkillImportListFilter) ([]service.SkillImportEvent, int64, error) {
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 5)
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions, fmt.Sprintf(`(e.event_type ILIKE $%d OR e.message ILIKE $%d)`, len(args), len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		conditions = append(conditions, fmt.Sprintf(`e.level=$%d`, len(args)))
	}
	if filter.RunID != nil {
		args = append(args, *filter.RunID)
		conditions = append(conditions, fmt.Sprintf(`e.run_id=$%d`, len(args)))
	}
	where := importWhere(conditions)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM skill_import_events e`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count skill import events: %w", err)
	}
	limit, offset := importPage(filter)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, `
SELECT e.id, e.run_id, e.run_item_id, e.level, e.event_type, e.message,
  e.payload, e.created_at
FROM skill_import_events e`+where+fmt.Sprintf(` ORDER BY e.id DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list skill import events: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.SkillImportEvent, 0)
	for rows.Next() {
		var event service.SkillImportEvent
		var runItemID sql.NullInt64
		var payload []byte
		if err := rows.Scan(&event.ID, &event.RunID, &runItemID, &event.Level,
			&event.EventType, &event.Message, &payload, &event.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan skill import event: %w", err)
		}
		event.RunItemID = nullInt64Pointer(runItemID)
		event.Payload = cloneImportJSON(payload, `{}`)
		items = append(items, event)
	}
	return items, total, rows.Err()
}

func (r *skillImportRepository) DeleteTerminalRunEventsBefore(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
DELETE FROM skill_import_events e
USING skill_import_runs r
WHERE e.run_id=r.id AND e.created_at < $1
  AND r.status IN ('succeeded','partial_succeeded','failed','cancelled')`, before)
	if err != nil {
		return 0, fmt.Errorf("delete expired skill import events: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("inspect expired skill import event cleanup: %w", err)
	}
	return affected, nil
}

const skillOriginColumns = `
o.id, o.source_id, o.namespace, o.external_id, o.skill_id, o.market_slug,
o.origin_url, o.source_revision, o.source_content_sha256, o.first_seen_run_id,
o.last_seen_run_id, o.first_seen_at, o.last_seen_at, o.last_rank, o.active,
o.created_at, o.updated_at`

func scanSkillOrigin(scanner skillImportScanner) (*service.SkillOrigin, error) {
	var origin service.SkillOrigin
	var rank sql.NullInt64
	if err := scanner.Scan(&origin.ID, &origin.StableKey.SourceID,
		&origin.StableKey.Namespace, &origin.StableKey.ExternalID, &origin.SkillID,
		&origin.MarketSlug, &origin.OriginURL, &origin.SourceRevision,
		&origin.SourceContentSHA256, &origin.FirstSeenRunID, &origin.LastSeenRunID,
		&origin.FirstSeenAt, &origin.LastSeenAt, &rank, &origin.Active,
		&origin.CreatedAt, &origin.UpdatedAt); err != nil {
		return nil, err
	}
	origin.LastRank = nullIntPointer(rank)
	return &origin, nil
}

func (r *skillImportRepository) GetOriginByStableKey(ctx context.Context, key service.SkillImportStableKey) (*service.SkillOrigin, error) {
	origin, err := scanSkillOrigin(r.db.QueryRowContext(ctx, `SELECT `+skillOriginColumns+`
FROM skill_origins o WHERE o.source_id=$1 AND o.namespace=$2 AND o.external_id=$3`,
		key.SourceID, key.Namespace, key.ExternalID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSkillImportOriginNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get skill origin: %w", err)
	}
	return origin, nil
}

func (r *skillImportRepository) UpsertOrigin(ctx context.Context, origin *service.SkillOrigin) error {
	err := r.db.QueryRowContext(ctx, `
INSERT INTO skill_origins (
  source_id, namespace, external_id, skill_id, market_slug, origin_url,
  source_revision, source_content_sha256, first_seen_run_id, last_seen_run_id,
  first_seen_at, last_seen_at, last_rank, active
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
  COALESCE(NULLIF($11,'0001-01-01 00:00:00+00'::timestamptz),NOW()),
  COALESCE(NULLIF($12,'0001-01-01 00:00:00+00'::timestamptz),NOW()),$13,$14)
ON CONFLICT (source_id, namespace, external_id) DO UPDATE SET
  market_slug=EXCLUDED.market_slug, origin_url=EXCLUDED.origin_url,
  source_revision=EXCLUDED.source_revision,
  source_content_sha256=EXCLUDED.source_content_sha256,
  last_seen_run_id=EXCLUDED.last_seen_run_id, last_seen_at=EXCLUDED.last_seen_at,
  last_rank=EXCLUDED.last_rank, active=EXCLUDED.active, updated_at=NOW()
WHERE skill_origins.skill_id=EXCLUDED.skill_id
RETURNING id, first_seen_at, last_seen_at, created_at, updated_at`,
		origin.StableKey.SourceID, origin.StableKey.Namespace,
		origin.StableKey.ExternalID, origin.SkillID, origin.MarketSlug,
		origin.OriginURL, origin.SourceRevision, origin.SourceContentSHA256,
		origin.FirstSeenRunID, origin.LastSeenRunID, origin.FirstSeenAt,
		origin.LastSeenAt, origin.LastRank, origin.Active,
	).Scan(&origin.ID, &origin.FirstSeenAt, &origin.LastSeenAt, &origin.CreatedAt, &origin.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrSkillImportConflict
	}
	if err != nil {
		return mapSkillImportWriteError(err, "upsert skill origin")
	}
	return nil
}

func (r *skillImportRepository) CreateVersionOrigin(ctx context.Context, origin *service.SkillVersionOrigin) error {
	provenance := normalizedImportJSON(origin.Provenance, `{}`)
	err := r.db.QueryRowContext(ctx, `
INSERT INTO skill_version_origins (
  version_id, origin_id, run_item_id, source_revision,
  source_content_sha256, transformed, provenance
) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb)
RETURNING id, created_at`, origin.VersionID, origin.OriginID, origin.RunItemID,
		origin.SourceRevision, origin.SourceContentSHA256, origin.Transformed,
		provenance).Scan(&origin.ID, &origin.CreatedAt)
	if err != nil {
		return mapSkillImportWriteError(err, "create skill version origin")
	}
	origin.Provenance = json.RawMessage(provenance)
	return nil
}

func (r *skillImportRepository) ListBootstrapCandidates(ctx context.Context, cursor service.SkillImportBootstrapCursor) ([]service.SkillImportBootstrapCandidate, error) {
	limit := cursor.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT s.id, s.slug, s.sort_order, s.origin_url, s.source_url, s.source_repository,
  v.id, v.version, v.sha256, v.package_data
FROM skills s
JOIN LATERAL (
  SELECT id, version, sha256, package_data FROM skill_versions
  WHERE skill_id=s.id
  ORDER BY (id=s.current_version_id) DESC, created_at DESC, id DESC LIMIT 1
) v ON TRUE
WHERE s.id > $1
ORDER BY s.id ASC LIMIT $2`, cursor.AfterSkillID, limit)
	if err != nil {
		return nil, fmt.Errorf("list skill import bootstrap candidates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.SkillImportBootstrapCandidate, 0)
	for rows.Next() {
		var item service.SkillImportBootstrapCandidate
		if err := rows.Scan(&item.SkillID, &item.Slug, &item.SortOrder, &item.OriginURL,
			&item.SourceURL, &item.SourceRepository, &item.VersionID,
			&item.Version, &item.SHA256, &item.PackageData); err != nil {
			return nil, fmt.Errorf("scan skill import bootstrap candidate: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// BootstrapOrigin atomically attaches explicit provenance evidence to an
// already-existing Skill/version. It never adopts a slug or stable key that is
// bound to another Skill.
func (r *skillImportRepository) BootstrapOrigin(ctx context.Context, input service.SkillImportBootstrapOriginInput) (*service.SkillImportBootstrapOriginResult, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin bootstrap skill origin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var runSourceID int64
	var triggerType, runStatus string
	var cancelRequestedAt sql.NullTime
	var runLeaseOwner sql.NullString
	var runLeaseExpiresAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `
SELECT source_id, trigger_type, status, cancel_requested_at,
  lease_owner, lease_expires_at
FROM skill_import_runs
WHERE id=$1 FOR UPDATE`, input.RunID).Scan(
		&runSourceID, &triggerType, &runStatus, &cancelRequestedAt,
		&runLeaseOwner, &runLeaseExpiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrSkillImportRunNotFound
		}
		return nil, fmt.Errorf("lock bootstrap skill import run: %w", err)
	}
	if runSourceID != input.StableKey.SourceID || triggerType != service.SkillImportTriggerBootstrap ||
		isSkillImportRunTerminal(runStatus) || cancelRequestedAt.Valid ||
		!runLeaseOwner.Valid || runLeaseOwner.String != input.RunWorkerID ||
		!runLeaseExpiresAt.Valid || !runLeaseExpiresAt.Time.After(time.Now().UTC()) {
		return nil, service.ErrSkillImportInvalidState
	}
	var slug string
	var sortOrder int
	var versionSkillID int64
	if err := tx.QueryRowContext(ctx, `
SELECT s.slug, s.sort_order, v.skill_id FROM skills s
JOIN skill_versions v ON v.id=$2
WHERE s.id=$1 FOR UPDATE OF s, v`, input.SkillID, input.VersionID).Scan(&slug, &sortOrder, &versionSkillID); err != nil {
		return nil, service.ErrSkillImportConflict
	}
	if slug != input.MarketSlug || versionSkillID != input.SkillID {
		return nil, service.ErrSkillImportConflict
	}
	var rank *int
	if sortOrder > 0 {
		rank = &sortOrder
	}

	provenance := normalizedImportJSON(input.Provenance, `{}`)
	result := &service.SkillImportBootstrapOriginResult{}
	err = tx.QueryRowContext(ctx, `
INSERT INTO skill_import_run_items (
  run_id, source_id, namespace, external_id, rank, market_slug, status, stage_action,
  upstream_name, origin_url, source_revision, source_content_sha256,
  package_sha256, staged_artifact, desired_skill, source_payload, validation_report,
  provenance, license_unverified, excluded_files, warnings, skill_id,
  version_id, completed_at
)
SELECT $1,$2,$3,$4,$12,$5::text,'unchanged','unchanged',$5::text,$6::text,$7,$8,v.sha256,
  jsonb_build_object(
    'bootstrap',TRUE,
    'manifest_name',$5::text,
    'package_sha256',v.sha256,
    'validation_report',v.validation_report
  ),
  jsonb_build_object('slug',$5::text,'origin_url',$6::text),
  '{"bootstrap":true}'::jsonb,v.validation_report,$9::jsonb,FALSE,
  '[]'::jsonb,'[]'::jsonb,$10,$11,NOW()
FROM skill_versions v WHERE v.id=$11 AND v.skill_id=$10
ON CONFLICT (run_id, source_id, namespace, external_id) DO UPDATE SET
  updated_at=NOW()
WHERE skill_import_run_items.skill_id=EXCLUDED.skill_id
  AND skill_import_run_items.version_id=EXCLUDED.version_id
RETURNING id`, input.RunID, input.StableKey.SourceID, input.StableKey.Namespace,
		input.StableKey.ExternalID, input.MarketSlug, input.OriginURL,
		input.SourceRevision, input.SourceContentSHA256, provenance,
		input.SkillID, input.VersionID, rank).Scan(&result.RunItemID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSkillImportConflict
	}
	if err != nil {
		return nil, mapSkillImportWriteError(err, "create bootstrap skill import item")
	}

	err = tx.QueryRowContext(ctx, `
INSERT INTO skill_origins (
  source_id, namespace, external_id, skill_id, market_slug, origin_url,
  source_revision, source_content_sha256, first_seen_run_id, last_seen_run_id,
  last_rank, active
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9,$10,TRUE)
ON CONFLICT (source_id, namespace, external_id) DO UPDATE SET
  source_revision=EXCLUDED.source_revision,
  source_content_sha256=EXCLUDED.source_content_sha256,
  last_seen_run_id=EXCLUDED.last_seen_run_id, last_seen_at=NOW(),
  last_rank=EXCLUDED.last_rank, active=TRUE, updated_at=NOW()
WHERE skill_origins.skill_id=EXCLUDED.skill_id
  AND skill_origins.market_slug=EXCLUDED.market_slug
RETURNING id`, input.StableKey.SourceID, input.StableKey.Namespace,
		input.StableKey.ExternalID, input.SkillID, input.MarketSlug,
		input.OriginURL, input.SourceRevision, input.SourceContentSHA256,
		input.RunID, rank).Scan(&result.OriginID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrSkillImportConflict
	}
	if err != nil {
		return nil, mapSkillImportWriteError(err, "create bootstrap skill origin")
	}

	err = tx.QueryRowContext(ctx, `
INSERT INTO skill_version_origins (
  version_id, origin_id, run_item_id, source_revision,
  source_content_sha256, transformed, provenance
) VALUES ($1,$2,$3,$4,$5,FALSE,$6::jsonb)
ON CONFLICT (run_item_id, version_id) DO NOTHING
RETURNING id`, input.VersionID, result.OriginID, result.RunItemID,
		input.SourceRevision, input.SourceContentSHA256, provenance).Scan(&result.VersionOriginID)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `
SELECT id FROM skill_version_origins
WHERE run_item_id=$1 AND version_id=$2 AND origin_id=$3`, result.RunItemID,
			input.VersionID, result.OriginID).Scan(&result.VersionOriginID)
	}
	if err != nil {
		return nil, mapSkillImportWriteError(err, "create bootstrap skill version origin")
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE skills s SET
  origin_url=CASE WHEN s.origin_url='' THEN $2 ELSE s.origin_url END,
  catalog_source_priority=src.catalog_priority,
  catalog_source_rank=$4,
  updated_at=NOW()
FROM skill_import_sources src
WHERE s.id=$1 AND src.id=$3`, input.SkillID, input.OriginURL,
		input.StableKey.SourceID, rank); err != nil {
		return nil, fmt.Errorf("update bootstrapped skill source fields: %w", err)
	}
	if _, err := refreshSkillImportCountsTx(ctx, tx, input.RunID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit bootstrap skill origin: %w", err)
	}
	return result, nil
}

func refreshSkillImportCountsTx(ctx context.Context, db skillImportDBTX, runID int64) (service.SkillImportRunCounts, error) {
	var counts service.SkillImportRunCounts
	err := db.QueryRowContext(ctx, `
WITH item_counts AS (
  SELECT
    COUNT(*)::int AS discovered,
    COUNT(*) FILTER (WHERE i.status IN ('ready','unchanged','published'))::int AS prepared,
    COUNT(*) FILTER (
	  WHERE i.stage_action='create' AND i.status IN ('ready','published')
    )::int AS created,
    COUNT(*) FILTER (
	  WHERE i.stage_action='new_version' AND i.status IN ('ready','published')
    )::int AS updated,
	COUNT(*) FILTER (
	  WHERE i.stage_action='unchanged' AND i.status='unchanged'
	)::int AS unchanged,
    COUNT(*) FILTER (WHERE i.status='skipped')::int AS skipped,
    COUNT(*) FILTER (WHERE i.status='blocked')::int AS blocked,
    COUNT(*) FILTER (WHERE i.status='failed')::int AS failed,
	COUNT(*) FILTER (WHERE i.status='published')::int AS published
  FROM skill_import_run_items i
  WHERE i.run_id=$1
), updated AS (
  UPDATE skill_import_runs r SET
    discovered_count=c.discovered, prepared_count=c.prepared,
    created_count=c.created, updated_count=c.updated,
    unchanged_count=c.unchanged, skipped_count=c.skipped,
    blocked_count=c.blocked, failed_count=c.failed, published_count=c.published,
    updated_at=NOW()
  FROM item_counts c WHERE r.id=$1
  RETURNING r.requested_count, r.discovered_count, r.prepared_count,
    r.created_count, r.updated_count, r.unchanged_count, r.skipped_count,
    r.blocked_count, r.failed_count, r.published_count
)
SELECT * FROM updated`, runID).Scan(&counts.Requested, &counts.Discovered,
		&counts.Prepared, &counts.Created, &counts.Updated, &counts.Unchanged,
		&counts.Skipped, &counts.Blocked, &counts.Failed, &counts.Published)
	if errors.Is(err, sql.ErrNoRows) {
		return counts, service.ErrSkillImportRunNotFound
	}
	if err != nil {
		return counts, fmt.Errorf("refresh skill import run counts: %w", err)
	}
	return counts, nil
}

func updatePublishedSkillMetadata(ctx context.Context, tx *sql.Tx, skillID int64, desired service.SkillImportDesiredSkill, actorID *int64) error {
	tags, prompts, err := encodeDesiredSkillLists(desired)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
UPDATE skills SET display_name=$2, summary=$3, description=$4, category=$5,
  tags=$6::jsonb, icon=$7, example_prompts=$8::jsonb, risk_notes=$9,
  source_url=$10, source_repository=$11,
  repository_stars=CASE WHEN source_url=$10 THEN repository_stars ELSE NULL END,
  repository_stars_fetched_at=CASE WHEN source_url=$10 THEN repository_stars_fetched_at ELSE NULL END,
  repository_stars_refresh_after=CASE
    WHEN source_url=$10 THEN repository_stars_refresh_after
    WHEN $10='' THEN NULL ELSE NOW() END,
  origin_url=$12, featured=$13, sort_order=$14,
  catalog_source_priority=$15, catalog_source_rank=$16,
  updated_by=$17, updated_at=NOW()
WHERE id=$1 AND slug=$18`, skillID, desired.DisplayName, desired.Summary,
		desired.Description, desired.Category, tags, desired.Icon, prompts,
		desired.RiskNotes, desired.SourceURL, desired.SourceRepository,
		desired.OriginURL, desired.Featured, desired.SortOrder,
		desired.CatalogSourcePriority, desired.CatalogSourceRank, actorID,
		desired.Slug)
	if err != nil {
		return mapSkillImportWriteError(err, "refresh imported skill metadata")
	}
	return requireSkillImportAffected(result, service.ErrSkillImportPublishInvalid)
}

func encodeDesiredSkillLists(desired service.SkillImportDesiredSkill) (string, string, error) {
	tags, err := json.Marshal(nonNilStrings(desired.Tags))
	if err != nil {
		return "", "", fmt.Errorf("encode desired skill tags: %w", err)
	}
	prompts, err := json.Marshal(nonNilStrings(desired.ExamplePrompts))
	if err != nil {
		return "", "", fmt.Errorf("encode desired skill prompts: %w", err)
	}
	return string(tags), string(prompts), nil
}

func nextSkillImportVersion(ctx context.Context, tx *sql.Tx, skillID int64) (string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT version FROM skill_versions WHERE skill_id=$1 ORDER BY id`, skillID)
	if err != nil {
		return "", fmt.Errorf("list imported skill versions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	highest := ""
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return "", fmt.Errorf("scan imported skill version: %w", err)
		}
		canonical := semver.Canonical("v" + version)
		if canonical != "" && (highest == "" || semver.Compare(canonical, highest) > 0) {
			highest = canonical
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate imported skill versions: %w", err)
	}
	if highest == "" {
		return "1.0.0", nil
	}
	base := strings.TrimPrefix(highest, "v")
	if index := strings.IndexAny(base, "-+"); index >= 0 {
		base = base[:index]
	}
	parts := strings.Split(base, ".")
	if len(parts) != 3 {
		return "", service.ErrSkillImportConflict
	}
	patch, err := strconv.ParseUint(parts[2], 10, 63)
	if err != nil || patch == uint64(^uint64(0)>>1) {
		return "", service.ErrSkillImportConflict
	}
	return parts[0] + "." + parts[1] + "." + strconv.FormatUint(patch+1, 10), nil
}

func resolveSkillImportMarketSlug(ctx context.Context, tx *sql.Tx, suggested string, key service.SkillImportStableKey, currentItemID int64) (string, error) {
	base := strings.Trim(strings.ToLower(strings.TrimSpace(suggested)), "-")
	if base == "" {
		base = "skill"
	}
	available := func(slug string) (bool, error) {
		var exists bool
		err := tx.QueryRowContext(ctx, `
SELECT EXISTS(SELECT 1 FROM skills WHERE slug=$1)
    OR EXISTS(SELECT 1 FROM skill_origins WHERE market_slug=$1)
    OR EXISTS(
      SELECT 1
      FROM skill_import_run_items i
      JOIN skill_import_runs r ON r.id=i.run_id
      WHERE i.market_slug=$1 AND i.id<>$2
        AND i.stage_action IN ('create','new_version')
        AND i.status IN ('ready','unchanged')
        AND r.status IN (
          'queued','discovering','preparing','waiting_retry','ready',
          'awaiting_review','publishing'
        )
    )`, slug, currentItemID).Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("check imported skill slug allocation: %w", err)
		}
		return !exists, nil
	}
	if len(base) <= 64 {
		free, err := available(base)
		if err != nil {
			return "", err
		}
		if free {
			return base, nil
		}
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s\x00%s", key.SourceID, key.Namespace, key.ExternalID)))
	hexDigest := fmt.Sprintf("%x", digest[:])
	for _, suffixLength := range []int{10, 16, 24, 32, 48, 56} {
		suffix := hexDigest[:suffixLength]
		prefixLength := 64 - 1 - len(suffix)
		prefix := base
		if len(prefix) > prefixLength {
			prefix = strings.TrimRight(prefix[:prefixLength], "-")
		}
		if prefix == "" {
			prefix = "skill"
		}
		candidate := prefix + "-" + suffix
		free, err := available(candidate)
		if err != nil {
			return "", err
		}
		if free {
			return candidate, nil
		}
	}
	return "", service.ErrSkillImportConflict
}

func importMetadataPolicy(raw []byte) string {
	var config struct {
		MetadataPolicy string `json:"metadata_policy"`
	}
	if json.Unmarshal(raw, &config) == nil && config.MetadataPolicy == service.SkillImportMetadataPolicyCreateOnly {
		return service.SkillImportMetadataPolicyCreateOnly
	}
	return service.SkillImportMetadataPolicyRefresh
}

func buildSkillImportFilter(filter service.SkillImportListFilter, searchColumn, statusColumn string) (string, []any) {
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		conditions = append(conditions, fmt.Sprintf(`%s ILIKE $%d`, searchColumn, len(args)))
	}
	if filter.Status != "" {
		enabled := filter.Status == "enabled" || filter.Status == "true"
		args = append(args, enabled)
		conditions = append(conditions, fmt.Sprintf(`%s=$%d`, statusColumn, len(args)))
	}
	return importWhere(conditions), args
}

func importWhere(conditions []string) string {
	if len(conditions) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(conditions, " AND ")
}

func importPage(filter service.SkillImportListFilter) (int, int) {
	page, size := filter.Page, filter.PageSize
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return size, (page - 1) * size
}

func normalizedImportJSON(raw json.RawMessage, fallback string) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return fallback
	}
	return trimmed
}

func cloneImportJSON(raw []byte, fallback string) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(fallback)
	}
	return append(json.RawMessage(nil), raw...)
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilIssues(values []service.SkillValidationIssue) []service.SkillValidationIssue {
	if values == nil {
		return []service.SkillValidationIssue{}
	}
	return values
}

func nullInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}

func nullIntPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int64)
	return &result
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func uniqueImportIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func isSkillImportRunTerminal(status string) bool {
	switch status {
	case service.SkillImportRunStatusSucceeded, service.SkillImportRunStatusPartialSucceeded,
		service.SkillImportRunStatusFailed, service.SkillImportRunStatusCancelled:
		return true
	default:
		return false
	}
}

func requireSkillImportAffected(result sql.Result, missing error) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect skill import write: %w", err)
	}
	if affected == 0 {
		return missing
	}
	return nil
}

func requireSkillImportLease(result sql.Result) error {
	return requireSkillImportAffected(result, service.ErrSkillImportLeaseLost)
}

func mapSkillImportWriteError(err error, action string) error {
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			return service.ErrSkillImportConflict
		case "23503", "23514", "22P02":
			return service.ErrSkillImportInvalidState
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}
