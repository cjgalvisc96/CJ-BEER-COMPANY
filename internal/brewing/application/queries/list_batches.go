package queries

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/domain"
)

type ListBatchesHandler struct {
	repository domain.BatchRepository
}

func NewListBatchesHandler(repository domain.BatchRepository) *ListBatchesHandler {
	return &ListBatchesHandler{repository: repository}
}

func (h *ListBatchesHandler) Handle(ctx context.Context) ([]dto.BatchOutput, error) {
	batches, err := h.repository.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	outputs := make([]dto.BatchOutput, 0, len(batches))
	for _, batch := range batches {
		outputs = append(outputs, dto.BatchOutputFromEntity(batch))
	}
	return outputs, nil
}
