package fiatfx

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

type fakeSource struct {
	currency Currency
	schedule Schedule
	rates    map[string]Rate
}

func (f *fakeSource) Currency() Currency { return f.currency }
func (f *fakeSource) Schedule() Schedule { return f.schedule }
func (f *fakeSource) Update(context.Context) error {
	return nil
}
func (f *fakeSource) Get(key time.Time) (Rate, bool) {
	v, ok := f.rates[dateKeyISO(dateOnly(key, f.schedule.Loc))]
	return v, ok
}

func TestFXRegistry_RegisterAndGet(t *testing.T) {
	reg := NewFXRegistry()
	src := &fakeSource{
		currency: RUB,
		schedule: Schedule{Loc: time.UTC, Hour: 1, Min: 0},
		rates:    map[string]Rate{},
	}
	reg.Register(src)

	got, ok := reg.GetSource(RUB)
	if !ok || got == nil {
		t.Fatal("expected source in registry")
	}
	if got.Currency() != RUB {
		t.Fatalf("unexpected currency: %s", got.Currency())
	}
	if len(reg.All()) != 1 {
		t.Fatalf("expected one source, got %d", len(reg.All()))
	}
	if len(reg.Currencies()) != 1 {
		t.Fatalf("expected one currency, got %d", len(reg.Currencies()))
	}
}

func TestFXProvider_GetUSDtoFiatRate_UnsupportedFiat(t *testing.T) {
	p := NewFXProvider(NewFXRegistry())
	_, err := p.GetUSDtoFiatRate(context.Background(), time.Now().UTC(), "EUR")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrUnsupportedFiat {
		t.Fatalf("expected UNSUPPORTED_FIAT, got %v", err)
	}
}

func TestFXProvider_GetUSDtoFiatRate_HitAndMiss(t *testing.T) {
	reg := NewFXRegistry()
	day := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	reg.Register(&fakeSource{
		currency: RUB,
		schedule: Schedule{Loc: time.UTC, Hour: 1, Min: 0},
		rates: map[string]Rate{
			dateKeyISO(day): decimal.RequireFromString("90.12"),
		},
	})
	p := NewFXProvider(reg)

	r, err := p.GetUSDtoFiatRate(context.Background(), day, "RUB")
	if err != nil {
		t.Fatalf("unexpected error on hit: %v", err)
	}
	if !r.Equal(decimal.RequireFromString("90.12")) {
		t.Fatalf("unexpected rate: %s", r.String())
	}

	_, err = p.GetUSDtoFiatRate(context.Background(), day.AddDate(0, 0, 1), "RUB")
	if err == nil {
		t.Fatal("expected miss error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrFXUnavailable {
		t.Fatalf("expected FX_UNAVAILABLE, got %v", err)
	}
}

func TestFXProvider_StartStopIdempotent(t *testing.T) {
	reg := NewFXRegistry()
	reg.Register(&fakeSource{
		currency: RUB,
		schedule: Schedule{Loc: time.UTC, Hour: 23, Min: 59},
		rates:    map[string]Rate{},
	})
	p := NewFXProvider(reg)

	ctx := context.Background()

	if err := p.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if err := p.Start(ctx); err != nil {
		t.Fatalf("second start failed: %v", err)
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Stop(stopCtx); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if err := p.Stop(stopCtx); err != nil {
		t.Fatalf("second stop failed: %v", err)
	}
}
