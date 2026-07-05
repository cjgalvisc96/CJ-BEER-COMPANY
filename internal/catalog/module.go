// Package catalog wires the catalog bounded context into the DI container.
// Cross-layer imports are confined to this file (composition root of the
// context), mirroring the per-context container pattern.
package catalog

import (
	"github.com/samber/do/v2"

	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/queries"
	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/infrastructure/persistence"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
)

func Register(injector do.Injector) {
	do.Provide(injector, func(i do.Injector) (domain.BeerRepository, error) {
		return persistence.NewMemoryBeerRepository(), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.CreateBeerHandler, error) {
		return commands.NewCreateBeerHandler(
			do.MustInvoke[domain.BeerRepository](i),
			do.MustInvoke[ports.EventPublisher](i),
		), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.ChangeBeerPriceHandler, error) {
		return commands.NewChangeBeerPriceHandler(
			do.MustInvoke[domain.BeerRepository](i),
			do.MustInvoke[ports.EventPublisher](i),
		), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.RetireBeerHandler, error) {
		return commands.NewRetireBeerHandler(
			do.MustInvoke[domain.BeerRepository](i),
			do.MustInvoke[ports.EventPublisher](i),
		), nil
	})
	do.Provide(injector, func(i do.Injector) (*queries.GetBeerHandler, error) {
		return queries.NewGetBeerHandler(do.MustInvoke[domain.BeerRepository](i)), nil
	})
	do.Provide(injector, func(i do.Injector) (*queries.ListBeersHandler, error) {
		return queries.NewListBeersHandler(do.MustInvoke[domain.BeerRepository](i)), nil
	})
}
