package web

import (
	"crypto/rand"
	"errors"
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
