package s3

import (
	"context"
	"testing"

	"github.com/nakamauwu/nakama/storage/tests"
)

func TestStore(t *testing.T) {
	s := &Store{
		Endpoint:   testEndpoint,
		Region:     testRegion,
		AccessKey:  testAccessKey,
		SecretKey:  testSecretKey,
		BucketList: []string{testBucket},
	}
	if err := s.Setup(context.Background()); err != nil {
		t.Fatal(err)
	}
	tests.RunStoreTests(t, s, testBucket)
}
