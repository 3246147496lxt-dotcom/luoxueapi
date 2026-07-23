package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/opsruntime"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type schedulerOutboxRepository struct {
	db *sql.DB
}

type schedulerOutboxCleanupLease struct {
	mu   sync.Mutex
	conn *sql.Conn
}

type schedulerOutboxConsumerLease struct {
	mu   sync.Mutex
	conn *sql.Conn
}

const (
	schedulerOutboxDefaultCleanSize = 5000
	schedulerOutboxMaxPollSize      = 200
	schedulerOutboxConsumerLockName = "scheduler_outbox_consumer"
	schedulerOutboxCleanupLockName  = "scheduler_outbox_cleanup"
	schedulerOutboxUnlockTimeout    = 2 * time.Second
)

func NewSchedulerOutboxRepository(db *sql.DB) service.SchedulerOutboxRepository {
	return &schedulerOutboxRepository{db: db}
}

func (r *schedulerOutboxRepository) TryAcquireConsumerLease(ctx context.Context) (service.SchedulerOutboxConsumerLease, bool, error) {
	if r == nil || r.db == nil {
		return nil, false, errors.New("scheduler outbox database is not configured")
	}
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("acquire scheduler outbox consumer connection: %w", err)
	}

	var acquired bool
	if err := conn.QueryRowContext(
		ctx,
		"SELECT pg_try_advisory_lock(hashtext($1))",
		schedulerOutboxConsumerLockName,
	).Scan(&acquired); err != nil {
		// The server may have acquired the session lock before the client observed
		// the query failure. Do not return an uncertain session to the pool.
		discardSQLConn(conn)
		_ = conn.Close()
		return nil, false, fmt.Errorf("acquire scheduler outbox consumer lease: %w", err)
	}
	if !acquired {
		if err := conn.Close(); err != nil {
			return nil, false, fmt.Errorf("release unclaimed scheduler outbox consumer connection: %w", err)
		}
		return nil, false, nil
	}
	return &schedulerOutboxConsumerLease{conn: conn}, true, nil
}

func (l *schedulerOutboxConsumerLease) Release() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	conn := l.conn
	l.conn = nil
	l.mu.Unlock()
	if conn == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), schedulerOutboxUnlockTimeout)
	defer cancel()
	var unlocked bool
	unlockErr := conn.QueryRowContext(
		ctx,
		"SELECT pg_advisory_unlock(hashtext($1))",
		schedulerOutboxConsumerLockName,
	).Scan(&unlocked)
	if unlockErr == nil && !unlocked {
		unlockErr = errors.New("current session did not hold the scheduler outbox consumer lease")
	}
	if unlockErr != nil {
		// An uncertain unlock must never put the physical session back in the pool.
		discardSQLConn(conn)
		closeErr := conn.Close()
		return errors.Join(fmt.Errorf("release scheduler outbox consumer lease: %w", unlockErr), closeErr)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("release scheduler outbox consumer connection: %w", err)
	}
	return nil
}

func (r *schedulerOutboxRepository) ListAfterAndReleaseDedup(ctx context.Context, afterID int64, limit int) ([]service.SchedulerOutboxEvent, error) {
	return listSchedulerOutboxAfterAndReleaseDedup(ctx, r.db, afterID, limit)
}

type schedulerOutboxTxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func (l *schedulerOutboxConsumerLease) ListAfterAndReleaseDedup(ctx context.Context, afterID int64, limit int) ([]service.SchedulerOutboxEvent, error) {
	if l == nil {
		return nil, errors.New("scheduler outbox consumer lease is not configured")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.conn == nil {
		return nil, errors.New("scheduler outbox consumer lease is already released")
	}
	return listSchedulerOutboxAfterAndReleaseDedup(ctx, l.conn, afterID, limit)
}

func listSchedulerOutboxAfterAndReleaseDedup(ctx context.Context, beginner schedulerOutboxTxBeginner, afterID int64, limit int) ([]service.SchedulerOutboxEvent, error) {
	if beginner == nil {
		return nil, errors.New("scheduler outbox database is not configured")
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > schedulerOutboxMaxPollSize {
		limit = schedulerOutboxMaxPollSize
	}

	// BIGSERIAL values are allocated before commit, so a later id can otherwise
	// become visible first and advance the consumer watermark past an older,
	// still-uncommitted event. The short table fence waits for all current
	// producers and prevents new INSERTs only while this bounded batch is read.
	tx, err := beginner.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin scheduler outbox poll: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, "SET LOCAL lock_timeout = '1s'"); err != nil {
		return nil, fmt.Errorf("set scheduler outbox poll lock timeout: %w", err)
	}
	fenceStartedAt := time.Now()
	if _, err := tx.ExecContext(ctx, "LOCK TABLE scheduler_outbox IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		opsruntime.ObserveOutboxFence(time.Since(fenceStartedAt), false, isOutboxFenceTimeout(ctx, err))
		return nil, fmt.Errorf("acquire scheduler outbox commit fence: %w", err)
	}
	opsruntime.ObserveOutboxFence(time.Since(fenceStartedAt), true, false)

	rows, err := tx.QueryContext(ctx, `
		WITH selected AS MATERIALIZED (
			SELECT id, event_type, account_id, group_id, payload, created_at
			FROM scheduler_outbox
			WHERE id > $1
			ORDER BY id ASC
			LIMIT $2
			FOR UPDATE
		), released AS (
			UPDATE scheduler_outbox AS o
			SET dedup_key = NULL
			FROM selected AS s
			WHERE o.id = s.id
				AND o.dedup_key IS NOT NULL
			RETURNING o.id
		)
		SELECT s.id, s.event_type, s.account_id, s.group_id, s.payload, s.created_at
		FROM selected AS s
		CROSS JOIN (SELECT COUNT(*) FROM released) AS release_barrier
		ORDER BY s.id ASC
	`, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("list scheduler outbox events: %w", err)
	}

	events := make([]service.SchedulerOutboxEvent, 0, limit)
	var readErr error
	for rows.Next() {
		var (
			payloadRaw []byte
			accountID  sql.NullInt64
			groupID    sql.NullInt64
			event      service.SchedulerOutboxEvent
		)
		if err := rows.Scan(&event.ID, &event.EventType, &accountID, &groupID, &payloadRaw, &event.CreatedAt); err != nil {
			readErr = err
			break
		}
		if accountID.Valid {
			v := accountID.Int64
			event.AccountID = &v
		}
		if groupID.Valid {
			v := groupID.Int64
			event.GroupID = &v
		}
		if len(payloadRaw) > 0 {
			var payload map[string]any
			if err := json.Unmarshal(payloadRaw, &payload); err != nil {
				readErr = err
				break
			}
			event.Payload = payload
		}
		events = append(events, event)
	}
	if readErr == nil {
		readErr = rows.Err()
	}
	if closeErr := rows.Close(); closeErr != nil {
		readErr = errors.Join(readErr, closeErr)
	}
	if readErr != nil {
		return nil, fmt.Errorf("read scheduler outbox events: %w", readErr)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit scheduler outbox poll: %w", err)
	}
	opsruntime.ObserveOutboxBatch(len(events))
	return events, nil
}

func isOutboxFenceTimeout(ctx context.Context, err error) bool {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "lock timeout") || strings.Contains(message, "55p03")
}

