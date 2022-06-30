package nakama

import (
	"context"
	"database/sql"
	_ "embed"
)

//go:embed schema.sql
var sqlSchema string

func MigrateSQL(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, sqlSchema)
	return err
}
