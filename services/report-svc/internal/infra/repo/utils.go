package repository

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func fromTimestamptz(v pgtype.Timestamptz) time.Time {
	return v.Time
}

func toTimestamptzPtr(v *time.Time) *pgtype.Timestamptz {
	if v == nil {
		return nil
	}
	return &pgtype.Timestamptz{
		Time:  *v,
		Valid: true,
	}
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	out := v
	return &out
}
