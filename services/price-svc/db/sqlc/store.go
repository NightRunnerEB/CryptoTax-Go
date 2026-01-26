package db

import (
	"context"
	"fmt"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/postgres"
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

// ExecTx executes a function within a database transaction
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.connPool.Begin(ctx)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}
