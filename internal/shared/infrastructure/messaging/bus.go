// Package messaging adapts Watermill to the shared EventPublisher port and
// offers helpers for subscribing context event handlers to the bus.
package messaging

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// Bus bundles the in-process pub/sub with the router that dispatches
// messages to handlers. GoChannel keeps everything in memory; swapping to
// Kafka/NATS only requires providing different Publisher/Subscriber
// implementations here.
type Bus struct {
	PubSub *gochannel.GoChannel
	Router *message.Router
}

func NewBus(logger *slog.Logger) (*Bus, error) {
	wmLogger := watermill.NewSlogLogger(logger)
	pubSub := gochannel.NewGoChannel(gochannel.Config{}, wmLogger)
	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		return nil, err
	}
	return &Bus{PubSub: pubSub, Router: router}, nil
}

// Handler processes one message payload from a topic. Returning an error
// signals the router to retry/nack according to its middleware.
type Handler func(msg *message.Message) error

// Subscribe registers a no-publisher handler on the router. The name must
// be unique across the application; use "<context>.<purpose>".
func (b *Bus) Subscribe(name, topic string, handler Handler) {
	b.Router.AddNoPublisherHandler(name, topic, b.PubSub, message.NoPublishHandlerFunc(handler))
}

func (b *Bus) Close() error {
	if err := b.Router.Close(); err != nil {
		return err
	}
	return b.PubSub.Close()
}
