package db

import (
	"context"

	sqlc "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store defines all functions to execute db queries and transactions
type Store interface {
	sqlc.Querier
	ExecTx(ctx context.Context, fn func(q *sqlc.Queries) error) error
	// SomeTx(ctx context.Context, arg SomeTxParams) (SomeTxResult, error) - sqlc генерирует 1 метод на запрос, а если нужно несколько запросов в транзакции, то можно это делать здесь
}

type PGStore struct {
	*sqlc.Queries
	pool *pgxpool.Pool
}

func NewStore(pg *postgres.Postgres) Store {
	return &PGStore{
		pool:    pg.Pool,
		Queries: sqlc.New(pg.Pool),
	}
}

/*
Example of transaction usage:
	err := store.ExecTx(ctx, func(q *sqlc.Queries) error {
		if err := q.UpsertTenantSymbol(ctx, params1); err != nil { return err }
		if err := q.UpsertHistoricalPrice(ctx, params2); err != nil { return err }
		return nil
	})
*/
func (s *PGStore) ExecTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.Queries.WithTx(tx)
	if err := fn(qtx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
