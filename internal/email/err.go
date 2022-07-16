package email

import "errors"

// Errors
var (
	ErrNotFound      = errors.New("email not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrQueryNotMatch = errors.New("query does not match with next cursor")

	// ErrNotTrashed is returned when trying to delete or untrash an untrashed email
	ErrNotTrashed = errors.New("email is not trashed")
	// ErrAlreadyTrashed is returned when trying to trash an already trashed email
	ErrAlreadyTrashed = errors.New("email is already trashed")

	// ErrEmailIsNotDraft is returned when expected draft type is not met
	ErrEmailIsNotDraft = errors.New("email type is not draft")
)
