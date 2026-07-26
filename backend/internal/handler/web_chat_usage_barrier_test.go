package handler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestWebChatUsageBarrierWaitsForEnqueuedTaskCompletion(t *testing.T) {
	pool := newUsageRecordTestPool(t)
	h := &OpenAIGatewayHandler{usageRecordWorkerPool: pool}
	barrier := newWebChatUsageBarrier()
	parent := withWebChatUsageBarrier(
		context.WithValue(context.Background(), ctxkey.WebChat, true),
		barrier,
	)

	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	h.submitOpenAIUsageRecordTask(parent, &service.OpenAIForwardResult{}, func(context.Context) {
		calls.Add(1)
		close(started)
		<-release
	})
	<-started

	waited := make(chan struct{})
	go func() {
		barrier.SealAndWait()
		close(waited)
	}()

	select {
	case <-waited:
		t.Fatal("barrier returned before the enqueued usage task completed")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-waited:
	case <-time.After(time.Second):
		t.Fatal("barrier did not return after the usage task completed")
	}
	require.EqualValues(t, 1, calls.Load())
}

func TestWebChatUsageBarrierMandatorySyncFallbackRunsExactlyOnce(t *testing.T) {
	pool := service.NewUsageRecordWorkerPoolWithOptions(service.UsageRecordWorkerPoolOptions{
		WorkerCount:           1,
		QueueSize:             1,
		TaskTimeout:           time.Second,
		OverflowPolicy:        "drop",
		OverflowSamplePercent: 0,
		AutoScaleEnabled:      false,
	})
	t.Cleanup(pool.Stop)
	h := &OpenAIGatewayHandler{usageRecordWorkerPool: pool}

	blocked := make(chan struct{})
	release := make(chan struct{})
	pool.Submit(func(context.Context) {
		close(blocked)
		<-release
	})
	<-blocked
	pool.Submit(func(context.Context) {})

	barrier := newWebChatUsageBarrier()
	parent := withWebChatUsageBarrier(
		context.WithValue(context.Background(), ctxkey.WebChat, true),
		barrier,
	)
	var calls atomic.Int32
	h.submitOpenAIUsageRecordTask(parent, &service.OpenAIForwardResult{}, func(context.Context) {
		calls.Add(1)
	})
	barrier.SealAndWait()
	close(release)

	require.EqualValues(t, 1, calls.Load())
}

func TestWebChatUsageBarrierRejectsRegistrationAfterSeal(t *testing.T) {
	h := &OpenAIGatewayHandler{}
	barrier := newWebChatUsageBarrier()
	parent := withWebChatUsageBarrier(
		context.WithValue(context.Background(), ctxkey.WebChat, true),
		barrier,
	)
	barrier.Seal()

	var calls atomic.Int32
	h.submitOpenAIUsageRecordTask(parent, &service.OpenAIForwardResult{}, func(context.Context) {
		calls.Add(1)
	})
	barrier.Wait()

	require.Zero(t, calls.Load())
}

func TestCyberPolicyUsageProducerPreservesWebChatCorrelation(t *testing.T) {
	h := &OpenAIGatewayHandler{}
	barrier := newWebChatUsageBarrier()
	parent := context.WithValue(context.Background(), ctxkey.ClientRequestID, "client-request-123")
	parent = context.WithValue(parent, ctxkey.RequestID, "local-request-456")
	parent = context.WithValue(parent, ctxkey.WebChat, true)
	parent = withWebChatUsageBarrier(parent, barrier)

	var gotClientRequestID string
	var gotRequestID string
	var gotWebChat bool
	var calls atomic.Int32
	h.submitCyberPolicyUsageRecordTask(parent, func(ctx context.Context) {
		calls.Add(1)
		gotClientRequestID, _ = ctx.Value(ctxkey.ClientRequestID).(string)
		gotRequestID, _ = ctx.Value(ctxkey.RequestID).(string)
		gotWebChat, _ = ctx.Value(ctxkey.WebChat).(bool)
	})
	barrier.SealAndWait()

	require.EqualValues(t, 1, calls.Load())
	require.Equal(t, "client-request-123", gotClientRequestID)
	require.Equal(t, "local-request-456", gotRequestID)
	require.True(t, gotWebChat)
}

