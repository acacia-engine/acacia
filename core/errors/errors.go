package errors

import "errors"

// Common application-wide errors.
var (
	ErrNotFound       = errors.New("resource not found")
	ErrInvalidInput   = errors.New("invalid input provided")
	ErrUnauthorized   = errors.New("unauthorized access")
	ErrForbidden      = errors.New("access forbidden")
	ErrInternalServer = errors.New("internal server error")
	ErrAlreadyExists  = errors.New("resource already exists")
)

// New creates a new error with the given message.
func New(message string) error {
	return errors.New(message)
}

// Wrap adds context to an existing error.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return errors.Join(errors.New(message), err)
}
