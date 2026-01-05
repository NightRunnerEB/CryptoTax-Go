package repository

import (
	"context"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/postgres"
)

type historicalPriceRepository struct {
	*postgres.Postgres
}

func NewHistoricalPriceRepo(pg *postgres.Postgres) domain.HistoricalPriceRepo {
	return &historicalPriceRepository{pg}
}

func (r *historicalPriceRepository) GetBatch(ctx context.Context, coinIDs []string, bucketStarts []time.Time, fiatCurrency, sourceProfile string) ([]domain.HistoricalPrice, error) {
	// Implementation goes here
	return nil, nil
}

func (r *historicalPriceRepository) Get(ctx context.Context, coinID, fiatCurrency, sourceProfile string, bucketStartUTC time.Time) (domain.HistoricalPrice, error) {
	// Implementation goes here
	return domain.HistoricalPrice{}, nil
}

func (r *historicalPriceRepository) Upsert(ctx context.Context, p domain.HistoricalPrice) error {
	// Implementation goes here
	return nil
}
