package muflone

import "errors"

// ErrInvalid is a domain-rule violation. Declared as a string-based error
// type so modules can define sentinel invariant errors as constants.
type ErrInvalid string

func (e ErrInvalid) Error() string { return string(e) }

// ErrNotFound signals a read-model miss; the REST layer maps it to 404.
var ErrNotFound = errors.New("not found")
