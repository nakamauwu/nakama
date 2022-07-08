package nakama

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func Test_isID(t *testing.T) {
	tt := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"cb3o6q7semviu8u64co0", true},
	}
	for _, tc := range tt {
		t.Run(tc.in, func(t *testing.T) {
			got := isID(tc.in)
			assert.Equal(t, tc.want, got, "isID(%q) = %v, want %v", tc.in, got, tc.want)
		})
	}
}
