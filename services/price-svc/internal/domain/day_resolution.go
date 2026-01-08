package domain

import (
	"context"
	"time"
)

type DayResolution struct {
	CoinID string    `json:"coin_id"`
	DayUtc time.Time `json:"day_utc"`
}

type DayResolutionUseCase interface {
	GetDayResolutions(ctx context.Context, coinIDs []string, days_utc []time.Time) ([]DayResolution, error)
}

type DayResolutionRepo interface {
	Upsert(ctx context.Context, dr DayResolution) error

	Get(ctx context.Context, coinID string, day_utc time.Time) (DayResolution, error)
	GetBatch(ctx context.Context, coinIDs []string, days_utc []time.Time) ([]DayResolution, error)
}
