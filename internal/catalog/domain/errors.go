package domain

import shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"

var (
	ErrEmptyBeerName = shared.NewValidationError("beer name cannot be empty")
	ErrBeerNotFound  = shared.NewNotFoundError("beer not found")
	ErrBeerNameTaken = shared.NewConflictError("a beer with that name already exists")
	ErrBeerRetired   = shared.NewUnprocessableError("beer is retired")
)
