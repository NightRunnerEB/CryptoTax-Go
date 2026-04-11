package domain

import (
	"context"
	"time"
)

type FXRate struct {
	Fiat   string
	Day    time.Time
	Rate   Rate
	IsReal bool
	Source string
}

type FXRateRepo interface {
	Upsert(ctx context.Context, r FXRate) error
	ListByFiat(ctx context.Context, fiat string) ([]FXRate, error)
}
