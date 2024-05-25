package cockroach

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasparada/go-db"
)

type Cockroach struct {
	db *db.DB
}

func New(pool *pgxpool.Pool) *Cockroach {
	return &Cockroach{
		db: db.New(pool),
	}
}
