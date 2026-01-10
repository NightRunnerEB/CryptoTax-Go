package repository

import (
	"fmt"
	"math/big"

	sqlc "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func mapHistoricalPriceDBToDomain(h sqlc.HistoricalPrice) (domain.HistoricalPrice, error) {
	price, err := numericToDecimal(h.PriceUsd)
	if err != nil {
		return domain.HistoricalPrice{}, err
	}

	return domain.HistoricalPrice{
		CoinID:             h.CoinID,
		BucketStartUtc:     h.BucketStartUtc,
		PriceUsd:           price,
		GranularitySeconds: int(h.GranularitySeconds),
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

func numericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, fmt.Errorf("NULL numeric")
	}
	if n.NaN {
		return decimal.Zero, fmt.Errorf("cannot convert NaN to decimal")
	}
	if n.InfinityModifier != 0 {
		return decimal.Zero, fmt.Errorf("cannot convert Infinity to decimal")
	}

	bi := n.Int
	if bi == nil {
		bi = big.NewInt(0)
	}

	return decimal.NewFromBigInt(bi, n.Exp), nil
}

func decimalToNumeric(d decimal.Decimal) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(d.String()); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}
