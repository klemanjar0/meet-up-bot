package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"meet-up-bot/internal/storage/db"
)

// Store wraps a pgx connection pool together with the sqlc-generated queries.
type Store struct {
	*db.Queries
	pool *pgxpool.Pool
}

// New opens a connection pool, verifies connectivity, and returns a Store.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Store{
		Queries: db.New(pool),
		pool:    pool,
	}, nil
}

// Close releases all pool connections.
func (s *Store) Close() {
	s.pool.Close()
}
