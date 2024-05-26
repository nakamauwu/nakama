package web

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"

	"github.com/btcsuite/btcutil/base58"
	"github.com/nicolasparada/go-errs/httperrs"
)

func err2code(err error) int {
	if errors.Is(err, errOAuth2StateMismatch) {
		return http.StatusTeapot
	}

	return httperrs.Code(err)
}

func genRandStr() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base58.Encode(b), nil
}

func ptr[T any](v T) *T {
	return &v
}

func (h *Handler) decodePostForm(r *http.Request, dest any) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("parse form: %w", err)
	}

	if err := h.formDecoder.Decode(dest, r.PostForm); err != nil {
		return fmt.Errorf("decode form values: %w", err)
	}

	return nil
}

func (h *Handler) decodeQuery(r *http.Request, dest any) error {
	if err := h.formDecoder.Decode(dest, r.URL.Query()); err != nil {
		return fmt.Errorf("decode query string params: %w", err)
	}

	return nil
}
