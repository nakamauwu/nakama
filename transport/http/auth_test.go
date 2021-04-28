package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicolasparada/nakama"
	"github.com/nicolasparada/nakama/testutil"
)

func Test_handler_sendMagicLink(t *testing.T) {
	type call struct {
		Ctx         context.Context
		Email       string
		RedirectURI string
	}

	tt := []struct {
		name     string
		body     []byte
		svc      *ServiceMock
		testResp func(*testing.T, *http.Response)
		testCall func(*testing.T, call)
	}{
		{
			name: "empty_request_body",
			testResp: func(t *testing.T, resp *http.Response) {
				testutil.AssertEqual(t, http.StatusBadRequest, resp.StatusCode, "status code")
				testutil.AssertEqual(t, "bad request", string(readAllAndTrim(t, resp.Body)), "body")
			},
		},
		{
			name: "malformed_request_body",
			body: []byte(`nope`),
			testResp: func(t *testing.T, resp *http.Response) {
				testutil.AssertEqual(t, http.StatusBadRequest, resp.StatusCode, "status code")
				testutil.AssertEqual(t, "bad request", string(readAllAndTrim(t, resp.Body)), "body")
			},
		},
		{
			name: "invalid_email",
			body: []byte(`{}`),
			svc: &ServiceMock{
				SendMagicLinkFunc: func(context.Context, string, string) error {
					return nakama.ErrInvalidEmail
				},
			},
			testResp: func(t *testing.T, resp *http.Response) {
				testutil.AssertEqual(t, http.StatusUnprocessableEntity, resp.StatusCode, "status code")
				testutil.AssertEqual(t, "invalid email", string(readAllAndTrim(t, resp.Body)), "body")
			},
		},
		{
			name: "invalid_redirect_uri",
			body: []byte(`{}`),
			svc: &ServiceMock{
				SendMagicLinkFunc: func(context.Context, string, string) error {
					return nakama.ErrInvalidRedirectURI
				},
			},
			testResp: func(t *testing.T, resp *http.Response) {
				testutil.AssertEqual(t, http.StatusUnprocessableEntity, resp.StatusCode, "status code")
				testutil.AssertEqual(t, "invalid redirect URI", string(readAllAndTrim(t, resp.Body)), "body")
			},
		},
		{
			name: "untrusted_redirect_uri",
			body: []byte(`{}`),
			svc: &ServiceMock{
				SendMagicLinkFunc: func(context.Context, string, string) error {
					return nakama.ErrUntrustedRedirectURI
				},
			},
			testResp: func(t *testing.T, resp *http.Response) {
				testutil.AssertEqual(t, http.StatusForbidden, resp.StatusCode, "status code")
				testutil.AssertEqual(t, "untrusted redirect URI", string(readAllAndTrim(t, resp.Body)), "body")
			},
		},
		{
			name: "internal_error",
			body: []byte(`{}`),
			svc: &ServiceMock{
				SendMagicLinkFunc: func(context.Context, string, string) error {
					return errors.New("internal error")
				},
			},
			testResp: func(t *testing.T, resp *http.Response) {
				testutil.AssertEqual(t, http.StatusInternalServerError, resp.StatusCode, "status code")
				testutil.AssertEqual(t, "internal server error", string(readAllAndTrim(t, resp.Body)), "body")
			},
		},
		{
			name: "ok",
			body: []byte(`{"email":"user@example.org","redirectURI":"https://example.org"}`),
			svc: &ServiceMock{
				SendMagicLinkFunc: func(context.Context, string, string) error {
					return nil
				},
			},
			testResp: func(t *testing.T, resp *http.Response) {
				testutil.AssertEqual(t, http.StatusNoContent, resp.StatusCode, "status code")
			},
			testCall: func(t *testing.T, call call) {
				testutil.AssertEqual(t, "user@example.org", call.Email, "email")
				testutil.AssertEqual(t, "https://example.org", call.RedirectURI, "redirect URI")
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			h := New(context.Background(), tc.svc, nil, nil, false, true, false)
			srv := httptest.NewServer(h)
			defer srv.Close()

			req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/send_magic_link", bytes.NewReader(tc.body))
			if err != nil {
				t.Fatalf("failed to create request to send magic link: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to do request to send magic link: %v", err)
			}

			defer resp.Body.Close()

			tc.testResp(t, resp)
			if tc.testCall != nil {
				tc.testCall(t, tc.svc.SendMagicLinkCalls()[0])
			}
		})
	}
}

func readAllAndTrim(t *testing.T, r io.Reader) []byte {
	t.Helper()

	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read all from reader: %v", err)
	}

	return bytes.TrimSpace(b)
}
