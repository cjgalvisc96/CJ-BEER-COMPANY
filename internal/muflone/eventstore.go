package muflone

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAggregateNotFound = errors.New("aggregate not found")
	// ErrConcurrency signals an optimistic-concurrency violation: someone
	// else appended to the stream since the aggregate was loaded.
	ErrConcurrency = errors.New("stream version conflict")
)

// StoredEvent is one event persisted in a stream, with the store-level
// metadata (commit headers in Muflone terms).
type StoredEvent struct {
	StreamID   string
	Version    int
	CommitID   uuid.UUID
	OccurredAt time.Time
	Event      DomainEvent
}

// EventStore is the source of truth of the system: append-only streams of
// domain events, one stream per aggregate instance.
type EventStore interface {
	ReadStream(ctx context.Context, streamID string) ([]StoredEvent, error)
	// Append adds events with an optimistic-concurrency check:
	// expectedVersion must equal the current stream length.
	Append(ctx context.Context, streamID string, expectedVersion int, commitID uuid.UUID, events []DomainEvent) error
}

// InMemoryEventStore keeps streams in memory. It doubles as the
// specification-test repository: Seed installs the Given events without
// tracking, and Appended returns everything the handler under test
// actually committed.
type InMemoryEventStore struct {
	mu       sync.RWMutex
	streams  map[string][]StoredEvent
	appended []DomainEvent
}

func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{streams: make(map[string][]StoredEvent)}
}

func (s *InMemoryEventStore) ReadStream(_ context.Context, streamID string) ([]StoredEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stream := s.streams[streamID]
	out := make([]StoredEvent, len(stream))
	copy(out, stream)
	return out, nil
}

func (s *InMemoryEventStore) Append(
	_ context.Context,
	streamID string,
	expectedVersion int,
	commitID uuid.UUID,
	events []DomainEvent,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream := s.streams[streamID]
	if len(stream) != expectedVersion {
		return fmt.Errorf("%w: stream %s is at version %d, expected %d",
			ErrConcurrency, streamID, len(stream), expectedVersion)
	}
	now := time.Now().UTC()
	for i, event := range events {
		stream = append(stream, StoredEvent{
			StreamID:   streamID,
			Version:    expectedVersion + i + 1,
			CommitID:   commitID,
			OccurredAt: now,
			Event:      event,
		})
		s.appended = append(s.appended, event)
	}
	s.streams[streamID] = stream
	return nil
}

// Seed installs prior history without recording it as appended — the
// equivalent of Muflone's Repository.ApplyGivenEvents used by
// specification tests.
func (s *InMemoryEventStore) Seed(streamID string, events []DomainEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stream := s.streams[streamID]
	for i, event := range events {
		stream = append(stream, StoredEvent{
			StreamID: streamID,
			Version:  len(stream) + i + 1,
			CommitID: event.CommitID(),
			Event:    event,
		})
	}
	s.streams[streamID] = stream
}

// Appended returns the events committed through Append, in order.
func (s *InMemoryEventStore) Appended() []DomainEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DomainEvent, len(s.appended))
	copy(out, s.appended)
	return out
}
