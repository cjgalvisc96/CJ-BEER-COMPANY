package queries

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/domain"
)

type ListBeersHandler struct {
	repository domain.BeerRepository
}

func NewListBeersHandler(repository domain.BeerRepository) *ListBeersHandler {
	return &ListBeersHandler{repository: repository}
}

func (h *ListBeersHandler) Handle(ctx context.Context) ([]dto.BeerOutput, error) {
	beers, err := h.repository.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	outputs := make([]dto.BeerOutput, 0, len(beers))
	for _, beer := range beers {
		outputs = append(outputs, dto.BeerOutputFromEntity(beer))
	}
	return outputs, nil
}
