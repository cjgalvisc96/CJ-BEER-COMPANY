package domain

import shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"

var (
	ErrBatchNotFound         = shared.NewNotFoundError("batch not found")
	ErrBatchAlreadyCompleted = shared.NewConflictError("batch is already completed")
	ErrBeerNotBrewable       = shared.NewUnprocessableError("beer does not exist or is retired")
)
