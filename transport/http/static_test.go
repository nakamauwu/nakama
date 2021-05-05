package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/nicolasparada/nakama/testutil"
)

func Test_spaFileSystem(t *testing.T) {
	tt := []struct {
		name            string
		embed           bool
		path            string
		wantContentType string
	}{
		{
			name:            "embeded_file_system",
			embed:           true,
			path:            "/index.html",
			wantContentType: "text/html; charset=utf-8",
		},
		{
			name:            "os_file_system",
			embed:           false,
			path:            "/index.html",
			wantContentType: "text/html; charset=utf-8",
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			h := New(nil, log.NewNopLogger(), nil, nil, tc.embed)
			srv := httptest.NewServer(h)
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, srv.URL+tc.path, nil)
			if err != nil {
				t.Fatalf("failed to create request to check spa: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to do request to check spa: %v", err)
			}

			defer resp.Body.Close()

			body := readAllAndTrim(t, resp.Body)
			t.Logf("\n%s body:\n%s\n\n", tc.path, body)

			ct := http.DetectContentType(body)
			testutil.AssertEqual(t, tc.wantContentType, ct, "content type")
		})
	}
}
