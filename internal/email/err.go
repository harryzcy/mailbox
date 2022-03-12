package email

import "errors"

// Errors
var (
	ErrNotFound     = errors.New("email not found")
	ErrInvalidInput = errors.New("invalid input")

	// ErrNotTrashed is returned when trying to delete or untrash an untrashed email
	ErrNotTrashed = errors.New("email is not trashed")
	// ErrAlreadyTrashed is returned when trying to trash an already trashed email
	ErrAlreadyTrashed = errors.New("email is already trashed")
)
