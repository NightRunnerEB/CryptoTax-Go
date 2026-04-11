package fiatfx

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UnlockFunc func(context.Context) error

type UpdateLocker interface {
	TryLock(ctx context.Context, key int64) (UnlockFunc, bool, error)
}

type PGAdvisoryLocker struct {
	pool *pgxpool.Pool
}

func NewPGAdvisoryLocker(pool *pgxpool.Pool) *PGAdvisoryLocker {
	return &PGAdvisoryLocker{pool: pool}
}

func (l *PGAdvisoryLocker) TryLock(ctx context.Context, key int64) (UnlockFunc, bool, error) {
	if l == nil || l.pool == nil {
		return nil, false, fmt.Errorf("pg advisory locker is not initialized")
	}

	conn, err := l.pool.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}

	var locked bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&locked); err != nil {
		conn.Release()
		return nil, false, err
	}

	if !locked {
		conn.Release()
		return nil, false, nil
	}

	unlock := func(ctx context.Context) error {
		var unlocked bool
		err := conn.QueryRow(ctx, "SELECT pg_advisory_unlock($1)", key).Scan(&unlocked)
		conn.Release()
		if err != nil {
			return err
		}
		if !unlocked {
			return fmt.Errorf("pg advisory unlock returned false for key %d", key)
		}
		return nil
	}

	return unlock, true, nil
}
