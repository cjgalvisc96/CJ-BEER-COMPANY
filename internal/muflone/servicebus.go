package muflone

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// Topic prefixes encode the message-handling pattern the book prescribes:
// commands use producer-consumer (one handler), events use pub/sub.
const (
	commandTopicPrefix          = "commands."
	domainEventTopicPrefix      = "events."
	integrationEventTopicPrefix = "integrationevents."
)

// CommandHandler consumes one command type — Muflone's
// ICommandHandlerAsync<TCommand>.
type CommandHandler[C Command] interface {
	Handle(ctx context.Context, command C) error
}

// ServiceBus delivers commands to their single handler and broadcasts
// domain/integration events to every subscriber. It replaces the mediator
// of the pre-events refactoring step, exactly as the book replaces
// BrewUp.Mediator with a service bus (RabbitMQ there, Watermill here —
// swapping brokers is a transport change, not a redesign).
type ServiceBus struct {
	pubSub *gochannel.GoChannel
	router *message.Router
	logger *slog.Logger
}

func NewServiceBus(logger *slog.Logger) (*ServiceBus, error) {
	wmLogger := watermill.NewSlogLogger(logger)
	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		return nil, err
	}
	return &ServiceBus{
		pubSub: gochannel.NewGoChannel(gochannel.Config{}, wmLogger),
		router: router,
		logger: logger,
	}, nil
}

// Send dispatches a command to its handler (fire-and-forget).
func (b *ServiceBus) Send(_ context.Context, command Command) error {
	return b.publish(commandTopicPrefix+command.MessageName(), command)
}

// PublishDomainEvent broadcasts a domain event inside the monolith. The
// event-store repository calls this after appending to the stream.
func (b *ServiceBus) PublishDomainEvent(_ context.Context, event DomainEvent) error {
	return b.publish(domainEventTopicPrefix+event.MessageName(), event)
}

// PublishIntegrationEvent broadcasts an integration event for other
// bounded contexts.
func (b *ServiceBus) PublishIntegrationEvent(_ context.Context, event IntegrationEvent) error {
	return b.publish(integrationEventTopicPrefix+event.MessageName(), event)
}

func (b *ServiceBus) publish(topic string, msg Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", topic, err)
	}
	return b.pubSub.Publish(topic, message.NewMessage(watermill.NewUUID(), payload))
}

// RegisterCommandHandler subscribes the single handler for command type C.
func RegisterCommandHandler[C Command](bus *ServiceBus, handler CommandHandler[C]) {
	var prototype C
	topic := commandTopicPrefix + prototype.MessageName()
	bus.subscribe("handler."+prototype.MessageName(), topic, func(msg *message.Message) error {
		var command C
		if err := json.Unmarshal(msg.Payload, &command); err != nil {
			return fmt.Errorf("unmarshal command %s: %w", topic, err)
		}
		return handler.Handle(msg.Context(), command)
	})
}

// RegisterDomainEventHandler subscribes one of possibly many handlers for
// domain event type E (read-model projections, integration publishers, …).
// The name distinguishes multiple subscriptions of the same event.
func RegisterDomainEventHandler[E DomainEvent](bus *ServiceBus, name string, handler func(ctx context.Context, event E) error) {
	var prototype E
	topic := domainEventTopicPrefix + prototype.MessageName()
	bus.subscribe(name, topic, func(msg *message.Message) error {
		var event E
		if err := json.Unmarshal(msg.Payload, &event); err != nil {
			return fmt.Errorf("unmarshal event %s: %w", topic, err)
		}
		return handler(msg.Context(), event)
	})
}

// SubscribeIntegrationEvent subscribes to another context's integration
// event by topic name. The payload is raw JSON on purpose: consumers
// deserialize their own contract struct instead of importing the
// producer's types, keeping the contexts decoupled.
func (b *ServiceBus) SubscribeIntegrationEvent(name, messageName string, handler func(ctx context.Context, payload []byte) error) {
	b.subscribe(name, integrationEventTopicPrefix+messageName, func(msg *message.Message) error {
		return handler(msg.Context(), msg.Payload)
	})
}

func (b *ServiceBus) subscribe(name, topic string, handler message.NoPublishHandlerFunc) {
	b.router.AddNoPublisherHandler(name, topic, b.pubSub, handler)
}

// Run starts routing messages and blocks until ctx is cancelled. Running()
// unblocks once the router is operational.
func (b *ServiceBus) Run(ctx context.Context) error {
	return b.router.Run(ctx)
}

func (b *ServiceBus) Running() <-chan struct{} {
	return b.router.Running()
}

func (b *ServiceBus) Close() error {
	if err := b.router.Close(); err != nil {
		return err
	}
	return b.pubSub.Close()
}
