package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	_defaultMaxPoolSize    = 1
	_defaultConnAttempts   = 10
	_defaultConnTimeout    = 3 * time.Second
	_defaultAttemptTimeout = 3 * time.Second
)

type Postgres struct {
	maxPoolSize    int
	connAttempts   int
	connTimeout    time.Duration
	attemptTimeout time.Duration

	Pool    *pgxpool.Pool
	Builder squirrel.StatementBuilderType
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

	pg.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
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
