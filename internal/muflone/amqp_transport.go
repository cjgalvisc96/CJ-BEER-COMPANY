package muflone

import (
	"fmt"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	amqp "github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
)

// AMQPTransport is the RabbitMQ wire (the book's broker for BrewUp,
// Muflone.Transport.RabbitMQ): topics become fanout exchanges, each
// handler binds its own DURABLE queue (named <topic>_<handler>), so
// messages survive broker and consumer restarts, every handler on a topic
// gets its copy, and scaled-out replicas of one handler compete for work.
type AMQPTransport struct {
	url       string
	logger    watermill.LoggerAdapter
	publisher message.Publisher
	// track subscribers for shutdown.
	subscribers []interface{ Close() error }

	// Constructor seams: production uses the watermill-amqp constructors;
	// tests inject fakes to prove this glue without a broker (the real
	// wire is exercised by the compose smoke test).
	newPublisher  func(config amqp.Config, logger watermill.LoggerAdapter) (message.Publisher, error)
	newSubscriber func(config amqp.Config, logger watermill.LoggerAdapter) (message.Subscriber, error)
}

// NewAMQPTransport connects to RabbitMQ, failing fast when the broker is
// unreachable.
func NewAMQPTransport(url string, logger *slog.Logger) (*AMQPTransport, error) {
	transport := &AMQPTransport{
		url:    url,
		logger: watermill.NewSlogLogger(logger),
		newPublisher: func(config amqp.Config, logger watermill.LoggerAdapter) (message.Publisher, error) {
			return amqp.NewPublisher(config, logger)
		},
		newSubscriber: func(config amqp.Config, logger watermill.LoggerAdapter) (message.Subscriber, error) {
			return amqp.NewSubscriber(config, logger)
		},
	}
	return transport, transport.connect()
}

func (t *AMQPTransport) connect() error {
	publisher, err := t.newPublisher(t.config("publisher"), t.logger)
	if err != nil {
		return fmt.Errorf("connect amqp publisher: %w", err)
	}
	t.publisher = publisher
	return nil
}

// config builds the durable pub/sub topology; the queue-name suffix keys
// the consumer group (queue = <topic>_<handler>).
func (t *AMQPTransport) config(handlerName string) amqp.Config {
	return amqp.NewDurablePubSubConfig(t.url, amqp.GenerateQueueNameTopicNameWithSuffix(handlerName))
}

func (t *AMQPTransport) Publisher() message.Publisher {
	return t.publisher
}

func (t *AMQPTransport) SubscriberFor(handlerName string) (message.Subscriber, error) {
	subscriber, err := t.newSubscriber(t.config(handlerName), t.logger)
	if err != nil {
		return nil, fmt.Errorf("connect amqp subscriber %s: %w", handlerName, err)
	}
	t.subscribers = append(t.subscribers, subscriber)
	return subscriber, nil
}

func (t *AMQPTransport) Close() error {
	return closeAll(append(t.subscribers, t.publisher))
}
