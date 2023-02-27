package nakama

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nakamauwu/nakama/crdbpgx"
)

var ctxKeyTx = struct{ name string }{"ctx-key-tx"}

func contextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, ctxKeyTx, tx)
}

func txFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(ctxKeyTx).(pgx.Tx)
	return tx, ok
}

// Store wrapper for SQL database with better semantics to run transactions.
type Store struct {
	pool           *pgxpool.Pool
	AvatarScanFunc func(dest **string) sql.Scanner
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		pool:           pool,
		AvatarScanFunc: MakePrefixedNullStringScanner(""),
	}
}

// QueryRow executes a query that is expected to return at most one row.
func (db *Store) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if tx, ok := txFromContext(ctx); ok {
		return tx.QueryRow(ctx, query, args...)
	}
	return db.pool.QueryRow(ctx, query, args...)
}

// Query executes a query that returns rows, typically a SELECT.
func (db *Store) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	if tx, ok := txFromContext(ctx); ok {
		return tx.Query(ctx, query, args...)
	}
	return db.pool.Query(ctx, query, args...)
}

// Exec executes a query without returning any rows.
func (db *Store) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if tx, ok := txFromContext(ctx); ok {
		return tx.Exec(ctx, query, args...)
	}
	return db.pool.Exec(ctx, query, args...)
}

// RunTx will start a new SQL transaction and hold a reference
// to the transaction inside the context.
// Next calls within the txFunc will use the new transaction from the context.
func (db *Store) RunTx(ctx context.Context, txFunc func(ctx context.Context) error) error {
	if _, ok := txFromContext(ctx); ok {
		return txFunc(ctx)
	}

	return crdbpgx.ExecuteTx(ctx, db.pool, func(tx pgx.Tx) error {
		return txFunc(contextWithTx(ctx, tx))
	})
}

func isPgError(err error, code string, columns ...string) bool {
	var e *pgconn.PgError
	if !errors.As(err, &e) {
		return false
	}

	if e.Code != code {
		return false
	}

	if len(columns) == 0 {
		return true
	}

	for _, col := range columns {
		if strings.Contains(strings.ToLower(e.Error()), strings.ToLower(col)) {
			return true
		}
	}

	return false
}

func isForeignKeyViolationError(err error, columns ...string) bool {
	return isPgError(err, pgerrcode.ForeignKeyViolation, columns...)
}

func MakePrefixedNullStringScanner(prefix string) func(**string) sql.Scanner {
	return func(dest **string) sql.Scanner {
		return &prefixedNullStringScanner{Prefix: prefix, Dest: dest}
	}
}

type prefixedNullStringScanner struct {
	Prefix string
	Dest   **string
}

func (s *prefixedNullStringScanner) Scan(src any) error {
	str, ok := src.(string)
	if !ok {
		return nil
	}

	*s.Dest = ptr(s.Prefix + str)
	return nil
}
