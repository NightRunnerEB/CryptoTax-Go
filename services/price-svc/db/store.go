package db

import (
	sqlc "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/postgres"
)

// Store defines all functions to execute db queries and transactions
type Store interface {
	sqlc.Querier
	// SomeTx(ctx context.Context, arg SomeTxParams) (SomeTxResult, error)
}

type PGStore struct {
	*postgres.Postgres
	*sqlc.Queries
}

// NewStore creates a new store
func NewStore(pg *postgres.Postgres) Store {
	return &PGStore{
		Postgres: pg,
		Queries:  sqlc.New(pg.ConnPool()),
	}
}

// func (store *PGStore) execTx(ctx context.Context, fn func(*sqlc.Querier) error) error {
// 	tx, err := store.connPool.Begin(ctx)
// 	if err != nil {
// 		return err
// 	}

// 	q := sqlc.New(tx)
// 	err = fn(q)
// 	if err != nil {
// 		if rbErr := tx.Rollback(ctx); rbErr != nil {
// 			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
// 		}
// 		return err
// 	}

// 	return tx.Commit(ctx)
// }
