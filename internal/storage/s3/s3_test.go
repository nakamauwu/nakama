package s3

import (
	"testing"

	"github.com/nicolasparada/nakama/internal/storage/tests"
)

func TestStore(t *testing.T) {
	tests.RunStoreTests(t, &Store{
		Endpoint:  testEndpoint,
		Region:    testRegion,
		Bucket:    testBucket,
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
	})
}
