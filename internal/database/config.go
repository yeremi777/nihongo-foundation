// Package database opens the Postgres connection the commands use.
package database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const connectTimeout = 10 * time.Second

// DSNFromEnv builds a keyword/value connection string from the DB_* variables.
// DB_PASSWORD may be empty; every other variable is required.
func DSNFromEnv(getenv func(string) string) (string, error) {
	keys := []struct{ env, keyword string }{
		{"DB_HOST", "host"},
		{"DB_PORT", "port"},
		{"DB_NAME", "dbname"},
		{"DB_USERNAME", "user"},
		{"DB_PASSWORD", "password"},
		{"DB_SSLMODE", "sslmode"},
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		value := getenv(k.env)
		if value == "" && k.env != "DB_PASSWORD" {
			return "", fmt.Errorf("%s is not set; copy .env.example to .env", k.env)
		}
		if k.env == "DB_PORT" {
			if port, err := strconv.Atoi(value); err != nil || port < 1 || port > 65535 {
				return "", fmt.Errorf("DB_PORT %q is not a port number", value)
			}
		}
		parts = append(parts, k.keyword+"="+quote(value))
	}
	return strings.Join(parts, " "), nil
}

// quote writes a keyword/value value so spaces, quotes, and backslashes survive parsing.
func quote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)
	return "'" + value + "'"
}

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
