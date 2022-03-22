package tests

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/nakamauwu/nakama/storage"
	"github.com/nakamauwu/nakama/testutil"
)

func RunStoreTests(t *testing.T, store storage.Store, bucket string) {
	ctx := context.Background()

	logoName := "go_logo.png"
	logoContentType := "image/png"

	logoFilePath := filepath.Join(testutil.CurrentDir(t), "testdata", logoName)
	logoFile, err := os.Open(logoFilePath)
	testutil.WantEq(t, nil, err, "os open")

	t.Cleanup(func() { logoFile.Close() })

	logoBytes, err := io.ReadAll(logoFile)
	testutil.WantEq(t, nil, err, "io read-all")

	/* -------------------------------- end setup ------------------------------- */

	err = store.Store(ctx, bucket, logoName, logoBytes, storage.StoreWithContentType(logoContentType))
	testutil.WantEq(t, nil, err, "error")

	f, err := store.Open(ctx, bucket, logoName)
	testutil.WantEq(t, nil, err, "error")

	t.Cleanup(func() { f.Close() })

	gotBytes, err := io.ReadAll(f)
	testutil.WantEq(t, nil, err, "error")

	testutil.WantEq(t, int64(len(logoBytes)), f.Size, "size")
	testutil.WantEq(t, logoContentType, f.ContentType, "content-type")
	testutil.WantEq(t, logoBytes, gotBytes, "bytes")

	err = store.Delete(ctx, bucket, logoName)
	testutil.WantEq(t, nil, err, "error")
}
