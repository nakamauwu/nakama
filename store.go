package nakama

import (
	"context"

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

func (s *Store) applyAvatarPrefix(dest **string) {
	if dest == nil || *dest == nil {
		return
	}

	*dest = ptr(s.s3Prefix + S3BucketAvatars + **dest)
}

func (s *Store) applyMediaPrefix(dest *[]Media) {
	for i, m := range *dest {
		if m.IsImage() {
			img := m.AsImage
			if img.Path == "" {
				continue
			}

			img.Path = s.s3Prefix + S3BucketMedia + img.Path
			(*dest)[i].AsImage = img
		}
	}
}
