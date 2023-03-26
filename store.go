package nakama

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasparada/go-db"
)

// Store wrapper for SQL database with better semantics to run transactions.
type Store struct {
	db       *db.DB
	s3Prefix string
}

func NewStore(pool *pgxpool.Pool, s3Prefix string) *Store {
	return &Store{
		db:       db.New(pool),
		s3Prefix: s3Prefix,
	}
}

func (s *Store) RunTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.db.RunTx(ctx, fn)
}

func (s *Store) scanAvatar(dest **string) sql.Scanner {
	return &prefixedNullStringScanner{
		Prefix: s.s3Prefix + S3BucketAvatars + "/",
		Dest:   dest,
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
