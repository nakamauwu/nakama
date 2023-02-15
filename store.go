package nakama

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroach-go/v2/crdb"
	"github.com/lib/pq"
)

var ctxKeyTx = struct{ name string }{"ctx-key-tx"}

func contextWithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, ctxKeyTx, tx)
}

func txFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(ctxKeyTx).(*sql.Tx)
	return tx, ok
}

// Store wrapper for SQL database with better semantics to run transactions.
type Store struct {
	pool           *sql.DB
	AvatarScanFunc func(dest **string) sql.Scanner
}

func NewStore(pool *sql.DB) *Store {
	return &Store{
		pool:           pool,
		AvatarScanFunc: MakePrefixedNullStringScanner(""),
	}
}

// QueryRowContext executes a query that is expected to return at most one row.
func (db *Store) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if tx, ok := txFromContext(ctx); ok {
		return tx.QueryRowContext(ctx, query, args...)
	}
	return db.pool.QueryRowContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows, typically a SELECT.
func (db *Store) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if tx, ok := txFromContext(ctx); ok {
		return tx.QueryContext(ctx, query, args...)
	}
	return db.pool.QueryContext(ctx, query, args...)
}

// ExecContext executes a query without returning any rows.
func (db *Store) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx, ok := txFromContext(ctx); ok {
		return tx.ExecContext(ctx, query, args...)
	}
	return db.pool.ExecContext(ctx, query, args...)
}

// RunTx will start a new SQL transaction and hold a reference
// to the transaction inside the context.
// Next calls within the txFunc will use the new transaction from the context.
func (db *Store) RunTx(ctx context.Context, txFunc func(ctx context.Context) error) error {
	if _, ok := txFromContext(ctx); ok {
		return txFunc(ctx)
	}

	return crdb.ExecuteTx(ctx, db.pool, nil, func(tx *sql.Tx) error {
		return txFunc(contextWithTx(ctx, tx))
	})
}

// scanner copies the columns in the current row into the values pointed
// at by dest. The number of values in dest must be the same as the
// number of columns in the selected rows.
type scanner interface {
	Scan(dest ...any) error
}

// collect rows into a slice.
func collect[T any](rows *sql.Rows, fn func(scanner scanner) (T, error)) ([]T, error) {
	defer rows.Close()

	var out []T
	for rows.Next() {
		item, err := fn(rows)
		if err != nil {
			return out, err
		}

		out = append(out, item)
	}

	return out, rows.Err()
}

func isPqError(err error, codeName string, columns ...string) bool {
	var e *pq.Error
	if !errors.As(err, &e) {
		return false
	}

	if e.Code.Name() != codeName {
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

// func isPqNotNullViolationError(err error, columns ...string) bool {
// 	return isPqError(err, "not_null_violation", columns...)
// }

func isPqForeignKeyViolationError(err error, columns ...string) bool {
	return isPqError(err, "foreign_key_violation", columns...)
}

// func isPqUniqueViolationError(err error, columns ...string) bool {
// 	return isPqError(err, "unique_violation", columns...)
// }

type jsonValue struct {
	Dst any
}

// Value implements sql driver Valuer interface.
func (jv jsonValue) Value() (driver.Value, error) {
	if jv.Dst == nil {
		return nil, nil
	}

	var buff bytes.Buffer
	enc := json.NewEncoder(&buff)
	enc.SetEscapeHTML(false)
	err := enc.Encode(jv.Dst)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), err
}

// Scan implements sql driver scanner interface.
func (jv *jsonValue) Scan(value any) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unexpected json, got %T", value)
	}

	return json.Unmarshal(b, &jv.Dst)
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
