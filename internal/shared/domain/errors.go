package domain

import "errors"

// ErrorKind classifies domain failures so the presentation layer can map
// them to transport concerns (HTTP status codes) without inspecting
// context-specific error types.
type ErrorKind int

const (
	KindValidation ErrorKind = iota
	KindNotFound
	KindConflict
	KindUnprocessable
)

// Error is the single error type crossing the domain boundary. Contexts
// create them through the New*Error constructors and may wrap them with
// additional context using fmt.Errorf and %w.
type Error struct {
	kind    ErrorKind
	message string
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) Kind() ErrorKind {
	return e.kind
}

func NewValidationError(message string) *Error {
	return &Error{kind: KindValidation, message: message}
}

func NewNotFoundError(message string) *Error {
	return &Error{kind: KindNotFound, message: message}
}

func NewConflictError(message string) *Error {
	return &Error{kind: KindConflict, message: message}
}

func NewUnprocessableError(message string) *Error {
	return &Error{kind: KindUnprocessable, message: message}
}

// KindOf extracts the ErrorKind from any error in the chain. The boolean is
// false when the error is not a domain error.
func KindOf(err error) (ErrorKind, bool) {
	var domainErr *Error
	if errors.As(err, &domainErr) {
		return domainErr.kind, true
	}
	return 0, false
}
