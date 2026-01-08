package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type HistoricalPrice struct {
	CoinID         string          `json:"coin_id"`
	BucketStartUtc time.Time       `json:"bucket_start_utc"`
	PriceUsd       decimal.Decimal `json:"price_usd"`
}

type HistoricalPriceUseCase interface {
	GetHistoricalPrices(ctx context.Context, coinIDs []string, bucket_start_utc []time.Time) ([]HistoricalPrice, error)
}

type HistoricalPriceRepo interface {
	Upsert(ctx context.Context, p HistoricalPrice) error
	Get(ctx context.Context, coinID string, bucket_start_utc time.Time) (HistoricalPrice, error)

	GetBatch(ctx context.Context, coinIDs []string, bucket_start_utc []time.Time) ([]HistoricalPrice, error)
}
