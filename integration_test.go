package nakama

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
)

var testQueries *Queries

func TestMain(m *testing.M) {
	code, err := setupT(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setupT() failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func setupT(m *testing.M) (int, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return 0, err
	}

	if err := pool.Client.Ping(); err != nil {
		return 0, err
	}

	cockroach, err := setupCockroach(pool)
	if err != nil {
		return 0, err
	}

	defer cockroach.Close()

	db, err := setupDB(cockroach, pool.Retry)
	if err != nil {
		return 0, err
	}

	defer db.Close()

	if err := MigrateSQL(context.Background(), db); err != nil {
		return 0, err
	}

	testQueries = New(db)

	return m.Run(), nil
}

func setupCockroach(pool *dockertest.Pool) (*dockertest.Resource, error) {
	return pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "cockroachdb/cockroach",
		Tag:        "latest",
		Cmd: []string{"start-single-node",
			"--insecure",
			"--store", "type=mem,size=0.25",
			"--advertise-addr", "localhost",
		},
	})
}

func setupDB(cockroach *dockertest.Resource, retry func(op func() error) error) (*sql.DB, error) {
	var db *sql.DB
	return db, retry(func() (err error) {
		hostPort := cockroach.GetHostPort("26257/tcp")
		db, err = sql.Open("postgres", "postgresql://root@"+hostPort+"/defaultdb?sslmode=disable")
		if err != nil {
			return err
		}

		return db.Ping()
	})
}
