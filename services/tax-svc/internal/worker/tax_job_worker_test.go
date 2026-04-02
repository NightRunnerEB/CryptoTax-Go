package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

type fakeProcessor struct {
	fn func(ctx context.Context) (bool, error)
}

func (f fakeProcessor) ProcessNextQueuedJob(ctx context.Context) (bool, error) {
	return f.fn(ctx)
}

func TestTaxJobWorker_Start_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls int32
	processor := fakeProcessor{fn: func(context.Context) (bool, error) {
		if atomic.AddInt32(&calls, 1) == 1 {
			cancel()
		}
		return false, nil
	}}

	w := NewTaxJobWorker(processor, zap.NewNop(), time.Millisecond, time.Millisecond)
	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if atomic.LoadInt32(&calls) == 0 {
		t.Fatal("expected at least one ProcessNextQueuedJob call")
	}
}

func TestTaxJobWorker_Start_HandlesProcessorErrorAndContinues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls int32
	processor := fakeProcessor{fn: func(context.Context) (bool, error) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			return false, errors.New("boom")
		}
		cancel()
		return false, nil
	}}

	w := NewTaxJobWorker(processor, zap.NewNop(), time.Millisecond, time.Millisecond)
	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected at least 2 calls, got %d", calls)
	}
}
