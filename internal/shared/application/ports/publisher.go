// Package ports declares the driven ports shared by all contexts'
// application layers. Infrastructure provides the adapters.
package ports

import (
	"context"

	"github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

// EventPublisher publishes domain events to the message bus. Each event is
// published on the topic returned by its EventName().
type EventPublisher interface {
	Publish(ctx context.Context, events ...domain.Event) error
}
