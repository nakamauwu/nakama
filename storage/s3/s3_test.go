package s3

import (
	"testing"

	"github.com/nicolasparada/nakama/storage/tests"
)

func TestStore(t *testing.T) {
	tests.RunStoreTests(t, &Store{
		Endpoint:   testEndpoint,
		Region:     testRegion,
		AccessKey:  testAccessKey,
		SecretKey:  testSecretKey,
		BucketList: []string{testBucket},
	}, testBucket)
}
