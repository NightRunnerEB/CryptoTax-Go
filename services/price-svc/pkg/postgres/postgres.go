package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
    _defaultMaxPoolSize    = 10
    _defaultConnAttempts   = 3
    _defaultConnTimeout    = 5 * time.Second
    _defaultAttemptTimeout = 2 * time.Second
)

type Postgres struct {
	maxPoolSize    int
	connAttempts   int
	connTimeout    time.Duration
	attemptTimeout time.Duration

	Pool *pgxpool.Pool
}

func New(url string, opts ...Option) (*Postgres, error) {
	pg := &Postgres{
		maxPoolSize:    _defaultMaxPoolSize,
		connAttempts:   _defaultConnAttempts,
		connTimeout:    _defaultConnTimeout,
		attemptTimeout: _defaultAttemptTimeout,
	}

	for _, opt := range opts {
		opt(pg)
	}

	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Postgres pool config: %w", err)
	}

	poolConfig.MaxConns = int32(pg.maxPoolSize)

	for pg.connAttempts > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), pg.connTimeout)
		pg.Pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		cancel()

		if err == nil {
			return pg, nil
		}

		log.Printf("Postgres is trying to connect, attempts left: %d", pg.connAttempts)

		time.Sleep(pg.attemptTimeout)

		pg.connAttempts--
	}

	return nil, fmt.Errorf("postgres - NewPostgres - connAttempts == 0: %w", err)
}

func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}

func (p *Postgres) WithTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	tx, err := p.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
