package fiatfx

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
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
	if err := src.Update(ctx); err != nil {
		log.Printf("fx: initial update failed for %s: %v", src.Currency(), err)
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
				log.Printf("fx: update failed for %s: %v", src.Currency(), err)
			}
		}
	}
}

func (r *FXProvider) GetUSDtoFiatRate(ctx context.Context, day time.Time, currency string) (domain.Fiat, error) {
	source, ok := r.registry.GetSource(currency)
	if !ok {
		return domain.Fiat{}, fmt.Errorf("GetUSDtoFiatRate: no source for currency %s", currency)
	}

	if rate, ok := source.Get(day); ok {
		return rate, nil
	}

	// Need to implement certain day update logic
	// rate, err := source.UpdateAt(ctx)
	// if err != nil {
	// 	return domain.Fiat{}, fmt.Errorf("GetUSDtoFiatRate: source update failed: %w", err)
	// }

	return domain.Fiat{}, fmt.Errorf("GetUSDtoFiatRate: no rate for currency %s at day %s", currency, day.Format("2006-01-02"))
}
