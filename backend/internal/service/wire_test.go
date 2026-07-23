package service

import (
	"errors"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
)

func TestProvideTimingWheelService_DefersInitializationErrorUntilStart(t *testing.T) {
	original := newTimingWheel
	t.Cleanup(func() { newTimingWheel = original })

	newTimingWheel = func(_ time.Duration, _ int, _ collection.Execute) (*collection.TimingWheel, error) {
		return nil, errors.New("boom")
	}

	svc, err := ProvideTimingWheelService()
	if err != nil {
		t.Fatalf("构造阶段不应启动 timing wheel: %v", err)
	}
	if svc == nil {
		t.Fatalf("期望构造出未启动的 service")
	}
	if err := svc.StartWithError(); err == nil {
		t.Fatalf("期望启动阶段返回 error，但得到 nil")
	}
}

func TestProvideTimingWheelService_Success(t *testing.T) {
	svc, err := ProvideTimingWheelService()
	if err != nil {
		t.Fatalf("期望 err 为 nil，但得到: %v", err)
	}
	if svc == nil {
		t.Fatalf("期望 svc 非空，但得到 nil")
	}
	if err := svc.StartWithError(); err != nil {
		t.Fatalf("启动 timing wheel: %v", err)
	}
	svc.Stop()
}
