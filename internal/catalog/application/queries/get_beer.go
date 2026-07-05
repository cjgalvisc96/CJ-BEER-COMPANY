// Package queries holds the read-side use cases of the catalog context.
package queries

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/domain"
)

type GetBeerHandler struct {
	repository domain.BeerRepository
}

func NewGetBeerHandler(repository domain.BeerRepository) *GetBeerHandler {
	return &GetBeerHandler{repository: repository}
}

func (h *GetBeerHandler) Handle(ctx context.Context, rawBeerID string) (dto.BeerOutput, error) {
	id, err := domain.ParseBeerID(rawBeerID)
	if err != nil {
		return dto.BeerOutput{}, err
	}
	beer, err := h.repository.FindByID(ctx, id)
	if err != nil {
		return dto.BeerOutput{}, err
	}
	return dto.BeerOutputFromEntity(beer), nil
}
