package testutil

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// WantEq -
func WantEq(t *testing.T, want, got any, msg string) {
	t.Helper()

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("%s: want %v; got %v", msg, want, got)
	}
}

// RandStr -
func RandStr(t *testing.T, size int) string {
	t.Helper()

	s, err := gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz", size)
	WantEq(t, nil, err, "nanoid")
	return s
}

// CurrentDir .
func CurrentDir(t *testing.T) string {
	t.Helper()

	_, file, _, _ := runtime.Caller(1)
	return filepath.Dir(file)
}
