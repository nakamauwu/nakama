package crdbpgx

import (
	"context"

	"github.com/cockroachdb/cockroach-go/v2/crdb"
	"github.com/jackc/pgx/v5"
)

// ExecuteTx runs fn inside a transaction and retries it as needed. On
// non-retryable failures, the transaction is aborted and rolled back; on
// success, the transaction is committed.
//
// See crdb.ExecuteTx() for more information.
//
// conn can be a pgx.Conn or a pgxpool.Pool.
func ExecuteTx(
	ctx context.Context, conn Conn, fn func(pgx.Tx) error,
) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	return crdb.ExecuteInTx(ctx, pgxTxAdapter{tx}, func() error { return fn(tx) })
}

// Conn abstracts pgx transactions creators: pgx.Conn and pgxpool.Pool.
type Conn interface {
	Begin(context.Context) (pgx.Tx, error)
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type pgxTxAdapter struct {
	tx pgx.Tx
}

var _ crdb.Tx = pgxTxAdapter{}

func (tx pgxTxAdapter) Commit(ctx context.Context) error {
	return tx.tx.Commit(ctx)
}

func (tx pgxTxAdapter) Rollback(ctx context.Context) error {
	return tx.tx.Rollback(ctx)
}

// Exec is part of the crdb.Tx interface.
func (tx pgxTxAdapter) Exec(ctx context.Context, q string, args ...interface{}) error {
	_, err := tx.tx.Exec(ctx, q, args...)
	return err
}
