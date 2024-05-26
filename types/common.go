package types

import (
	"strings"
	"time"

	"github.com/nicolasparada/go-errs"
	"github.com/oklog/ulid/v2"
)

const (
	pageSizeDefault = uint(3)
	pageSizeMax     = uint(100)
	pageSizeMin     = uint(1)
)

type Created struct {
	ID        string    `db:"id"`
	CreatedAt time.Time `db:"created_at"`
}

type List[T any] struct {
	Items       []T
	LastID      *string
	HasNextPage bool
}

func validateID(s *string) error {
	if s == nil {
		return nil
	}

	*s = strings.TrimSpace(*s)
	if !validID(*s) {
		return errs.InvalidArgumentError("invalid ID")
	}

	return nil
}

func validatePageSize(p *uint) error {
	if p == nil {
		return nil
	}

	if *p < pageSizeMin {
		return errs.InvalidArgumentError("page size too small")
	}

	if *p > pageSizeMax {
		return errs.InvalidArgumentError("page size too large")
	}

	return nil
}

func validID(s string) bool {
	_, err := ulid.ParseStrict(s)
	return err == nil
}

func ptr[T any](v T) *T {
	return &v
}
