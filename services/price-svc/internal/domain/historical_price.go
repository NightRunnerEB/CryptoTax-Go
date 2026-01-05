package domain

import (
	"context"
	"time"
)

type HistoricalPrice struct {
	CoinID         string    `json:"coin_id"`
	BucketStartUtc time.Time `json:"bucket_start_utc"`
	FiatCurrency   string    `json:"fiat_currency"`
	SourceProfile  string    `json:"source_profile"`
	Rate           float64   `json:"rate"`
	FetchedAt      time.Time `json:"fetched_at"`
}

type HistoricalPriceUseCase interface {
	GetHistoricalPrices(ctx context.Context, coinIDs []string, times []time.Time, fiatCurrency string, sourceProfile string) ([]HistoricalPrice, error)
}

type HistoricalPriceRepo interface {
	Upsert(ctx context.Context, p HistoricalPrice) error
	Get(ctx context.Context, coinID, fiatCurrency, sourceProfile string, time time.Time) (HistoricalPrice, error)

	GetBatch(ctx context.Context, coinIDs []string, time []time.Time, fiatCurrency, sourceProfile string) ([]HistoricalPrice, error)
}
