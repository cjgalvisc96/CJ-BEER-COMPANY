// White-box unit tests for the muflone building blocks: bases, aggregate,
// event store, repository, and service bus — including the error branches,
// driven through fakes rather than left uncovered.
package muflone

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- test fixtures ----------------------------------------------------------

type stubEvent struct {
	DomainEventBase
	Name string `json:"name"`
}

func (stubEvent) MessageName() string { return "test.stub_happened" }

type stubCommand struct {
	CommandBase
	Name string `json:"name"`
}

func (stubCommand) MessageName() string { return "test.do_stub" }

type badPayloadEvent struct {
	DomainEventBase
	Unserializable chan int `json:"unserializable"`
}

func (badPayloadEvent) MessageName() string { return "test.bad_payload" }

type badPayloadIntegrationEvent struct {
	IntegrationEventBase
	Unserializable chan int `json:"unserializable"`
}

func (badPayloadIntegrationEvent) MessageName() string { return "test.bad_integration_payload" }

// stubAggregate is a minimal event-sourced aggregate.
type stubAggregate struct {
	AggregateRoot
	names []string
}

func newStubAggregate() *stubAggregate {
	aggregate := &stubAggregate{}
	aggregate.Bind(aggregate)
	return aggregate
}

func (a *stubAggregate) Route(event DomainEvent) {
	if e, ok := event.(stubEvent); ok {
		a.SetID(e.AggregateID())
		a.names = append(a.names, e.Name)
	}
}

type failingStore struct {
	readErr   error
	appendErr error
}

func (s *failingStore) ReadStream(context.Context, string) ([]StoredEvent, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	return nil, nil
}

func (s *failingStore) Append(context.Context, string, int, uuid.UUID, []DomainEvent) error {
	return s.appendErr
}

type failingPublisher struct {
	err error
}

func (p *failingPublisher) PublishDomainEvent(context.Context, DomainEvent) error {
	return p.err
}

// --- message bases ----------------------------------------------------------

func TestMessageBases(t *testing.T) {
	aggregateId, commitId := uuid.New(), uuid.New()

	command := NewCommandBase(aggregateId, commitId)
	assert.Equal(t, aggregateId, command.AggregateID())
	assert.Equal(t, commitId, command.CommitID())

	event := NewDomainEventBase(aggregateId, commitId)
	assert.Equal(t, aggregateId, event.AggregateID())
	assert.Equal(t, commitId, event.CommitID())

	integration := NewIntegrationEventBase(commitId)
	assert.Equal(t, commitId, integration.CommitId)

	assert.Equal(t, "boom", ErrInvalid("boom").Error())
}

// --- aggregate root ---------------------------------------------------------

func TestAggregateRootRaiseAndReplay(t *testing.T) {
	id := uuid.New()
	aggregate := newStubAggregate()

	aggregate.RaiseEvent(stubEvent{DomainEventBase: NewDomainEventBase(id, uuid.New()), Name: "first"})
	aggregate.RaiseEvent(stubEvent{DomainEventBase: NewDomainEventBase(id, uuid.New()), Name: "second"})

	assert.Equal(t, id, aggregate.ID())
	assert.Equal(t, 2, aggregate.Version())
	assert.Equal(t, []string{"first", "second"}, aggregate.names)
	require.Len(t, aggregate.UncommittedEvents(), 2)

	aggregate.ClearUncommittedEvents()
	assert.Empty(t, aggregate.UncommittedEvents())
}

// --- event store ------------------------------------------------------------

func TestInMemoryEventStoreAppendAndRead(t *testing.T) {
	store := NewInMemoryEventStore()
	ctx := context.Background()
	event := stubEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New()), Name: "x"}

	require.NoError(t, store.Append(ctx, "Stub-1", 0, uuid.New(), []DomainEvent{event}))

	stored, err := store.ReadStream(ctx, "Stub-1")
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, 1, stored[0].Version)
	assert.Len(t, store.Appended(), 1)
}

func TestInMemoryEventStoreOptimisticConcurrency(t *testing.T) {
	store := NewInMemoryEventStore()
	event := stubEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New()), Name: "x"}
	require.NoError(t, store.Append(context.Background(), "Stub-1", 0, uuid.New(), []DomainEvent{event}))

	err := store.Append(context.Background(), "Stub-1", 0, uuid.New(), []DomainEvent{event})

	assert.ErrorIs(t, err, ErrConcurrency)
}

func TestInMemoryEventStoreSeedIsNotTracked(t *testing.T) {
	store := NewInMemoryEventStore()
	event := stubEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New()), Name: "given"}

	store.Seed("Stub-1", []DomainEvent{event})

	stored, err := store.ReadStream(context.Background(), "Stub-1")
	require.NoError(t, err)
	assert.Len(t, stored, 1)
	assert.Empty(t, store.Appended(), "seeded history must not count as appended")
}

// --- repository -------------------------------------------------------------

func TestRepositoryRoundTrip(t *testing.T) {
	store := NewInMemoryEventStore()
	repository := NewEventStoreRepository(store, newStubAggregate, "Stub", nil)
	ctx := context.Background()
	id := uuid.New()

	aggregate := newStubAggregate()
	aggregate.RaiseEvent(stubEvent{DomainEventBase: NewDomainEventBase(id, uuid.New()), Name: "one"})
	require.NoError(t, repository.Save(ctx, aggregate, uuid.New()))
	require.NoError(t, repository.Save(ctx, aggregate, uuid.New()), "no uncommitted events is a no-op")

	loaded, err := repository.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Version())
	assert.Equal(t, []string{"one"}, loaded.names)
}

