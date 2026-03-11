package repository

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func fromTimestamptz(v pgtype.Timestamptz) time.Time {
	return v.Time
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	out := v
	return &out
}
