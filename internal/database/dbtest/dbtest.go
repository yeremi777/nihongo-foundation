//go:build integration

// Package dbtest runs integration tests against a throwaway test database,
// never the .env one.
package dbtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Main runs a package's integration tests against the database TEST_DB_DSN
// names, and exits. It rebuilds the schema from the migrations, hands the
// connection to setup, runs the tests, and empties the database again, so
// no table outlives the run. It refuses any database whose name does not
// start with "test".
func Main(m *testing.M, setup func(ctx context.Context, conn *pgx.Conn) error) {
	ctx := context.Background()
	conn, err := open(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := 1
	if err := applyMigrations(ctx, conn); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if err := setup(ctx, conn); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		code = m.Run()
	}
	if err := emptySchema(ctx, conn); err != nil {
		fmt.Fprintln(os.Stderr, "empty the test database:", err)
		code = 1
	}
	conn.Close(ctx)
	os.Exit(code)
}

// open connects to the database TEST_DB_DSN names, refusing one whose name
// does not start with "test".
func open(ctx context.Context) (*pgx.Conn, error) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("TEST_DB_DSN is not set; run make test-integration")
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	var name string
	if err := conn.QueryRow(ctx, "SELECT current_database()").Scan(&name); err != nil || !strings.HasPrefix(name, "test") {
		conn.Close(ctx)
		return nil, fmt.Errorf("refusing to reset database %q: its name must start with \"test\" (%v)", name, err)
	}
	return conn, nil
}

// emptySchema drops every table, leaving an empty public schema.
func emptySchema(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public")
	return err
}

// applyMigrations empties the schema and runs the goose Up section of every
// migration, oldest first.
func applyMigrations(ctx context.Context, conn *pgx.Conn) error {
	if err := emptySchema(ctx, conn); err != nil {
		return err
	}
	_, here, _, _ := runtime.Caller(0)
	files, err := filepath.Glob(filepath.Join(filepath.Dir(here), "..", "migrations", "*.sql"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no migrations found next to %s", here)
	}
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		up, _, ok := strings.Cut(string(b), "-- +goose Down")
		if !ok {
			return fmt.Errorf("%s has no Down section", f)
		}
		if _, err := conn.PgConn().Exec(ctx, up).ReadAll(); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
	}
	return nil
}
