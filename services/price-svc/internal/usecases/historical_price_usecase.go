package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/coingecko"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/logger"
)

const USD = "usd"

var precision = "3"

type historicalPriceUC struct {
	logger         logger.Logger
	repo           domain.HistoricalPriceRepo
	fxProvider     domain.FXProvider
	cgClient       *coingecko.CGClient
	contextTimeout time.Duration
}

func NewHistoricalPriceUC(
	logger logger.Logger,
	repo domain.HistoricalPriceRepo,
	fx domain.FXProvider,
	cgClient *coingecko.CGClient,
	timeout time.Duration,
) domain.HistoricalPriceUseCase {
	return &historicalPriceUC{
		logger:         logger,
		repo:           repo,
		fxProvider:     fx,
		cgClient:       cgClient,
		contextTimeout: timeout,
	}
}

func (u *historicalPriceUC) GetHistoricalPrices(ctx context.Context, fiatCurrency string, priceKeys []domain.PriceKey) ([]domain.Fiat, error) {
	if fiatCurrency == "" {
		return nil, apperr.ErrInvalidArgument
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
		dayEnd   time.Time
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
		dayEnd := dayStart.Add(24*time.Hour + 59*time.Minute + 59*time.Second)

		w[i] = wanted{
			coinID:   k.CoinID,
			txTime:   txTime,
			bucket:   bucket,
			dayStart: dayStart,
			dayEnd:   dayEnd,
			desiredG: desired,
		}
		// repo expects bucket_start_utc
		repoKeys[i] = domain.PriceKey{CoinID: k.CoinID, BucketStartUtc: bucket}
	}

	// read batch from DB (LEFT JOIN order-preserving)
	rows, err := u.repo.GetBatch(ctx, repoKeys)
	if err != nil {
		return nil, fmt.Errorf("repo.GetBatch: %w", err)
	}
	if len(rows) != len(repoKeys) {
		return nil, fmt.Errorf("pricing invariant violated: got %d rows for %d keys", len(rows), len(repoKeys))
	}

	// plan provider fetches for missing/upgrade
	type fetchKey struct {
		coinID   string
		dayStart time.Time
		dayEnd   time.Time
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
			if errors.Is(err, apperr.ErrProviderUnavailable) || errors.Is(err, apperr.ErrProviderBadResponse) {
				return nil, err
			}
			return nil, fmt.Errorf("fetchAndUpsertDay: %w", err)
		}
	}

	// re-read after upserts
	if len(needFetch) > 0 {
		rows, err = u.repo.GetBatch(ctx, repoKeys)
		if err != nil {
			return nil, fmt.Errorf("repo.GetBatch (after fetch): %w", err)
		}
		if len(rows) != len(repoKeys) {
			return nil, fmt.Errorf("pricing invariant violated after fetch: got %d rows for %d keys", len(rows), len(repoKeys))
		}
	}

	out := make([]domain.Fiat, len(rows))

	for i, p := range rows {
		if p.PriceUsd == nil {
			u.logger.Error("price still missing after fetch", "coinID", w[i].coinID, "bucket", w[i].bucket)
			return nil, fmt.Errorf("coin=%s bucket=%s: %w", w[i].coinID, w[i].bucket.Format(time.RFC3339), apperr.ErrPriceUnavailable)
		}

		rate, err := u.fxProvider.GetUSDtoFiatRate(ctx, w[i].dayStart, fiatCurrency)
		if err != nil {
			// distinguish unsupported fiat vs fx unavailable if your fxProvider does it
			u.logger.Error("fxProvider.GetUSDtoFiatRate: fx rate fetch failed", "fiat", fiatCurrency, "day", w[i].dayStart, "error", err)
			return nil, fmt.Errorf("fxProvider.GetUSDtoFiatRate: %w", err)
		}

		usd := *p.PriceUsd
		out[i] = usd.Mul(rate)
	}

	return out, nil
}

func (u *historicalPriceUC) fetchAndUpsertDay(ctx context.Context, coinID string, dayStartUTC time.Time, granularitySeconds time.Duration) error {

	to := dayStartUTC.Add(24*time.Hour - time.Second)

	// CoinGecko returns points; per our agreement we normalize sequentially into buckets without flooring by timestamp.
	resp, err := u.cgClient.CoinsMarketChartRange(ctx, coinID, "usd", dayStartUTC, to, &precision)
	if err != nil {
		return fmt.Errorf("%w: %v", apperr.ErrProviderUnavailable, err)
	}

	if resp == nil || len(resp.Prices) == 0 {
		return fmt.Errorf("%w: empty prices for coin=%s day=%s", apperr.ErrProviderBadResponse, coinID, dayStartUTC.Format(time.DateOnly))
	}

	// normalize points to buckets "by order"
	buckets, err := normalizeByOrder(coinID, dayStartUTC, granularitySeconds, resp.Prices)
	if err != nil {
		u.logger.Error("normalizeByOrder failed", "coinID", coinID, "dayStartUTC", dayStartUTC, "error", err)
		return fmt.Errorf("%w: %v", apperr.ErrProviderBadResponse, err)
	}

	if err := u.repo.UpsertBatch(ctx, buckets); err != nil {
		return fmt.Errorf("repo.UpsertBatch: %w", err)
	}

	return nil
}
