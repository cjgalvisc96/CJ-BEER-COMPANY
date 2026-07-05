package domain

import shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"

var (
	ErrEmptyCustomerName   = shared.NewValidationError("customer name cannot be empty")
	ErrEmptyOrder          = shared.NewValidationError("order needs at least one line")
	ErrOrderNotFound       = shared.NewNotFoundError("order not found")
	ErrOrderNotPending     = shared.NewConflictError("order is not pending")
	ErrOrderNotCancellable = shared.NewConflictError("order can no longer be cancelled")
	ErrBeerNotSellable     = shared.NewUnprocessableError("beer does not exist or is retired")
)
