package db

import (
	"context"
	"database/sql"

	"github.com/cockroachdb/cockroach-go/v2/crdb"
)

var ctxKeyTx = struct{ name string }{"ctx-key-tx"}

func contextWithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, ctxKeyTx, tx)
}

func txFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(ctxKeyTx).(*sql.Tx)
	return tx, ok
}

// DB wrapper for SQL database with better semantics to run transactions.
type DB struct {
	pool *sql.DB
}

// New database.
func New(pool *sql.DB) *DB {
	return &DB{pool: pool}
}

// QueryRowContext executes a query that is expected to return at most one row.
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if tx, ok := txFromContext(ctx); ok {
		return tx.QueryRowContext(ctx, query, args...)
	}
	return db.pool.QueryRowContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows, typically a SELECT.
func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if tx, ok := txFromContext(ctx); ok {
		return tx.QueryContext(ctx, query, args...)
	}
	return db.pool.QueryContext(ctx, query, args...)
}

// ExecContext executes a query without returning any rows.
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx, ok := txFromContext(ctx); ok {
		return tx.ExecContext(ctx, query, args...)
	}
	return db.pool.ExecContext(ctx, query, args...)
}

// RunTx will start a new SQL transaction and hold a reference
// to the transaction inside the context.
// Next calls within the txFunc will use the new transaction from the context.
func (db *DB) RunTx(ctx context.Context, txFunc func(ctx context.Context) error) error {
	if _, ok := txFromContext(ctx); ok {
		return txFunc(ctx)
	}

	return crdb.ExecuteTx(ctx, db.pool, nil, func(tx *sql.Tx) error {
		return txFunc(contextWithTx(ctx, tx))
	})
}

// ScanFunc copies the columns in the current row into the values pointed
// at by dest. The number of values in dest must be the same as the
// number of columns in the selected rows.
type ScanFunc func(dest ...any) error

// Collect rows into a slice.
func Collect[T any](rows *sql.Rows, scanFunc func(scan ScanFunc) (T, error)) ([]T, error) {
	defer rows.Close()

	var out []T
	for rows.Next() {
		item, err := scanFunc(rows.Scan)
		if err != nil {
			return out, err
		}

		out = append(out, item)
	}

	return out, rows.Err()
}
