package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

type HistoricalPrice struct {
	CoinID             string           `json:"coin_id"`
	Time               time.Time        `json:"bucket_start_utc"`
	PriceUsd           *decimal.Decimal `json:"price_usd"`
	GranularitySeconds *int             `json:"granularity_seconds"`
}

type PriceKey struct {
	CoinID         string
	BucketStartUtc time.Time
}

type Fiat = decimal.Decimal
type Rate = decimal.Decimal

type HistoricalPriceUseCase interface {
	GetHistoricalPrices(ctx context.Context, fiatCurrency string, priceKeys []PriceKey) ([]Fiat, error)
}

type HistoricalPriceRepo interface {
	Upsert(ctx context.Context, p HistoricalPrice) error
	UpsertBatch(ctx context.Context, prices []HistoricalPrice) error

	Get(ctx context.Context, coinID string, bucketStartUtc time.Time) (HistoricalPrice, error)
	GetBatch(ctx context.Context, priceKeys []PriceKey) ([]HistoricalPrice, error)
}

type FXProvider interface {
	Start(context.Context) error
	GetUSDtoFiatRate(ctx context.Context, day time.Time, fiat string) (Fiat, error)
}

type CoinIdResolver interface {
	Resolve(symbol string) (string, error)
}
