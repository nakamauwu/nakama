package nakama

import "github.com/nicolasparada/go-errs"

const ErrInvalidEmoji = errs.InvalidArgumentError("invalid emoji")

// TODO: implement emoji validation.
func validEmoji(_ string) bool {
	return true
}
