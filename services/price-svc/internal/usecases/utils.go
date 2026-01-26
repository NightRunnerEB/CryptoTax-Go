package usecase

import (
	"fmt"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/shopspring/decimal"
)

// normalizeByOrder maps sequential points to sequential buckets.
// For 5m => 288 points; 1h => 24 points; 1d => 1 point.
// We assign the first point to 00:00 bucket, second to 00:05, etc (as agreed).
func normalizeByOrder(
	coinID string,
	dayStartUTC time.Time,
	granularity time.Duration,
	prices [][]float64, // [ [ts_ms, price], ... ]
) ([]domain.HistoricalPrice, error) {
	if granularity <= 0 {
		return nil, fmt.Errorf("bad granularity=%d", granularity)
	}

	out := make([]domain.HistoricalPrice, 0, len(prices))
	for i, pt := range prices {
		if len(pt) < 2 {
			return nil, fmt.Errorf("bad point at idx=%d", i)
		}
		price := pt[1]

		// bucket start strictly by index
		bucket := dayStartUTC.Add(time.Duration(i) * granularity)

		dec := decimal.NewFromFloat(price)
		g := int(granularity / time.Second)

		out = append(out, domain.HistoricalPrice{
			CoinID:             coinID,
			Time:               bucket,
			PriceUsd:           &dec,
			GranularitySeconds: &g,
		})
	}
	return out, nil
}

func truncateDayUTC(t time.Time) time.Time {
	t = t.UTC()
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func floorToBucket(t time.Time, granularity time.Duration) time.Time {
	t = t.UTC()
	g := int64(granularity / time.Second)
	sec := t.Unix()
	floored := (sec / g) * g
	return time.Unix(floored, 0).UTC()
}
