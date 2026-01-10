package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/db"
	sqlc "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/jackc/pgx/v5"
)

type historicalPriceRepository struct {
	store db.Store
}

func NewHistoricalPriceRepo(store db.Store) domain.HistoricalPriceRepo {
	return &historicalPriceRepository{store: store}
}

func (r *historicalPriceRepository) GetBatch(ctx context.Context, coinIDs []string, bucketStarts []time.Time) ([]domain.HistoricalPrice, error) {
	if len(coinIDs) == 0 || len(bucketStarts) == 0 {
		return []domain.HistoricalPrice{}, nil
	}
	if len(coinIDs) != len(bucketStarts) {
		return nil, fmt.Errorf(
			"GetBatch: coinIDs and bucketStarts must have same length (coinIDs=%d, bucketStarts=%d)",
			len(coinIDs), len(bucketStarts),
		)
	}

	rows, err := r.store.GetHistoricalPricesBatch(ctx, sqlc.GetHistoricalPricesBatchParams{
		Column1: coinIDs,
		Column2: bucketStarts,
	})
	if err != nil {
		return nil, fmt.Errorf("GetBatch: query failed: %w", err)
	}

	out := make([]domain.HistoricalPrice, 0, len(rows))
	for _, row := range rows {
		p, err := mapHistoricalPriceDBToDomain(row)
		if err != nil {
			return nil, fmt.Errorf("GetBatch: map db->domain failed: %w", err)
		}
		out = append(out, p)
	}

	return out, nil
}

func (r *historicalPriceRepository) Get(ctx context.Context, coinID string, bucketStartUTC time.Time) (domain.HistoricalPrice, error) {
	if coinID == "" {
		return domain.HistoricalPrice{}, fmt.Errorf("Get: coinID is empty")
	}

	row, err := r.store.GetHistoricalPrice(ctx, sqlc.GetHistoricalPriceParams{
		CoinID:         coinID,
		BucketStartUtc: bucketStartUTC,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.HistoricalPrice{}, err
		}
		return domain.HistoricalPrice{}, fmt.Errorf("Get: query failed: %w", err)
	}

	p, err := mapHistoricalPriceDBToDomain(row)
	if err != nil {
		return domain.HistoricalPrice{}, fmt.Errorf("Get: map db->domain failed: %w", err)
	}

	return p, nil
}

func (r *historicalPriceRepository) Upsert(ctx context.Context, p domain.HistoricalPrice) error {
	if p.CoinID == "" {
		return fmt.Errorf("Upsert: CoinID is empty")
	}

	priceNumeric, err := decimalToNumeric(p.PriceUsd)
	if err != nil {
		return fmt.Errorf("Upsert: invalid PriceUSD: %w", err)
	}

	if err := r.store.UpsertHistoricalPrice(ctx, sqlc.UpsertHistoricalPriceParams{
		CoinID:             p.CoinID,
		BucketStartUtc:     p.BucketStartUtc,
		PriceUsd:           priceNumeric,
		GranularitySeconds: int32(p.GranularitySeconds),
	}); err != nil {
		return fmt.Errorf("Upsert: query failed: %w", err)
	}

	return nil
}
