// White-box tests for the AMQP transport glue. The constructor seams stand
// in for a broker: this proves the transport's own logic (config, wiring,
// error paths, shutdown); the real RabbitMQ wire is exercised end to end
// by the compose smoke test in CI.
package muflone

import (
	"context"
	"log/slog"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	amqp "github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePublisher struct{ closed bool }

func (*fakePublisher) Publish(string, ...*message.Message) error { return nil }
func (p *fakePublisher) Close() error                            { p.closed = true; return nil }

type fakeSubscriber struct{ closed bool }

func (*fakeSubscriber) Subscribe(context.Context, string) (<-chan *message.Message, error) {
	return nil, nil
}
func (s *fakeSubscriber) Close() error { s.closed = true; return nil }

func TestAMQPTransportFailsFastWithoutBroker(t *testing.T) {
	// Nothing listens on port 1: the eager publisher connection fails,
	// and so does any subscriber attempted against the same broker.
	transport, err := NewAMQPTransport("amqp://guest:guest@127.0.0.1:1/", slog.Default())
	assert.ErrorContains(t, err, "connect amqp publisher")

	_, err = transport.SubscriberFor("some.handler")
	assert.ErrorContains(t, err, "connect amqp subscriber")
}

func TestAMQPTransportWiresPublisherAndSubscribers(t *testing.T) {
	publisher := &fakePublisher{}
	subscriberA, subscriberB := &fakeSubscriber{}, &fakeSubscriber{}
	remaining := []*fakeSubscriber{subscriberA, subscriberB}
	var seenQueues []string

	transport := &AMQPTransport{
		url:    "amqp://ignored",
		logger: watermill.NewSlogLogger(slog.Default()),
		newPublisher: func(config amqp.Config, _ watermill.LoggerAdapter) (message.Publisher, error) {
			return publisher, nil
		},
		newSubscriber: func(config amqp.Config, _ watermill.LoggerAdapter) (message.Subscriber, error) {
			seenQueues = append(seenQueues, config.Queue.GenerateName("events.some_topic"))
			next := remaining[0]
			remaining = remaining[1:]
			return next, nil
		},
	}
	require.NoError(t, transport.connect())
	assert.Same(t, publisher, transport.Publisher())

	first, err := transport.SubscriberFor("sales.projection")
	require.NoError(t, err)
	assert.Same(t, subscriberA, first)
	second, err := transport.SubscriberFor("warehouses.saga")
	require.NoError(t, err)
	assert.Same(t, subscriberB, second)

	// Each handler gets its own durable queue → fan-out per handler,
	// competing consumers per replica.
	assert.Equal(t, []string{
		"events.some_topic_sales.projection",
		"events.some_topic_warehouses.saga",
	}, seenQueues)

	require.NoError(t, transport.Close())
	assert.True(t, publisher.closed)
	assert.True(t, subscriberA.closed)
	assert.True(t, subscriberB.closed)
}

func TestAMQPTransportSurfacesSubscriberFailures(t *testing.T) {
	transport := &AMQPTransport{
		url:    "amqp://ignored",
		logger: watermill.NewSlogLogger(slog.Default()),
		newSubscriber: func(amqp.Config, watermill.LoggerAdapter) (message.Subscriber, error) {
			return nil, assert.AnError
		},
	}

	_, err := transport.SubscriberFor("some.handler")

	assert.ErrorContains(t, err, "connect amqp subscriber")
}

// TestBusOnFailingTransport: a transport whose subscribers cannot connect
// fails Run before any message flows.
func TestBusOnFailingTransport(t *testing.T) {
	transport := &AMQPTransport{
		url:       "amqp://ignored",
		logger:    watermill.NewSlogLogger(slog.Default()),
		publisher: &fakePublisher{},
		newSubscriber: func(amqp.Config, watermill.LoggerAdapter) (message.Subscriber, error) {
			return nil, assert.AnError
		},
	}
	bus := NewServiceBusWithTransport(transport, slog.Default())

	err := bus.Run(context.Background())

	assert.ErrorIs(t, err, assert.AnError)
	assert.NoError(t, bus.Close())
}
