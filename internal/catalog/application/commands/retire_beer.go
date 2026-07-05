package commands

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
)

type RetireBeerHandler struct {
	repository domain.BeerRepository
	publisher  ports.EventPublisher
}

func NewRetireBeerHandler(repository domain.BeerRepository, publisher ports.EventPublisher) *RetireBeerHandler {
	return &RetireBeerHandler{repository: repository, publisher: publisher}
}

func (h *RetireBeerHandler) Handle(ctx context.Context, rawBeerID string) error {
	id, err := domain.ParseBeerID(rawBeerID)
	if err != nil {
		return err
	}
	beer, err := h.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}
	beer.Retire()
	if err := h.repository.Save(ctx, beer); err != nil {
		return err
	}
	return h.publisher.Publish(ctx, beer.PullEvents()...)
}
