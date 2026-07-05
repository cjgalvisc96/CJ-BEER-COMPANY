package muflone

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CommandSpecification is the Go rendition of
// Muflone.SpecificationTests.CommandSpecification<TCommand>: it tests the
// entire lifecycle of an aggregate. Given brings the aggregate to a known
// state through past events, When issues the command, OnHandler supplies
// the command handler under test (built on the provided event store), and
// Expect declares the events that must have been committed.
type CommandSpecification[C Command] struct {
	// StreamName is the aggregate's stream prefix (e.g. "SalesOrder").
	StreamName string
	Given      func() []DomainEvent
	When       func() C
	OnHandler  func(store EventStore) CommandHandler[C]
	Expect     func() []DomainEvent
	// ExpectedError, when set, asserts the handler fails with this error
	// and that nothing was committed.
	ExpectedError error
}

// Run executes the specification: seed Given, handle When, compare the
// committed events with Expect — by type and value, exactly like the
// book's CompareEvents.
func (s CommandSpecification[C]) Run(t *testing.T) {
	t.Helper()

	store := NewInMemoryEventStore()
	command := s.When()
	if given := s.Given(); len(given) > 0 {
		store.Seed(s.StreamName+"-"+command.AggregateID().String(), given)
	}

	handler := s.OnHandler(store)
	err := handler.Handle(context.Background(), command)

	if s.ExpectedError != nil {
		require.Error(t, err)
		assert.True(t, errors.Is(err, s.ExpectedError),
			"expected error %v, got %v", s.ExpectedError, err)
		assert.Empty(t, store.Appended(), "a failed command must not commit events")
		return
	}
	require.NoError(t, err)

	expected := s.Expect()
	published := store.Appended()
	require.Len(t, published, len(expected), "different number of expected/published events")
	for i := range expected {
		assert.IsType(t, expected[i], published[i])
		assert.Equal(t, expected[i], published[i],
			"events at position %d differ", i)
	}
}
