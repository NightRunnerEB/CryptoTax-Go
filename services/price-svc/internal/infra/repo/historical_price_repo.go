package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

type historicalPriceRepository struct {
	store db.Store
}

func NewHistoricalPriceRepo(store db.Store) domain.HistoricalPriceRepo {
	return &historicalPriceRepository{store: store}
}

func (r *historicalPriceRepository) GetBatch(ctx context.Context, priceKeys []domain.PriceKey) ([]domain.HistoricalPrice, error) {
	if len(priceKeys) == 0 {
		return nil, nil
	}

	coinIDs := make([]string, 0, len(priceKeys))
	bucketStarts := make([]time.Time, 0, len(priceKeys))
	for _, k := range priceKeys {
		coinIDs = append(coinIDs, k.CoinID)
		bucketStarts = append(bucketStarts, k.BucketStartUtc)
	}

	rows, err := r.store.GetHistoricalPricesBatch(ctx, db.GetHistoricalPricesBatchParams{
		Column1: coinIDs,
		Column2: toTimestamptzSlice(bucketStarts),
	})
	if err != nil {
		return nil, apperr.Internal("get historical price batch failed", err, map[string]string{
			"keys": strconv.Itoa(len(priceKeys)),
		})
	}

	out := make([]domain.HistoricalPrice, 0, len(rows))
	for _, row := range rows {
		p, err := mapHistoricalPriceRowDBToDomain(row)
		if err != nil {
			return nil, apperr.Internal("map historical price row failed", err, nil)
		}
		out = append(out, p)
	}

	return out, nil
}

func (r *historicalPriceRepository) Get(ctx context.Context, coinID string, bucketStartUTC time.Time) (domain.HistoricalPrice, error) {
	if coinID == "" {
		return domain.HistoricalPrice{}, apperr.InvalidArgument("coin id is required", nil, apperr.FieldViolation{
			Field:       "coin_id",
			Description: "required",
		})
	}

	row, err := r.store.GetHistoricalPrice(ctx, db.GetHistoricalPriceParams{
		CoinID:         coinID,
		BucketStartUtc: pgtype.Timestamptz{Time: bucketStartUTC, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			name := coinID + "@" + bucketStartUTC.UTC().Format(time.RFC3339)
			return domain.HistoricalPrice{}, apperr.NotFound("historical price not found", apperr.Resource{
				Type: "historical_price",
				Name: name,
			}, err)
		}
		return domain.HistoricalPrice{}, apperr.Internal("get historical price failed", err, map[string]string{
			"coin_id":      coinID,
			"bucket_start": bucketStartUTC.UTC().Format(time.RFC3339),
		})
	}

	p, err := mapHistoricalPriceDBToDomain(row)
	if err != nil {
		return domain.HistoricalPrice{}, apperr.Internal("map historical price failed", err, nil)
	}

	return p, nil
}

func (r *historicalPriceRepository) Upsert(ctx context.Context, p domain.HistoricalPrice) error {
	if p.CoinID == "" {
		return apperr.InvalidArgument("coin id is required", nil, apperr.FieldViolation{
			Field:       "coin_id",
			Description: "required",
		})
	}

	priceNumeric, err := decimalToNumeric(p.PriceUsd)
	if err != nil {
		return apperr.InvalidArgument("price_usd is invalid", err, apperr.FieldViolation{
			Field:       "price_usd",
			Description: "invalid",
		})
	}

	if err := r.store.UpsertHistoricalPrice(ctx, db.UpsertHistoricalPriceParams{
		CoinID:             p.CoinID,
		BucketStartUtc:     pgtype.Timestamptz{Time: p.Time, Valid: true},
		PriceUsd:           priceNumeric,
		GranularitySeconds: int32(*p.GranularitySeconds),
	}); err != nil {
		return apperr.Internal("upsert historical price failed", err, map[string]string{
			"coin_id":      p.CoinID,
			"bucket_start": p.Time.UTC().Format(time.RFC3339),
		})
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
			return apperr.InvalidArgument("invalid historical price", nil, apperr.FieldViolation{
				Field:       "prices",
				Description: "missing required fields",
			})
		}

		num, err := decimalToNumeric(p.PriceUsd)
		if err != nil {
			return apperr.InvalidArgument("price_usd is invalid", err, apperr.FieldViolation{
				Field:       "price_usd",
				Description: "invalid",
			})
		}

		coinIDs = append(coinIDs, p.CoinID)
		bucketStarts = append(bucketStarts, pgtype.Timestamptz{Time: p.Time, Valid: true})
		priceNums = append(priceNums, num)
		grans = append(grans, int32(*p.GranularitySeconds))
	}

	if err := r.store.UpsertHistoricalPricesBatch(
		ctx,
		db.UpsertHistoricalPricesBatchParams{
			Column1: coinIDs,
			Column2: bucketStarts,
			Column3: priceNums,
			Column4: grans,
		},
	); err != nil {
		return apperr.Internal("upsert historical prices batch failed", err, map[string]string{
			"count": strconv.Itoa(len(prices)),
		})
	}

	return nil
}
