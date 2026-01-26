package repository

import (
	"math/big"
	"time"

	sqlc "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func mapHistoricalPriceRowDBToDomain(h sqlc.GetHistoricalPricesBatchRow) (domain.HistoricalPrice, error) {
	price := numericToDecimal(h.PriceUsd)

	var gsPtr *int
	if h.GranularitySeconds != nil {
		gs := int(*h.GranularitySeconds)
		gsPtr = &gs
	}

	return domain.HistoricalPrice{
		CoinID:             h.CoinID,
		Time:               h.BucketStartUtc.Time, // guaranteed to be valid
		PriceUsd:           price,
		GranularitySeconds: gsPtr,
	}, nil
}

func mapHistoricalPriceDBToDomain(h sqlc.HistoricalPrice) (domain.HistoricalPrice, error) {
	price := numericToDecimal(h.PriceUsd)

	granularitySeconds := int(h.GranularitySeconds)

	return domain.HistoricalPrice{
		CoinID:             h.CoinID,
		Time:               h.BucketStartUtc.Time, // guaranteed to be valid
		PriceUsd:           price,
		GranularitySeconds: &granularitySeconds,
	}, nil
}

func mapTenantSymbolDBToDomain(s sqlc.TenantSymbol) domain.TenantSymbol {
	return domain.TenantSymbol{
		TenantID: s.TenantID,
		Source:   s.Source,
		Symbol:   s.Symbol,
		CoinID:   s.CoinID,
	}
}

func numericToDecimal(n pgtype.Numeric) *decimal.Decimal {
	if !n.Valid {
		return nil
	}

	bi := n.Int
	if bi == nil {
		bi = big.NewInt(0)
	}

	result := decimal.NewFromBigInt(bi, n.Exp)

	return &result
}

func decimalToNumeric(d *decimal.Decimal) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(d.String()); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}

func toTimestamptzSlice(times []time.Time) []pgtype.Timestamptz {
	res := make([]pgtype.Timestamptz, len(times))
	for i, t := range times {
		res[i] = pgtype.Timestamptz{
			Time:  t,
			Valid: true,
		}
	}
	return res
}
