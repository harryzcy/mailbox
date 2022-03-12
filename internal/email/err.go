package email

import "errors"

// Errors
var (
	ErrNotFound     = errors.New("email not found")
	ErrInvalidInput = errors.New("invalid input")

	// ErrNotTrashed is returned when trying to delete an untrashed email
	ErrNotTrashed = errors.New("email is not trashed")
)
