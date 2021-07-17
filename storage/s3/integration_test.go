package s3

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/ory/dockertest/v3"
)

const (
	testRegion    = "test_region"
	testAccessKey = "test_access_key"
	testSecretKey = "test_secret_key"
)

var testEndpoint string

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	pool, err := dockertest.NewPool("")
	if err != nil {
		fmt.Printf("could not create docker pool: %v\n", err)
		return 1
	}

	cleanup, err := setupTestMinio(pool)
	if err != nil {
		fmt.Printf("could not setup test minio: %v\n", err)
		return 1
	}

	defer func() {
		if err := cleanup(); err != nil {
			fmt.Printf("could not cleanup minio container: %v\n", err)
		}
	}()

	return m.Run()
}

func setupTestMinio(pool *dockertest.Pool) (func() error, error) {
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "minio/minio",
		Env: []string{
			fmt.Sprintf("MINIO_ACCESS_KEY=%s", testAccessKey),
			fmt.Sprintf("MINIO_SECRET_KEY=%s", testSecretKey),
		},
		Cmd: []string{"server", "/data"},
	})
	if err != nil {
		return nil, fmt.Errorf("could not create minio resource: %w", err)
	}

	err = pool.Retry(func() (err error) {
		testEndpoint = resource.GetHostPort("9000/tcp")
		_, err = minio.New(testEndpoint, &minio.Options{
			Secure: true,
		})
		if err != nil {
			return fmt.Errorf("could not create minio client: %w", err)
		}

		resp, err := http.Get("http://" + testEndpoint + "/minio/health/live")
		if err != nil {
			return fmt.Errorf("could not request minio health endpoint: %w", err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.New("minio not alive")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return func() error {
		return pool.Purge(resource)
	}, nil
}
