package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/NightRunner/CryptoTax-Go/pkg/postgres"
)

type Store interface {
	Querier
	ExecTx(ctx context.Context, fn func(*Queries) error) error
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

func (store *SQLStore) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.connPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := store.Queries.WithTx(tx)
	if err := fn(q); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ Store = (*SQLStore)(nil)
