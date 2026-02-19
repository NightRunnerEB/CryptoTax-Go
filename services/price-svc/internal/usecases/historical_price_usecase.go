package usecase

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/coingecko"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

const USD = "usd"

var precision = "3"

const providerCoinGecko = "coingecko"

type historicalPriceUC struct {
	repo           domain.HistoricalPriceRepo
	fxProvider     domain.FXProvider
	cgClient       *coingecko.CGClient
	contextTimeout time.Duration
}

func NewHistoricalPriceUC(
	repo domain.HistoricalPriceRepo,
	fx domain.FXProvider,
	cgClient *coingecko.CGClient,
	timeout time.Duration,
) domain.HistoricalPriceUseCase {
	return &historicalPriceUC{
		repo:           repo,
		fxProvider:     fx,
		cgClient:       cgClient,
		contextTimeout: timeout,
	}
}

func (u *historicalPriceUC) GetHistoricalPrices(ctx context.Context, fiatCurrency string, priceKeys []domain.PriceKey) ([]domain.Fiat, error) {
	if fiatCurrency == "" {
		return nil, apperr.InvalidArgument(
			"fiat currency is required",
			nil,
			apperr.FieldViolation{
				Field:       "fiat_currency",
				Description: "required",
			},
		)
	}

	if len(priceKeys) == 0 {
		return []domain.Fiat{}, nil
	}

	if u.contextTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, u.contextTimeout)
		defer cancel()
	}

	now := time.Now().UTC()

	type wanted struct {
		coinID   string
		txTime   time.Time
		bucket   time.Time
		dayStart time.Time
		desiredG time.Duration
	}

	w := make([]wanted, len(priceKeys))
	repoKeys := make([]domain.PriceKey, len(priceKeys))

	// compute desired granularity + bucketStart per key
	for i, k := range priceKeys {
		txTime := k.BucketStartUtc.UTC() // NOTE: this is actually tx time
		desired := u.cgClient.GetGranularitySeconds(txTime, now)
		bucket := floorToBucket(txTime, desired)
		dayStart := truncateDayUTC(txTime)

		w[i] = wanted{
			coinID:   k.CoinID,
			txTime:   txTime,
			bucket:   bucket,
			dayStart: dayStart,
			desiredG: desired,
		}
		// repo expects bucket_start_utc
		repoKeys[i] = domain.PriceKey{CoinID: k.CoinID, BucketStartUtc: bucket}
	}

	// read batch from DB (LEFT JOIN order-preserving)
	rows, err := u.repo.GetBatch(ctx, repoKeys)
	if err != nil {
		return nil, apperr.Internal("repo GetBatch failed", err, map[string]string{
			"keys": strconv.Itoa(len(repoKeys)),
		})
	}
	if len(rows) != len(repoKeys) {
		return nil, apperr.Internal("pricing invariant violated", nil, map[string]string{
			"got":      strconv.Itoa(len(rows)),
			"expected": strconv.Itoa(len(repoKeys)),
		})
	}

	// plan provider fetches for missing/upgrade
	type fetchKey struct {
		coinID   string
		dayStart time.Time
		g        time.Duration
	}
	needFetch := make(map[fetchKey]struct{})

	for i, p := range rows {
		missing := p.PriceUsd == nil
		upgrade := false
		if !missing {
			if *p.GranularitySeconds > int(w[i].desiredG.Seconds()) {
				upgrade = true
			}
		}
		if missing || upgrade {
			needFetch[fetchKey{coinID: w[i].coinID, dayStart: w[i].dayStart, g: w[i].desiredG}] = struct{}{}
		}
	}

	// fetch day data from CoinGecko and upsert buckets
	for fk := range needFetch {
		if err := u.fetchAndUpsertDay(ctx, fk.coinID, fk.dayStart, fk.g); err != nil {
			var ae *apperr.Error
			if errors.As(err, &ae) {
				return nil, ae
			}
			return nil, apperr.Internal("fetch and upsert failed", err, map[string]string{
				"coin_id":             fk.coinID,
				"day_start":           fk.dayStart.Format(time.DateOnly),
				"granularity_seconds": strconv.FormatInt(int64(fk.g/time.Second), 10),
			})
		}
	}

	// re-read after upserts
	if len(needFetch) > 0 {
		rows, err = u.repo.GetBatch(ctx, repoKeys)
		if err != nil {
			return nil, apperr.Internal("repo GetBatch after fetch failed", err, map[string]string{
				"keys": strconv.Itoa(len(repoKeys)),
			})
		}
		if len(rows) != len(repoKeys) {
			return nil, apperr.Internal("pricing invariant violated after fetch", nil, map[string]string{
				"got":      strconv.Itoa(len(rows)),
				"expected": strconv.Itoa(len(repoKeys)),
			})
		}
	}

	out := make([]domain.Fiat, len(rows))

	for i, p := range rows {
		if p.PriceUsd == nil {
			return nil, apperr.PriceUnavailable(
				"price unavailable",
				w[i].coinID,
				map[string]string{
					"bucket_start": w[i].bucket.Format(time.RFC3339),
					"date":         w[i].dayStart.Format(time.DateOnly),
				},
				nil,
			)
		}

		rate, err := u.fxProvider.GetUSDtoFiatRate(ctx, w[i].dayStart, fiatCurrency)
		if err != nil {
			// distinguish unsupported fiat vs fx unavailable if your fxProvider does it
			var ae *apperr.Error
			if errors.As(err, &ae) {
				return nil, ae
			}
			return nil, apperr.FXUnavailable(
				"fx rate fetch failed",
				fiatCurrency,
				map[string]string{
					"day": w[i].dayStart.Format(time.DateOnly),
				},
				err,
			)
		}

		usd := *p.PriceUsd
		out[i] = usd.Mul(rate)
	}

	return out, nil
}

func (u *historicalPriceUC) fetchAndUpsertDay(ctx context.Context, coinID string, dayStartUTC time.Time, granularity time.Duration) error {
	metadata := map[string]string{
		"coin_id": coinID,
		"date":    dayStartUTC.Format(time.DateOnly),
	}

	to := dayStartUTC.Add(24*time.Hour - time.Second)

	// CoinGecko returns points; per our agreement we normalize sequentially into buckets without flooring by timestamp.
	resp, err := u.cgClient.CoinsMarketChartRange(ctx, coinID, "usd", dayStartUTC, to, &precision)
	if err != nil {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			return ae
		}
		return apperr.ProviderUnavailable(
			"coingecko request failed",
			providerCoinGecko,
			err,
			metadata,
		)
	}

	if resp == nil || len(resp.Prices) == 0 {
		return apperr.ProviderBadResponse(
			"empty prices",
			providerCoinGecko,
			nil,
			metadata,
		)
	}

	// normalize points to buckets "by order"
	buckets, err := normalizeByOrder(coinID, dayStartUTC, granularity, resp.Prices)
	if err != nil {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			return ae
		}
		return apperr.ProviderBadResponse(
			"normalize by order failed",
			providerCoinGecko,
			err,
			metadata,
		)
	}

	if err := u.repo.UpsertBatch(ctx, buckets); err != nil {
		return apperr.Internal("repo UpsertBatch failed", err, metadata)
	}

	return nil
}
