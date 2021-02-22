package testutil

import (
	"reflect"
	"testing"

	gonanoid "github.com/matoous/go-nanoid"
)

// AssertEqual -
func AssertEqual(t *testing.T, want, got interface{}, msg string) {
	t.Helper()

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("%s: want %v; got %v", msg, want, got)
	}
}

// RandStr -
func RandStr(t *testing.T, size int) string {
	s, err := gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz", size)
	AssertEqual(t, nil, err, "nanoid")
	return s
}
