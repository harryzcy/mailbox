package email

import "errors"

// Errors
var (
	ErrNotFound     = errors.New("email not found")
	ErrInvalidInput = errors.New("invalid input")
)
