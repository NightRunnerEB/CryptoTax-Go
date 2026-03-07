package worker

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Processor interface {
	ProcessNextQueuedJob(ctx context.Context) (bool, error)
}

type TaxJobWorker struct {
	processor    Processor
	log          *zap.Logger
	pollInterval time.Duration
	idleSleep    time.Duration
}

func NewTaxJobWorker(
	processor Processor,
	log *zap.Logger,
	pollInterval time.Duration,
	idleSleep time.Duration,
) *TaxJobWorker {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	if idleSleep <= 0 {
		idleSleep = 500 * time.Millisecond
	}

	return &TaxJobWorker{
		processor:    processor,
		log:          log,
		pollInterval: pollInterval,
		idleSleep:    idleSleep,
	}
}

func (w *TaxJobWorker) Start(ctx context.Context) error {
	w.log.Info("TaxJobWorker: started")

	for {
		select {
		case <-ctx.Done():
			w.log.Info("TaxJobWorker: stopped")
			return nil
		default:
		}

		processed, err := w.processor.ProcessNextQueuedJob(ctx)
		if err != nil {
			w.log.Warn("TaxJobWorker: processing failed", zap.Error(err))
			if !wait(ctx, w.pollInterval) {
				return nil
			}
			continue
		}

		if processed {
			continue
		}

		if !wait(ctx, w.idleSleep) {
			return nil
		}
	}
}

func wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
