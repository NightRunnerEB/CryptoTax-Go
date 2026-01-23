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
	"github.com/jackc/pgx/v5/pgtype"
)

type historicalPriceRepository struct {
	store db.Store
}

func NewHistoricalPriceRepo(store db.Store) domain.HistoricalPriceRepo {
	return &historicalPriceRepository{store: store}
}

func (r *historicalPriceRepository) GetBatch(ctx context.Context, priceKeys []domain.PriceKey) ([]domain.HistoricalPrice, error) {
	if len(priceKeys) == 0 {
		return []domain.HistoricalPrice{}, nil
	}

	coinIDs := make([]string, 0, len(priceKeys))
	bucketStarts := make([]time.Time, 0, len(priceKeys))
	for _, k := range priceKeys {
		coinIDs = append(coinIDs, k.CoinID)
		bucketStarts = append(bucketStarts, k.BucketStartUtc)
	}

	rows, err := r.store.GetHistoricalPricesBatch(ctx, sqlc.GetHistoricalPricesBatchParams{
		Column1: coinIDs,
		Column2: toTimestamptzSlice(bucketStarts),
	})
	if err != nil {
		return nil, fmt.Errorf("GetBatch: query failed: %w", err)
	}

	out := make([]domain.HistoricalPrice, 0, len(rows))
	for _, row := range rows {
		p, err := mapHistoricalPriceRowDBToDomain(row)
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
		BucketStartUtc: pgtype.Timestamptz{Time: bucketStartUTC, Valid: true},
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
		BucketStartUtc:     pgtype.Timestamptz{Time: p.Time, Valid: true},
		PriceUsd:           priceNumeric,
		GranularitySeconds: int32(*p.GranularitySeconds),
	}); err != nil {
		return fmt.Errorf("Upsert: query failed: %w", err)
	}

	return nil
}

func (r *historicalPriceRepository) UpsertBatch(
	ctx context.Context,
	prices []domain.HistoricalPrice,
) error {
	if len(prices) == 0 {
		return nil
	}

	coinIDs := make([]string, 0, len(prices))
	bucketStarts := make([]pgtype.Timestamptz, 0, len(prices))
	priceNums := make([]pgtype.Numeric, 0, len(prices))
	grans := make([]int32, 0, len(prices))

	for _, p := range prices {
		if p.CoinID == "" || p.PriceUsd == nil || p.GranularitySeconds == nil {
			return fmt.Errorf("UpsertBatch: invalid HistoricalPrice %+v", p)
		}

		num, err := decimalToNumeric(p.PriceUsd)
		if err != nil {
			return fmt.Errorf("UpsertBatch: price_usd: %w", err)
		}

		coinIDs = append(coinIDs, p.CoinID)
		bucketStarts = append(bucketStarts, pgtype.Timestamptz{Time: p.Time, Valid: true})
		priceNums = append(priceNums, num)
		grans = append(grans, int32(*p.GranularitySeconds))
	}

	if err := r.store.UpsertHistoricalPricesBatch(
		ctx,
		sqlc.UpsertHistoricalPricesBatchParams{
			Column1: coinIDs,
			Column2: bucketStarts,
			Column3: priceNums,
			Column4: grans,
		},
	); err != nil {
		return fmt.Errorf("UpsertBatch: query failed: %w", err)
	}

	return nil
}
