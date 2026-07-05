package domain

// AggregateRoot collects the domain events an aggregate records while
// executing a command. The application layer pulls and publishes them after
// the aggregate has been persisted, so events are only emitted for state
// that was actually saved.
type AggregateRoot struct {
	events []Event
}

func (a *AggregateRoot) RecordEvent(event Event) {
	a.events = append(a.events, event)
}

// PullEvents returns the recorded events and clears the buffer.
func (a *AggregateRoot) PullEvents() []Event {
	events := a.events
	a.events = nil
	return events
}
