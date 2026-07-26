package handler

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type webChatUsageBarrierContextKey struct{}

// webChatUsageBarrier proves that every usage producer registered by one
// first-party Web Chat request has finished executing. Sealing prevents a task
// discovered after the gateway returns from escaping the final wait.
type webChatUsageBarrier struct {
	mu            sync.Mutex
	done          chan struct{}
	pending       int
	producerCount int
	results       []webChatUsageProducerResult
	sealed        bool
	complete      bool
}

type webChatUsageProducerTask func(context.Context) error

type webChatUsageProducerResult struct {
	Err error
}

type webChatUsageBarrierResult struct {
	ProducerCount int
	Results       []webChatUsageProducerResult
}

func (r webChatUsageBarrierResult) Err() error {
	errs := make([]error, 0, len(r.Results))
	for _, result := range r.Results {
		if result.Err != nil {
			errs = append(errs, result.Err)
		}
	}
	return errors.Join(errs...)
}

func newWebChatUsageBarrier() *webChatUsageBarrier {
	return &webChatUsageBarrier{done: make(chan struct{})}
}

func withWebChatUsageBarrier(
	ctx context.Context,
	barrier *webChatUsageBarrier,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if barrier == nil {
		return ctx
	}
	return context.WithValue(ctx, webChatUsageBarrierContextKey{}, barrier)
}

func webChatUsageBarrierFromContext(ctx context.Context) *webChatUsageBarrier {
	if ctx == nil {
		return nil
	}
	barrier, _ := ctx.Value(webChatUsageBarrierContextKey{}).(*webChatUsageBarrier)
	return barrier
}

func (b *webChatUsageBarrier) Register(
	task service.UsageRecordTask,
) (service.UsageRecordTask, bool) {
	if b == nil || task == nil {
		return nil, false
	}
	return b.RegisterResult(func(ctx context.Context) error {
		task(ctx)
		return nil
	})
}

func (b *webChatUsageBarrier) RegisterResult(
	task webChatUsageProducerTask,
) (service.UsageRecordTask, bool) {
	if b == nil || task == nil {
		return nil, false
	}

	b.mu.Lock()
	if b.sealed {
		b.mu.Unlock()
		return nil, false
	}
	b.pending++
	b.producerCount++
	b.mu.Unlock()

	var runOnce sync.Once
	return func(ctx context.Context) {
		runOnce.Do(func() {
			var taskErr error
			defer func() {
				if recovered := recover(); recovered != nil {
					taskErr = fmt.Errorf("web chat usage producer panic: %v", recovered)
				}
				b.finish(taskErr)
			}()
			taskErr = task(ctx)
		})
	}, true
}

func (b *webChatUsageBarrier) Seal() {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.sealed = true
	b.completeIfReadyLocked()
	b.mu.Unlock()
}

func (b *webChatUsageBarrier) Wait() webChatUsageBarrierResult {
	if b == nil {
		return webChatUsageBarrierResult{}
	}
	<-b.done
	b.mu.Lock()
	defer b.mu.Unlock()
	results := append([]webChatUsageProducerResult(nil), b.results...)
	return webChatUsageBarrierResult{
		ProducerCount: b.producerCount,
		Results:       results,
	}
}

func (b *webChatUsageBarrier) SealAndWait() webChatUsageBarrierResult {
	if b == nil {
		return webChatUsageBarrierResult{}
	}
	b.Seal()
	return b.Wait()
}

func (b *webChatUsageBarrier) finish(err error) {
	b.mu.Lock()
	if b.pending > 0 {
		b.pending--
	}
	b.results = append(b.results, webChatUsageProducerResult{Err: err})
	b.completeIfReadyLocked()
	b.mu.Unlock()
}

func (b *webChatUsageBarrier) completeIfReadyLocked() {
	if b.complete || !b.sealed || b.pending != 0 {
		return
	}
	b.complete = true
	close(b.done)
}
