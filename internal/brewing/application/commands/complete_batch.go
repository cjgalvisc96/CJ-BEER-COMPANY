package commands

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/domain"
	sharedports "github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
)

type CompleteBatchInput struct {
	BatchID       string
	ProducedUnits int
}

type CompleteBatchHandler struct {
	repository domain.BatchRepository
	publisher  sharedports.EventPublisher
}

func NewCompleteBatchHandler(repository domain.BatchRepository, publisher sharedports.EventPublisher) *CompleteBatchHandler {
	return &CompleteBatchHandler{repository: repository, publisher: publisher}
}

func (h *CompleteBatchHandler) Handle(ctx context.Context, input CompleteBatchInput) (dto.BatchOutput, error) {
	id, err := domain.ParseBatchID(input.BatchID)
	if err != nil {
		return dto.BatchOutput{}, err
	}
	batch, err := h.repository.FindByID(ctx, id)
	if err != nil {
		return dto.BatchOutput{}, err
	}
	if err := batch.Complete(input.ProducedUnits); err != nil {
		return dto.BatchOutput{}, err
	}
	if err := h.repository.Save(ctx, batch); err != nil {
		return dto.BatchOutput{}, err
	}
	if err := h.publisher.Publish(ctx, batch.PullEvents()...); err != nil {
		return dto.BatchOutput{}, err
	}
	return dto.BatchOutputFromEntity(batch), nil
}
