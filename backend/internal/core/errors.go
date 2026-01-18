package core

import "errors"

var (
	ErrNotFound    = errors.New("resource not found")
	ErrConflict    = errors.New("resource already exists")
	ErrRateLimited = errors.New("rate limit exceeded")
	ErrBadRequest  = errors.New("invalid request")
)
