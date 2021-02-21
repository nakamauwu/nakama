package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/nicolasparada/nakama/internal/service"
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
				assertEqual(t, http.StatusBadRequest, resp.StatusCode, "status code")
				assertEqual(t, "bad request", readerText(t, resp.Body), "body")
			},
		},
		{
			name: "malformed_request_body",
			body: []byte(`nope`),
			testResp: func(t *testing.T, resp *http.Response) {
				assertEqual(t, http.StatusBadRequest, resp.StatusCode, "status code")
				assertEqual(t, "bad request", readerText(t, resp.Body), "body")
			},
		},
		{
			name: "invalid_email",
			body: []byte(`{}`),
			svc: &ServiceMock{
				SendMagicLinkFunc: func(context.Context, string, string) error {
					return service.ErrInvalidEmail
				},
			},
			testResp: func(t *testing.T, resp *http.Response) {
				assertEqual(t, http.StatusUnprocessableEntity, resp.StatusCode, "status code")
				assertEqual(t, "invalid email", readerText(t, resp.Body), "body")
			},
		},
		{
			name: "invalid_redirect_uri",
			body: []byte(`{}`),
			svc: &ServiceMock{
				SendMagicLinkFunc: func(context.Context, string, string) error {
					return service.ErrInvalidRedirectURI
				},
			},
			testResp: func(t *testing.T, resp *http.Response) {
				assertEqual(t, http.StatusUnprocessableEntity, resp.StatusCode, "status code")
				assertEqual(t, "invalid redirect URI", readerText(t, resp.Body), "body")
			},
		},
		{
			name: "user_not_found",
			body: []byte(`{}`),
			svc: &ServiceMock{
				SendMagicLinkFunc: func(context.Context, string, string) error {
					return service.ErrUserNotFound
				},
			},
			testResp: func(t *testing.T, resp *http.Response) {
				assertEqual(t, http.StatusNotFound, resp.StatusCode, "status code")
				assertEqual(t, "user not found", readerText(t, resp.Body), "body")
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
				assertEqual(t, http.StatusInternalServerError, resp.StatusCode, "status code")
				assertEqual(t, "internal server error", readerText(t, resp.Body), "body")
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
				assertEqual(t, http.StatusNoContent, resp.StatusCode, "status code")
			},
			testCall: func(t *testing.T, call call) {
				assertEqual(t, "user@example.org", call.Email, "email")
				assertEqual(t, "https://example.org", call.RedirectURI, "redirect URI")
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			h := New(tc.svc, true)
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

			tc.testResp(t, resp)
			if tc.testCall != nil {
				tc.testCall(t, tc.svc.SendMagicLinkCalls()[0])
			}
		})
	}
}

func assertEqual(t *testing.T, want, got interface{}, msg string) {
	t.Helper()

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("%s: want %v; got %v", msg, want, got)
	}
}

func readerText(t *testing.T, rc io.ReadCloser) string {
	t.Helper()

	defer rc.Close()

	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read all from reader: %v", err)
	}

	return string(bytes.TrimSpace(b))
}
