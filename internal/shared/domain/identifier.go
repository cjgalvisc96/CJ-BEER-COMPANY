// Package domain holds the shared kernel: primitives every bounded context
// builds on (identifiers, aggregates, events, error kinds). It must not
// import from any context.
package domain

import (
	"github.com/google/uuid"
)

// EntityID is an opaque, immutable identifier for entities and aggregates.
// Contexts define their own named types on top of it (e.g. BeerID) so that
// identifiers from different aggregates cannot be mixed up by accident.
type EntityID struct {
	value uuid.UUID
}

func NewEntityID() EntityID {
	return EntityID{value: uuid.New()}
}

func ParseEntityID(raw string) (EntityID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return EntityID{}, NewValidationError("invalid identifier: " + raw)
	}
	return EntityID{value: parsed}, nil
}

func (id EntityID) String() string {
	return id.value.String()
}

func (id EntityID) IsZero() bool {
	return id.value == uuid.Nil
}
