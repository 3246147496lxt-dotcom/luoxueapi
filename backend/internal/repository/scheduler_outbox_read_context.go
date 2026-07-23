package repository

import (
	"context"
	"errors"
	"sync"

	dbent "github.com/Wei-Shaw/sub2api/ent"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

var (
	errSchedulerOutboxConsumerLeaseReleased = errors.New("scheduler outbox consumer lease is already released")
	errSchedulerOutboxLeaseReadOnly         = errors.New("scheduler outbox consumer lease context is read-only")
)

type schedulerOutboxReadContextKey struct{}

type schedulerOutboxReadDriver struct {
	lease *schedulerOutboxConsumerLease
}

type schedulerOutboxLeaseRows struct {
	entsql.ColumnScanner
	once   sync.Once
	unlock func()
}

func (r *schedulerOutboxLeaseRows) Close() error {
	err := r.ColumnScanner.Close()
	r.once.Do(r.unlock)
	return err
}

func (d *schedulerOutboxReadDriver) Exec(context.Context, string, any, any) error {
	return errSchedulerOutboxLeaseReadOnly
}

func (d *schedulerOutboxReadDriver) Query(ctx context.Context, query string, args, v any) error {
	if d == nil || d.lease == nil {
		return errSchedulerOutboxConsumerLeaseReleased
	}
	d.lease.mu.Lock()
	if d.lease.conn == nil {
		d.lease.mu.Unlock()
		return errSchedulerOutboxConsumerLeaseReleased
	}
	conn := entsql.Conn{ExecQuerier: d.lease.conn}
	if err := conn.Query(ctx, query, args, v); err != nil {
		d.lease.mu.Unlock()
		return err
	}
	rows, ok := v.(*entsql.Rows)
	if !ok || rows.ColumnScanner == nil {
		d.lease.mu.Unlock()
		return errors.New("scheduler outbox lease query returned invalid rows")
	}
	rows.ColumnScanner = &schedulerOutboxLeaseRows{
		ColumnScanner: rows.ColumnScanner,
		unlock:        d.lease.mu.Unlock,
	}
	return nil
}

func (d *schedulerOutboxReadDriver) Tx(context.Context) (dialect.Tx, error) {
	return nil, errSchedulerOutboxLeaseReadOnly
}

func (d *schedulerOutboxReadDriver) Close() error {
	return nil
}

func (d *schedulerOutboxReadDriver) Dialect() string {
	return dialect.Postgres
}

func (l *schedulerOutboxConsumerLease) BindContext(ctx context.Context) (context.Context, error) {
	if l == nil {
		return nil, errors.New("scheduler outbox consumer lease is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.conn == nil {
		return nil, errSchedulerOutboxConsumerLeaseReleased
	}
	client := dbent.NewClient(dbent.Driver(&schedulerOutboxReadDriver{lease: l}))
	return context.WithValue(ctx, schedulerOutboxReadContextKey{}, client), nil
}

func schedulerOutboxReadClientFromContext(ctx context.Context) (*dbent.Client, bool) {
	if ctx == nil {
		return nil, false
	}
	client, ok := ctx.Value(schedulerOutboxReadContextKey{}).(*dbent.Client)
	return client, ok && client != nil
}
