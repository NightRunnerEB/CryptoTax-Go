package db

import (
	"github.com/NightRunner/CryptoTax-Go/pkg/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store defines all functions to execute db queries and transactions
type Store interface {
	Querier
	// SomeTx(ctx context.Context, arg SomeTxParams) (SomeTxResult, error) // by execTx
}

type SQLStore struct {
	*Queries
	connPool *pgxpool.Pool
}

func NewStore(pg *postgres.Postgres) Store {
	return &SQLStore{
		Queries:  New(pg.Pool),
		connPool: pg.Pool,
	}
}
