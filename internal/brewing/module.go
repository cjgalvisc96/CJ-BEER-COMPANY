// Package brewing wires the brewing bounded context into the DI container.
package brewing

import (
	"github.com/samber/do/v2"

	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/ports"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/queries"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/domain"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/infrastructure/acl"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/infrastructure/persistence"
	catalogqueries "github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/queries"
	sharedports "github.com/cjgalvisc96/cj-beer-company/internal/shared/application/ports"
)

func Register(injector do.Injector) {
	do.Provide(injector, func(i do.Injector) (domain.BatchRepository, error) {
		return persistence.NewMemoryBatchRepository(), nil
	})
	do.Provide(injector, func(i do.Injector) (ports.BeerCatalog, error) {
		return acl.NewCatalogAdapter(do.MustInvoke[*catalogqueries.GetBeerHandler](i)), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.StartBatchHandler, error) {
		return commands.NewStartBatchHandler(
			do.MustInvoke[domain.BatchRepository](i),
			do.MustInvoke[ports.BeerCatalog](i),
			do.MustInvoke[sharedports.EventPublisher](i),
		), nil
	})
	do.Provide(injector, func(i do.Injector) (*commands.CompleteBatchHandler, error) {
		return commands.NewCompleteBatchHandler(
			do.MustInvoke[domain.BatchRepository](i),
			do.MustInvoke[sharedports.EventPublisher](i),
		), nil
	})
	do.Provide(injector, func(i do.Injector) (*queries.GetBatchHandler, error) {
		return queries.NewGetBatchHandler(do.MustInvoke[domain.BatchRepository](i)), nil
	})
	do.Provide(injector, func(i do.Injector) (*queries.ListBatchesHandler, error) {
		return queries.NewListBatchesHandler(do.MustInvoke[domain.BatchRepository](i)), nil
	})
}
