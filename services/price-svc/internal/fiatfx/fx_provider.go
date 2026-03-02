package fiatfx

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	applogger "github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/grpcerr"
)

type FXProvider struct {
	registry *FXSourceRegistry

	mu        sync.Mutex
	waitGroup sync.WaitGroup
	started   bool
	cancel    context.CancelFunc
}

func NewFXProvider(registry *FXSourceRegistry) domain.FXProvider {
	return &FXProvider{
		registry: registry,
	}
}

func (r *FXProvider) Start(ctx context.Context) error {
	log := applogger.FromContext(ctx)

	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		log.Warn("fx provider start skipped: already running")
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.started = true
	r.mu.Unlock()

	sources := r.registry.All()
	log.Info("fx provider starting", zap.Int("sources", len(sources)))
	for _, source := range sources {
		srs := source
		r.waitGroup.Add(1)
		go func() {
			defer r.waitGroup.Done()
			r.runSource(runCtx, srs)
		}()
	}

	return nil
}

func (r *FXProvider) Stop(ctx context.Context) error {
	log := applogger.FromContext(ctx)

	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		log.Info("fx provider stop skipped: not running")
		return nil
	}
	cancel := r.cancel
	r.mu.Unlock()

	log.Info("fx provider graceful shutdown started")
	if cancel != nil {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		r.waitGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
		r.mu.Lock()
		r.cancel = nil
		r.started = false
		r.mu.Unlock()

		log.Info("fx provider graceful shutdown completed")
		return nil
	case <-ctx.Done():
		log.Warn("fx provider graceful shutdown timed out", zap.Error(ctx.Err()))
		return ctx.Err()
	}
}

func (r *FXProvider) runSource(ctx context.Context, src FXSource) {
	log := applogger.FromContext(ctx)
	currency := src.Currency()

	if err := src.Update(ctx); err != nil {
		fields := append(
			[]zap.Field{
				zap.String("fiat", currency),
			},
			grpcerr.LogFields(err)...,
		)
		log.Fatal("fx initial update failed", fields...)
	}

	for {
		next := nextRunTime(time.Now(), src.Schedule())
		timer := time.NewTimer(time.Until(next))

		select {
		case <-ctx.Done():
			timer.Stop()
			log.Info("fx source worker stopped", zap.String("fiat", currency))
			return
		case <-timer.C:
			if err := src.Update(ctx); err != nil {
				log.Error("fx update failed", zap.String("fiat", currency), zap.Error(err))
			}
		}
	}
}

func (r *FXProvider) GetUSDtoFiatRate(ctx context.Context, day time.Time, currency string) (domain.Fiat, error) {
	log := applogger.FromContext(ctx)

	source, ok := r.registry.GetSource(currency)
	if !ok {
		log.Warn("fx source not found", zap.String("fiat", currency))
		return domain.Fiat{}, apperr.UnsupportedFiat("unsupported fiat currency", currency)
	}

	if rate, ok := source.Get(day); ok {
		return rate, nil
	}

	log.Warn(
		"fx rate cache miss",
		zap.String("fiat", currency),
		zap.String("day", day.Format("2006-01-02")),
	)
	// Need to implement certain day update logic
	// rate, err := source.UpdateAt(ctx)
	// if err != nil {
	// 	return domain.Fiat{}, fmt.Errorf("GetUSDtoFiatRate: source update failed: %w", err)
	// }

	return domain.Fiat{}, apperr.FXUnavailable(
		"fx rate unavailable",
		currency,
		map[string]string{
			"day": day.Format("2006-01-02"),
		},
		nil,
	)
}
