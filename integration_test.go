package nakama

import (
	"context"
	"fmt"
	"io"
	mathrand "math/rand"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ory/dockertest/v3"
)

var testService *Service

func TestMain(m *testing.M) {
	code, err := setupT(m)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "setupT() failed: %v\n", err)
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

	dbPool, err := setupDB(cockroach, pool.Retry)
	if err != nil {
		return 0, err
	}

	defer dbPool.Close()

	if err := MigrateSQL(context.Background(), dbPool); err != nil {
		return 0, err
	}

	testService = &Service{
		Store:       NewStore(dbPool, "localhost/"),
		Logger:      log.New(io.Discard),
		BaseContext: context.Background,
	}

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

func setupDB(cockroach *dockertest.Resource, retry func(op func() error) error) (*pgxpool.Pool, error) {
	ctx := context.Background()

	var db *pgxpool.Pool
	return db, retry(func() (err error) {
		hostPort := cockroach.GetHostPort("26257/tcp")
		db, err = pgxpool.New(ctx, "postgresql://root@"+hostPort+"/defaultdb?sslmode=disable")
		if err != nil {
			return err
		}

		return db.Ping(ctx)
	})
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randString(n int) string {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[r.Intn(len(letterRunes))]
	}
	return string(b)
}
