package muflone

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Topic prefixes encode the message-handling pattern the book prescribes:
// commands use producer-consumer (one handler), events use pub/sub.
const (
	commandTopicPrefix          = "commands."
	domainEventTopicPrefix      = "events."
	integrationEventTopicPrefix = "integrationevents."

	// DeadLetterTopic receives poison messages after the retries are
	// exhausted (book Ch. 12: failed messages must never be lost — they
	// are parked for inspection instead of blocking the bus).
	DeadLetterTopic = "brewup.dead_letter"

	poisonRetries       = 3
	poisonRetryInterval = 50 * time.Millisecond
)

// CommandHandler consumes one command type — Muflone's
// ICommandHandlerAsync<TCommand>.
type CommandHandler[C Command] interface {
	Handle(ctx context.Context, command C) error
}

// pendingSubscription is a handler registered before the bus runs;
// subscribers are attached at Run, when the transport must be reachable.
type pendingSubscription struct {
	name    string
	topic   string
	handler message.NoPublishHandlerFunc
}

// ServiceBus delivers commands to their single handler and broadcasts
// domain/integration events to every subscriber. It replaces the mediator
// of the pre-events refactoring step, exactly as the book replaces
// BrewUp.Mediator with a service bus. The wire is pluggable (Transport):
// in-memory GoChannel by default, RabbitMQ (the book's broker) in
// production — swapping brokers is configuration, not a redesign.
type ServiceBus struct {
	transport Transport
	publisher message.Publisher
	router    *message.Router
	logger    *slog.Logger

	pending  []pendingSubscription
	attach   sync.Once
	attached error
}

// NewServiceBus builds a bus on the in-memory transport.
func NewServiceBus(logger *slog.Logger) *ServiceBus {
	return NewServiceBusWithTransport(NewInMemoryTransport(logger), logger)
}

// NewServiceBusWithTransport builds a bus on any Transport (e.g. the
// RabbitMQ one when BROKER_URL is configured).
func NewServiceBusWithTransport(transport Transport, logger *slog.Logger) *ServiceBus {
	wmLogger := watermill.NewSlogLogger(logger)
	// NewRouter only errors on an invalid config; the zero RouterConfig is
	// always valid, so the error is unreachable here.
	router, _ := message.NewRouter(message.RouterConfig{}, wmLogger)

	// Dead-letter handling: the poison queue is the outermost middleware,
	// so a message that still fails after the retries is parked on the
	// dead-letter topic instead of being redelivered forever. PoisonQueue
	// only errors on an empty topic — unreachable with the constant.
	poison, _ := middleware.PoisonQueue(transport.Publisher(), DeadLetterTopic)
	router.AddMiddleware(poison)
	router.AddMiddleware(middleware.Retry{
		MaxRetries:      poisonRetries,
		InitialInterval: poisonRetryInterval,
		Logger:          wmLogger,
	}.Middleware)

	bus := &ServiceBus{
		transport: transport,
		publisher: transport.Publisher(),
		router:    router,
		logger:    logger,
	}
	// Park-and-log: operators inspect dead letters via this structured log
	// (or by subscribing to the topic themselves).
	bus.subscribe("muflone.dead_letter_monitor", DeadLetterTopic, func(msg *message.Message) error {
		logger.Error("bus.dead_letter",
			slog.String("reason", msg.Metadata.Get(middleware.ReasonForPoisonedKey)),
			slog.String("topic", msg.Metadata.Get(middleware.PoisonedTopicKey)),
			slog.String("payload", string(msg.Payload)),
		)
		return nil
	})
	return bus
}

// Send dispatches a command to its handler (fire-and-forget).
func (b *ServiceBus) Send(ctx context.Context, command Command) error {
	return b.publish(ctx, commandTopicPrefix+command.MessageName(), command)
}

// PublishDomainEvent broadcasts a domain event inside the monolith. The
// event-store repository calls this after appending to the stream.
func (b *ServiceBus) PublishDomainEvent(ctx context.Context, event DomainEvent) error {
	return b.publish(ctx, domainEventTopicPrefix+event.MessageName(), event)
}

// PublishIntegrationEvent broadcasts an integration event for other
// bounded contexts.
func (b *ServiceBus) PublishIntegrationEvent(ctx context.Context, event IntegrationEvent) error {
	return b.publish(ctx, integrationEventTopicPrefix+event.MessageName(), event)
}

func (b *ServiceBus) publish(ctx context.Context, topic string, msg Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", topic, err)
	}
	// Distributed tracing across bus hops: a publish span, with the trace
	// context carried in the message metadata (W3C traceparent).
	ctx, span := otel.Tracer("muflone").Start(ctx, "publish "+topic)
	defer span.End()
	wireMessage := message.NewMessage(watermill.NewUUID(), payload)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(wireMessage.Metadata))
	return b.publisher.Publish(topic, wireMessage)
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
	traced := func(msg *message.Message) error {
		ctx := otel.GetTextMapPropagator().Extract(msg.Context(), propagation.MapCarrier(msg.Metadata))
		ctx, span := otel.Tracer("muflone").Start(ctx, "consume "+topic)
		defer span.End()
		msg.SetContext(ctx)
		return handler(msg)
	}
	b.pending = append(b.pending, pendingSubscription{name: name, topic: topic, handler: traced})
}

// attachHandlers turns the pending registrations into live consumers, one
// subscriber per handler (fan-out on shared topics; competing consumers
// across replicas of the same handler).
func (b *ServiceBus) attachHandlers() error {
	b.attach.Do(func() {
		for _, pending := range b.pending {
			subscriber, err := b.transport.SubscriberFor(pending.name)
			if err != nil {
				b.attached = err
				return
			}
			b.router.AddConsumerHandler(pending.name, pending.topic, subscriber, pending.handler)
		}
	})
	return b.attached
}

// Run starts routing messages and blocks until ctx is cancelled. Running()
// unblocks once the router is operational.
func (b *ServiceBus) Run(ctx context.Context) error {
	if err := b.attachHandlers(); err != nil {
		return err
	}
	return b.router.Run(ctx)
}

func (b *ServiceBus) Running() <-chan struct{} {
	return b.router.Running()
}

func (b *ServiceBus) Close() error {
	select {
	case <-b.router.Running():
		return errors.Join(b.router.Close(), b.transport.Close())
	default:
		// The router never ran (construction-only bus, e.g. a failed
		// boot): closing it would block on handlers that never started.
		return b.transport.Close()
	}
}
