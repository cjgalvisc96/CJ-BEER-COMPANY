package domain

import shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"

var (
	ErrStockItemNotFound      = shared.NewNotFoundError("stock item not found")
	ErrStockItemAlreadyExists = shared.NewConflictError("stock item already tracked for that beer")
	ErrInsufficientStock      = shared.NewUnprocessableError("insufficient stock")
)
