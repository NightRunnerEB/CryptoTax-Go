package db

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NightRunner/CryptoTax-Go/pkg/postgres"
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
// TODO: add transactional helpers here if needed.
//
// func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
// 	tx, err := store.connPool.Begin(ctx)
// 	if err != nil {
// 		return err
// 	}
// 	defer func() {
// 		_ = tx.Rollback(ctx)
// 	}()
//
// 	q := New(tx)
// 	if err := fn(q); err != nil {
// 		return err
// 	}
// 	return tx.Commit(ctx)
// }
