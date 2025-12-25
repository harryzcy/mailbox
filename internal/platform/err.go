package platform

import "errors"

// Errors
var (
	ErrTooManyRequests = errors.New("too many requests")

	ErrNotFound      = errors.New("email not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrQueryNotMatch = errors.New("query does not match with next cursor")

	// ErrReadActionFailed is returned when a read action or unread action fails
	ErrReadActionFailed = errors.New("read action failed")

	// ErrEmailIsNotDraft is returned when expected draft type is not met
	ErrEmailIsNotDraft = errors.New("email type is not draft")
)

// NotTrashedError is returned when trying to delete or untrash an untrashed email/thread
type NotTrashedError struct {
	Type string // 'email' or 'thread'
}

func (e *NotTrashedError) Error() string {
	return e.Type + " is not trashed"
}

func (e *NotTrashedError) Is(target error) bool {
	t, ok := target.(*NotTrashedError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

type AlreadyTrashedError struct {
	Type string // 'email' or 'thread'
}

func (e *AlreadyTrashedError) Error() string {
	return e.Type + " is already trashed"
}

func (e *AlreadyTrashedError) Is(target error) bool {
	t, ok := target.(*AlreadyTrashedError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}