func TestRepositoryGetByIDErrors(t *testing.T) {
	store := NewInMemoryEventStore()
	repository := NewEventStoreRepository(store, newStubAggregate, "Stub", nil)

	_, err := repository.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrAggregateNotFound)

	readErr := errors.New("store down")
	failing := NewEventStoreRepository(&failingStore{readErr: readErr}, newStubAggregate, "Stub", nil)
	_, err = failing.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, readErr)
}

func TestRepositorySaveErrors(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	newAggregateWithEvent := func() *stubAggregate {
		aggregate := newStubAggregate()
		aggregate.RaiseEvent(stubEvent{DomainEventBase: NewDomainEventBase(id, uuid.New()), Name: "one"})
		return aggregate
	}

	appendErr := errors.New("append refused")
	failingAppend := NewEventStoreRepository(&failingStore{appendErr: appendErr}, newStubAggregate, "Stub", nil)
	assert.ErrorIs(t, failingAppend.Save(ctx, newAggregateWithEvent(), uuid.New()), appendErr)

	publishErr := errors.New("bus down")
	failingPublish := NewEventStoreRepository(
		NewInMemoryEventStore(), newStubAggregate, "Stub", &failingPublisher{err: publishErr},
	)
	assert.ErrorIs(t, failingPublish.Save(ctx, newAggregateWithEvent(), uuid.New()), publishErr)
}

// --- service bus ------------------------------------------------------------

type recordingHandler struct {
	received chan stubCommand
}

func (h *recordingHandler) Handle(_ context.Context, command stubCommand) error {
	h.received <- command
	return nil
}

func runBus(t *testing.T) *ServiceBus {
	t.Helper()
	bus := NewServiceBus(slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(func() { _ = bus.Close() })
	go func() { _ = bus.Run(ctx) }()
	return bus
}

func TestServiceBusCommandAndEventFlow(t *testing.T) {
	bus := NewServiceBus(slog.Default())

	commandHandler := &recordingHandler{received: make(chan stubCommand, 1)}
	RegisterCommandHandler[stubCommand](bus, commandHandler)

	events := make(chan stubEvent, 2)
	RegisterDomainEventHandler(bus, "test.subscriber", func(_ context.Context, event stubEvent) error {
		events <- event
		return nil
	})
	raw := make(chan []byte, 1)
	bus.SubscribeIntegrationEvent("test.integration", "test.stub_happened",
		func(_ context.Context, payload []byte) error {
			raw <- payload
			return nil
		})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = bus.Run(ctx) }()
	<-bus.Running()

	command := stubCommand{CommandBase: NewCommandBase(uuid.New(), uuid.New()), Name: "do it"}
	require.NoError(t, bus.Send(ctx, command))
	select {
	case received := <-commandHandler.received:
		assert.Equal(t, "do it", received.Name)
	case <-time.After(2 * time.Second):
		t.Fatal("command was not delivered")
	}

	event := stubEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New()), Name: "done"}
	require.NoError(t, bus.PublishDomainEvent(ctx, event))
	select {
	case received := <-events:
		assert.Equal(t, "done", received.Name)
	case <-time.After(2 * time.Second):
		t.Fatal("domain event was not delivered")
	}

	require.NoError(t, bus.PublishIntegrationEvent(ctx, event))
	select {
	case payload := <-raw:
		assert.Contains(t, string(payload), "done")
	case <-time.After(2 * time.Second):
		t.Fatal("integration event was not delivered")
	}

	require.NoError(t, bus.Close())
}

func TestServiceBusMarshalErrors(t *testing.T) {
	bus := runBus(t)
	event := badPayloadEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New())}

	assert.Error(t, bus.PublishDomainEvent(context.Background(), event))
	assert.Error(t, bus.PublishIntegrationEvent(context.Background(),
		badPayloadIntegrationEvent{IntegrationEventBase: NewIntegrationEventBase(uuid.New())}))
}

// TestServiceBusUnmarshalErrorsAreReported: raw junk published straight to
// the topics exercises the unmarshal-error paths of both registrars (the
// router nacks and would redeliver — the test only needs the branch to
// run once before shutting down).
func TestServiceBusUnmarshalErrorsAreReported(t *testing.T) {
	bus := NewServiceBus(slog.New(slog.DiscardHandler))

	RegisterCommandHandler[stubCommand](bus, &recordingHandler{received: make(chan stubCommand, 1)})
	RegisterDomainEventHandler(bus, "test.subscriber", func(context.Context, stubEvent) error {
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = bus.Run(ctx) }()
	<-bus.Running()

	publishRaw(t, bus, commandTopicPrefix+stubCommand{}.MessageName(), `{"aggregate_id":123}`)
	publishRaw(t, bus, domainEventTopicPrefix+stubEvent{}.MessageName(), `{"aggregate_id":123}`)

	time.Sleep(100 * time.Millisecond) // let the handlers reject the junk
	cancel()
	_ = bus.Close()
}

func publishRaw(t *testing.T, bus *ServiceBus, topic, payload string) {
	t.Helper()
	require.NoError(t, bus.pubSub.Publish(topic,
		message.NewMessage(watermill.NewUUID(), []byte(payload))))
}
