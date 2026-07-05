// Package commands holds the write-side use cases of the catalog context.
// One file, one command handler (CQRS).
package commands

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/dto"
	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

type CreateBeerInput struct {
	Name        string
	Style       string
	ABV         float64
	PriceCents  int64
	Currency    string
	Description string
}

type CreateBeerHandler struct {
	repository domain.BeerRepository
	publisher  ports.EventPublisher
}

func NewCreateBeerHandler(repository domain.BeerRepository, publisher ports.EventPublisher) *CreateBeerHandler {
	return &CreateBeerHandler{repository: repository, publisher: publisher}
}

func (h *CreateBeerHandler) Handle(ctx context.Context, input CreateBeerInput) (dto.BeerOutput, error) {
	style, err := domain.ParseStyle(input.Style)
	if err != nil {
		return dto.BeerOutput{}, err
	}
	abv, err := domain.NewABV(input.ABV)
	if err != nil {
		return dto.BeerOutput{}, err
	}
	price, err := shared.NewMoney(input.PriceCents, input.Currency)
	if err != nil {
		return dto.BeerOutput{}, err
	}

	taken, err := h.repository.ExistsByName(ctx, input.Name)
	if err != nil {
		return dto.BeerOutput{}, err
	}
	if taken {
		return dto.BeerOutput{}, domain.ErrBeerNameTaken
	}

	beer, err := domain.NewBeer(input.Name, style, abv, price, input.Description)
	if err != nil {
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
