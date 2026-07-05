package domain

import "time"

// Event is a domain event: a fact that already happened, named in the past
// tense. Events double as integration messages: EventName() is the topic
// they are published on, and consumers in other contexts unmarshal only the
// fields they care about (consumer-driven contracts).
type Event interface {
	// EventName returns the fully qualified topic, e.g. "catalog.beer_created".
	EventName() string
	// OccurredAt returns when the fact happened.
	OccurredAt() time.Time
}

// BaseEvent carries the metadata common to all events. Embed it in concrete
// event structs.
type BaseEvent struct {
	At time.Time `json:"occurred_at"`
}

func NewBaseEvent() BaseEvent {
	return BaseEvent{At: time.Now().UTC()}
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.At
}
