package fs

import (
	"testing"

	"github.com/nakamauwu/nakama/storage/tests"
)

func TestStore(t *testing.T) {
	tests.RunStoreTests(t, &Store{
		Root: t.TempDir(),
	}, "")
}
