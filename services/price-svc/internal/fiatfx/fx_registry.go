package fiatfx

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type Currency = string
type Rate = decimal.Decimal

const (
	USD Currency = "USD"
	RUB Currency = "RUB"
	KZT Currency = "KZT"
)

type Schedule struct {
	Loc  *time.Location // Europe/Moscow for RUB, Asia/Almaty for KZT
	Hour int
	Min  int
}

type FXSource interface {
	Currency() Currency
	Get(key time.Time) (Rate, bool)
	Schedule() Schedule
	Update(ctx context.Context) error
}

type FXSourceRegistry struct {
	sources map[Currency]FXSource
}

func NewFXRegistry() *FXSourceRegistry {
	return &FXSourceRegistry{
		sources: make(map[Currency]FXSource),
	}
}

func (r *FXSourceRegistry) GetSource(currency Currency) (FXSource, bool) {
	source, ok := r.sources[currency]
	return source, ok
}

func (r *FXSourceRegistry) Register(source FXSource) {
	r.sources[source.Currency()] = source
}

func (r *FXSourceRegistry) All() []FXSource {
	result := make([]FXSource, 0, len(r.sources))
	for _, source := range r.sources {
		result = append(result, source)
	}
	return result
}

func (r *FXSourceRegistry) Currencies() []Currency {
	result := make([]Currency, 0, len(r.sources))
	for c := range r.sources {
		result = append(result, c)
	}
	return result
}
