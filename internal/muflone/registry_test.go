package muflone

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryDeserializesRegisteredEvents(t *testing.T) {
	registry := NewEventRegistry()
	RegisterEvent[stubEvent](registry)
	original := stubEvent{DomainEventBase: NewDomainEventBase(uuid.New(), uuid.New()), Name: "brewed"}
	payload, err := json.Marshal(original)
	require.NoError(t, err)

	event, err := registry.Deserialize(original.MessageName(), payload)

	require.NoError(t, err)
	assert.Equal(t, original, event)
}

func TestRegistryRejectsUnknownAndMalformed(t *testing.T) {
	registry := NewEventRegistry()
	RegisterEvent[stubEvent](registry)

	_, err := registry.Deserialize("test.never_registered", []byte(`{}`))
	assert.ErrorContains(t, err, "unknown event type")

	_, err = registry.Deserialize(stubEvent{}.MessageName(), []byte(`{"aggregate_id":123}`))
	assert.ErrorContains(t, err, "unmarshal event")
}

// TestRegistryUpcastsOldVersions models the book's Chapter 11 weak-schema
// evolution: a retired event name+shape is rewritten into the current one
// at read time, chaining through intermediate versions.
func TestRegistryUpcastsOldVersions(t *testing.T) {
	registry := NewEventRegistry()
	RegisterEvent[stubEvent](registry)

	// v1 stored {"label": ...}; today's stubEvent has {"name": ...}.
	registry.RegisterUpcaster("test.stub_happened.v1", func(payload []byte) (string, []byte, error) {
		var old struct {
			DomainEventBase
			Label string `json:"label"`
		}
		if err := json.Unmarshal(payload, &old); err != nil {
			return "", nil, err
		}
		upgraded, err := json.Marshal(stubEvent{DomainEventBase: old.DomainEventBase, Name: old.Label})
		return stubEvent{}.MessageName(), upgraded, err
	})

	aggregateId, commitId := uuid.New(), uuid.New()
	oldPayload := []byte(`{"aggregate_id":"` + aggregateId.String() +
		`","commit_id":"` + commitId.String() + `","label":"from-v1"}`)

	event, err := registry.Deserialize("test.stub_happened.v1", oldPayload)

	require.NoError(t, err)
	assert.Equal(t, stubEvent{
		DomainEventBase: NewDomainEventBase(aggregateId, commitId),
		Name:            "from-v1",
	}, event)
}

func TestRegistryUpcasterErrorsSurface(t *testing.T) {
	registry := NewEventRegistry()
	registry.RegisterUpcaster("test.broken.v1", func([]byte) (string, []byte, error) {
		return "", nil, assert.AnError
	})

	_, err := registry.Deserialize("test.broken.v1", []byte(`{}`))

	assert.ErrorIs(t, err, assert.AnError)
}
