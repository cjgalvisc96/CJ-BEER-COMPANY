// Package queries holds the read-side use cases of the brewing context.
package queries

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/domain"
)

type GetBatchHandler struct {
	repository domain.BatchRepository
}

func NewGetBatchHandler(repository domain.BatchRepository) *GetBatchHandler {
	return &GetBatchHandler{repository: repository}
}

func (h *GetBatchHandler) Handle(ctx context.Context, rawBatchID string) (dto.BatchOutput, error) {
	id, err := domain.ParseBatchID(rawBatchID)
	if err != nil {
		return dto.BatchOutput{}, err
	}
	batch, err := h.repository.FindByID(ctx, id)
	if err != nil {
		return dto.BatchOutput{}, err
	}
	return dto.BatchOutputFromEntity(batch), nil
}