func TestWebChatUsageBarrierReturnsStructuredProducerError(t *testing.T) {
	barrier := newWebChatUsageBarrier()
	wantErr := errors.New("settlement failed")
	task, ok := barrier.RegisterResult(func(context.Context) error {
		return wantErr
	})
	require.True(t, ok)

	task(context.Background())
	result := barrier.SealAndWait()

	require.Equal(t, 1, result.ProducerCount)
	require.Len(t, result.Results, 1)
	require.ErrorIs(t, result.Results[0].Err, wantErr)
	require.ErrorIs(t, result.Err(), wantErr)
}

func TestRunWebChatUsageRecordTaskRetriesWithDetachedBudgetContext(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	parent = context.WithValue(parent, ctxkey.ClientRequestID, "client-request-123")
	parent = context.WithValue(parent, ctxkey.WebChat, true)
	cancel()

	wantErr := errors.New("transient billing failure")
	var calls atomic.Int32
	err := runWebChatUsageRecordTask(parent, func(ctx context.Context) error {
		call := calls.Add(1)
		require.NoError(t, ctx.Err())
		require.Equal(t, "client-request-123", ctx.Value(ctxkey.ClientRequestID))
		require.Equal(t, true, ctx.Value(ctxkey.WebChat))
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), webChatUsageSettlementBudget)
		if call < 3 {
			return wantErr
		}
		return nil
	})

	require.NoError(t, err)
	require.EqualValues(t, webChatUsageSettlementMaxAttempts, calls.Load())
}

func TestRunWebChatUsageRecordTaskSettlementClosedIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	err := runWebChatUsageRecordTask(context.Background(), func(context.Context) error {
		calls.Add(1)
		return service.ErrUsageBillingSettlementClosed
	})

	require.ErrorIs(t, err, service.ErrUsageBillingSettlementClosed)
	require.EqualValues(t, 1, calls.Load())
}

func TestRunWebChatUsageRecordTaskStructuralFailureIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	err := runWebChatUsageRecordTask(context.Background(), func(context.Context) error {
		calls.Add(1)
		return service.ErrUsageBillingRequestConflict
	})

	require.ErrorIs(t, err, service.ErrUsageBillingRequestConflict)
	require.EqualValues(t, 1, calls.Load())
}

func TestSubmitOpenAIUsageRecordResultTaskReportsFailureToBarrier(t *testing.T) {
	h := &OpenAIGatewayHandler{}
	barrier := newWebChatUsageBarrier()
	parent := context.WithValue(context.Background(), ctxkey.WebChat, true)
	parent = withWebChatUsageBarrier(parent, barrier)

	var calls atomic.Int32
	h.submitOpenAIUsageRecordResultTask(
		parent,
		&service.OpenAIForwardResult{},
		func(context.Context) error {
			calls.Add(1)
			return service.ErrUsageBillingSettlementClosed
		},
	)
	result := barrier.SealAndWait()

	require.EqualValues(t, 1, calls.Load())
	require.Equal(t, 1, result.ProducerCount)
	require.ErrorIs(t, result.Err(), service.ErrUsageBillingSettlementClosed)
}

func TestSubmitOpenAIUsageRecordResultTaskBypassesSharedWorkerPool(t *testing.T) {
	pool := service.NewUsageRecordWorkerPoolWithOptions(service.UsageRecordWorkerPoolOptions{
		WorkerCount:           1,
		QueueSize:             4,
		TaskTimeout:           time.Second,
		OverflowPolicy:        "drop",
		OverflowSamplePercent: 0,
		AutoScaleEnabled:      false,
	})
	t.Cleanup(pool.Stop)
	blocked := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	pool.Submit(func(context.Context) {
		close(blocked)
		<-release
	})
	<-blocked

	h := &OpenAIGatewayHandler{usageRecordWorkerPool: pool}
	barrier := newWebChatUsageBarrier()
	parent := context.WithValue(context.Background(), ctxkey.WebChat, true)
	parent = withWebChatUsageBarrier(parent, barrier)
	var calls atomic.Int32

	h.submitOpenAIUsageRecordResultTask(
		parent,
		&service.OpenAIForwardResult{},
		func(context.Context) error {
			calls.Add(1)
			return nil
		},
	)

	require.EqualValues(t, 1, calls.Load())
	require.NoError(t, barrier.SealAndWait().Err())
}
