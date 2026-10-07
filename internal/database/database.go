// Package database opens the Postgres connection the commands use.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const connectTimeout = 10 * time.Second

// Connect opens one connection, giving up after connectTimeout.
func Connect(ctx context.Context, dsn string) (*pgx.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	return conn, nil
}
