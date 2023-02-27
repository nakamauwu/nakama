package nakama

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var sqlSchema string

func MigrateSQL(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, sqlSchema)
	return err
}
