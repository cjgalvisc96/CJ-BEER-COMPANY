// Package commands holds the write-side use cases of the brewing context.
package commands

import (
	"context"
	"fmt"

	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/ports"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/domain"
	sharedports "github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
)

type StartBatchInput struct {
	BeerID string
	Units  int
}

type StartBatchHandler struct {
	repository domain.BatchRepository
	catalog    ports.BeerCatalog
	publisher  sharedports.EventPublisher
}

func NewStartBatchHandler(
	repository domain.BatchRepository,
	catalog ports.BeerCatalog,
	publisher sharedports.EventPublisher,
) *StartBatchHandler {
	return &StartBatchHandler{repository: repository, catalog: catalog, publisher: publisher}
}

func (h *StartBatchHandler) Handle(ctx context.Context, input StartBatchInput) (dto.BatchOutput, error) {
	beerID, err := domain.ParseBeerRef(input.BeerID)
	if err != nil {
		return dto.BatchOutput{}, err
	}
	brewable, err := h.catalog.IsBrewable(ctx, input.BeerID)
	if err != nil || !brewable {
		return dto.BatchOutput{}, fmt.Errorf("%w: %s", domain.ErrBeerNotBrewable, input.BeerID)
	}
	batch, err := domain.StartBatch(beerID, input.Units)
	if err != nil {
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
