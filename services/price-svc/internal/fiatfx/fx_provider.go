package fiatfx

import (
	"context"
	"time"

	applogger "github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/grpcerr"
	"go.uber.org/zap"
)

type FXProvider struct {
	registry *FXSourceRegistry
}

func NewFXProvider(registry *FXSourceRegistry) domain.FXProvider {
	return &FXProvider{
		registry: registry,
	}
}

func (r *FXProvider) Start(ctx context.Context) error {
	sources := r.registry.All()
	for _, source := range sources {
		srs := source
		go r.runSource(ctx, srs)
	}
	return nil
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
