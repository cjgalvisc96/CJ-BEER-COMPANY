package muflone

import (
	"encoding/json"
	"fmt"
)

// EventRegistry maps stored event-type names back to concrete Go types so
// a durable event store can rehydrate streams. It also hosts upcasters —
// the book's Chapter 11 ("Dealing with Events and Their Evolution")
// weak-schema strategy: when an event's shape changes, an upcaster
// rewrites the OLD stored payload into the NEW shape at read time, so
// history never has to be migrated.
type EventRegistry struct {
	factories map[string]func(payload []byte) (DomainEvent, error)
	upcasters map[string]Upcaster
}

// Upcaster rewrites a stored payload (and possibly its type name) into the
// current version. Returning a different name chains into that name's
// upcaster, so multi-step evolutions compose.
type Upcaster func(payload []byte) (newName string, newPayload []byte, err error)

func NewEventRegistry() *EventRegistry {
	return &EventRegistry{
		factories: make(map[string]func(payload []byte) (DomainEvent, error)),
		upcasters: make(map[string]Upcaster),
	}
}

// RegisterEvent makes event type E rehydratable by its MessageName.
func RegisterEvent[E DomainEvent](registry *EventRegistry) {
	var prototype E
	registry.factories[prototype.MessageName()] = func(payload []byte) (DomainEvent, error) {
		var event E
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf("unmarshal event %s: %w", prototype.MessageName(), err)
		}
		return event, nil
	}
}

// RegisterUpcaster installs the evolution step for a (possibly retired)
// event-type name.
func (r *EventRegistry) RegisterUpcaster(eventName string, upcaster Upcaster) {
	r.upcasters[eventName] = upcaster
}

// Deserialize applies any upcaster chain, then rehydrates the event.
func (r *EventRegistry) Deserialize(eventName string, payload []byte) (DomainEvent, error) {
	for {
		upcaster, ok := r.upcasters[eventName]
		if !ok {
			break
		}
		newName, newPayload, err := upcaster(payload)
		if err != nil {
			return nil, fmt.Errorf("upcast event %s: %w", eventName, err)
		}
		eventName, payload = newName, newPayload
	}
	factory, ok := r.factories[eventName]
	if !ok {
		return nil, fmt.Errorf("unknown event type %q — is it registered?", eventName)
	}
	return factory(payload)
}
