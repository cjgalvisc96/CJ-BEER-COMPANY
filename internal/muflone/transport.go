package muflone

import (
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// Transport is the wire under the ServiceBus. Swapping brokers (the book
// moves BrewUp from the mediator to RabbitMQ) is providing another
// implementation — the bus, the handlers, and the modules never change.
type Transport interface {
	// Publisher publishes to a topic.
	Publisher() message.Publisher
	// SubscriberFor returns the subscriber feeding one handler. Each
	// handler gets its own subscription so a topic fans out to every
	// interested handler, while replicas of the SAME handler (same name)
	// compete for messages — consumer-group semantics.
	SubscriberFor(handlerName string) (message.Subscriber, error)
	Close() error
}

// InMemoryTransport runs everything in process over Watermill's GoChannel
// — zero dependencies (dev, tests). Messages die with the process; the
// event store is the durability there.
type InMemoryTransport struct {
	pubSub *gochannel.GoChannel
}

func NewInMemoryTransport(logger *slog.Logger) *InMemoryTransport {
	return &InMemoryTransport{
		pubSub: gochannel.NewGoChannel(gochannel.Config{}, watermill.NewSlogLogger(logger)),
	}
}

func (t *InMemoryTransport) Publisher() message.Publisher {
	return t.pubSub
}

// SubscriberFor returns the shared GoChannel: it already delivers a copy
// of each message to every Subscribe call, which is exactly the
// per-handler fan-out the bus needs.
func (t *InMemoryTransport) SubscriberFor(string) (message.Subscriber, error) {
	return t.pubSub, nil
}

func (t *InMemoryTransport) Close() error {
	return t.pubSub.Close()
}

var _ Transport = (*InMemoryTransport)(nil)
var _ Transport = (*AMQPTransport)(nil)

// closeAll closes many closers, joining the errors.
func closeAll(closers []interface{ Close() error }) error {
	var errs []error
	for _, closer := range closers {
		errs = append(errs, closer.Close())
	}
	return errors.Join(errs...)
}
