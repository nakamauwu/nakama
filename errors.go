package nakama

import "errors"

var ErrInvalidArgument = errors.New("invalid argument")

type InvalidArgumentError string

func (e InvalidArgumentError) Error() string {
	return string(e)
}

func (e InvalidArgumentError) Unwrap() error {
	return ErrInvalidArgument
}

// -----------------------------------------------------------------------------

var ErrNotFound = errors.New("not found")

type NotFoundError string

func (e NotFoundError) Error() string {
	return string(e)
}

func (e NotFoundError) Unwrap() error {
	return ErrNotFound
}

// -----------------------------------------------------------------------------

var ErrAlreadyExists = errors.New("already exists")

type AlreadyExistsError string

func (e AlreadyExistsError) Error() string {
	return string(e)
}

func (e AlreadyExistsError) Unwrap() error {
	return ErrAlreadyExists
}

// -----------------------------------------------------------------------------

var ErrPermissionDenied = errors.New("permission denied")

type PermissionDeniedError string

func (e PermissionDeniedError) Error() string {
	return string(e)
}

func (e PermissionDeniedError) Unwrap() error {
	return ErrPermissionDenied
}

// -----------------------------------------------------------------------------

// ErrUnauthenticated denotes no authenticated user in context.
var ErrUnauthenticated = errors.New("unauthenticated")

type UnauthenticatedError string

func (e UnauthenticatedError) Error() string {
	return string(e)
}

func (e UnauthenticatedError) Unwrap() error {
	return ErrUnauthenticated
}

// -----------------------------------------------------------------------------

var ErrUnimplemented = errors.New("unimplemented")

type UnimplementedError string

func (e UnimplementedError) Error() string {
	return string(e)
}

func (e UnimplementedError) Unwrap() error {
	return ErrUnimplemented
}

// -----------------------------------------------------------------------------

var ErrGone = errors.New("gone")

type GoneError string

func (e GoneError) Error() string {
	return string(e)
}

func (e GoneError) Unwrap() error {
	return ErrGone
}
