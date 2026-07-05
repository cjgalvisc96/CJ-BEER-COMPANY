package muflone

// ErrInvalid is a domain-rule violation. Declared as a string-based error
// type so modules can define sentinel invariant errors as constants.
type ErrInvalid string

func (e ErrInvalid) Error() string { return string(e) }