func (r *schedulerOutboxRepository) FirstCreatedAtAfter(ctx context.Context, afterID int64) (time.Time, bool, error) {
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT created_at
		FROM scheduler_outbox
		WHERE id > $1
		ORDER BY id ASC
		LIMIT 1
	`, afterID).Scan(&createdAt)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return createdAt, true, nil
}

func (r *schedulerOutboxRepository) MaxID(ctx context.Context) (int64, error) {
	var maxID int64
	if err := r.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM scheduler_outbox").Scan(&maxID); err != nil {
		return 0, err
	}
	return maxID, nil
}

func (r *schedulerOutboxRepository) DeleteConsumedUpTo(ctx context.Context, watermark int64, limit int) (int64, error) {
	return deleteSchedulerOutboxConsumedUpTo(ctx, r.db, watermark, limit)
}

type schedulerOutboxExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (l *schedulerOutboxCleanupLease) DeleteConsumedUpTo(ctx context.Context, watermark int64, limit int) (int64, error) {
	if l == nil {
		return 0, errors.New("scheduler outbox cleanup lease is not configured")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.conn == nil {
		return 0, errors.New("scheduler outbox cleanup lease is already released")
	}
	return deleteSchedulerOutboxConsumedUpTo(ctx, l.conn, watermark, limit)
}

func deleteSchedulerOutboxConsumedUpTo(ctx context.Context, execer schedulerOutboxExecer, watermark int64, limit int) (int64, error) {
	if watermark <= 0 {
		return 0, nil
	}
	if limit <= 0 {
		limit = schedulerOutboxDefaultCleanSize
	}
	// 提交顺序由 ListAfterAndReleaseDedup 的表级 fence 保证。这里额外保留 10 秒
	// 仅作为清理宽限，避免刚推进 watermark 就立即删除事件，也让滚动升级中的
	// worker 有时间结束当前批次；它本身不承担消费顺序保证。
	result, err := execer.ExecContext(ctx, `
		WITH doomed AS (
			SELECT id
			FROM scheduler_outbox
			WHERE id <= $1
				AND created_at < NOW() - INTERVAL '10 seconds'
			ORDER BY id ASC
			LIMIT $2
		)
		DELETE FROM scheduler_outbox o
		USING doomed d
		WHERE o.id = d.id
	`, watermark, limit)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *schedulerOutboxRepository) TryAcquireCleanupLock(ctx context.Context) (service.SchedulerOutboxCleanupLease, bool, error) {
	if r == nil || r.db == nil {
		return nil, false, errors.New("scheduler outbox database is not configured")
	}
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("acquire scheduler outbox cleanup connection: %w", err)
	}

	var acquired bool
	if err := conn.QueryRowContext(
		ctx,
		"SELECT pg_try_advisory_lock(hashtext($1))",
		schedulerOutboxCleanupLockName,
	).Scan(&acquired); err != nil {
		// As with the consumer lease, a failed response cannot prove that the
		// server did not acquire the session-scoped lock. Discard the session.
		discardSQLConn(conn)
		_ = conn.Close()
		return nil, false, fmt.Errorf("acquire scheduler outbox cleanup lease: %w", err)
	}
	if !acquired {
		if err := conn.Close(); err != nil {
			return nil, false, fmt.Errorf("release unclaimed scheduler outbox cleanup connection: %w", err)
		}
		return nil, false, nil
	}
	return &schedulerOutboxCleanupLease{conn: conn}, true, nil
}

func (l *schedulerOutboxCleanupLease) Release() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	conn := l.conn
	l.conn = nil
	l.mu.Unlock()
	if conn == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), schedulerOutboxUnlockTimeout)
	defer cancel()
	var unlocked bool
	unlockErr := conn.QueryRowContext(
		ctx,
		"SELECT pg_advisory_unlock(hashtext($1))",
		schedulerOutboxCleanupLockName,
	).Scan(&unlocked)
	if unlockErr == nil && !unlocked {
		unlockErr = errors.New("current session did not hold the scheduler outbox cleanup lease")
	}
	if unlockErr != nil {
		discardSQLConn(conn)
		closeErr := conn.Close()
		return errors.Join(fmt.Errorf("release scheduler outbox cleanup lease: %w", unlockErr), closeErr)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("release scheduler outbox cleanup connection: %w", err)
	}
	return nil
}

func enqueueSchedulerOutbox(ctx context.Context, exec sqlExecutor, eventType string, accountID *int64, groupID *int64, payload any) error {
	if exec == nil {
		return errors.New("scheduler outbox SQL executor is not configured")
	}
	var payloadArg any
	var payloadJSON []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		payloadArg = encoded
		payloadJSON = encoded
	}
	query := `
		INSERT INTO scheduler_outbox (event_type, account_id, group_id, payload)
		VALUES ($1, $2, $3, $4)
	`
	args := []any{eventType, accountID, groupID, payloadArg}
	if schedulerOutboxEventSupportsDedup(eventType) {
		dedupKey := schedulerOutboxDedupKey(eventType, accountID, groupID, payloadJSON)
		query = `
			INSERT INTO scheduler_outbox (event_type, account_id, group_id, payload, dedup_key)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (dedup_key) WHERE dedup_key IS NOT NULL DO NOTHING
		`
		args = append(args, dedupKey)
	}
	_, err := exec.ExecContext(ctx, query, args...)
	return err
}

func schedulerOutboxDedupKey(eventType string, accountID *int64, groupID *int64, payloadJSON []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(eventType))
	_, _ = h.Write([]byte{0})
	if accountID != nil {
		_, _ = h.Write([]byte(strconv.FormatInt(*accountID, 10)))
	}
	_, _ = h.Write([]byte{0})
	if groupID != nil {
		_, _ = h.Write([]byte(strconv.FormatInt(*groupID, 10)))
	}
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(payloadJSON)
	return fmt.Sprintf("scheduler_outbox:%s", hex.EncodeToString(h.Sum(nil)))
}

func schedulerOutboxEventSupportsDedup(eventType string) bool {
	switch eventType {
	case service.SchedulerOutboxEventAccountChanged,
		service.SchedulerOutboxEventGroupChanged,
		service.SchedulerOutboxEventFullRebuild:
		return true
	default:
		return false
	}
}
