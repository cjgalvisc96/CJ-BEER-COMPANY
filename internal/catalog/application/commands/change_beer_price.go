package commands

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

type ChangeBeerPriceInput struct {
	BeerID     string
	PriceCents int64
	Currency   string
}

type ChangeBeerPriceHandler struct {
	repository domain.BeerRepository
	publisher  ports.EventPublisher
}

func NewChangeBeerPriceHandler(repository domain.BeerRepository, publisher ports.EventPublisher) *ChangeBeerPriceHandler {
	return &ChangeBeerPriceHandler{repository: repository, publisher: publisher}
}

func (h *ChangeBeerPriceHandler) Handle(ctx context.Context, input ChangeBeerPriceInput) (dto.BeerOutput, error) {
	id, err := domain.ParseBeerID(input.BeerID)
	if err != nil {
		return dto.BeerOutput{}, err
	}
	price, err := shared.NewMoney(input.PriceCents, input.Currency)
	if err != nil {
		return dto.BeerOutput{}, err
	}
	beer, err := h.repository.FindByID(ctx, id)
	if err != nil {
		return dto.BeerOutput{}, err
	}
	if err := beer.ChangePrice(price); err != nil {
		return dto.BeerOutput{}, err
	}
	if err := h.repository.Save(ctx, beer); err != nil {
		return dto.BeerOutput{}, err
	}
	if err := h.publisher.Publish(ctx, beer.PullEvents()...); err != nil {
		return dto.BeerOutput{}, err
	}
	return dto.BeerOutputFromEntity(beer), nil
}
