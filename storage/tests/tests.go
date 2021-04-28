package tests

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/nicolasparada/nakama/storage"
	"github.com/nicolasparada/nakama/testutil"
)

func RunStoreTests(t *testing.T, store storage.Store) {
	ctx := context.Background()

	logoName := "go_logo.png"
	logoContentType := "image/png"

	logoFilePath := filepath.Join(testutil.CurrentDir(t), "testdata", logoName)
	logoFile, err := os.Open(logoFilePath)
	testutil.AssertEqual(t, nil, err, "os open")

	t.Cleanup(func() { logoFile.Close() })

	logoBytes, err := io.ReadAll(logoFile)
	testutil.AssertEqual(t, nil, err, "io read-all")

	/* -------------------------------- end setup ------------------------------- */

	err = store.Store(ctx, logoName, logoBytes, storage.StoreWithContentType(logoContentType))
	testutil.AssertEqual(t, nil, err, "error")

	f, err := store.Open(ctx, logoName)
	testutil.AssertEqual(t, nil, err, "error")

	t.Cleanup(func() { f.Close() })

	gotBytes, err := io.ReadAll(f)
	testutil.AssertEqual(t, nil, err, "error")

	testutil.AssertEqual(t, int64(len(logoBytes)), f.Size, "size")
	testutil.AssertEqual(t, logoContentType, f.ContentType, "content-type")
	testutil.AssertEqual(t, logoBytes, gotBytes, "bytes")

	err = store.Delete(ctx, logoName)
	testutil.AssertEqual(t, nil, err, "error")
}
