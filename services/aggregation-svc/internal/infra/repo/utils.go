package repository

import (
	"encoding/json"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func fromTimestamptz(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}

func moneyLegToJSON(leg *domain.MoneyLeg) ([]byte, error) {
	if leg == nil {
		return nil, nil
	}
	return json.Marshal(leg)
}

func moneyLegFromJSON(b []byte) (*domain.MoneyLeg, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var leg domain.MoneyLeg
	if err := json.Unmarshal(b, &leg); err != nil {
		return nil, err
	}
	return &leg, nil
}
